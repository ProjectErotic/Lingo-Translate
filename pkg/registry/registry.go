package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/storage"
)

// ProjectEntry represents metadata of a registered NST translation project
type ProjectEntry struct {
	FilePath          string    `json:"file_path"`           // Absolute path to .nst workspace file
	DisplayName       string    `json:"display_name"`        // User-friendly project name
	EngineName        string    `json:"engine_name"`         // Engine type (rpgm, renpy, etc.)
	ProjectPath       string    `json:"project_path"`        // Original game root directory
	SourceLang        string    `json:"source_lang"`
	TargetLang        string    `json:"target_lang"`
	LastModified      time.Time `json:"last_modified"`
	TotalEntries      int       `json:"total_entries"`
	TranslatedEntries int       `json:"translated_entries"`
	TranslatedPercent float64   `json:"translated_percent"`
}

type Registry struct {
	filePath string
	mu       sync.RWMutex
	projects map[string]ProjectEntry // key: absolute FilePath
}

// New creates a Registry backed by the default user configuration directory
func New() (*Registry, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = "."
	}
	regPath := filepath.Join(cfgDir, "lingo", "projects.json")
	return NewWithFile(regPath)
}

// NewWithFile creates a Registry backed by a specific file path
func NewWithFile(path string) (*Registry, error) {
	r := &Registry{
		filePath: path,
		projects: make(map[string]ProjectEntry),
	}
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return r, nil
}

func (r *Registry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	var list []ProjectEntry
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	r.projects = make(map[string]ProjectEntry)
	for _, p := range list {
		r.projects[p.FilePath] = p
	}
	return nil
}

func (r *Registry) save() error {
	if dir := filepath.Dir(r.filePath); dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	list := make([]ProjectEntry, 0, len(r.projects))
	for _, p := range r.projects {
		list = append(list, p)
	}

	// Sort newest first
	sort.Slice(list, func(i, j int) bool {
		return list[i].LastModified.After(list[j].LastModified)
	})

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}

// Register adds or updates a project from its .nst workspace file
func (r *Registry) Register(workspacePath string) (*ProjectEntry, error) {
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, err
	}

	store, err := storage.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workspace: %w", err)
	}
	defer store.Close()

	proj, err := store.GetProject()
	if err != nil {
		return nil, fmt.Errorf("failed to get project metadata: %w", err)
	}
	if proj == nil {
		proj = &model.Project{
			Name:       filepath.Base(absPath),
			Engine:     "rpgm",
			SourcePath: filepath.Dir(absPath),
		}
	}

	entries, err := store.GetEntries("all")
	if err != nil {
		return nil, err
	}

	total := len(entries)
	translated := 0
	for _, e := range entries {
		if e.Target != "" && e.Status != model.StatusUntranslated {
			translated++
		}
	}

	pct := 0.0
	if total > 0 {
		pct = float64(translated) / float64(total) * 100
	}

	fi, err := os.Stat(absPath)
	lastMod := time.Now()
	if err == nil {
		lastMod = fi.ModTime()
	}

	entry := ProjectEntry{
		FilePath:          absPath,
		DisplayName:       proj.Name,
		EngineName:        proj.Engine,
		ProjectPath:       proj.SourcePath,
		SourceLang:        proj.SourceLang,
		TargetLang:        proj.TargetLang,
		LastModified:      lastMod,
		TotalEntries:      total,
		TranslatedEntries: translated,
		TranslatedPercent: pct,
	}

	r.mu.Lock()
	r.projects[absPath] = entry
	r.mu.Unlock()

	_ = r.save()
	return &entry, nil
}

// Remove removes a workspace from the registry (does NOT delete the file)
func (r *Registry) Remove(workspacePath string) error {
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return err
	}

	r.mu.Lock()
	delete(r.projects, absPath)
	r.mu.Unlock()

	return r.save()
}

// List returns all registered projects sorted by newest first
func (r *Registry) List() []ProjectEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]ProjectEntry, 0, len(r.projects))
	for _, p := range r.projects {
		list = append(list, p)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].LastModified.After(list[j].LastModified)
	})

	return list
}

// RefreshAll re-checks all known projects on disk and refreshes stats
func (r *Registry) RefreshAll() {
	r.mu.RLock()
	paths := make([]string, 0, len(r.projects))
	for p := range r.projects {
		paths = append(paths, p)
	}
	r.mu.RUnlock()

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			// Workspace file no longer exists, remove from registry
			_ = r.Remove(p)
			continue
		}
		_, _ = r.Register(p)
	}
}
