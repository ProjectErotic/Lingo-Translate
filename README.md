# Lingo-Translate (Game Translation Suite - Go Edition)

[![Go Report Card](https://goreportcard.com/badge/github.com/ProjectErotic/Lingo-Translate)](https://goreportcard.com/report/github.com/ProjectErotic/Lingo-Translate)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Lingo-Translate** is a next-generation game translation suite redesigned from the ground up in **Pure Go** (`CGO_ENABLED=0`) and modern web technologies. It is engineered for lightning-fast performance, zero-dependency distribution, and automated translation workflows for visual novels and RPG games.

---

## Key Highlights

- **Pure Go CLI (`CGO_ENABLED=0`):** Compiles into a single self-contained binary for Windows, Linux, and macOS without requiring any C/C++ runtimes or external dependencies.
- **Multi-Engine Support:**
  - **RPG Maker (MV / MZ):** Dialogue codes (`401`), choices (`102`), speaker names (`101`), and database files.
  - **Ren'Py:** `.rpy` scripts, dialogue strings, menu items, and native `tl/Thai/` export.
  - **Godot:** `.tscn` scene files, GDScript `tr()` calls, and `.csv` translation catalogs.
  - **Unity:** YAML scene `.asset` / `.prefab`, StreamingAssets JSON/CSV, and XUnity.AutoTranslator export.
  - **libGDX:** Java `.properties` bundles, narrative script JSONs (e.g. *Tales of Androgyny* encounters/trees), non-destructive `translations/` modding export, and standalone JAR patching.
- **Non-Destructive In-Game Injection:** Deploy translations into RPG Maker games via `Lingo_TranslationLayer.js` without touching original game data files.
- **Tag Masking & Protection:** Intelligent masking ensures engine escape codes (such as `\c[1]`, `\v[n]`, `\fs[20]`) remain intact across translation providers with fuzzy restoration.
- **Translation Memory (TM Cache):** Embedded SQLite workspace (`.nst`) prevents duplicate API calls and maintains translation consistency across project revisions.
- **Multiple Translation Providers:** Built-in drivers for Google Gemini, OpenAI, Ollama / Local LLMs, and Google Translate.
- **Model Context Protocol (MCP) Server:** Native stdio MCP server (`lingo mcp`) allowing AI coding agents (Claude Desktop, Cursor, Antigravity) to directly inspect, extract, and translate game projects.
- **Modern Desktop GUI:** Built with Wails v3 + React 19 + Vite + TypeScript + Tailwind CSS + shadcn/ui.

---

## Quickstart (CLI)

### 1. Build from Source

```bash
# Build standalone CLI binary
make build

# Output binary will be located at bin/lingo
./bin/lingo version
```

### 2. Basic Commands

```bash
# Extract game text into an NST workspace
./bin/lingo extract --game /path/to/game --output workspace.nst

# View project statistics
./bin/lingo status --workspace workspace.nst

# Translate entries using Gemini
export GEMINI_API_KEY="your-api-key"
./bin/lingo translate --workspace workspace.nst --provider gemini --source ja --target th

# Inject translations directly back into game files
./bin/lingo inject --workspace workspace.nst --game /path/to/game --dest /path/to/game_translated

# Or deploy non-destructively (RPG Maker MV/MZ)
./bin/lingo deploy --workspace workspace.nst --game /path/to/game

# Run MCP Server for AI Agent interaction
./bin/lingo mcp
```

---

## Cross-Platform Compilation

Build binaries for Windows, Linux, and macOS in seconds from any host machine:

```bash
make cross-compile
```

Generated binaries will be stored in `dist/`:
- `nst-linux-amd64`
- `nst-linux-arm64`
- `nst-windows-amd64.exe`
- `nst-darwin-arm64` (Apple Silicon M1/M2/M3/M4)
- `nst-darwin-amd64` (Intel Mac)

---

## Automated Build & Release (CI/CD)

This repository features automated GitHub Actions workflows:

1. **CI Workflow (`.github/workflows/ci.yml`):**
   - Automatically runs `go vet` and all package unit tests on Pull Requests to `main`.
2. **Release Workflow (`.github/workflows/release.yml`):**
   - **Trigger:** Whenever `VERSION` is updated in `main` (or a `v*` tag is pushed), GitHub Actions will:
     1. Run full unit test suites.
     2. Cross-compile standalone CLI binaries for Windows (`.exe`), Linux (`amd64`, `arm64`), and macOS (`arm64`, `amd64`).
     3. Package each platform archive (`.zip` / `.tar.gz`) along with documentation.
     4. Calculate SHA256 checksums (`checksums.txt`).
     5. Automatically create a GitHub Release with generated release notes and attach all binary packages.

---

## Desktop GUI (Wails v3)

The modern desktop interface is located in `cmd/lingo-desktop/` with frontend sources in `frontend/`.

```bash
# Development mode
cd frontend && npm install
cd .. && wails3 dev

# Production build
wails3 build
```

---

## Architecture

```
NST-V2/
├── cmd/
│   ├── nst/              # Unified standalone CLI
│   └── nst-desktop/      # Modern desktop GUI application (Wails v3)
├── pkg/
│   ├── analyzer/         # RPG Maker event and variable dependency analyzer
│   ├── app/              # Shared business logic and use cases
│   ├── filter/           # Smart exclusion filter engine
│   ├── injection/rpgm/   # Non-destructive JS runtime injection layer
│   ├── masker/           # Control code masking & fuzzy unmasking
│   ├── mcp/              # Model Context Protocol (MCP) server
│   ├── merger/           # Version reconciler & translation migrator
│   ├── model/            # Core domain models
│   ├── parser/           # Multi-engine parsers (RPGM, Ren'Py, Godot, Unity)
│   ├── pipeline/         # Concurrent translation worker pool
│   ├── plugins/          # External ecosystem plugins (e.g. Chanomhub)
│   ├── registry/         # Local project registry manager
│   ├── storage/          # Pure Go SQLite workspace & TM cache
│   ├── translator/       # Translation provider drivers (Gemini, OpenAI, Google)
│   └── webui/            # Embedded Web UI and REST API server
├── frontend/             # React 19 + TypeScript + Vite UI for Desktop
├── build/                # Desktop packaging assets (Windows, macOS, Linux)
├── VERSION               # Single source of truth for application version
└── LICENSE               # MIT License
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
