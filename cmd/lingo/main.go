package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"lingo-translate/pkg/app"
	"lingo-translate/pkg/mcp"
	"lingo-translate/pkg/model"
	"lingo-translate/pkg/plugins/chanomhub"
	"lingo-translate/pkg/registry"
	"lingo-translate/pkg/storage"
	"lingo-translate/pkg/translator/prompts"
	appVersion "lingo-translate/pkg/version"
	"lingo-translate/pkg/updater"
	"lingo-translate/pkg/webui"

	"golang.org/x/term"
)

var version = ""

func init() {
	if version == "" {
		version = appVersion.FullVersion()
	}
}

func main() {
	// Enforce 64 MiB soft memory ceiling on Go runtime to prevent heap ballooning
	debug.SetMemoryLimit(64 * 1024 * 1024)

	if len(os.Args) < 2 {
		printUsage()
		fmt.Println("\nTip: Run 'lingo ui' for web dashboard, or 'lingo-desktop' for the GUI.")
		return
	}

	command := os.Args[1]

	// Backward compatibility: if invoked with flags directly (e.g. -e rpgm -p /path), route to handleExtract
	if strings.HasPrefix(command, "-") {
		switch command {
		case "-v", "--version", "-version":
			fmt.Printf("Lingo CLI %s\n", version)
			return
		case "-h", "--help", "-help":
			printUsage()
			return
		default:
			handleExtract(os.Args[1:])
			return
		}
	}

	switch command {
	case "app", "desktop", "gui":
		handleApp(os.Args[2:])
	case "ui", "web":
		handleWeb(os.Args[2:])
	case "extract":
		handleExtract(os.Args[2:])
	case "translate":
		handleTranslate(os.Args[2:])
	case "inject":
		handleInject(os.Args[2:])
	case "deploy":
		handleDeploy(os.Args[2:])
	case "merge":
		handleMerge(os.Args[2:])
	case "projects":
		handleProjects(os.Args[2:])
	case "providers":
		handleProviders(os.Args[2:])
	case "status":
		handleStatus(os.Args[2:])
	case "mcp":
		handleMCP(os.Args[2:])
	case "publish":
		handlePublish(os.Args[2:])
	case "login", "adduser":
		handleLogin(os.Args[2:])
	case "logout":
		handleLogout(os.Args[2:])
	case "whoami":
		handleWhoami(os.Args[2:])
	case "import-cache":
		handleImportCache(os.Args[2:])
	case "meta":
		handleMeta(os.Args[2:])
	case "export-patch":
		handleExportPatch(os.Args[2:])
	case "import-patch":
		handleImportPatch(os.Args[2:])
	case "apply-patch":
		handleApplyPatch(os.Args[2:])
	case "styles", "templates":
		handleStyles(os.Args[2:])
	case "update", "upgrade":
		handleUpdate(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("Lingo CLI %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Lingo Translate - Next-Gen Game Translation Suite

Usage:
  lingo <command> [arguments]

Commands:
  app          Launch the Lingo Desktop Window Application (standalone native window)
  ui           Launch the interactive Web Dashboard in browser (recommended)
  extract      Extract translatable texts from a game into a .nst workspace file
  translate    Translate extracted texts using AI or Translation APIs
  inject       Apply translated texts back into game files
  deploy       Export non-destructive runtime translation mod
  merge        Merge existing translations into an updated game version
  projects     List registered projects and translation progress
  providers    List built-in and external custom translation providers
  styles       List available translation styles and persona templates (standard, nsfw, etc.)
  status       Show translation statistics of a workspace
  publish      Compress and publish translation mod to Chanomhub
  login        Authenticate with Chanomhub registry (npm login style)
  logout       Log out from Chanomhub and clear saved credentials
  whoami       Display currently logged-in Chanomhub user
  meta         Manage workspace metadata (Chanomhub slug, game version, tags)
  export-patch Export translated workspace to ultra-compact distribution patch (.patch.json.gz)
  import-patch Re-hydrate/merge distribution patch into workspace and TM cache (0 API cost)
  apply-patch  Directly install distribution patch onto a game folder without workspace
  update       Check and update Lingo CLI to the latest release
  mcp          Start the Model Context Protocol (MCP) server over stdio
  version      Show version info

Examples:
  lingo ui
  lingo providers
  lingo login
  lingo whoami
  lingo extract -path ./MyGame -workspace ./project.nst
  lingo translate -workspace ./project.nst -provider gpt -source Japanese -target Thai
  lingo inject -path ./MyGame -workspace ./project.nst -dest ./MyGame_Translated
  lingo deploy -path ./MyGame -workspace ./project.nst
  lingo publish -path ./MyGame -slug my-game-slug
  lingo mcp`)
}

func handleExtract(args []string) {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	var gamePath string
	var wsPath string
	var srcLang string
	var tgtLang string
	var engineFlag string

	fs.StringVar(&gamePath, "path", "", "Path to game root directory (required)")
	fs.StringVar(&gamePath, "p", "", "Path to game root directory (shorthand)")
	fs.StringVar(&wsPath, "workspace", "", "Output .nst workspace file (defaults to ~/.lingo/<game_name>.nst)")
	fs.StringVar(&wsPath, "output", "", "Output .nst workspace file (alias)")
	fs.StringVar(&wsPath, "o", "", "Output .nst workspace file (shorthand)")
	fs.StringVar(&srcLang, "source-lang", "Japanese", "Source language")
	fs.StringVar(&srcLang, "s", "Japanese", "Source language (shorthand)")
	fs.StringVar(&tgtLang, "target-lang", "Thai", "Target language")
	fs.StringVar(&tgtLang, "t", "Thai", "Target language (shorthand)")
	fs.StringVar(&engineFlag, "engine", "", "Game engine (optional: rpgm, renpy, godot, unity)")
	fs.StringVar(&engineFlag, "e", "", "Game engine (shorthand)")
	fs.Parse(args)

	if gamePath == "" {
		fmt.Println("Error: -path (or -p) is required")
		fs.Usage()
		os.Exit(1)
	}

	resolvedWs := app.ResolveWorkspacePath(wsPath, gamePath)
	ws, stats, err := app.CreateFromGame(gamePath, resolvedWs, srcLang, tgtLang, engineFlag)
	if err != nil {
		fmt.Printf("Extraction failed: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	fmt.Println("------------------------------------------")
	fmt.Printf("✅ Extracted successfully in %v\n", stats.Duration)
	fmt.Printf("   Files Scanned : %d\n", stats.FilesScanned)
	fmt.Printf("   Total Entries : %d\n", stats.TotalEntries)
	fmt.Printf("   Unique Texts  : %d\n", stats.UniqueTexts)
	fmt.Printf("   Total Words   : %d\n", stats.TotalWords)
	fmt.Printf("   Workspace File: %s\n", ws.Path())
	fmt.Println("------------------------------------------")
}

func handleTranslate(args []string) {
	fs := flag.NewFlagSet("translate", flag.ExitOnError)
	wsPath := fs.String("workspace", "", "Path to .nst workspace file (or game name in ~/.lingo/)")
	providerName := fs.String("provider", "mock", "Provider: mock, gemini, openai, google, or custom (e.g. gpt)")
	envKey := os.Getenv("LINGO_API_KEY")
	if envKey == "" {
		envKey = os.Getenv("NST_API_KEY") // legacy fallback
	}
	apiKey := fs.String("api-key", envKey, "API Key (or env LINGO_API_KEY)")
	modelName := fs.String("model", "", "Model name (e.g. gemini-2.5-flash, gpt-4o-mini, deepseek-v4-pro-0813)")
	baseURL := fs.String("base-url", "", "Custom Base URL for OpenAI/Ollama")
	srcLang := fs.String("source", "Japanese", "Source language")
	tgtLang := fs.String("target", "Thai", "Target language")
	styleName := fs.String("style", "standard", "Translation style/persona: standard, nsfw, vn_romance, fantasy_rpg, comedy, dan_uncensored, or path to custom template file")
	customPrompt := fs.String("prompt", "", "Custom system prompt instruction override")
	batchSize := fs.Int("batch-size", 10, "Batch size")
	concurrency := fs.Int("concurrency", 4, "Number of concurrent workers")
	streamMode := fs.Bool("stream", false, "Use streaming line-by-line mode (SSE)")
	formatMode := fs.String("format", "json", "Translation format: 'json' (default) or 'line' ([ID] ||| [Text])")
	timeout := fs.Duration("timeout", 60*time.Second, "API request timeout (e.g. 60s, 300s)")
	megaBatch := fs.Bool("mega-batch", false, "Enable mega-batch streaming preset (stream=true, format=line, timeout=300s, batch-size=300)")
	fs.Parse(args)

	if *megaBatch {
		if !*streamMode {
			*streamMode = true
		}
		if *formatMode == "json" {
			*formatMode = "line"
		}
		if *batchSize == 10 {
			*batchSize = 300
		}
		if *timeout == 60*time.Second {
			*timeout = 300 * time.Second
		}
		if *concurrency == 4 {
			*concurrency = 1
		}
	}

	ws, err := app.Open(*wsPath)
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	fmt.Println("🚀 Starting Translation Pipeline...")
	fmt.Printf("   Provider:    %s\n", strings.ToUpper(*providerName))
	fmt.Printf("   Language:    %s -> %s\n", *srcLang, *tgtLang)
	fmt.Printf("   Style:       %s\n", strings.ToUpper(*styleName))
	if *customPrompt != "" {
		fmt.Printf("   Prompt:      %s\n", *customPrompt)
	}
	fmt.Printf("   Batch Size:  %d\n", *batchSize)
	fmt.Printf("   Concurrency: %d workers\n", *concurrency)
	if *streamMode || *megaBatch {
		fmt.Printf("   Mode:        STREAMING (Format: %s, Timeout: %v)\n", *formatMode, *timeout)
	}
	fmt.Println("------------------------------------------")

	startTime := time.Now()
	ctx := context.Background()

	err = ws.Translate(ctx, app.TranslateOptions{
		Provider: app.ProviderConfig{
			Name:    *providerName,
			APIKey:  *apiKey,
			Model:   *modelName,
			BaseURL: *baseURL,
			Timeout: *timeout,
		},
		SourceLang:  *srcLang,
		TargetLang:  *tgtLang,
		Style:       *styleName,
		Prompt:      *customPrompt,
		BatchSize:   *batchSize,
		Concurrency: *concurrency,
		Scope:       "all",
		Stream:      *streamMode,
		Format:      *formatMode,
	}, func(p model.TranslationProgress) {
		fmt.Printf("\r⏳ Progress: %5.1f%% (%d/%d) | Current: %-25s",
			p.Percent, p.Completed, p.Total, p.CurrentFile)
	})

	if err != nil {
		fmt.Printf("\nTranslation failed: %v\n", err)
		os.Exit(1)
	}

	stats, _ := ws.Stats()
	fmt.Printf("\n------------------------------------------\n")
	fmt.Printf("🎉 Translation finished in %v\n", time.Since(startTime).Round(time.Millisecond))
	fmt.Printf("   Total:      %d\n", stats.Total)
	fmt.Printf("   Translated: %d\n", stats.Translated)
	fmt.Printf("   Pending:    %d\n", stats.Pending)
	fmt.Println("------------------------------------------")
}

func handleInject(args []string) {
	fs := flag.NewFlagSet("inject", flag.ExitOnError)
	gamePath := fs.String("path", "", "Path to game root directory (required)")
	wsPath := fs.String("workspace", "", "Path to .nst workspace file (defaults to ~/.lingo/<game_name>.nst)")
	destPath := fs.String("dest", "", "Destination path for patched game (required)")
	fs.Parse(args)

	if *gamePath == "" || *destPath == "" {
		fmt.Println("Error: -path and -dest are required")
		fs.Usage()
		os.Exit(1)
	}

	ws, err := app.Open(app.ResolveWorkspacePath(*wsPath, *gamePath))
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	ctx := context.Background()
	if err := ws.ExportCopy(ctx, *gamePath, *destPath); err != nil {
		fmt.Printf("Injection failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("------------------------------------------")
	fmt.Println("✅ Injection completed successfully!")
	fmt.Printf("   Patched game written to: %s\n", *destPath)
	fmt.Println("------------------------------------------")
}

func handleDeploy(args []string) {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	gamePath := fs.String("path", "", "Path to game root directory (required)")
	wsPath := fs.String("workspace", "", "Path to .nst workspace file (defaults to ~/.lingo/<game_name>.nst)")
	langName := fs.String("lang", "Thai", "Display name of translated language")
	fs.Parse(args)

	if *gamePath == "" {
		fmt.Println("Error: -path is required")
		fs.Usage()
		os.Exit(1)
	}

	ws, err := app.Open(app.ResolveWorkspacePath(*wsPath, *gamePath))
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	fmt.Printf("🚀 Deploying non-destructive translation layer to %s ...\n", *gamePath)
	if err := ws.DeployLayer(*gamePath, *langName); err != nil {
		fmt.Printf("Deploy failed: %v\n", err)
		os.Exit(1)
	}

	engine := ""
	if ws.Project() != nil {
		engine = ws.Project().Engine
	}

	fmt.Println("------------------------------------------")
	fmt.Println("✅ Deployment completed successfully!")
	fmt.Printf("   Target: %s\n", *gamePath)
	if engine == "renpy" {
		fmt.Printf("   Layer : %s/game/tl/%s/ (00_lingo_font_layer.rpy, script.rpy, screens.rpy)\n", *gamePath, *langName)
		fmt.Printf("   Fonts : %s/game/fonts/ (Zero-tofu embedded fonts)\n", *gamePath)
	} else {
		fmt.Printf("   Layer : %s/js/plugins/Lingo_TranslationLayer.js\n", *gamePath)
		fmt.Printf("   Files : %s/lingo_translations/\n", *gamePath)
	}
	fmt.Println("------------------------------------------")
}

func handleMerge(args []string) {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	newGamePath := fs.String("new-game", "", "Path to new/updated game folder (required)")
	wsPath := fs.String("workspace", "", "Path to existing .nst workspace (defaults to ~/.lingo/<game_name>.nst)")
	fs.Parse(args)

	if *newGamePath == "" {
		fmt.Println("Error: -new-game is required")
		fs.Usage()
		os.Exit(1)
	}

	resolvedWs := app.ResolveWorkspacePath(*wsPath, *newGamePath)
	ws, err := app.Open(resolvedWs)
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	ctx := context.Background()
	stats, err := ws.MergeNewVersion(ctx, *newGamePath)
	if err != nil {
		fmt.Printf("Merge failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("------------------------------------------")
	fmt.Printf("✅ Version Merge Complete in %v\n", stats.Duration)
	fmt.Printf("   Total New Texts : %d\n", stats.TotalNew)
	fmt.Printf("   Exact Preserved : %d\n", stats.ExactMatches)
	fmt.Printf("   Fuzzy Preserved : %d\n", stats.FuzzyMatches)
	fmt.Printf("   New To Translate: %d\n", stats.NewUntranslated)
	fmt.Printf("   Obsolete Lines  : %d\n", stats.ObsoleteCount)
	fmt.Println("------------------------------------------")
}

func handleStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	wsPath := fs.String("workspace", "", "Path to .nst workspace file (or game name in ~/.lingo/)")
	fs.Parse(args)

	resolvedWs := app.ResolveWorkspacePath(*wsPath)
	ws, err := app.Open(resolvedWs)
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	proj := ws.Project()
	stats, err := ws.Stats()
	if err != nil {
		fmt.Printf("Failed to read workspace stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("------------------------------------------")
	if proj != nil {
		fmt.Printf("Project:    %s\n", proj.Name)
		fmt.Printf("Engine:     %s\n", strings.ToUpper(proj.Engine))
		fmt.Printf("Source Path:%s\n", proj.SourcePath)
		fmt.Printf("Languages:  %s -> %s\n", proj.SourceLang, proj.TargetLang)
	}
	fmt.Printf("Total:      %d\n", stats.Total)
	fmt.Printf("Translated: %d (%.1f%%)\n", stats.Translated, stats.Percent)
	fmt.Printf("Pending:    %d\n", stats.Pending)
	fmt.Println("------------------------------------------")
}

func handleProjects(args []string) {
	reg, err := registry.New()
	if err != nil {
		fmt.Printf("Failed to open project registry: %v\n", err)
		os.Exit(1)
	}
	reg.RefreshAll()
	list := reg.List()

	fmt.Println("================================================================================")
	fmt.Println("📋 Registered Translation Projects")
	fmt.Println("================================================================================")
	if len(list) == 0 {
		fmt.Println("No registered projects found. Extract or load a project to register it.")
		fmt.Println("================================================================================")
		return
	}

	for i, p := range list {
		fmt.Printf("[%d] %s (%s) - Progress: %.1f%% (%d/%d)\n",
			i+1, p.DisplayName, p.EngineName, p.TranslatedPercent, p.TranslatedEntries, p.TotalEntries)
		fmt.Printf("    Workspace: %s\n", p.FilePath)
		fmt.Printf("    Game Dir : %s\n", p.ProjectPath)
		fmt.Printf("    Languages: %s -> %s | Modified: %s\n", p.SourceLang, p.TargetLang, p.LastModified.Format("2006-01-02 15:04"))
		fmt.Println("--------------------------------------------------------------------------------")
	}
}

func handleProviders(args []string) {
	providers := app.ListAvailableProviders()

	fmt.Println("================================================================================")
	fmt.Println("Lingo Translation Providers (Built-in & Custom External Plugins)")
	fmt.Println("================================================================================")

	for _, p := range providers {
		tag := "[Built-in]"
		if p.IsCustom {
			tag = "[Custom]  "
		}
		modelInfo := ""
		if p.DefaultModel != "" {
			modelInfo = fmt.Sprintf(" (Default Model: %s)", p.DefaultModel)
		}
		fmt.Printf("%s %-12s : %s%s\n", tag, p.Name, p.DisplayName, modelInfo)
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("💡 To add a new external provider securely without git exposure:")
	fmt.Println("   Place a JSON config in ~/.lingo/providers/<name>.json (recommended) or ~/.config/lingo/providers/")
	fmt.Println("================================================================================")
}

func handleApp(args []string) {
	// 1. Prefer native Wails desktop application if lingo-desktop binary exists
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)

	candidates := []string{
		filepath.Join(execDir, "lingo-desktop"),
		filepath.Join(".", "bin", "lingo-desktop"),
		filepath.Join(".", "lingo-desktop"),
	}
	if p, err := exec.LookPath("lingo-desktop"); err == nil {
		candidates = append([]string{p}, candidates...)
	}

	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
			fmt.Printf("🚀 Launching Native Lingo Desktop GUI (%s)...\n", c)
			cmd := exec.Command(c, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err == nil {
				return
			}
		}
	}

	// 2. Fallback to embedded web server desktop window
	fs := flag.NewFlagSet("app", flag.ExitOnError)
	port := fs.Int("port", 18080, "App server port")
	fs.Parse(args)

	srv := webui.New(*port, webui.ModeDesktop)
	if err := srv.Start(); err != nil {
		fmt.Printf("Application window error: %v\n", err)
		os.Exit(1)
	}
}

func handleWeb(args []string) {
	fs := flag.NewFlagSet("ui", flag.ExitOnError)
	port := fs.Int("port", 18080, "Web server port")
	mode := fs.String("mode", webui.ModeBrowser, "UI mode: 'browser' (web browser tab), 'desktop' (native standalone window), or 'none'")
	open := fs.Bool("open", true, "Open UI automatically")
	fs.Parse(args)

	launchMode := *mode
	if !*open {
		launchMode = webui.ModeHeadless
	}

	srv := webui.New(*port, launchMode)
	if err := srv.Start(); err != nil {
		fmt.Printf("Web server error: %v\n", err)
		os.Exit(1)
	}
}

func handleMCP(args []string) {
	server := mcp.NewServer()
	ctx := context.Background()
	if err := server.ServeStdio(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server stopped: %v\n", err)
	}
}

func handlePublish(args []string) {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	wsPath := fs.String("workspace", "", "Path to .nst workspace file")
	patchPath := fs.String("patch", "", "Path to .patch.json.gz distribution package")
	gamePath := fs.String("path", "", "Legacy path to game folder containing lingo_translations/")
	slug := fs.String("slug", "", "Chanomhub game article slug (optional if stored in workspace)")
	token := fs.String("token", "", "Chanomhub API token (defaults to CHANOMHUB_TOKEN or saved login)")
	lang := fs.String("lang", "", "Target language name")
	credit := fs.String("credit", "Lingo Translate", "Credit to translator/group")
	apiBase := fs.String("api-base", "", "Custom API base URL (defaults to saved registry)")
	storageURL := fs.String("storage-url", "", "Custom storage URL (defaults to saved storage URL)")
	fs.Parse(args)

	if *wsPath == "" && *patchPath == "" && *gamePath == "" {
		fmt.Println("Error: -workspace, -patch, or -path is required")
		fs.Usage()
		os.Exit(1)
	}

	effectiveToken := strings.TrimSpace(*token)
	if effectiveToken == "" {
		effectiveToken = chanomhub.GetEffectiveToken()
	}
	if effectiveToken == "" {
		fmt.Println("Error: Authentication required to publish translation mods.")
		fmt.Println("Please run 'lingo login' to authenticate, or provide -token / CHANOMHUB_TOKEN.")
		os.Exit(1)
	}

	effectiveAPIBase := *apiBase
	if effectiveAPIBase == "" {
		effectiveAPIBase = chanomhub.GetEffectiveAPIBase()
	}

	effectiveStorageURL := *storageURL
	if effectiveStorageURL == "" {
		effectiveStorageURL = chanomhub.GetEffectiveStorageURL()
	}

	fmt.Printf("📦 Publishing translation mod to Chanomhub (%s)...\n", effectiveAPIBase)
	if uInfo, err := chanomhub.ParseTokenUserInfo(effectiveToken); err == nil && uInfo != nil {
		if uInfo.Username != "" {
			fmt.Printf("   User:        %s (from Token)\n", uInfo.Username)
		}
	}

	ctx := context.Background()
	res, err := app.Publish(ctx, app.PublishOptions{
		Workspace:  *wsPath,
		PatchFile:  *patchPath,
		GameDir:    *gamePath,
		Slug:       *slug,
		Token:      effectiveToken,
		Language:   *lang,
		CreditTo:   *credit,
		APIBase:    effectiveAPIBase,
		StorageURL: effectiveStorageURL,
	})
	if err != nil {
		fmt.Printf("Publish failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ %s\n", res.Message)
	fmt.Printf("   Download URL: %s\n", res.DownloadURL)
	fmt.Printf("   Archive Size: %d bytes (%.2f KB)\n", res.FileSizeBytes, float64(res.FileSizeBytes)/1024.0)
}

func handleLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	apiBase := fs.String("registry", "", "Chanomhub registry / API base URL")
	fs.StringVar(apiBase, "api-base", "", "Chanomhub registry / API base URL (alias)")
	storageURL := fs.String("storage-url", "", "Custom storage URL")
	token := fs.String("token", "", "Direct token authentication (bypasses interactive login)")
	fs.StringVar(token, "t", "", "Direct token authentication (shorthand)")
	username := fs.String("username", "", "Username or email")
	fs.StringVar(username, "u", "", "Username or email (shorthand)")
	password := fs.String("password", "", "Password")
	fs.StringVar(password, "p", "", "Password (shorthand)")
	fs.Parse(args)

	targetRegistry := *apiBase
	if targetRegistry == "" {
		targetRegistry = chanomhub.GetEffectiveAPIBase()
	}

	// 1. Direct token login
	if *token != "" {
		fmt.Printf("🔐 Chanomhub Login (%s)\n\n", targetRegistry)
		ctx := context.Background()
		res, err := chanomhub.Login(ctx, chanomhub.LoginRequest{
			Token:      *token,
			APIBase:    *apiBase,
			StorageURL: *storageURL,
		})
		if err != nil {
			fmt.Printf("❌ Login failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Logged in as %s to %s\n", res.UserInfo.Username, res.APIBase)
		return
	}

	// 2. Direct username & password login if provided via flags
	if *username != "" && *password != "" {
		fmt.Printf("🔐 Chanomhub Login (%s)\n\n", targetRegistry)
		ctx := context.Background()
		res, err := chanomhub.Login(ctx, chanomhub.LoginRequest{
			UsernameOrEmail: *username,
			Password:        *password,
			APIBase:         *apiBase,
			StorageURL:      *storageURL,
		})
		if err != nil {
			fmt.Printf("❌ Login failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Logged in as %s to %s\n", res.UserInfo.Username, res.APIBase)
		return
	}

	// 3. Interactive email + password prompt, or guide user to generate token
	fmt.Printf("🔐 Chanomhub Login (%s)\n\n", targetRegistry)
	fmt.Println("Choose login method:")
	fmt.Println("  1) Email + Password")
	fmt.Println("  2) Paste API token (for Google/social login users)")
	fmt.Print("\nChoice [1/2]: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "2" || choice == "" {
		webBase := chanomhub.GetEffectiveWebURL()
		fmt.Printf("\n👉 Go to %s/member/dashboard/tokens\n", webBase)
		fmt.Println("   → Create a new token (select duration: forever)")
		fmt.Println("   → Copy the token and paste it below")
		fmt.Println()
		fmt.Print("Token: ")
		tokenInput, _ := reader.ReadString('\n')
		tokenInput = strings.TrimSpace(tokenInput)
		if tokenInput == "" {
			fmt.Println("❌ Token is required")
			os.Exit(1)
		}
		ctx := context.Background()
		res, err := chanomhub.Login(ctx, chanomhub.LoginRequest{
			Token:      tokenInput,
			APIBase:    targetRegistry,
			StorageURL: *storageURL,
		})
		if err != nil {
			fmt.Printf("❌ Login failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Logged in as %s to %s\n", res.UserInfo.Username, res.APIBase)
		return
	}

	// Email + password
	fmt.Print("\nEmail or username: ")
	emailInput, _ := reader.ReadString('\n')
	emailInput = strings.TrimSpace(emailInput)
	if emailInput == "" {
		fmt.Println("❌ Email/username is required")
		os.Exit(1)
	}

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		line, _ := reader.ReadString('\n')
		passwordBytes = []byte(line)
	}
	fmt.Println()
	passwordInput := strings.TrimSpace(string(passwordBytes))
	if passwordInput == "" {
		fmt.Println("❌ Password is required")
		os.Exit(1)
	}

	ctx := context.Background()
	res, err := chanomhub.Login(ctx, chanomhub.LoginRequest{
		UsernameOrEmail: emailInput,
		Password:        passwordInput,
		APIBase:         targetRegistry,
		StorageURL:      *storageURL,
	})
	if err != nil {
		fmt.Printf("❌ Login failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Logged in as %s to %s\n", res.UserInfo.Username, res.APIBase)
}

func handleLogout(args []string) {
	if err := chanomhub.ClearConfig(); err != nil {
		fmt.Printf("Failed to clear credentials: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("👋 Successfully logged out from Chanomhub.")
}

func handleWhoami(args []string) {
	ctx := context.Background()
	info, apiBase, err := chanomhub.Whoami(ctx)
	if err != nil {
		fmt.Printf("Not logged in to Chanomhub. Run 'lingo login' to authenticate.\n")
		os.Exit(1)
	}

	if info.Email != "" && info.Email != info.Username {
		fmt.Printf("Logged in as %s (%s) on %s\n", info.Username, info.Email, apiBase)
	} else {
		fmt.Printf("Logged in as %s on %s\n", info.Username, apiBase)
	}
}

func handleImportCache(args []string) {
	fs := flag.NewFlagSet("import-cache", flag.ExitOnError)
	wsPath := fs.String("workspace", "", "Path to .nst workspace file (required)")
	cachePath := fs.String("cache", "", "Path to JSON cache file (required)")
	srcLang := fs.String("source-lang", "", "Source language (optional)")
	tgtLang := fs.String("target-lang", "", "Target language (optional)")
	fs.Parse(args)

	if *wsPath == "" || *cachePath == "" {
		fmt.Println("Error: -workspace and -cache are required")
		fs.Usage()
		os.Exit(1)
	}

	resolvedWS := app.ResolveWorkspacePath(*wsPath)
	store, err := storage.Open(resolvedWS)
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	proj, err := store.GetProject()
	if err == nil && proj != nil {
		if *srcLang == "" {
			*srcLang = proj.SourceLang
		}
		if *tgtLang == "" {
			*tgtLang = proj.TargetLang
		}
	}
	if *srcLang == "" {
		*srcLang = "Korean"
	}
	if *tgtLang == "" {
		*tgtLang = "Thai"
	}

	data, err := os.ReadFile(*cachePath)
	if err != nil {
		fmt.Printf("Failed to read cache file: %v\n", err)
		os.Exit(1)
	}

	var cacheMap map[string]string
	if err := json.Unmarshal(data, &cacheMap); err != nil {
		fmt.Printf("Failed to parse cache JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📥 Importing %d translations from %s into %s ...\n", len(cacheMap), *cachePath, *wsPath)
	importedTM := 0
	for src, tgt := range cacheMap {
		if strings.TrimSpace(tgt) != "" {
			_ = store.SetCache(src, tgt, *srcLang, *tgtLang, "imported_cache")
			importedTM++
		}
	}

	entries, err := store.GetEntries("untranslated")
	if err != nil {
		fmt.Printf("Failed to read entries: %v\n", err)
		os.Exit(1)
	}

	updatedEntries := 0
	for _, e := range entries {
		if tgt, ok := cacheMap[e.Source]; ok && strings.TrimSpace(tgt) != "" {
			_ = store.UpdateEntryTarget(e.ID, tgt, model.StatusTranslated, "cache_import")
			updatedEntries++
		}
	}

	fmt.Println("------------------------------------------")
	fmt.Printf("✅ Cache import complete!\n")
	fmt.Printf("   TM Cache entries populated  : %d\n", importedTM)
	fmt.Printf("   Workspace entries translated: %d\n", updatedEntries)
	fmt.Println("------------------------------------------")
}

type metaSliceFlags []string

func (s *metaSliceFlags) String() string {
	return strings.Join(*s, ", ")
}

func (s *metaSliceFlags) Set(val string) error {
	*s = append(*s, val)
	return nil
}

func handleMeta(args []string) {
	fs := flag.NewFlagSet("meta", flag.ExitOnError)
	wsPath := fs.String("workspace", "", "Path to .nst workspace file (required)")
	var setFlags metaSliceFlags
	fs.Var(&setFlags, "set", "Set metadata key=value (can be used multiple times)")
	getKey := fs.String("get", "", "Get value of a specific metadata key")
	listAll := fs.Bool("list", false, "List all metadata key-values")
	fs.Parse(args)

	if *wsPath == "" {
		fmt.Println("Error: -workspace is required")
		fs.Usage()
		os.Exit(1)
	}

	ws, err := app.Open(app.ResolveWorkspacePath(*wsPath))
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	if len(setFlags) > 0 {
		for _, pair := range setFlags {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				fmt.Printf("Warning: ignoring invalid metadata pair (expected key=value): %s\n", pair)
				continue
			}
			k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if err := ws.SetMetadata(k, v); err != nil {
				fmt.Printf("Failed to set metadata %s: %v\n", k, err)
			} else {
				fmt.Printf("✓ Set metadata: %s = %s\n", k, v)
			}
		}
	}

	if *getKey != "" {
		val, found, err := ws.GetMetadata(*getKey)
		if err != nil {
			fmt.Printf("Failed to get metadata: %v\n", err)
		} else if !found {
			fmt.Printf("Metadata key '%s' not found\n", *getKey)
		} else {
			fmt.Printf("%s = %s\n", *getKey, val)
		}
	}

	if *listAll || (len(setFlags) == 0 && *getKey == "") {
		meta, err := ws.GetAllMetadata()
		if err != nil {
			fmt.Printf("Failed to list metadata: %v\n", err)
			return
		}
		fmt.Println("------------------------------------------")
		fmt.Printf("Workspace Metadata (%s):\n", *wsPath)
		if len(meta) == 0 {
			fmt.Println("  (No metadata set)")
		} else {
			for k, v := range meta {
				fmt.Printf("  %-22s: %s\n", k, v)
			}
		}
		fmt.Println("------------------------------------------")
	}
}

func handleExportPatch(args []string) {
	fs := flag.NewFlagSet("export-patch", flag.ExitOnError)
	wsPath := fs.String("workspace", "", "Path to .nst workspace file (required)")
	outputPath := fs.String("output", "", "Output path for patch file (default: <workspace>.patch.json.gz)")
	fs.Parse(args)

	if *wsPath == "" {
		fmt.Println("Error: -workspace is required")
		fs.Usage()
		os.Exit(1)
	}

	ws, err := app.Open(app.ResolveWorkspacePath(*wsPath))
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	fmt.Printf("📦 Exporting distribution patch from %s ...\n", app.ResolveWorkspacePath(*wsPath))
	pkg, actualPath, err := ws.ExportPatch(*outputPath)
	if err != nil {
		fmt.Printf("Export failed: %v\n", err)
		os.Exit(1)
	}

	fi, err := os.Stat(actualPath)
	var sizeBytes int64
	if err == nil {
		sizeBytes = fi.Size()
	}

	fmt.Println("------------------------------------------")
	fmt.Printf("✅ Distribution patch exported successfully!\n")
	fmt.Printf("   Output File      : %s\n", actualPath)
	fmt.Printf("   File Size        : %d bytes (%.2f KB)\n", sizeBytes, float64(sizeBytes)/1024.0)
	fmt.Printf("   Engine           : %s\n", pkg.Engine)
	fmt.Printf("   Game Title       : %s\n", pkg.GameTitle)
	fmt.Printf("   Game Version     : %s\n", pkg.GameVersion)
	fmt.Printf("   Chanomhub Slug   : %s\n", pkg.ChanomhubSlug)
	fmt.Printf("   Translated Count : %d / %d\n", pkg.Stats.TranslatedEntries, pkg.Stats.TotalEntries)
	fmt.Printf("   Unique Texts     : %d\n", pkg.Stats.UniqueTexts)
	fmt.Println("------------------------------------------")
}

func handleImportPatch(args []string) {
	fs := flag.NewFlagSet("import-patch", flag.ExitOnError)
	wsPath := fs.String("workspace", "", "Path to target .nst workspace file (required)")
	patchPath := fs.String("patch", "", "Path to .patch.json or .patch.json.gz file (required)")
	fs.Parse(args)

	if *wsPath == "" || *patchPath == "" {
		fmt.Println("Error: -workspace and -patch are required")
		fs.Usage()
		os.Exit(1)
	}

	resolvedWS := app.ResolveWorkspacePath(*wsPath)
	ws, err := app.Open(resolvedWS)
	if err != nil {
		fmt.Printf("Failed to open workspace: %v\n", err)
		os.Exit(1)
	}
	defer ws.Close()

	fmt.Printf("🔄 Importing & re-hydrating patch [%s] into [%s] ...\n", *patchPath, resolvedWS)
	stats, err := ws.ImportPatch(*patchPath)
	if err != nil {
		fmt.Printf("Import failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("------------------------------------------")
	fmt.Printf("✅ Distribution patch imported & TM cache populated!\n")
	fmt.Printf("   Exact Key Matches   : %d (100%% reused, $0.00)\n", stats.ExactMatches)
	fmt.Printf("   Fuzzy TM Matches    : %d (re-aligned strings, $0.00)\n", stats.FuzzyMatches)
	fmt.Printf("   Remaining Unmatched : %d (new strings to translate)\n", stats.NewUntranslated)
	fmt.Printf("   Obsolete in Game    : %d\n", stats.ObsoleteCount)
	fmt.Println("------------------------------------------")
}

func handleApplyPatch(args []string) {
	fs := flag.NewFlagSet("apply-patch", flag.ExitOnError)
	gameDir := fs.String("game", "", "Path to game directory to patch (required)")
	patchPath := fs.String("patch", "", "Path to .patch.json or .patch.json.gz file (required)")
	outDir := fs.String("output", "", "Optional target output directory (defaults to modifying game in-place)")
	fs.Parse(args)

	if *gameDir == "" || *patchPath == "" {
		fmt.Println("Error: -game and -patch are required")
		fs.Usage()
		os.Exit(1)
	}

	dest := *outDir
	if dest == "" {
		dest = *gameDir
	}

	fmt.Printf("⚡ Directly applying patch [%s] to game [%s] ...\n", *patchPath, dest)
	ctx := context.Background()
	if err := app.ApplyPatch(ctx, *patchPath, *gameDir, dest); err != nil {
		fmt.Printf("Apply patch failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("------------------------------------------")
	fmt.Printf("✅ Patch applied directly to game successfully!\n")
	fmt.Printf("   Target Directory: %s\n", dest)
	fmt.Println("------------------------------------------")
}

func handleStyles(args []string) {
	fmt.Println("🎨 Available Translation Styles & Persona Templates:")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, s := range prompts.AvailableStyles() {
		fmt.Printf("  • %-16s %s\n    %s\n\n", s.ID, s.Name, s.Description)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("📁 Custom Template Files:")
	fmt.Println("   Place custom templates in './templates/<name>.txt' or pass a path:")
	fmt.Println("   Example: lingo translate -workspace game.nst -provider maxplus -style nsfw")
}

func handleUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	checkOnly := fs.Bool("check", false, "Only check if an update is available without installing")
	force := fs.Bool("force", false, "Force re-download even if already up to date")
	_ = fs.Parse(args)

	if *checkOnly {
		fmt.Println("🔍 Checking for updates...")
		hasUpdate, currentVer, latestVer, err := updater.CheckUpdate()
		if err != nil {
			fmt.Printf("❌ Failed to check for updates: %v\n", err)
			os.Exit(1)
		}
		if hasUpdate {
			fmt.Printf("📦 An update is available: v%s (current: v%s)\n", latestVer, currentVer)
			fmt.Println("   Run 'lingo update' to upgrade.")
		} else {
			fmt.Printf("✅ Lingo CLI is up to date (v%s)\n", currentVer)
		}
		return
	}

	if err := updater.Update(*force); err != nil {
		fmt.Printf("❌ Update failed: %v\n", err)
		os.Exit(1)
	}
}
