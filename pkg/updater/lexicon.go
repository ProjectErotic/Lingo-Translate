package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lingo-translate/pkg/translator/prompts"
)

// LexiconUpdateResult summarizes changes for a single language lexicon file
type LexiconUpdateResult struct {
	Language   string `json:"language"`
	FilePath   string `json:"file_path"`
	NewTerms   int    `json:"new_terms"`
	TotalTerms int    `json:"total_terms"`
	Created    bool   `json:"created"`
	Error      error  `json:"error,omitempty"`
}

// UpdateLexicons fetches the latest central knowledge base files from GitHub and merges them non-destructively
func UpdateLexicons(targetLangs ...string) ([]LexiconUpdateResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	lingoDir := filepath.Join(homeDir, ".lingo")
	if err := os.MkdirAll(lingoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", lingoDir, err)
	}

	langs := targetLangs
	if len(langs) == 0 {
		langs = []string{"th", "en", "zh"}
	}

	client := &http.Client{Timeout: 20 * time.Second}
	var results []LexiconUpdateResult

	for _, lang := range langs {
		code := prompts.NormalizeLangCode(lang)
		var filename string
		if code != "" && code != "th" {
			filename = fmt.Sprintf("nsfw_lexicon_%s.json", code)
		} else {
			filename = "nsfw_lexicon.json"
		}

		localPath := filepath.Join(lingoDir, filename)
		res := updateSingleLexicon(client, filename, localPath, lang)
		results = append(results, res)
	}

	return results, nil
}

func updateSingleLexicon(client *http.Client, filename, localPath, lang string) LexiconUpdateResult {
	res := LexiconUpdateResult{
		Language: lang,
		FilePath: localPath,
	}

	remoteData, err := fetchRemoteLexicon(client, filename)
	if err != nil {
		// Fallback to built-in default if offline/network fails and local file doesn't exist
		if _, statErr := os.Stat(localPath); os.IsNotExist(statErr) {
			def := prompts.DefaultNSFWLexicon(lang)
			if bytes, mErr := json.MarshalIndent(def, "", "  "); mErr == nil {
				_ = os.WriteFile(localPath, bytes, 0644)
				res.Created = true
				res.NewTerms = countTerms(def)
				res.TotalTerms = res.NewTerms
				return res
			}
		}
		res.Error = fmt.Errorf("failed to fetch central lexicon %s: %w", filename, err)
		return res
	}

	var remoteLex prompts.NSFWLexicon
	if err := json.Unmarshal(remoteData, &remoteLex); err != nil {
		res.Error = fmt.Errorf("invalid JSON in remote lexicon %s: %w", filename, err)
		return res
	}

	// If local file exists, merge non-destructively
	if localData, err := os.ReadFile(localPath); err == nil {
		var localLex prompts.NSFWLexicon
		if err := json.Unmarshal(localData, &localLex); err == nil {
			beforeCount := countTerms(&localLex)
			merged := mergeLexicons(&localLex, &remoteLex)
			afterCount := countTerms(merged)

			data, err := json.MarshalIndent(merged, "", "  ")
			if err != nil {
				res.Error = fmt.Errorf("failed to encode merged lexicon: %w", err)
				return res
			}
			if err := os.WriteFile(localPath, data, 0644); err != nil {
				res.Error = fmt.Errorf("failed to save merged lexicon: %w", err)
				return res
			}

			res.NewTerms = afterCount - beforeCount
			res.TotalTerms = afterCount
			return res
		}
	}

	// New local file
	if err := os.WriteFile(localPath, remoteData, 0644); err != nil {
		res.Error = fmt.Errorf("failed to write lexicon to %s: %w", localPath, err)
		return res
	}

	res.Created = true
	res.NewTerms = countTerms(&remoteLex)
	res.TotalTerms = res.NewTerms
	return res
}

func fetchRemoteLexicon(client *http.Client, filename string) ([]byte, error) {
	urls := []string{
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/templates/%s", repoOwner, repoName, filename),
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/templates/%s", repoOwner, repoName, filename),
	}

	var lastErr error
	for _, u := range urls {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "Lingo-CLI-Updater")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return io.ReadAll(resp.Body)
		}
		lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, u)
	}

	return nil, lastErr
}

func mergeLexicons(local, remote *prompts.NSFWLexicon) *prompts.NSFWLexicon {
	if local == nil {
		return remote
	}
	if remote == nil {
		return local
	}

	return &prompts.NSFWLexicon{
		Sensations: mergeSlices(local.Sensations, remote.Sensations),
		Actions:    mergeSlices(local.Actions, remote.Actions),
		Anatomy: prompts.AnatomyTerms{
			Male:   mergeSlices(local.Anatomy.Male, remote.Anatomy.Male),
			Female: mergeSlices(local.Anatomy.Female, remote.Anatomy.Female),
		},
		Fluids: mergeSlices(local.Fluids, remote.Fluids),
		Moans:  mergeSlices(local.Moans, remote.Moans),
		DirtyTalk: prompts.DirtyTalkTerms{
			Submissive: mergeSlices(local.DirtyTalk.Submissive, remote.DirtyTalk.Submissive),
			Dominant:   mergeSlices(local.DirtyTalk.Dominant, remote.DirtyTalk.Dominant),
		},
		Custom: mergeSlices(local.Custom, remote.Custom),
	}
}

func mergeSlices(base, incoming []string) []string {
	seen := make(map[string]bool)
	var res []string
	for _, s := range base {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			res = append(res, s)
		}
	}
	for _, s := range incoming {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			res = append(res, s)
		}
	}
	return res
}

func countTerms(l *prompts.NSFWLexicon) int {
	if l == nil {
		return 0
	}
	return len(l.Sensations) + len(l.Actions) +
		len(l.Anatomy.Male) + len(l.Anatomy.Female) +
		len(l.Fluids) + len(l.Moans) +
		len(l.DirtyTalk.Submissive) + len(l.DirtyTalk.Dominant) +
		len(l.Custom)
}
