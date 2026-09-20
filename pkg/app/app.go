package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	renpyInj "nst-go/pkg/injection/renpy"
	rpgmInj "nst-go/pkg/injection/rpgm"
	"nst-go/pkg/merger"
	"nst-go/pkg/model"
	"nst-go/pkg/parser"
	"nst-go/pkg/patch"
	"nst-go/pkg/pipeline"
	"nst-go/pkg/plugins/chanomhub"
	"nst-go/pkg/registry"
	"nst-go/pkg/storage"
	"nst-go/pkg/translator"
	"nst-go/pkg/translator/custom"
	"nst-go/pkg/translator/gemini"
	"nst-go/pkg/translator/google"
	"nst-go/pkg/translator/mock"
	"nst-go/pkg/translator/openai"
)

// ProviderConfig holds configuration for constructing a translator provider
type ProviderConfig struct {
	Name    string        `json:"name"` // "mock", "gemini", "openai", "google"
	APIKey  string        `json:"api_key"`
	Model   string        `json:"model"`
	BaseURL string        `json:"base_url"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

// TranslateOptions configures a batch translation run
type TranslateOptions struct {
	Provider    ProviderConfig `json:"provider"`
	SourceLang  string         `json:"source_lang"`
	TargetLang  string         `json:"target_lang"`
	BatchSize   int            `json:"batch_size"`
	Concurrency int            `json:"concurrency"`
	Scope       string         `json:"scope"` // "all", "untranslated", or file path
	Stream      bool           `json:"stream,omitempty"`
	Format      string         `json:"format,omitempty"` // "json" (default) or "line"
	Style       string         `json:"style,omitempty"`  // e.g. "standard", "nsfw", "vn_romance", "fantasy_rpg", etc.
	Prompt      string         `json:"prompt,omitempty"` // Custom system prompt instruction
}

// PublishOptions configures translation mod publishing to Chanomhub
type PublishOptions struct {
	Workspace  string `json:"workspace,omitempty"`
	PatchFile  string `json:"patch_file,omitempty"`
	GameDir    string `json:"game_dir,omitempty"`
	Slug       string `json:"slug,omitempty"`
	Token      string `json:"token"`
	Language   string `json:"language,omitempty"`
	CreditTo   string `json:"credit_to,omitempty"`
	APIBase    string `json:"api_base,omitempty"`
	StorageURL string `json:"storage_url,omitempty"`
}

// Workspace manages an open translation project workspace (.nst)
type Workspace struct {
	mu      sync.RWMutex
	path    string
	store   *storage.Storage
	project *model.Project
}

// GetNSTHomeDir returns ~/.nst directory path, ensuring it exists
func GetNSTHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	nstDir := filepath.Join(home, ".nst")
	_ = os.MkdirAll(nstDir, 0755)
	return nstDir
}

// ResolveWorkspacePath automatically resolves workspace paths to ~/.nst/<name>.nst
// if no explicit relative/absolute directory path is provided.
func ResolveWorkspacePath(wsPath string, gamePath ...string) string {
	nstDir := GetNSTHomeDir()
	trimmed := strings.TrimSpace(wsPath)

	// 1. If wsPath is empty or legacy default "workspace.nst", and gamePath is given:
	if (trimmed == "" || trimmed == "workspace.nst") && len(gamePath) > 0 && strings.TrimSpace(gamePath[0]) != "" {
		gameBase := filepath.Base(filepath.Clean(gamePath[0]))
		if gameBase != "" && gameBase != "." && gameBase != "/" {
			return filepath.Join(nstDir, gameBase+".nst")
		}
	}

	// 2. If wsPath contains directory separators (e.g. "./custom.nst", "/tmp/ws.nst", "subdir/ws.nst"):
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return trimmed
	}

	// 3. If wsPath is empty or legacy default and no gamePath given:
	if trimmed == "" || trimmed == "workspace.nst" {
		// If workspace.nst exists in current dir, keep backward compatibility
		if _, err := os.Stat("workspace.nst"); err == nil {
			return "workspace.nst"
		}
		return filepath.Join(nstDir, "workspace.nst")
	}

	// 4. Plain name provided (e.g. "idol", "idol.nst"):
	// Check if already exists in current working directory
	if fi, err := os.Stat(trimmed); err == nil && !fi.IsDir() {
		return trimmed
	}

	name := trimmed
	if !strings.HasSuffix(strings.ToLower(name), ".nst") {
		name += ".nst"
	}

	if fi, err := os.Stat(name); err == nil && !fi.IsDir() {
		return name
	}

	// Check if exists in ~/.nst/workspaces/
	wsSubdir := filepath.Join(nstDir, "workspaces", name)
	if fi, err := os.Stat(wsSubdir); err == nil && !fi.IsDir() {
		return wsSubdir
	}

	// Check if exists in ~/.nst/<name>
	nstFile := filepath.Join(nstDir, name)
	if fi, err := os.Stat(nstFile); err == nil && !fi.IsDir() {
		return nstFile
	}

	// Default destination for new workspaces: ~/.nst/<name>
	return nstFile
}

// Open opens an existing workspace file. It returns an error if the file does not exist.
func Open(wsPath string) (*Workspace, error) {
	wsPath = ResolveWorkspacePath(wsPath)
	if _, err := os.Stat(wsPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workspace file does not exist: %s", wsPath)
		}
		return nil, fmt.Errorf("failed to check workspace file: %w", err)
	}

	store, err := storage.Open(wsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workspace storage: %w", err)
	}

	proj, err := store.GetProject()
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to read project metadata: %w", err)
	}
	if proj == nil {
		store.Close()
		return nil, fmt.Errorf("workspace has no project metadata: %s", wsPath)
	}

	return &Workspace{
		path:    wsPath,
		store:   store,
		project: proj,
	}, nil
}

// CreateFromGame extracts texts from a game directory, creates a new workspace (.nst), and registers it
func CreateFromGame(gamePath, wsPath, srcLang, tgtLang string, engine ...string) (*Workspace, *model.ExtractionStats, error) {
	absGamePath, err := filepath.Abs(gamePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid game path: %w", err)
	}

	var engineParser parser.EngineParser
	if len(engine) > 0 && engine[0] != "" {
		engineParser, err = parser.GetParser(engine[0])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid engine '%s': %w", engine[0], err)
		}
	} else {
		engineParser, err = parser.DetectEngine(absGamePath)
		if err != nil {
			return nil, nil, fmt.Errorf("engine detection failed: %w", err)
		}
	}

	ctx := context.Background()
	entries, stats, err := engineParser.Extract(ctx, absGamePath)
	if err != nil {
		return nil, nil, fmt.Errorf("extraction failed: %w", err)
	}

	wsPath = ResolveWorkspacePath(wsPath, absGamePath)

	store, err := storage.Open(wsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create workspace storage: %w", err)
	}

	proj := &model.Project{
		ID:         filepath.Base(absGamePath),
		Name:       filepath.Base(absGamePath),
		Engine:     engineParser.Name(),
		SourcePath: absGamePath,
		SourceLang: srcLang,
		TargetLang: tgtLang,
		CreatedAt:  time.Now(),
	}

	if err := store.SaveProject(proj); err != nil {
		store.Close()
		return nil, nil, fmt.Errorf("failed to save project: %w", err)
	}

	if err := store.SaveEntries(entries); err != nil {
		store.Close()
		return nil, nil, fmt.Errorf("failed to save entries: %w", err)
	}

	// Register in project registry
	if reg, err := registry.New(); err == nil {
		_, _ = reg.Register(wsPath)
	}

	ws := &Workspace{
		path:    wsPath,
		store:   store,
		project: proj,
	}

	return ws, stats, nil
}

// Path returns the path of the workspace file
func (w *Workspace) Path() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.path
}

// Store returns the underlying storage
func (w *Workspace) Store() *storage.Storage {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.store
}

// GetPrimaryTranslator returns the primary model/translator used in the workspace
func (w *Workspace) GetPrimaryTranslator() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.store == nil {
		return ""
	}
	t, _ := w.store.GetPrimaryTranslator()
	return t
}

// Project returns a copy of current project metadata
func (w *Workspace) Project() *model.Project {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.project == nil {
		return nil
	}
	cp := *w.project
	return &cp
}

// Close closes the underlying workspace storage
func (w *Workspace) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.store != nil {
		return w.store.Close()
	}
	return nil
}

// Stats returns workspace metrics
func (w *Workspace) Stats() (storage.WorkspaceStats, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.store.Stats()
}

// SetMetadata saves or updates metadata key-value on the workspace
func (w *Workspace) SetMetadata(key, value string) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.store.SetMetadata(key, value)
}

// GetMetadata retrieves a metadata value by key
func (w *Workspace) GetMetadata(key string) (string, bool, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.store.GetMetadata(key)
}

// GetAllMetadata returns all workspace metadata
func (w *Workspace) GetAllMetadata() (map[string]string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.store.GetAllMetadata()
}

// ListFiles returns summary stats per file
func (w *Workspace) ListFiles() ([]storage.FileSummary, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.store.ListFiles()
}

// QueryEntries queries entries with filtering and pagination
func (w *Workspace) QueryEntries(q storage.EntryQuery) ([]model.TextEntry, int, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.store.QueryEntries(q)
}

// UpdateEntry updates a single entry target text
func (w *Workspace) UpdateEntry(id, target string, status model.TranslationStatus, translator string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if status == "" {
		if target == "" {
			status = model.StatusUntranslated
		} else {
			status = model.StatusTranslated
		}
	}
	if translator == "" {
		translator = "manual"
	}
	return w.store.UpdateEntryTarget(id, target, status, translator)
}

// Translate runs a translation batch pipeline
func (w *Workspace) Translate(ctx context.Context, opts TranslateOptions, progressCb func(model.TranslationProgress)) error {
	w.mu.RLock()
	store := w.store
	proj := w.project
	wsPath := w.path
	w.mu.RUnlock()

	trans, err := CreateTranslator(opts.Provider)
	if err != nil {
		return fmt.Errorf("failed to create translator: %w", err)
	}

	var query storage.EntryQuery
	switch opts.Scope {
	case "untranslated":
		query.Status = "untranslated"
	case "", "all":
		// all entries
	default:
		// Specific file
		query.File = opts.Scope
	}

	entries, _, err := store.QueryEntries(query)
	if err != nil {
		return fmt.Errorf("failed to query entries for translation: %w", err)
	}

	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}

	pipe := pipeline.New(store, trans, pipeline.Config{
		BatchSize:   batchSize,
		Concurrency: concurrency,
	})

	srcLang := opts.SourceLang
	if srcLang == "" && proj != nil {
		srcLang = proj.SourceLang
	}
	tgtLang := opts.TargetLang
	if tgtLang == "" && proj != nil {
		tgtLang = proj.TargetLang
	}

	tOpts := translator.Options{
		SourceLang: srcLang,
		TargetLang: tgtLang,
		Model:      opts.Provider.Model,
		Stream:     opts.Stream,
		Format:     opts.Format,
		Style:      opts.Style,
		Prompt:     opts.Prompt,
	}

	_, err = pipe.Run(ctx, entries, tOpts, progressCb)

	// Refresh in registry to update completion stats
	if reg, regErr := registry.New(); regErr == nil {
		_, _ = reg.Register(wsPath)
	}

	return err
}

// DeployLayer deploys non-destructive translation layer (RPGM & Ren'Py)
func (w *Workspace) DeployLayer(gamePath, langName string) error {
	w.mu.RLock()
	store := w.store
	proj := w.project
	w.mu.RUnlock()

	targetGamePath := gamePath
	if targetGamePath == "" && proj != nil {
		targetGamePath = proj.SourcePath
	}
	if targetGamePath == "" {
		return fmt.Errorf("game path is required for deploy")
	}

	engine := ""
	if proj != nil {
		engine = proj.Engine
	}

	entries, err := store.GetEntries("all")
	if err != nil {
		return fmt.Errorf("failed to read entries: %w", err)
	}

	if langName == "" && proj != nil {
		langName = proj.TargetLang
	}
	if langName == "" {
		langName = "Thai"
	}

	switch engine {
	case "renpy":
		exporter := renpyInj.NewExporter()
		return exporter.Deploy(targetGamePath, entries, renpyInj.DeployOptions{
			LanguageName:   langName,
			OnlyTranslated: true,
		})
	case "rpgm", "rpgm-mv", "rpgm-mz":
		exporter := rpgmInj.NewExporter()
		return exporter.Deploy(targetGamePath, entries, rpgmInj.DeployOptions{
			LanguageName:   langName,
			OnlyTranslated: true,
		})
	default:
		// Attempt auto-detection if engine string wasn't explicitly set
		if p, err := parser.DetectEngine(targetGamePath); err == nil && p != nil {
			if p.Name() == "renpy" {
				exporter := renpyInj.NewExporter()
				return exporter.Deploy(targetGamePath, entries, renpyInj.DeployOptions{
					LanguageName:   langName,
					OnlyTranslated: true,
				})
			} else if p.Name() == "rpgm" || p.Name() == "rpgm-mv" || p.Name() == "rpgm-mz" {
				exporter := rpgmInj.NewExporter()
				return exporter.Deploy(targetGamePath, entries, rpgmInj.DeployOptions{
					LanguageName:   langName,
					OnlyTranslated: true,
				})
			}
		}
		return fmt.Errorf("non-destructive additive layer is currently supported for RPG Maker (MV/MZ) and Ren'Py engines (engine: %s)", engine)
	}
}

// ExportCopy injects translations into a destination directory (copy mode)
func (w *Workspace) ExportCopy(ctx context.Context, gamePath, destPath string) error {
	w.mu.RLock()
	store := w.store
	proj := w.project
	w.mu.RUnlock()

	targetGamePath := gamePath
	if targetGamePath == "" && proj != nil {
		targetGamePath = proj.SourcePath
	}
	if targetGamePath == "" {
		return fmt.Errorf("source game path is required")
	}
	if destPath == "" {
		return fmt.Errorf("destination directory is required")
	}

	entries, err := store.GetEntries("all")
	if err != nil {
		return fmt.Errorf("failed to read entries: %w", err)
	}

	var engineParser parser.EngineParser
	if proj != nil && proj.Engine != "" {
		engineParser, _ = parser.GetParser(proj.Engine)
	}
	if engineParser == nil {
		engineParser, err = parser.DetectEngine(targetGamePath)
		if err != nil {
			return fmt.Errorf("engine detection failed: %w", err)
		}
	}

	return engineParser.Inject(ctx, targetGamePath, destPath, entries)
}

// MergeNewVersion merges existing translations with an updated version of the game
func (w *Workspace) MergeNewVersion(ctx context.Context, newGamePath string) (*merger.MergeStats, error) {
	w.mu.RLock()
	proj := w.project
	wsPath := w.path
	w.mu.RUnlock()

	var engineParser parser.EngineParser
	var err error
	if proj != nil && proj.Engine != "" {
		engineParser, _ = parser.GetParser(proj.Engine)
	}
	if engineParser == nil {
		engineParser, err = parser.DetectEngine(newGamePath)
		if err != nil {
			return nil, fmt.Errorf("failed to detect engine for new game: %w", err)
		}
	}

	m := merger.New()
	stats, err := m.UpdateWorkspaceWithNewGame(ctx, engineParser, newGamePath, wsPath)
	if err != nil {
		return stats, err
	}

	// Update registry
	if reg, regErr := registry.New(); regErr == nil {
		_, _ = reg.Register(wsPath)
	}

	return stats, nil
}

// ProviderInfo represents metadata about an available translation provider
type ProviderInfo struct {
	Name            string   `json:"name"`
	DisplayName     string   `json:"display_name"`
	Description     string   `json:"description,omitempty"`
	IsCustom        bool     `json:"is_custom"`
	BaseURL         string   `json:"base_url,omitempty"`
	DefaultModel    string   `json:"default_model,omitempty"`
	AvailableModels []string `json:"available_models,omitempty"`
}

// ListAvailableProviders returns all built-in and detected external custom providers
func ListAvailableProviders() []ProviderInfo {
	list := []ProviderInfo{
		{
			Name:        "mock",
			DisplayName: "Mock (Debug / Test)",
			Description: "Fast offline mock provider for testing without API usage",
			IsCustom:    false,
		},
		{
			Name:            "gemini",
			DisplayName:     "Google Gemini AI",
			Description:     "Official Google Gemini models (Gemini 2.5 Flash, Gemini 1.5 Pro)",
			IsCustom:        false,
			DefaultModel:    "gemini-2.5-flash",
			AvailableModels: []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-1.5-flash", "gemini-1.5-pro"},
		},
		{
			Name:            "openai",
			DisplayName:     "OpenAI / Compatible",
			Description:     "Standard OpenAI API or local LLM server (Ollama, LM Studio)",
			IsCustom:        false,
			BaseURL:         "https://api.openai.com/v1",
			DefaultModel:    "gpt-4o-mini",
			AvailableModels: []string{"gpt-4o-mini", "gpt-4o", "o3-mini", "gpt-3.5-turbo"},
		},
		{
			Name:        "google",
			DisplayName: "Google Translate API",
			Description: "Google Cloud Translation API v2",
			IsCustom:    false,
		},
	}

	if customList, err := custom.List(); err == nil {
		for _, c := range customList {
			list = append(list, ProviderInfo{
				Name:            c.Name,
				DisplayName:     c.DisplayName,
				Description:     c.Description,
				IsCustom:        true,
				BaseURL:         c.BaseURL,
				DefaultModel:    c.DefaultModel,
				AvailableModels: c.AvailableModels,
			})
		}
	}
	return list
}

// CreateTranslator constructs a translator implementation from ProviderConfig
func CreateTranslator(cfg ProviderConfig) (translator.Translator, error) {
	switch cfg.Name {
	case "mock", "":
		return mock.New("[TH] "), nil
	case "gemini":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key is required for Gemini provider")
		}
		return gemini.New(gemini.Config{
			APIKey: cfg.APIKey,
			Model:  cfg.Model,
		}), nil
	case "openai":
		return openai.New(openai.Config{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
			Timeout: cfg.Timeout,
		}), nil
	case "google":
		return google.New(google.Config{
			APIKey: cfg.APIKey,
		}), nil
	case "uchs":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key is required for UCHS provider (generate one via Chanomhub login)")
		}
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://ilms.uchs-th.com/v1"
		}
		model := cfg.Model
		if model == "" {
			model = "deepseek-v4.1-flash"
		}
		return openai.New(openai.Config{
			APIKey:  cfg.APIKey,
			BaseURL: baseURL,
			Model:   model,
			Timeout: cfg.Timeout,
		}), nil
	default:
		// Attempt to load external custom provider definition (e.g. from ~/.config/nst/providers/*.json or ./providers/*.json)
		if customDef, err := custom.Find(cfg.Name); err == nil {
			return custom.NewTranslator(*customDef, cfg.APIKey, cfg.Model, cfg.BaseURL)
		}
		return nil, fmt.Errorf("unsupported provider: %s (available built-in: mock, gemini, openai, google, or custom providers in providers/)", cfg.Name)
	}
}

// Publish uploads a translation mod to Chanomhub
func Publish(ctx context.Context, opts PublishOptions) (*chanomhub.PublishResult, error) {
	if opts.Token == "" {
		opts.Token = chanomhub.GetEffectiveToken()
	}
	if opts.Token == "" {
		return nil, fmt.Errorf("token is required to publish (run 'nst login' to authenticate)")
	}
	if opts.APIBase == "" {
		opts.APIBase = chanomhub.GetEffectiveAPIBase()
	}
	if opts.StorageURL == "" {
		opts.StorageURL = chanomhub.GetEffectiveStorageURL()
	}

	var patchFileToUpload string
	var tempPatchToDelete string
	engine := "rpgm"
	slug := opts.Slug
	lang := opts.Language
	credit := opts.CreditTo
	gameVersion := "1.0.0"
	sourceLang := ""
	translatorModel := ""
	var configData map[string]interface{}
	var statsData map[string]interface{}

	if opts.Workspace != "" {
		ws, err := Open(opts.Workspace)
		if err != nil {
			return nil, fmt.Errorf("failed to open workspace: %w", err)
		}
		defer ws.Close()

		meta, _ := ws.GetAllMetadata()
		if slug == "" {
			slug = meta["chanomhub_slug"]
		}
		if lang == "" {
			if tl, ok := meta["target_lang"]; ok && tl != "" {
				lang = tl
			} else if ws.project != nil {
				lang = ws.project.TargetLang
			}
		}
		if ws.project != nil {
			sourceLang = ws.project.SourceLang
		}
		translatorModel = ws.GetPrimaryTranslator()
		if gv, ok := meta["game_version"]; ok && gv != "" {
			gameVersion = gv
		}
		if ws.project != nil && ws.project.Engine != "" {
			engine = ws.project.Engine
		}

		// Export patch to temp file
		tempPatch := filepath.Join(os.TempDir(), fmt.Sprintf("chanomhub_patch_%d.patch.json.gz", time.Now().UnixMilli()))
		pkg, _, err := ws.ExportPatch(tempPatch)
		if err != nil {
			return nil, fmt.Errorf("failed to export patch for publishing: %w", err)
		}
		patchFileToUpload = tempPatch
		tempPatchToDelete = tempPatch

		pct := 0.0
		if pkg.Stats.TotalEntries > 0 {
			pct = (float64(pkg.Stats.TranslatedEntries) / float64(pkg.Stats.TotalEntries)) * 100.0
		}
		statsData = map[string]interface{}{
			"total_entries":      pkg.Stats.TotalEntries,
			"translated_entries": pkg.Stats.TranslatedEntries,
			"unique_texts":       pkg.Stats.UniqueTexts,
			"progress_percent":   pct,
		}

		configData = map[string]interface{}{
			"format":             "patch.json.gz",
			"game_version":       gameVersion,
			"total_entries":      pkg.Stats.TotalEntries,
			"translated_entries": pkg.Stats.TranslatedEntries,
			"unique_texts":       pkg.Stats.UniqueTexts,
		}
	} else if opts.PatchFile != "" {
		patchFileToUpload = opts.PatchFile
		pkg, err := patch.LoadPatch(opts.PatchFile)
		if err == nil {
			if slug == "" {
				slug = pkg.ChanomhubSlug
			}
			if lang == "" {
				lang = pkg.TargetLang
			}
			sourceLang = pkg.SourceLang
			if tm, ok := pkg.Metadata["translator_model"]; ok {
				translatorModel = tm
			}
			if pkg.Engine != "" {
				engine = pkg.Engine
			}
			if pkg.GameVersion != "" {
				gameVersion = pkg.GameVersion
			}
			pct := 0.0
			if pkg.Stats.TotalEntries > 0 {
				pct = (float64(pkg.Stats.TranslatedEntries) / float64(pkg.Stats.TotalEntries)) * 100.0
			}
			statsData = map[string]interface{}{
				"total_entries":      pkg.Stats.TotalEntries,
				"translated_entries": pkg.Stats.TranslatedEntries,
				"unique_texts":       pkg.Stats.UniqueTexts,
				"progress_percent":   pct,
			}
			configData = map[string]interface{}{
				"format":             "patch.json.gz",
				"game_version":       gameVersion,
				"total_entries":      pkg.Stats.TotalEntries,
				"translated_entries": pkg.Stats.TranslatedEntries,
				"unique_texts":       pkg.Stats.UniqueTexts,
			}
		}
	}

	if tempPatchToDelete != "" {
		defer os.Remove(tempPatchToDelete)
	}

	if slug == "" {
		return nil, fmt.Errorf("slug is required (specify -slug or set 'chanomhub_slug' in workspace metadata)")
	}
	if patchFileToUpload == "" && opts.GameDir == "" {
		return nil, fmt.Errorf("either -workspace, -patch, or -path (game directory) is required to publish")
	}

	// Customizable credit: public pen name / alias.
	// Defaults to "NST Translator". Sensitive account information is never sent;
	// the server verifies user identity safely via the Bearer token.
	if credit == "" || credit == "NST" {
		credit = "NST Translator"
	}

	if configData == nil {
		configData = make(map[string]interface{})
	}
	configData["credit_to"] = credit
	configData["translator_tool"] = "NST-V2"
	configData["ai_model"] = translatorModel
	configData["source_language"] = sourceLang
	configData["target_language"] = lang
	configData["translation_summary"] = fmt.Sprintf("แปลโดย: %s | โมเดล: %s | ภาษา: %s -> %s", credit, translatorModel, sourceLang, lang)

	client := chanomhub.NewClient(opts.APIBase, opts.StorageURL, opts.Token)
	return client.PublishTranslation(ctx, chanomhub.PublishRequest{
		PatchFile:       patchFileToUpload,
		GameDir:         opts.GameDir,
		Slug:            slug,
		Language:        lang,
		Engine:          engine,
		CreditTo:        credit,
		GameVersion:     gameVersion,
		TranslatorModel: translatorModel,
		SourceLanguage:  sourceLang,
		TargetLanguage:  lang,
		Stats:           statsData,
		Config:          configData,
	})
}

// ExportPatch exports translated entries into a lightweight distribution patch package (.patch.json.gz)
func (w *Workspace) ExportPatch(outputPath string) (*patch.PatchPackage, string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	proj := w.project
	if proj == nil {
		return nil, "", fmt.Errorf("project metadata not found in workspace")
	}

	meta, err := w.store.GetAllMetadata()
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	gameTitle := proj.Name
	if gt, ok := meta["game_title"]; ok && gt != "" {
		gameTitle = gt
	}
	gameVersion := "1.0.0"
	if gv, ok := meta["game_version"]; ok && gv != "" {
		gameVersion = gv
	}
	slug := ""
	if s, ok := meta["chanomhub_slug"]; ok {
		slug = s
	}

	entries, err := w.store.GetEntries("all")
	if err != nil {
		return nil, "", fmt.Errorf("failed to load entries: %w", err)
	}

	var patchEntries []patch.PatchEntry
	uniqueMap := make(map[string]bool)
	translatedCount := 0

	for _, e := range entries {
		if e.Target != "" && e.Target != e.Source && e.Status != model.StatusUntranslated {
			patchEntries = append(patchEntries, patch.PatchEntry{
				FilePath: e.FilePath,
				KeyPath:  e.KeyPath,
				Source:   e.Source,
				Target:   e.Target,
			})
			uniqueMap[e.Source] = true
			translatedCount++
		}
	}

	if outputPath == "" {
		baseName := strings.TrimSuffix(filepath.Base(w.path), filepath.Ext(w.path))
		if baseName == "" {
			baseName = "distribution"
		}
		outputPath = filepath.Join(filepath.Dir(w.path), baseName+".patch.json.gz")
	}

	pkg := &patch.PatchPackage{
		FormatVersion: "1.0",
		Engine:        proj.Engine,
		GameTitle:     gameTitle,
		GameVersion:   gameVersion,
		ChanomhubSlug: slug,
		SourceLang:    proj.SourceLang,
		TargetLang:    proj.TargetLang,
		CreatedAt:     time.Now().UTC(),
		Metadata:      meta,
		Stats: patch.PatchStats{
			TotalEntries:      len(entries),
			TranslatedEntries: translatedCount,
			UniqueTexts:       len(uniqueMap),
		},
		Entries: patchEntries,
	}

	if err := patch.SavePatch(pkg, outputPath); err != nil {
		return nil, "", err
	}

	return pkg, outputPath, nil
}

// ImportPatch merges a distribution patch into the workspace, populates TM cache, and restores translations
func (w *Workspace) ImportPatch(patchPath string) (*merger.MergeStats, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	pkg, err := patch.LoadPatch(patchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load patch: %w", err)
	}

	// 1. Populate TM cache from all patch entries
	srcLang := w.project.SourceLang
	if srcLang == "" {
		srcLang = pkg.SourceLang
	}
	tgtLang := w.project.TargetLang
	if tgtLang == "" {
		tgtLang = pkg.TargetLang
	}

	for _, pe := range pkg.Entries {
		if pe.Target != "" {
			_ = w.store.SetCache(pe.Source, pe.Target, srcLang, tgtLang, "patch_import")
		}
	}

	// 2. Fetch current workspace entries
	currentEntries, err := w.store.GetEntries("all")
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace entries: %w", err)
	}

	// Convert patch entries to model.TextEntry
	patchModelEntries := pkg.ToModelEntries()

	// 3. Merge entries using Merger
	m := merger.New()
	mergedEntries, stats := m.MergeEntries(patchModelEntries, currentEntries)

	// 4. Save merged entries to storage
	if err := w.store.UpdateEntriesTargetBatch(mergedEntries); err != nil {
		return nil, fmt.Errorf("failed to save merged entries: %w", err)
	}

	// 5. Populate workspace metadata if not already set
	if pkg.ChanomhubSlug != "" {
		if _, exists, _ := w.store.GetMetadata("chanomhub_slug"); !exists {
			_ = w.store.SetMetadata("chanomhub_slug", pkg.ChanomhubSlug)
		}
	}
	if pkg.GameTitle != "" {
		if _, exists, _ := w.store.GetMetadata("game_title"); !exists {
			_ = w.store.SetMetadata("game_title", pkg.GameTitle)
		}
	}
	if pkg.GameVersion != "" {
		if _, exists, _ := w.store.GetMetadata("game_version"); !exists {
			_ = w.store.SetMetadata("game_version", pkg.GameVersion)
		}
	}

	return &stats, nil
}

// ApplyPatch applies a distribution patch directly to a game folder without requiring a workspace
func ApplyPatch(ctx context.Context, patchPath, gameDir, outDir string) error {
	return patch.ApplyPatchToGame(ctx, patchPath, gameDir, outDir)
}

