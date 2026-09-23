package main

import (
	"fmt"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lingo-translate/pkg/app"
	"lingo-translate/pkg/model"
	"lingo-translate/pkg/parser"
	"lingo-translate/pkg/registry"
)

type ProjectService struct {
	session      *Session
	registryPath string // optional custom path for unit testing
}

func NewProjectService(session *Session, customRegistryPath ...string) *ProjectService {
	regPath := ""
	if len(customRegistryPath) > 0 {
		regPath = customRegistryPath[0]
	}
	return &ProjectService{
		session:      session,
		registryPath: regPath,
	}
}

func (p *ProjectService) getRegistry() (*registry.Registry, error) {
	if p.registryPath != "" {
		return registry.NewWithFile(p.registryPath)
	}
	return registry.New()
}

// List returns all registered projects with up-to-date translation progress
func (p *ProjectService) List() ([]registry.ProjectEntry, error) {
	reg, err := p.getRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to open project registry: %w", err)
	}
	reg.RefreshAll()
	list := reg.List()
	if list == nil {
		list = []registry.ProjectEntry{}
	}
	return list, nil
}

// OpenWorkspace opens a .nst workspace file and attaches it to the current session
func (p *ProjectService) OpenWorkspace(path string) (*model.Project, error) {
	if path == "" {
		return nil, fmt.Errorf("workspace path cannot be empty")
	}
	ws, err := app.Open(path)
	if err != nil {
		return nil, err
	}

	p.session.SetWorkspace(ws)

	// Register / refresh in registry
	if reg, err := p.getRegistry(); err == nil {
		_, _ = reg.Register(path)
	}

	return ws.Project(), nil
}

// CreateFromGame extracts translatable strings and creates a new workspace (.nst)
func (p *ProjectService) CreateFromGame(gamePath, wsPath, srcLang, tgtLang string, engine ...string) (*model.ExtractionStats, error) {
	if gamePath == "" {
		return nil, fmt.Errorf("game directory is required")
	}
	if wsPath == "" {
		wsPath = filepath.Join(gamePath, "workspace.nst")
	}

	ws, stats, err := app.CreateFromGame(gamePath, wsPath, srcLang, tgtLang, engine...)
	if err != nil {
		return nil, err
	}

	p.session.SetWorkspace(ws)

	// Register in registry
	if reg, err := p.getRegistry(); err == nil {
		_, _ = reg.Register(wsPath)
	}

	return stats, nil
}

// Close closes the currently open workspace
func (p *ProjectService) Close() error {
	return p.session.Close()
}

// Current returns current open project metadata, or nil if none open
func (p *ProjectService) Current() *model.Project {
	ws, err := p.session.GetWorkspace()
	if err != nil || ws == nil {
		return nil
	}
	return ws.Project()
}

// RemoveFromRegistry removes a workspace entry from the registry (does NOT delete file)
func (p *ProjectService) RemoveFromRegistry(path string) error {
	reg, err := p.getRegistry()
	if err != nil {
		return err
	}
	return reg.Remove(path)
}

// DetectEngine checks game engine type for a directory
func (p *ProjectService) DetectEngine(dir string) (string, error) {
	parserInst, err := parser.DetectEngine(dir)
	if err != nil {
		return "", err
	}
	return parserInst.Name(), nil
}

// PickDirectory opens a native directory picker dialog
func (p *ProjectService) PickDirectory(title string) (string, error) {
	wailsApp := application.Get()
	if wailsApp == nil {
		return "", fmt.Errorf("Wails application not initialized")
	}

	dlg := wailsApp.Dialog.OpenFile()
	if title != "" {
		dlg.SetTitle(title)
	} else {
		dlg.SetTitle("Select Game Directory")
	}
	dlg.CanChooseDirectories(true)
	dlg.CanChooseFiles(false)
	return dlg.PromptForSingleSelection()
}

// PickWorkspace opens a native file picker dialog for .nst workspace
func (p *ProjectService) PickWorkspace(save bool) (string, error) {
	wailsApp := application.Get()
	if wailsApp == nil {
		return "", fmt.Errorf("Wails application not initialized")
	}

	if save {
		dlg := wailsApp.Dialog.SaveFile()
		dlg.SetMessage("Save Lingo Workspace File")
		dlg.AddFilter("Lingo Workspace (*.nst)", "*.nst")
		return dlg.PromptForSingleSelection()
	}

	dlg := wailsApp.Dialog.OpenFile()
	dlg.SetTitle("Open Lingo Workspace File")
	dlg.AddFilter("Lingo Workspace (*.nst)", "*.nst")
	dlg.CanChooseFiles(true)
	dlg.CanChooseDirectories(false)
	return dlg.PromptForSingleSelection()
}
