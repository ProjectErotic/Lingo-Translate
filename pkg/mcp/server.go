package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/parser"
	"lingo-translate/pkg/pipeline"
	"lingo-translate/pkg/plugins/chanomhub"
	"lingo-translate/pkg/storage"
	"lingo-translate/pkg/translator"
	"lingo-translate/pkg/translator/gemini"
	"lingo-translate/pkg/translator/mock"
	"lingo-translate/pkg/translator/openai"
	"lingo-translate/pkg/version"
)

// Server represents an MCP (Model Context Protocol) Server
type Server struct {
	mu sync.Mutex
}

func NewServer() *Server {
	return &Server{}
}

// JSON-RPC Request and Response structures
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ServeStdio starts the MCP server over standard input and output
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	writer := bufio.NewWriter(out)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req RPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			errResp := RPCResponse{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    -32700,
					Message: "Parse error",
				},
			}
			bytes, _ := json.Marshal(errResp)
			_, _ = writer.Write(append(bytes, '\n'))
			_ = writer.Flush()
			continue
		}

		resp := s.HandleRequest(ctx, &req)
		if resp != nil {
			bytes, _ := json.Marshal(resp)
			_, _ = writer.Write(append(bytes, '\n'))
			_ = writer.Flush()
		}
	}

	return scanner.Err()
}

// HandleRequest processes an incoming JSON-RPC request
func (s *Server) HandleRequest(ctx context.Context, req *RPCRequest) *RPCResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch req.Method {
	case "initialize":
		return &RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]string{
					"name":    "lingo-server",
					"version": version.Get() + "-go",
				},
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
			},
		}

	case "notifications/initialized", "initialized":
		// Handshake completed
		return nil

	case "tools/list":
		return &RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.ListTools(),
			},
		}

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    -32602,
					Message: "Invalid tool call parameters",
				},
			}
		}

		result, err := s.CallTool(ctx, params.Name, params.Arguments)
		if err != nil {
			return &RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolCallResult{
					Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
					IsError: true,
				},
			}
		}

		return &RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}

	default:
		return &RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

// ListTools returns definitions of all tools exposed by the MCP server
func (s *Server) ListTools() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "lingo_load_project",
			"description": "Load a game project, auto-detect the engine, and extract translatable texts into a workspace",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the game directory",
					},
					"workspace": map[string]interface{}{
						"type":        "string",
						"description": "Path to output .nst workspace file (default: workspace.nst)",
					},
					"engine": map[string]interface{}{
						"type":        "string",
						"description": "Optional engine override (rpgm, renpy, godot, unity, libgdx)",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "lingo_get_status",
			"description": "Get workspace stats, total texts, translated count, and engine information",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workspace": map[string]interface{}{
						"type":        "string",
						"description": "Path to .nst workspace file",
					},
				},
				"required": []string{"workspace"},
			},
		},
		{
			"name":        "lingo_query_entries",
			"description": "Query or search extracted text entries from a workspace",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workspace": map[string]interface{}{
						"type":        "string",
						"description": "Path to .nst workspace file",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Filter by status: 'all', 'untranslated', 'translated'",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of entries to return (default: 50)",
					},
				},
				"required": []string{"workspace"},
			},
		},
		{
			"name":        "lingo_update_entry",
			"description": "Update the translation of a specific text entry in the workspace",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workspace": map[string]interface{}{
						"type":        "string",
						"description": "Path to .nst workspace file",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Text entry ID",
					},
					"target": map[string]interface{}{
						"type":        "string",
						"description": "The translated text string",
					},
				},
				"required": []string{"workspace", "id", "target"},
			},
		},
		{
			"name":        "lingo_translate",
			"description": "Run automated AI translation batch on untranslated texts in the workspace",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workspace": map[string]interface{}{
						"type":        "string",
						"description": "Path to .nst workspace file",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "Translation provider: 'mock', 'gemini', 'openai'",
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "Optional model name",
					},
				},
				"required": []string{"workspace", "provider"},
			},
		},
		{
			"name":        "lingo_deploy",
			"description": "Deploy translated texts back into game files or export runtime translation mod",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workspace": map[string]interface{}{
						"type":        "string",
						"description": "Path to .nst workspace file",
					},
					"game_path": map[string]interface{}{
						"type":        "string",
						"description": "Original game root directory",
					},
					"dest_path": map[string]interface{}{
						"type":        "string",
						"description": "Destination directory",
					},
				},
				"required": []string{"workspace", "game_path", "dest_path"},
			},
		},
		{
			"name":        "lingo_publish_chanomhub",
			"description": "Compress translations and publish directly to Chanomhub mod portal",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"game_path": map[string]interface{}{
						"type":        "string",
						"description": "Game root path containing lingo_translations/",
					},
					"slug": map[string]interface{}{
						"type":        "string",
						"description": "Chanomhub game article slug",
					},
					"token": map[string]interface{}{
						"type":        "string",
						"description": "Chanomhub user JWT API token",
					},
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Target language (default: Thai)",
					},
				},
				"required": []string{"game_path", "slug", "token"},
			},
		},
	}
}

// CallTool executes a specific tool by name with arguments
func (s *Server) CallTool(ctx context.Context, name string, args json.RawMessage) (*ToolCallResult, error) {
	switch name {
	case "lingo_load_project", "nst_load_project":
		var p struct {
			Path      string `json:"path"`
			Workspace string `json:"workspace"`
			Engine    string `json:"engine"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}
		if p.Workspace == "" {
			p.Workspace = "workspace.nst"
		}

		var engineParser parser.EngineParser
		var err error
		if p.Engine != "" {
			engineParser, err = parser.GetParser(p.Engine)
		} else {
			engineParser, err = parser.DetectEngine(p.Path)
		}
		if err != nil {
			return nil, fmt.Errorf("engine detection failed: %w", err)
		}

		entries, stats, err := engineParser.Extract(ctx, p.Path)
		if err != nil {
			return nil, fmt.Errorf("extraction failed: %w", err)
		}

		store, err := storage.Open(p.Workspace)
		if err != nil {
			return nil, fmt.Errorf("failed to open workspace: %w", err)
		}
		defer store.Close()

		proj := &model.Project{
			ID:         filepath.Base(p.Path),
			Name:       filepath.Base(p.Path),
			Engine:     engineParser.Name(),
			SourcePath: p.Path,
			SourceLang: "Japanese",
			TargetLang: "Thai",
			CreatedAt:  time.Now(),
		}
		_ = store.SaveProject(proj)
		_ = store.SaveEntries(entries)

		return &ToolCallResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Successfully loaded project: %s (Engine: %s). Extracted %d entries (%d unique texts) in %v.",
						proj.Name, engineParser.Name(), stats.TotalEntries, stats.UniqueTexts, stats.Duration),
				},
			},
		}, nil

	case "lingo_get_status", "nst_get_status":
		var p struct {
			Workspace string `json:"workspace"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}
		store, err := storage.Open(p.Workspace)
		if err != nil {
			return nil, err
		}
		defer store.Close()

		proj, _ := store.GetProject()
		entries, err := store.GetEntries("all")
		if err != nil {
			return nil, err
		}

		translated := 0
		for _, e := range entries {
			if e.Target != "" && e.Status != model.StatusUntranslated {
				translated++
			}
		}
		pct := 0.0
		if len(entries) > 0 {
			pct = float64(translated) / float64(len(entries)) * 100.0
		}

		resText := fmt.Sprintf("Project: %s\nEngine: %s\nTotal Entries: %d\nTranslated: %d (%.1f%%)",
			proj.Name, proj.Engine, len(entries), translated, pct)

		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: resText}},
		}, nil

	case "lingo_query_entries", "nst_query_entries":
		var p struct {
			Workspace string `json:"workspace"`
			Status    string `json:"status"`
			Limit     int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}
		if p.Status == "" {
			p.Status = "all"
		}
		if p.Limit <= 0 {
			p.Limit = 50
		}

		store, err := storage.Open(p.Workspace)
		if err != nil {
			return nil, err
		}
		defer store.Close()

		entries, err := store.GetEntries(p.Status)
		if err != nil {
			return nil, err
		}

		if len(entries) > p.Limit {
			entries = entries[:p.Limit]
		}

		data, _ := json.MarshalIndent(entries, "", "  ")
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		}, nil

	case "lingo_update_entry", "nst_update_entry":
		var p struct {
			Workspace string `json:"workspace"`
			ID        string `json:"id"`
			Target    string `json:"target"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}

		store, err := storage.Open(p.Workspace)
		if err != nil {
			return nil, err
		}
		defer store.Close()

		if err := store.UpdateEntryTarget(p.ID, p.Target, model.StatusTranslated, "manual"); err != nil {
			return nil, err
		}

		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Updated entry %s successfully.", p.ID)}},
		}, nil

	case "lingo_translate", "nst_translate":
		var p struct {
			Workspace string `json:"workspace"`
			Provider  string `json:"provider"`
			Model     string `json:"model"`
			APIKey    string `json:"api_key"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}

		store, err := storage.Open(p.Workspace)
		if err != nil {
			return nil, err
		}
		defer store.Close()

		apiKey := p.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("LINGO_API_KEY")
		}

		var trans translator.Translator
		switch strings.ToLower(p.Provider) {
		case "mock":
			trans = mock.New("[TH] ")
		case "gemini":
			trans = gemini.New(gemini.Config{
				APIKey: apiKey,
				Model:  p.Model,
			})
		case "openai":
			trans = openai.New(openai.Config{
				APIKey: apiKey,
				Model:  p.Model,
			})
		default:
			return nil, fmt.Errorf("unsupported translation provider: %s", p.Provider)
		}

		entries, err := store.GetEntries(string(model.StatusUntranslated))
		if err != nil {
			return nil, err
		}

		pipe := pipeline.New(store, trans, pipeline.Config{BatchSize: 10, Concurrency: 2})
		opts := translator.Options{
			SourceLang: "Japanese",
			TargetLang: "Thai",
			Model:      p.Model,
		}

		_, err = pipe.Run(ctx, entries, opts, nil)
		if err != nil {
			return nil, fmt.Errorf("pipeline execution error: %w", err)
		}

		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Translation completed for %d entries.", len(entries))}},
		}, nil

	case "lingo_deploy", "nst_deploy":
		var p struct {
			Workspace string `json:"workspace"`
			GamePath  string `json:"game_path"`
			DestPath  string `json:"dest_path"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}

		store, err := storage.Open(p.Workspace)
		if err != nil {
			return nil, err
		}
		defer store.Close()

		proj, _ := store.GetProject()
		entries, err := store.GetEntries("all")
		if err != nil {
			return nil, err
		}

		engineParser, err := parser.GetParser(proj.Engine)
		if err != nil {
			engineParser, _ = parser.DetectEngine(p.GamePath)
		}
		if engineParser == nil {
			return nil, fmt.Errorf("could not determine game engine for deployment")
		}

		if err := engineParser.Inject(ctx, p.GamePath, p.DestPath, entries); err != nil {
			return nil, fmt.Errorf("deploy injection failed: %w", err)
		}

		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Deployed translations into %s successfully.", p.DestPath)}},
		}, nil

	case "lingo_publish_chanomhub", "nst_publish_chanomhub":
		var p struct {
			GamePath string `json:"game_path"`
			Slug     string `json:"slug"`
			Token    string `json:"token"`
			Language string `json:"language"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}

		client := chanomhub.NewClient("", "", p.Token)
		res, err := client.PublishTranslation(ctx, chanomhub.PublishRequest{
			GameDir:  p.GamePath,
			Slug:     p.Slug,
			Language: p.Language,
		})
		if err != nil {
			return nil, fmt.Errorf("chanomhub publish failed: %w", err)
		}

		return &ToolCallResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Submission successful! Download URL: %s (Size: %d bytes). %s",
						res.DownloadURL, res.FileSizeBytes, res.Message),
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}
