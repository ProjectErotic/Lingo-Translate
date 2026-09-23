package rpgm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lingo-translate/pkg/model"
)

// Inject applies translated text entries back into the game files
func (p *Parser) Inject(ctx context.Context, sourceDir, outputDir string, entries []model.TextEntry) error {
	srcDataDir, err := p.FindDataDir(sourceDir)
	if err != nil {
		return err
	}

	targetDataDir := srcDataDir
	if outputDir != "" && outputDir != sourceDir {
		// Determine relative path of dataDir to sourceDir
		rel, err := filepath.Rel(sourceDir, srcDataDir)
		if err != nil {
			rel = "data"
		}
		targetDataDir = filepath.Join(outputDir, rel)
	}

	if err := os.MkdirAll(targetDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create target data dir: %w", err)
	}

	// Group entries by file
	entriesByFile := make(map[string][]model.TextEntry)
	for _, e := range entries {
		if e.Target != "" && e.Target != e.Source {
			entriesByFile[e.FilePath] = append(entriesByFile[e.FilePath], e)
		}
	}

	// Read and patch each file
	for fileName, fileEntries := range entriesByFile {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		srcPath := filepath.Join(srcDataDir, fileName)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read source file %s: %w", srcPath, err)
		}

		var root interface{}
		if err := json.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("failed to parse JSON %s: %w", srcPath, err)
		}

		for _, entry := range fileEntries {
			updateJSONValue(&root, entry.KeyPath, entry.Target)
		}

		outData, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize JSON for %s: %w", fileName, err)
		}

		targetPath := filepath.Join(targetDataDir, fileName)
		if err := os.WriteFile(targetPath, outData, 0644); err != nil {
			return fmt.Errorf("failed to write output file %s: %w", targetPath, err)
		}
	}

	return nil
}

// updateJSONValue updates a nested JSON tree by key path
func updateJSONValue(root *interface{}, keyPath, newVal string) bool {
	parts := strings.Split(keyPath, ".")
	return updateRecursive(root, parts, 0, newVal)
}

func updateRecursive(current *interface{}, parts []string, index int, newVal string) bool {
	if current == nil || index >= len(parts) {
		return false
	}

	part := parts[index]
	isLast := index == len(parts)-1

	// Handle array indexing like "events[1]" or "[1]"
	if strings.Contains(part, "[") {
		bracketPos := strings.Index(part, "[")
		baseKey := part[:bracketPos]
		indicesStr := part[bracketPos:]

		indicesMatches := arrayIndexRegex.FindAllStringSubmatch(indicesStr, -1)
		if len(indicesMatches) == 0 {
			return false
		}

		var targetNode *interface{} = current

		if baseKey != "" {
			m, ok := (*current).(map[string]interface{})
			if !ok {
				return false
			}
			val, exists := m[baseKey]
			if !exists {
				return false
			}
			temp := val
			targetNode = &temp
			defer func() {
				m[baseKey] = *targetNode
			}()
		}

		currArrayNode := targetNode
		for i, match := range indicesMatches {
			idx, err := strconv.Atoi(match[1])
			if err != nil {
				return false
			}

			arr, ok := (*currArrayNode).([]interface{})
			if !ok || idx >= len(arr) {
				return false
			}

			isLastIndex := (i == len(indicesMatches)-1) && isLast
			if isLastIndex {
				arr[idx] = newVal
				*currArrayNode = arr
				return true
			}

			if i < len(indicesMatches)-1 {
				subItem := arr[idx]
				currArrayNode = &subItem
				defer func(a []interface{}, idx int, sub *interface{}) {
					a[idx] = *sub
				}(arr, idx, currArrayNode)
			} else {
				// Move to next dot part
				subItem := arr[idx]
				res := updateRecursive(&subItem, parts, index+1, newVal)
				arr[idx] = subItem
				*currArrayNode = arr
				return res
			}
		}
		return false
	}

	// Normal Map key
	m, ok := (*current).(map[string]interface{})
	if !ok {
		return false
	}

	if isLast {
		m[part] = newVal
		return true
	}

	nextVal, exists := m[part]
	if !exists {
		return false
	}

	res := updateRecursive(&nextVal, parts, index+1, newVal)
	m[part] = nextVal
	return res
}
