# Lingo-Translate: Task-Based AI Routing & System One (Jev) Integration Tracker

**Project:** Lingo-Translate (`ProjectErotic/Lingo-Translate`)  
**Status:** In Progress 🚀  
**Created:** 2026-09-23  
**Architecture Theme:** Hermes-style Task Delegation + System One (Jev) Non-Autoregressive Decision Engine  

---

## 1. Guiding Principles & Rules
1. **High-Repetition Reduction First:** Never call Jev/LLMs for identical or structurally repetitive strings. Use O(1) Exact Hash, TM Cache, and Parametric Skeletons (Research Track 01) before reaching decision layers.
2. **Selective / High-ROI Placement:** Apply Jev ONLY to high-impact gray areas where deterministic regex/code fails and calling frontier LLMs is wasteful.
3. **Graceful Fallback (Zero Disruption):** System One is strictly auxiliary/optional. When disabled or unconfigured, the pipeline must seamlessly fall back to deterministic heuristics.

---

## 2. Architecture & Work Breakdown

```
[Level 0] Exact Cache / TM Lookup (0ms, $0)
    │
[Level 1] Deterministic Fast-Path: Go Regex / Lexer / Skeletons (0ms, $0)
    │
[Level 2] Task Router (Hermes-Style: Narrative vs Fast UI)
    │
[Level 3] System One Auxiliary Engine (Jev: Ambiguous Code Gate & Speculative Acceptor)
    │
[Level 4] Frontier LLM Generation (System Two: Gemini Pro / Claude / GPT-4o)
```

---

## 3. Implementation Phases & Checklist

### Phase 1: Task-Based Settings & Type Definitions ✅ (Completed)
- [x] **1.1 Backend Settings Extension (`cmd/lingo-desktop/settings_service.go`)**
  - Define `TaskBinding` struct: `Provider`, `Model`, `Temperature`, `Prompt`
  - Define `SystemOneConfig` struct: `Enabled`, `Provider`, `APIKey`, `ConfidenceThreshold`, `Features`
  - Add `Tasks` and `SystemOne` fields to `Settings` struct with backward-compatible defaults.
- [x] **1.2 Settings Serialization & Persistence Verification**
  - Verify loading/saving in `~/.config/lingo/settings.json`.
  - Add unit test coverage in `cmd/lingo-desktop/services_test.go`.
- [x] **1.3 Desktop Frontend Settings Dialog (`frontend/src/components/dialogs/SettingsDialog.tsx`)**
  - Add "Task Routing" (Hermes Architecture) configuration section for Primary vs Fast Model.
  - Add dedicated "System One (Jev)" Auxiliary card with feature checkboxes (`filter_ambiguous_code`, `accept_ui_drafts`, `verify_qa`).
  - Add TypeSafe AI API Key management in API Keys tab.

### Phase 2: System One Client & Memoization Layer (`pkg/decision`) ✅ (Completed)
- [x] **2.1 Core Decision Engine Interface (`pkg/decision/engine.go`)**
  - `Noul(ctx, input, question) (bool, float64, error)`
  - `Choice(ctx, input, choices) (string, float64, error)`
  - `Score(ctx, input, criteria) (float64, error)`
- [x] **2.2 Jev API Driver (`pkg/decision/jev/client.go`)**
  - Implement TypeSafe AI REST client for Jev primitives with graceful fallback.
- [x] **2.3 Decision Memoization & Deduplication Cache (`pkg/decision/cache.go`)**
  - Thread-safe bounded memory cache (`hash(primitive + "|" + query + "|" + input) -> cached decision`).
  - Guarantees 0ms response for identical queries across batches.
  - Tested with `-race` condition detector.

### Phase 3: High-ROI Integration Hooks ✅ (Completed)
- [x] **3.1 Hook A: Ambiguous Code Gatekeeper (`pkg/filter/filter.go`)**
  - Keep existing regex/prefixes as primary Fast-Path.
  - Only invoke `decisionEngine.Noul()` when regex result is in the gray-zone.
  - Added unit test `TestFilterWithDecisionEngine`.
- [x] **3.2 Hook B: Speculative Draft Acceptor (`pkg/pipeline/pipeline.go`)**
  - For short UI/item strings (<60 chars) translated via fast draft, invoke `decisionEngine.Noul("Is draft accurate?")`.
  - If approved, mark entry as `StatusTranslated` and skip Frontier LLM (saving tokens & cost).
  - Added QA refusal sentinel check.
  - Added unit tests `TestPipelineSpeculativeDraftAcceptor` and `TestPipelineQAValidation`.

### Phase 4: Verification, Benchmarking & Documentation ✅ (Completed)
- [x] Full Go test suite: `CGO_ENABLED=0 go test ./pkg/...` (100% pass)
- [x] Desktop test suite: `go test -tags gtk3 ./cmd/lingo-desktop/...` (100% pass)
- [x] Concurrency race detector: `go test -race ./pkg/decision/... ./pkg/pipeline/...` (100% pass)
- [x] Frontend build verification: `npm run build` in `frontend/` (Clean build)

### Phase 5: Desktop & CLI End-to-End Integration ✅ (Completed)
- [x] **5.1 Provider Auth Resolution (`cmd/lingo-desktop/settings_service.go`, `translation_service.go`)**
  - Added `ResolveProviderAuth` helper for auto-populating API keys and custom base URLs into `opts.FastProvider` and `opts.Provider`.
  - Added test coverage in `cmd/lingo-desktop/services_test.go`.
- [x] **5.2 CLI Task Routing & System One Flags (`cmd/lingo/main.go`)**
  - Added `-fast-provider`, `-fast-model`, `-fast-api-key`, `-fast-base-url`, `-auto-route`, `-max-short-len`.
  - Added `-system-one`, `-jev-key`, `-jev-base-url`, `-jev-threshold`.
  - Rich CLI banner output summarizing active routing and auxiliary decision engines.
- [x] **5.3 Desktop Translate Dialog (`frontend/src/components/dialogs/TranslateDialog.tsx`)**
  - Added visual "AI Routing & Decision System" card with Hermes Fast-Route and System One Jev badges.
  - Interactive run-time toggles for Fast Model Auto-Route and System One Auxiliary Engine.
  - Added TypeScript bindings for `SystemOneOptions` in `frontend/bindings/lingo-translate/pkg/app/models.ts`.
- [x] **5.4 Pipeline Thread-Safety Hardening (`pkg/pipeline/pipeline.go`)**
  - Added `progressMu sync.Mutex` in `report()` to serialize progress callbacks and eliminate data races across concurrent workers.


