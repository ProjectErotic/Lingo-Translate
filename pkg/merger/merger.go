package merger

import (
	"context"
	"fmt"
	"time"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/parser"
	"lingo-translate/pkg/storage"
)

// MergeStats tracks the metrics of updating a project with a new game version
type MergeStats struct {
	TotalNew        int           `json:"total_new"`
	ExactMatches    int           `json:"exact_matches"`
	FuzzyMatches    int           `json:"fuzzy_matches"`
	NewUntranslated int           `json:"new_untranslated"`
	ObsoleteCount   int           `json:"obsolete_count"`
	Duration        time.Duration `json:"duration"`
}

type Merger struct{}

func New() *Merger {
	return &Merger{}
}

// MergeEntries reconciles existing translations with newly extracted entries from an updated game
func (m *Merger) MergeEntries(existing []model.TextEntry, incoming []model.TextEntry) ([]model.TextEntry, MergeStats) {
	startTime := time.Now()

	// 1. Build lookup tables from existing translations
	// Exact match: filePath + "|" + keyPath + "|" + source
	exactMap := make(map[string]model.TextEntry)
	// Fallback match: source text only (in case events shifted lines)
	sourceMap := make(map[string]model.TextEntry)
	existingIDs := make(map[string]bool)

	for _, e := range existing {
		existingIDs[e.ID] = true
		if e.Target != "" && e.Status != model.StatusUntranslated {
			exactKey := fmt.Sprintf("%s|%s|%s", e.FilePath, e.KeyPath, e.Source)
			exactMap[exactKey] = e
			if _, exists := sourceMap[e.Source]; !exists {
				sourceMap[e.Source] = e
			}
		}
	}

	merged := make([]model.TextEntry, len(incoming))
	exactCount := 0
	fuzzyCount := 0
	newCount := 0
	matchedExistingIDs := make(map[string]bool)

	// 2. Map translations onto incoming new entries
	for i, in := range incoming {
		entry := in
		exactKey := fmt.Sprintf("%s|%s|%s", in.FilePath, in.KeyPath, in.Source)

		if matched, ok := exactMap[exactKey]; ok {
			entry.Target = matched.Target
			entry.Status = matched.Status
			entry.Translator = matched.Translator
			entry.UpdatedAt = matched.UpdatedAt
			exactCount++
			matchedExistingIDs[matched.ID] = true
		} else if matched, ok := sourceMap[in.Source]; ok {
			entry.Target = matched.Target
			entry.Status = model.StatusReviewed // Needs quick review because location moved
			entry.Translator = matched.Translator + " (merged-fuzzy)"
			entry.UpdatedAt = time.Now()
			fuzzyCount++
			matchedExistingIDs[matched.ID] = true
		} else {
			entry.Target = ""
			entry.Status = model.StatusUntranslated
			entry.UpdatedAt = time.Now()
			newCount++
		}

		merged[i] = entry
	}

	// Calculate obsolete lines (lines that existed before but were removed in the new game update)
	obsoleteCount := len(existing) - len(matchedExistingIDs)
	if obsoleteCount < 0 {
		obsoleteCount = 0
	}

	stats := MergeStats{
		TotalNew:        len(incoming),
		ExactMatches:    exactCount,
		FuzzyMatches:    fuzzyCount,
		NewUntranslated: newCount,
		ObsoleteCount:   obsoleteCount,
		Duration:        time.Since(startTime),
	}

	return merged, stats
}

// UpdateWorkspaceWithNewGame extracts texts from newGameDir and merges them into the workspace .nst file
func (m *Merger) UpdateWorkspaceWithNewGame(ctx context.Context, p parser.EngineParser, newGameDir, workspacePath string) (*MergeStats, error) {
	store, err := storage.Open(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workspace: %w", err)
	}
	defer store.Close()

	// 1. Get existing translations
	existing, err := store.GetEntries("all")
	if err != nil {
		return nil, fmt.Errorf("failed to read existing entries: %w", err)
	}

	// 2. Extract from new game folder
	incoming, _, err := p.Extract(ctx, newGameDir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract new game texts: %w", err)
	}

	// 3. Merge
	merged, stats := m.MergeEntries(existing, incoming)

	// 4. Save merged entries back into workspace
	if err := store.SaveEntries(merged); err != nil {
		return nil, fmt.Errorf("failed to save merged entries: %w", err)
	}

	// Update project source path to new game folder
	if proj, _ := store.GetProject(); proj != nil {
		proj.SourcePath = newGameDir
		proj.UpdatedAt = time.Now()
		_ = store.SaveProject(proj)
	}

	return &stats, nil
}
