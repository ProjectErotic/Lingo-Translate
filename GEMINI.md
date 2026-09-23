# Lingo-Translate Development Guide

**Repository:** `ProjectErotic/Lingo-Translate` (https://github.com/ProjectErotic/Lingo-Translate)  
**Organization:** ProjectErotic

---

## AI Coding Rules & Project Conventions

### 1. Project Naming & Brand Identity
- The project is **Lingo-Translate** (or **Lingo**).
- Binary outputs:
  - CLI: `lingo` (compiled into `bin/lingo`)
  - Desktop GUI: `lingo-desktop` (compiled into `bin/lingo-desktop`)
- Go module: `lingo-translate` in `go.mod`.
- Config directories:
  - Workspaces / models: `~/.lingo/`
  - Settings / registry: `~/.config/lingo/`
- **NEVER** introduce new files, packages, functions, or UI elements named `nst` or `NST`.
- Any legacy `nst` references must only exist as backward-compatible fallback symlinks/aliases (e.g. symlink `nst -> lingo`, `NST_API_KEY` fallback if `LINGO_API_KEY` is unset).

### 2. Code Style & Logging
- Keep console output clean, concise, and professional.
- Avoid unnecessary emojis in system, parser, and daemon logs. Use clear bracketed tags such as `[Lingo]`, `[Parser]`, `[Deploy]`, `[Error]`, `[Info]`.

### 3. Architecture & Build Standards
- **Pure Go CLI (`CGO_ENABLED=0`)**: The CLI must compile without external C runtime dependencies.
- **Non-Destructive Modding**: Always preserve original game files. Runtime patches and injected layers (such as `Lingo_TranslationLayer.js` or `00_lingo_font_layer.rpy`) must never mutate raw source assets.
- **Tests**: Always run `CGO_ENABLED=0 go test ./pkg/...` and ensure 100% test coverage pass before publishing.
