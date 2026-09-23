package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lingo-translate/frontend"
	"lingo-translate/pkg/app"
	"lingo-translate/pkg/model"
)

func init() {
	// Register custom events for strongly typed TS bindings
	application.RegisterEvent[model.TranslationProgress]("translation:progress")
	application.RegisterEvent[TranslationDonePayload]("translation:done")
}

func main() {
	session := NewSession()

	projectService := NewProjectService(session)
	entryService := NewEntryService(session)
	translationService := NewTranslationService(session)
	deployService := NewDeployService(session)
	settingsService := NewSettingsService()

	// Auto-open workspace or game folder passed via CLI arguments
	if len(os.Args) > 1 {
		var targetPath string
		var engine string
		for i := 1; i < len(os.Args); i++ {
			arg := os.Args[i]
			if arg == "-p" || arg == "--path" {
				if i+1 < len(os.Args) {
					targetPath = os.Args[i+1]
					i++
				}
			} else if arg == "-e" || arg == "--engine" {
				if i+1 < len(os.Args) {
					engine = os.Args[i+1]
					i++
				}
			} else if !strings.HasPrefix(arg, "-") && targetPath == "" {
				targetPath = arg
			}
		}

		if targetPath != "" {
			if fi, err := os.Stat(targetPath); err == nil {
				if fi.IsDir() {
					wsPath := filepath.Join(targetPath, "workspace.nst")
					resolvedWs := app.ResolveWorkspacePath("", targetPath)
					if _, err := os.Stat(wsPath); err == nil {
						_, _ = projectService.OpenWorkspace(wsPath)
					} else if _, err := os.Stat(resolvedWs); err == nil {
						_, _ = projectService.OpenWorkspace(resolvedWs)
					} else {
						var engines []string
						if engine != "" {
							engines = append(engines, engine)
						}
						_, _ = projectService.CreateFromGame(targetPath, wsPath, "Japanese", "Thai", engines...)
					}
				} else if strings.HasSuffix(targetPath, ".nst") {
					_, _ = projectService.OpenWorkspace(targetPath)
				}
			}
		}
	}

	app := application.New(application.Options{
		Name:        "lingo-desktop",
		Description: "Lingo Translate - Game Translation Suite",
		Services: []application.Service{
			application.NewService(projectService),
			application.NewService(entryService),
			application.NewService(translationService),
			application.NewService(deployService),
			application.NewService(settingsService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontend.Assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Lingo Translate - Game Translation Suite",
		Width:  1320,
		Height: 880,
		MinWidth: 1000,
		MinHeight: 650,
		BackgroundColour: application.NewRGB(26, 26, 26),
		URL: "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
