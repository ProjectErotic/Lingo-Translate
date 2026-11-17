package engine

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	gmlSymbolRegex = regexp.MustCompile(`gml_(?:Script|Object|RoomCC)_[a-zA-Z0-9_@]+`)
	placeholderRegex = regexp.MustCompile(`\{(\d+)\}`)
)

// GameMakerInspector analyzes GameMaker game engines (both standard VM and compiled YYC).
type GameMakerInspector struct {
	GameDir     string
	DataWinPath string
	ExePath     string
}

// NewGameMakerInspector creates an inspector if data.win is found.
func NewGameMakerInspector(gameDir string) (*GameMakerInspector, error) {
	dataWin := filepath.Join(gameDir, "data.win")
	if _, err := os.Stat(dataWin); err != nil {
		return nil, fmt.Errorf("data.win not found in %s: %w", gameDir, err)
	}

	// Look for executable
	var exePath string
	entries, _ := os.ReadDir(gameDir)
	for _, entry := range entries {
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
			exePath = filepath.Join(gameDir, entry.Name())
			break
		}
	}

	return &GameMakerInspector{
		GameDir:     gameDir,
		DataWinPath: dataWin,
		ExePath:     exePath,
	}, nil
}

// DataWinInfo contains extracted metadata from data.win
type DataWinInfo struct {
	Chunks       []string
	TotalStrings int
	Templates    []string
}

// ParseDataWin parses chunks and strings from data.win
func (g *GameMakerInspector) ParseDataWin() (*DataWinInfo, error) {
	f, err := os.Open(g.DataWinPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var formMagic [4]byte
	if _, err := io.ReadFull(f, formMagic[:]); err != nil || string(formMagic[:]) != "FORM" {
		return nil, fmt.Errorf("invalid GameMaker FORM header")
	}

	var formSize uint32
	if err := binary.Read(f, binary.LittleEndian, &formSize); err != nil {
		return nil, err
	}

	chunks := make(map[string]int64)
	chunkOrder := []string{}
	offset := int64(8)

	for offset < int64(formSize) {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			break
		}
		var cName [4]byte
		if _, err := io.ReadFull(f, cName[:]); err != nil {
			break
		}
		var cLen uint32
		if err := binary.Read(f, binary.LittleEndian, &cLen); err != nil {
			break
		}

		nameStr := string(cName[:])
		chunks[nameStr] = offset + 8
		chunkOrder = append(chunkOrder, nameStr)
		offset += 8 + int64(cLen)
	}

	info := &DataWinInfo{
		Chunks: chunkOrder,
	}

	// Parse STRG chunk
	if strgOff, ok := chunks["STRG"]; ok {
		if _, err := f.Seek(strgOff, io.SeekStart); err == nil {
			var strCount uint32
			if err := binary.Read(f, binary.LittleEndian, &strCount); err == nil {
				info.TotalStrings = int(strCount)
				// Sample templates
				limit := strCount
				if limit > 2000 {
					limit = 2000
				}
				ptrTable := make([]uint32, limit)
				if err := binary.Read(f, binary.LittleEndian, &ptrTable); err == nil {
					for _, ptr := range ptrTable {
						if _, err := f.Seek(int64(ptr), io.SeekStart); err == nil {
							var sLen uint32
							if err := binary.Read(f, binary.LittleEndian, &sLen); err == nil && sLen < 512 {
								sBytes := make([]byte, sLen)
								if _, err := io.ReadFull(f, sBytes); err == nil {
									sStr := string(sBytes)
									if placeholderRegex.MatchString(sStr) {
										info.Templates = append(info.Templates, sStr)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return info, nil
}

// AnalyzeKey inspects the GameMaker engine and executable to find caller context for a key.
func (g *GameMakerInspector) AnalyzeKey(key string) (*KeyContext, error) {
	ctx := &KeyContext{
		Key:        key,
		EngineType: "GameMaker",
	}

	if g.ExePath == "" {
		return ctx, nil
	}

	exeData, err := os.ReadFile(g.ExePath)
	if err != nil {
		return ctx, err
	}

	// Check if key is a known stat/numeric fragment
	if strings.Contains(key, "act_check") || strings.Contains(key, "hornymeter") || strings.Contains(key, "stamina") {
		ctx.DynamicSuffix = "%"
		ctx.TrailingSpaceNeeded = true
	}

	// Search all variants of key in executable
	searchTargets := [][]byte{[]byte(key)}
	parts := strings.Split(key, "_")
	if len(parts) > 3 {
		// Strip trailing numeric indexes (e.g. _0_0 -> base key)
		trimCount := 0
		for i := len(parts) - 1; i >= 0; i-- {
			if _, err := strconv.Atoi(parts[i]); err == nil {
				trimCount++
			} else {
				break
			}
		}
		if trimCount > 0 {
			basePrefix := strings.Join(parts[:len(parts)-trimCount], "_")
			searchTargets = append(searchTargets, []byte(basePrefix))
		}
	}

	callerSet := make(map[string]bool)
	for _, target := range searchTargets {
		pos := 0
		for {
			hit := bytes.Index(exeData[pos:], target)
			if hit == -1 {
				break
			}
			absPos := pos + hit

			// Scan window 1500 bytes before and after for gml symbols
			start := absPos - 1500
			if start < 0 {
				start = 0
			}
			end := absPos + 1500
			if end > len(exeData) {
				end = len(exeData)
			}

			window := exeData[start:end]
			matches := gmlSymbolRegex.FindAll(window, -1)
			for _, m := range matches {
				callerSet[string(m)] = true
			}

			pos = absPos + len(target)
		}
	}

	for c := range callerSet {
		ctx.Callers = append(ctx.Callers, c)
	}

	// Detect placeholders like {0}, {1}
	if matches := placeholderRegex.FindAllString(key, -1); len(matches) > 0 {
		ctx.Placeholders = matches
	}

	return ctx, nil
}

// GenerateFullReport runs comprehensive engine inspection on the game directory.
func (g *GameMakerInspector) GenerateFullReport(sampleKeys []string) (*EngineReport, error) {
	dataInfo, err := g.ParseDataWin()
	if err != nil {
		return nil, fmt.Errorf("failed to parse data.win: %w", err)
	}

	report := &EngineReport{
		EngineType:   "GameMaker",
		TotalStrings: dataInfo.TotalStrings,
		KeyAnalyses:  make(map[string]*KeyContext),
	}

	// Check if YYC
	if g.ExePath != "" {
		peFile, err := pe.Open(g.ExePath)
		if err == nil {
			defer peFile.Close()
			for _, sec := range peFile.Sections {
				if sec.Name == ".text" {
					report.IsYYC = true
					break
				}
			}
		}

		// Count gml_Scripts in exe
		exeData, err := os.ReadFile(g.ExePath)
		if err == nil {
			allScripts := gmlSymbolRegex.FindAll(exeData, -1)
			uniqueScripts := make(map[string]bool)
			for _, s := range allScripts {
				uniqueScripts[string(s)] = true
			}
			report.TotalScripts = len(uniqueScripts)
		}
	}

	// Analyze sample keys
	for _, k := range sampleKeys {
		analysis, err := g.AnalyzeKey(k)
		if err == nil {
			analysis.SimulatedOutput = SimulateInGameRender(analysis, "ระดับความเงี่ยนของเธออยู่ที่ ")
			report.KeyAnalyses[k] = analysis
		}
	}
	report.TargetKeysTested = len(report.KeyAnalyses)

	return report, nil
}
