package rpgm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"lingo-translate/pkg/model"
)

// Event Codes for RPG Maker MV/MZ
const (
	CodeShowTextSetup         = 101
	CodeShowChoices           = 102
	CodeShowScrollingText     = 105
	CodeShowTextLine          = 401
	CodeShowScrollingTextLine = 405
	CodeChangeName            = 320
	CodeChangeNickname        = 324
)

var (
	urlRegex         = regexp.MustCompile(`(?i)^(https?|ftp|file)://`)
	fileExtRegex     = regexp.MustCompile(`(?i)\.(png|jpg|jpeg|gif|bmp|wav|ogg|m4a|mp3|json|js)$`)
	controlCodeRegex = regexp.MustCompile(`(?i)^\\[a-z]\[\d+\]$`)
	eventNameRegex   = regexp.MustCompile(`(?i)^EV\d{3,}$`)
	pluginCmdRegex   = regexp.MustCompile(`(?i)^[A-Z][a-zA-Z]+ (open|close|add|remove|set|get|show|hide|enable|disable)`)
	arrayIndexRegex  = regexp.MustCompile(`\[(\d+)\]`)
)

var whitelistedKeys = map[string]bool{
	"name": true, "description": true, "message1": true, "message2": true,
	"message3": true, "message4": true, "note": true, "nickname": true,
	"profile": true, "gameTitle": true, "currencyUnit": true, "terms": true,
	"basic": true, "commands": true, "params": true, "messages": true,
	"actionFailure": true, "actorDamage": true, "actorDrain": true, "actorGain": true,
	"actorLoss": true, "actorNoDamage": true, "actorNoHit": true, "actorRecovery": true,
	"alwaysDash": true, "commandRemember": true, "counterAttack": true, "criticalToActor": true,
	"criticalToEnemy": true, "defeat": true, "emerge": true, "enemyDamage": true,
	"enemyDrain": true, "enemyGain": true, "enemyLoss": true, "enemyNoDamage": true,
	"enemyNoHit": true, "enemyRecovery": true, "escapeFailure": true, "escapeStart": true,
	"evasion": true, "expNext": true, "expTotal": true, "levelUp": true,
	"loadMessage": true, "magicEvasion": true, "magicReflection": true, "obtainExp": true,
	"obtainGold": true, "obtainItem": true, "obtainSkill": true, "partyName": true,
	"possession": true, "preemptive": true, "saveMessage": true, "substitute": true,
	"surprise": true, "useItem": true, "victory": true, "title": true,
	"memo": true, "text": true, "caption": true, "label": true, "comment": true,
}

var blacklistedKeys = map[string]bool{
	"se": true, "bgm": true, "bgs": true, "me": true,
	"animation1Name": true, "animation2Name": true, "battlerName": true,
	"characterName": true, "faceName": true, "motion": true,
	"overlay1Name": true, "overlay2Name": true, "tileset": true,
	"parallaxName": true, "battleback1Name": true, "battleback2Name": true,
	"script": true, "url": true,
}

var systemPrefixes = []string{
	"img/", "audio/", "data/", "js/", "fonts/",
	"Actor", "Class", "Skill", "Item", "Weapon", "Armor", "Enemy", "Troop",
	"State", "Animation", "Tileset", "CommonEvent", "System", "MapInfo",
}

// Parser handles RPG Maker MV and MZ games
type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Name() string {
	return "rpgm"
}

// FindDataDir locates the game's data folder (e.g. data/ or www/data/)
func (p *Parser) FindDataDir(rootDir string) (string, error) {
	candidates := []string{
		filepath.Join(rootDir, "data"),
		filepath.Join(rootDir, "www", "data"),
	}

	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			systemPath := filepath.Join(c, "System.json")
			if _, err := os.Stat(systemPath); err == nil {
				return c, nil
			}
		}
	}
	return "", fmt.Errorf("could not find RPG Maker data folder with System.json in %s", rootDir)
}

func (p *Parser) Detect(dir string) bool {
	_, err := p.FindDataDir(dir)
	return err == nil
}

func isSystemString(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return true
	}
	if strings.Contains(text, "/") || urlRegex.MatchString(text) || fileExtRegex.MatchString(text) ||
		controlCodeRegex.MatchString(trimmed) || eventNameRegex.MatchString(text) ||
		pluginCmdRegex.MatchString(text) || strings.Contains(strings.ToLower(text), "$game") {
		return true
	}
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(strings.ToLower(text), strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

func isAudioObject(m map[string]interface{}) bool {
	_, hasName := m["name"]
	_, hasVol := m["volume"]
	_, hasPitch := m["pitch"]
	_, hasPan := m["pan"]
	return hasName && hasVol && hasPitch && hasPan
}

// Extract extracts all dialogue, choices, and database texts
func (p *Parser) Extract(ctx context.Context, dir string) ([]model.TextEntry, *model.ExtractionStats, error) {
	startTime := time.Now()
	dataDir, err := p.FindDataDir(dir)
	if err != nil {
		return nil, nil, err
	}

	files, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var entries []model.TextEntry
	filesScanned := 0
	uniqueSet := make(map[string]bool)

	for _, file := range files {
		lowerName := strings.ToLower(file.Name())
		if file.IsDir() || !strings.HasSuffix(lowerName, ".json") || lowerName == "tilesets.json" || lowerName == "animations.json" {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		fileName := file.Name()
		filePath := filepath.Join(dataDir, fileName)

		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var root interface{}
		if err := json.Unmarshal(data, &root); err != nil {
			continue
		}

		filesScanned++
		p.extractFromNode(root, &entries, fileName, "")
	}

	// 2. Discover encryption keys from JS plugins
	keyFinder := NewHeuristicKeyFinder()
	keys := keyFinder.DiscoverKeys(dir)

	// 3. Scan external/custom data files (CSV, TSV, RCSV, TXT, DAT)
	customFilesScanned := p.ScanCustomDataFiles(dir, dataDir, keys, &entries)
	filesScanned += customFilesScanned

	// 4. Harvest hardcoded UI strings from JS plugins
	p.HarvestPluginStrings(dir, &entries)

	totalWords := 0
	for _, entry := range entries {
		uniqueSet[entry.Source] = true
		totalWords += len(strings.Fields(entry.Source))
	}

	stats := &model.ExtractionStats{
		TotalEntries: len(entries),
		UniqueTexts:  len(uniqueSet),
		TotalWords:   totalWords,
		FilesScanned: filesScanned,
		Duration:     time.Since(startTime),
	}

	return entries, stats, nil
}

func (p *Parser) extractFromNode(node interface{}, entries *[]model.TextEntry, fileName, keyPath string) {
	switch v := node.(type) {
	case string:
		if !isSystemString(v) {
			*entries = append(*entries, model.TextEntry{
				ID:        fmt.Sprintf("%s:%s", fileName, keyPath),
				Source:    v,
				FilePath:  fileName,
				KeyPath:   keyPath,
				Status:    model.StatusUntranslated,
				UpdatedAt: time.Now(),
			})
		}

	case map[string]interface{}:
		if isAudioObject(v) {
			return
		}

		// Check for Event Command structure (code + parameters)
		if codeVal, hasCode := v["code"]; hasCode {
			if paramsVal, hasParams := v["parameters"]; hasParams {
				if codeNum, ok := toInt64(codeVal); ok {
					if params, ok := paramsVal.([]interface{}); ok {
						p.extractFromEventCommand(codeNum, params, entries, fileName, keyPath)
						return
					}
				}
			}
		}

		// General JSON Object scanning
		for k, val := range v {
			if blacklistedKeys[k] {
				continue
			}

			newPath := k
			if keyPath != "" {
				newPath = keyPath + "." + k
			}

			switch strVal := val.(type) {
			case string:
				if whitelistedKeys[k] && !isSystemString(strVal) {
					*entries = append(*entries, model.TextEntry{
						ID:        fmt.Sprintf("%s:%s", fileName, newPath),
						Source:    strVal,
						FilePath:  fileName,
						KeyPath:   newPath,
						Status:    model.StatusUntranslated,
						UpdatedAt: time.Now(),
					})
				}
			default:
				p.extractFromNode(val, entries, fileName, newPath)
			}
		}

	case []interface{}:
		for i, item := range v {
			itemPath := fmt.Sprintf("[%d]", i)
			if keyPath != "" {
				itemPath = fmt.Sprintf("%s[%d]", keyPath, i)
			}
			p.extractFromNode(item, entries, fileName, itemPath)
		}
	}
}

func (p *Parser) extractFromEventCommand(code int64, params []interface{}, entries *[]model.TextEntry, fileName, keyPath string) bool {
	makeParamPath := func(idx int) string {
		if keyPath == "" {
			return fmt.Sprintf("parameters[%d]", idx)
		}
		return fmt.Sprintf("%s.parameters[%d]", keyPath, idx)
	}

	switch code {
	case CodeShowTextSetup:
		// MZ character speaker name is at parameter index 4
		if len(params) > 4 {
			if name, ok := params[4].(string); ok && !isSystemString(name) {
				*entries = append(*entries, model.TextEntry{
					ID:        fmt.Sprintf("%s:%s", fileName, makeParamPath(4)),
					Source:    name,
					FilePath:  fileName,
					KeyPath:   makeParamPath(4),
					Status:    model.StatusUntranslated,
					Context:   "Speaker Name",
					UpdatedAt: time.Now(),
				})
			}
		}
		return true

	case CodeShowTextLine, CodeShowScrollingText, CodeShowScrollingTextLine:
		if len(params) > 0 {
			if text, ok := params[0].(string); ok && !isSystemString(text) {
				*entries = append(*entries, model.TextEntry{
					ID:        fmt.Sprintf("%s:%s", fileName, makeParamPath(0)),
					Source:    text,
					FilePath:  fileName,
					KeyPath:   makeParamPath(0),
					Status:    model.StatusUntranslated,
					UpdatedAt: time.Now(),
				})
			}
		}
		return true

	case CodeShowChoices:
		if len(params) > 0 {
			if choices, ok := params[0].([]interface{}); ok {
				for i, choice := range choices {
					if text, ok := choice.(string); ok && !isSystemString(text) {
						choicePath := fmt.Sprintf("parameters[0][%d]", i)
						if keyPath != "" {
							choicePath = fmt.Sprintf("%s.parameters[0][%d]", keyPath, i)
						}
						*entries = append(*entries, model.TextEntry{
							ID:        fmt.Sprintf("%s:%s", fileName, choicePath),
							Source:    text,
							FilePath:  fileName,
							KeyPath:   choicePath,
							Status:    model.StatusUntranslated,
							Context:   "Choice",
							UpdatedAt: time.Now(),
						})
					}
				}
			}
		}
		return true

	case CodeChangeName, CodeChangeNickname:
		if len(params) > 1 {
			if name, ok := params[1].(string); ok && !isSystemString(name) {
				*entries = append(*entries, model.TextEntry{
					ID:        fmt.Sprintf("%s:%s", fileName, makeParamPath(1)),
					Source:    name,
					FilePath:  fileName,
					KeyPath:   makeParamPath(1),
					Status:    model.StatusUntranslated,
					UpdatedAt: time.Now(),
				})
			}
		}
		return true
	}

	return false
}

func toInt64(val interface{}) (int64, bool) {
	switch v := val.(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n, true
		}
	}
	return 0, false
}
