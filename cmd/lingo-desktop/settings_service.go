package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lingo-translate/pkg/app"
	"lingo-translate/pkg/plugins/chanomhub"
	"lingo-translate/pkg/translator"
	"lingo-translate/pkg/translator/custom"
)

// TaskBinding specifies model and options for a specific translation role
type TaskBinding struct {
	Provider    string  `json:"provider"`              // "gemini", "openai", "google", "mock", or plugin name
	Model       string  `json:"model"`                 // e.g. "gemini-2.5-pro", "gpt-4o", etc.
	Temperature float64 `json:"temperature,omitempty"` // optional sampling temperature
	MaxTokens   int     `json:"max_tokens,omitempty"`  // optional token limit
}

// SystemOneFeatures defines which decision delegations are enabled
type SystemOneFeatures struct {
	FilterAmbiguousCode bool `json:"filter_ambiguous_code"` // Filter code vs text in gray-zone (Noul)
	AcceptUIDrafts      bool `json:"accept_ui_drafts"`       // Speculatively accept MT UI drafts without LLM (Noul)
	VerifyQA            bool `json:"verify_qa"`              // Quality & refusal/hallucination sentinel (Score/Noul)
}

// SystemOneConfig specifies the auxiliary non-autoregressive decision engine (e.g. TypeSafe AI Jev)
type SystemOneConfig struct {
	Enabled             bool              `json:"enabled"`              // Auxiliary toggle (ON/OFF)
	Provider            string            `json:"provider"`             // "typesafe_jev", "heuristic"
	APIKey              string            `json:"api_key"`              // TypeSafe AI API Key
	BaseURL             string            `json:"base_url,omitempty"`   // Endpoint override (e.g. https://api.typesafe.ai/v1)
	ConfidenceThreshold float64           `json:"confidence_threshold"` // Minimum confidence score (e.g. 0.85)
	Features            SystemOneFeatures `json:"features"`             // Fine-grained delegation toggles
}

// TasksConfig manages Hermes-style task routing across different models
type TasksConfig struct {
	PrimaryTranslation  TaskBinding `json:"primary_translation"`   // Narrative / Complex dialogue
	FastTranslation     TaskBinding `json:"fast_translation"`      // UI, items, skills, bulk short text
	FallbackTranslation TaskBinding `json:"fallback_translation"`  // Failover engine if primary fails
	AutoRouteShortText  bool        `json:"auto_route_short_text"` // Route lines shorter than MaxShortLength to FastTranslation
	MaxShortLength      int         `json:"max_short_length"`       // Character length threshold (default: 60)
	EnableFallback      bool        `json:"enable_fallback"`       // Auto failover toggle
}

type Settings struct {
	DefaultProvider    string            `json:"default_provider"`     // "mock", "gemini", "openai", "google", or any custom plugin name
	DefaultModel       string            `json:"default_model"`        // e.g. "gemini-2.5-flash", "gpt-4o-mini", etc.
	GeminiAPIKey       string            `json:"gemini_api_key"`
	OpenAIAPIKey       string            `json:"openai_api_key"`
	OpenAIBaseURL      string            `json:"openai_base_url"`
	GoogleAPIKey       string            `json:"google_api_key"`
	ChanomhubToken     string            `json:"chanomhub_token"`
	UchsAPIKey         string            `json:"uchs_api_key"`
	PluginKeys         map[string]string `json:"plugin_keys,omitempty"`      // dynamic map: provider_name -> api_key
	PluginBaseURLs     map[string]string `json:"plugin_base_urls,omitempty"`  // dynamic map: provider_name -> base_url
	DefaultSourceLang  string            `json:"default_source_lang"`  // "Japanese"
	DefaultTargetLang  string            `json:"default_target_lang"`  // "Thai"
	DefaultBatchSize   int               `json:"default_batch_size"`   // 10
	DefaultConcurrency int               `json:"default_concurrency"`  // 4
	Theme              string            `json:"theme"`                // "dark", "light", "midnight", "system"
	Density            string            `json:"density,omitempty"`    // "comfortable", "compact"
	FontSize           string            `json:"font_size,omitempty"`  // "small", "medium", "large"

	// Context & Memory
	ContextLore        string            `json:"context_lore,omitempty"`        // Custom world lore & prompt directives
	TranslationStyle   string            `json:"translation_style,omitempty"`    // Persona style (e.g. "standard", "nsfw", "vn_romance")
	EnableMemoryCache  bool              `json:"enable_memory_cache"`            // Reuse identical translations from TM cache

	// Task-Based AI Routing & Auxiliary System One (Hermes-Style Architecture)
	Tasks     TasksConfig     `json:"tasks"`
	SystemOne SystemOneConfig `json:"system_one"`
}

type SettingsService struct {
	mu           sync.RWMutex
	filePath     string
}

func NewSettingsService(customPath ...string) *SettingsService {
	path := ""
	if len(customPath) > 0 {
		path = customPath[0]
	} else {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			cfgDir = "."
		}
		path = filepath.Join(cfgDir, "lingo", "settings.json")
	}
	return &SettingsService{filePath: path}
}

// defaultSettings returns sensible defaults populated with any existing environment variables
func defaultSettings() Settings {
	s := Settings{
		DefaultProvider:    "mock",
		DefaultModel:       "gemini-3.8-flash",
		DefaultSourceLang:  "Japanese",
		DefaultTargetLang:  "Thai",
		DefaultBatchSize:   10,
		DefaultConcurrency: 4,
		Theme:              "dark",
		Density:            "comfortable",
		FontSize:           "medium",
		TranslationStyle:   "standard",
		EnableMemoryCache:  true,
		Tasks: TasksConfig{
			PrimaryTranslation: TaskBinding{
				Provider: "gemini",
				Model:    "gemini-3.8-flash",
			},
			FastTranslation: TaskBinding{
				Provider: "google",
				Model:    "google-translate",
			},
			FallbackTranslation: TaskBinding{
				Provider: "gemini",
				Model:    "gemini-3.8-flash",
			},
			AutoRouteShortText: false,
			MaxShortLength:     60,
			EnableFallback:     false,
		},
		SystemOne: SystemOneConfig{
			Enabled:             false,
			Provider:            "typesafe_jev",
			ConfidenceThreshold: 0.85,
			Features: SystemOneFeatures{
				FilterAmbiguousCode: true,
				AcceptUIDrafts:      true,
				VerifyQA:            false,
			},
		},
	}

	if envKey := os.Getenv("LINGO_API_KEY"); envKey != "" {
		s.GeminiAPIKey = envKey
		s.OpenAIAPIKey = envKey
		s.GoogleAPIKey = envKey
	} else if envKey := os.Getenv("NST_API_KEY"); envKey != "" { // legacy fallback
		s.GeminiAPIKey = envKey
		s.OpenAIAPIKey = envKey
		s.GoogleAPIKey = envKey
	}
	if envToken := os.Getenv("CHANOMHUB_TOKEN"); envToken != "" {
		s.ChanomhubToken = envToken
	}
	if jevKey := os.Getenv("TYPESAFE_API_KEY"); jevKey != "" {
		s.SystemOne.APIKey = jevKey
	} else if jevKey := os.Getenv("JEV_API_KEY"); jevKey != "" {
		s.SystemOne.APIKey = jevKey
	}
	return s
}

// GetSettings reads persisted settings or returns defaults
func (s *SettingsService) GetSettings() (Settings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultSettings(), nil
		}
		return defaultSettings(), err
	}

	settings := defaultSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings(), err
	}

	// Ensure fallback for legacy settings
	if settings.Tasks.PrimaryTranslation.Provider == "" {
		settings.Tasks.PrimaryTranslation.Provider = settings.DefaultProvider
		settings.Tasks.PrimaryTranslation.Model = settings.DefaultModel
	}
	if settings.Tasks.FastTranslation.Provider == "" {
		settings.Tasks.FastTranslation.Provider = "google"
		settings.Tasks.FastTranslation.Model = "google-translate"
	}
	if settings.Tasks.FallbackTranslation.Provider == "" {
		settings.Tasks.FallbackTranslation.Provider = "gemini"
		settings.Tasks.FallbackTranslation.Model = "gemini-3.8-flash"
	}
	if settings.Tasks.MaxShortLength <= 0 {
		settings.Tasks.MaxShortLength = 60
	}
	if settings.SystemOne.Provider == "" {
		settings.SystemOne.Provider = "typesafe_jev"
	}
	if settings.SystemOne.ConfidenceThreshold <= 0 {
		settings.SystemOne.ConfidenceThreshold = 0.85
	}
	if settings.Theme == "" {
		settings.Theme = "dark"
	}
	if settings.Density == "" {
		settings.Density = "comfortable"
	}
	if settings.FontSize == "" {
		settings.FontSize = "medium"
	}
	if settings.TranslationStyle == "" {
		settings.TranslationStyle = "standard"
	}

	return settings, nil
}

// GetProviders returns list of all built-in and plugin translation providers
func (s *SettingsService) GetProviders() []app.ProviderInfo {
	providers := app.ListAvailableProviders()

	// If settings exist, query real endpoints dynamically for live models
	settings, err := s.GetSettings()
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		for i := range providers {
			switch providers[i].Name {
			case "uchs":
				key := strings.TrimSpace(settings.UchsAPIKey)
				if key != "" {
					baseURL := providers[i].BaseURL
					if baseURL == "" {
						baseURL = "https://ilms.uchs-th.com/v1"
					}
					if liveModels, err := queryEndpointModels(ctx, baseURL, key); err == nil && len(liveModels) > 0 {
						providers[i].AvailableModels = liveModels
					}
				}
			case "gemini":
				key := strings.TrimSpace(settings.GeminiAPIKey)
				if key != "" {
					if liveModels, err := queryEndpointModels(ctx, "https://generativelanguage.googleapis.com", key); err == nil && len(liveModels) > 0 {
						providers[i].AvailableModels = liveModels
					}
				}
			case "openai":
				key := strings.TrimSpace(settings.OpenAIAPIKey)
				baseURL := strings.TrimSpace(settings.OpenAIBaseURL)
				if baseURL != "" && baseURL != "https://api.openai.com/v1" {
					if liveModels, err := queryEndpointModels(ctx, baseURL, key); err == nil && len(liveModels) > 0 {
						providers[i].AvailableModels = liveModels
					}
				}
			default:
				if providers[i].IsCustom && providers[i].BaseURL != "" {
					key, base := settings.ResolveProviderAuth(providers[i].Name)
					if base == "" {
						base = providers[i].BaseURL
					}
					if liveModels, err := queryEndpointModels(ctx, base, key); err == nil && len(liveModels) > 0 {
						providers[i].AvailableModels = liveModels
					}
				}
			}
		}
	}

	return providers
}

// SaveSettings writes updated settings to disk
func (s *SettingsService) SaveSettings(settings Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0600)
}

// RequestUchsKey requests a new Virtual Key from Chanomhub and saves it into settings.
// If token is provided, it updates the saved Chanomhub token; otherwise it uses the persisted token.
func (s *SettingsService) RequestUchsKey(token string) (string, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token) != "" {
		settings.ChanomhubToken = strings.TrimSpace(token)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	key, err := chanomhub.RequestUchsKey(ctx, settings.ChanomhubToken, "")
	if err != nil {
		return "", err
	}

	settings.UchsAPIKey = key
	if err := s.SaveSettings(settings); err != nil {
		return key, fmt.Errorf("key generated but failed to save settings: %w", err)
	}
	return key, nil
}

// OpenUchsPortal opens the UCHS Account Portal in the default web browser via Chanomhub SSO.
// If token is provided, it updates the saved Chanomhub token; otherwise it uses the persisted token.
func (s *SettingsService) OpenUchsPortal(token string) error {
	settings, err := s.GetSettings()
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) != "" {
		settings.ChanomhubToken = strings.TrimSpace(token)
		_ = s.SaveSettings(settings)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	return chanomhub.OpenUchsPortal(ctx, settings.ChanomhubToken, "")
}

// ResolveProviderAuth returns the configured API key and BaseURL for the specified provider.
func (s Settings) ResolveProviderAuth(providerName string) (apiKey string, baseURL string) {
	switch providerName {
	case "gemini":
		return s.GeminiAPIKey, ""
	case "openai":
		return s.OpenAIAPIKey, s.OpenAIBaseURL
	case "google":
		return s.GoogleAPIKey, ""
	case "uchs":
		return s.UchsAPIKey, "https://ilms.uchs-th.com/v1"
	case "deepseek":
		base := "https://api.deepseek.com/v1"
		if s.PluginBaseURLs != nil && s.PluginBaseURLs["deepseek"] != "" {
			base = s.PluginBaseURLs["deepseek"]
		}
		var key string
		if s.PluginKeys != nil {
			key = s.PluginKeys["deepseek"]
		}
		return key, base
	case "groq":
		base := "https://api.groq.com/openai/v1"
		if s.PluginBaseURLs != nil && s.PluginBaseURLs["groq"] != "" {
			base = s.PluginBaseURLs["groq"]
		}
		var key string
		if s.PluginKeys != nil {
			key = s.PluginKeys["groq"]
		}
		return key, base
	case "ollama":
		base := "http://localhost:11434/v1"
		if s.PluginBaseURLs != nil && s.PluginBaseURLs["ollama"] != "" {
			base = s.PluginBaseURLs["ollama"]
		}
		var key string
		if s.PluginKeys != nil {
			key = s.PluginKeys["ollama"]
		}
		return key, base
	case "openrouter":
		base := "https://openrouter.ai/api/v1"
		if s.PluginBaseURLs != nil && s.PluginBaseURLs["openrouter"] != "" {
			base = s.PluginBaseURLs["openrouter"]
		}
		var key string
		if s.PluginKeys != nil {
			key = s.PluginKeys["openrouter"]
		}
		return key, base
	default:
		var key, base string
		if s.PluginKeys != nil {
			key = s.PluginKeys[providerName]
		}
		if s.PluginBaseURLs != nil {
			base = s.PluginBaseURLs[providerName]
		}
		return key, base
	}
}

// SaveCustomProvider saves or updates a custom provider plugin definition (~/.lingo/providers/<name>.json)
func (s *SettingsService) SaveCustomProvider(def custom.Definition) error {
	if strings.TrimSpace(def.Name) == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if strings.TrimSpace(def.BaseURL) == "" {
		return fmt.Errorf("base URL cannot be empty")
	}
	return custom.Save(def)
}

// DeleteCustomProvider removes a custom provider plugin definition
func (s *SettingsService) DeleteCustomProvider(name string) error {
	return custom.Delete(name)
}

// ListCustomProviders returns all saved custom provider definitions
func (s *SettingsService) ListCustomProviders() ([]custom.Definition, error) {
	return custom.List()
}

// FetchRemoteModels queries an OpenAI-compatible or Ollama endpoint for available models
func (s *SettingsService) FetchRemoteModels(baseURL, apiKey string) ([]string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	// Auto-resolve saved API key if apiKey was not explicitly supplied
	if strings.TrimSpace(apiKey) == "" {
		if settings, err := s.GetSettings(); err == nil {
			if strings.Contains(baseURL, "uchs-th.com") || strings.Contains(baseURL, "ilms.uchs") {
				apiKey = settings.UchsAPIKey
			} else if strings.Contains(baseURL, "generativelanguage.googleapis.com") || baseURL == "gemini" {
				apiKey = settings.GeminiAPIKey
			} else if strings.Contains(baseURL, "api.openai.com") {
				apiKey = settings.OpenAIAPIKey
			} else if settings.PluginKeys != nil {
				for pName, key := range settings.PluginKeys {
					pBase := settings.PluginBaseURLs[pName]
					if pBase != "" && (pBase == baseURL || strings.TrimRight(pBase, "/") == strings.TrimRight(baseURL, "/")) {
						apiKey = key
						break
					}
				}
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return queryEndpointModels(ctx, baseURL, apiKey)
}

// TestCustomProvider verifies connection by translating a sample word
func (s *SettingsService) TestCustomProvider(def custom.Definition) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tr, err := custom.NewTranslator(def, def.APIKey, def.DefaultModel, def.BaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to initialize translator: %w", err)
	}

	results, err := tr.Translate(ctx, []string{"Hello"}, translator.Options{
		SourceLang: "English",
		TargetLang: "Thai",
	})
	if err != nil {
		return "", fmt.Errorf("connection test failed: %w", err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("no response returned from provider")
	}
	return results[0].Target, nil
}

func queryEndpointModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	baseURL = strings.TrimRight(baseURL, "/")

	// 1. Google Gemini check
	if strings.Contains(baseURL, "generativelanguage.googleapis.com") || baseURL == "gemini" {
		endpoint := "https://generativelanguage.googleapis.com/v1beta/models"
		if apiKey != "" {
			endpoint += "?key=" + apiKey
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err == nil {
			client := &http.Client{Timeout: 8 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("failed to connect to Gemini API: %w", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode == 400 || resp.StatusCode == 401 || resp.StatusCode == 403 {
				var gErr struct {
					Error struct {
						Message string `json:"message"`
					} `json:"error"`
				}
				_ = json.Unmarshal(body, &gErr)
				errMsg := gErr.Error.Message
				if errMsg == "" {
					errMsg = string(body)
				}
				return nil, fmt.Errorf("Gemini authentication/request failed (HTTP %d): %s", resp.StatusCode, errMsg)
			}
			var geminiResp struct {
				Models []struct {
					Name                       string   `json:"name"`
					SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
				} `json:"models"`
			}
			if err := json.Unmarshal(body, &geminiResp); err == nil && len(geminiResp.Models) > 0 {
				var models []string
				for _, m := range geminiResp.Models {
					canGenerate := false
					for _, method := range m.SupportedGenerationMethods {
						if method == "generateContent" {
							canGenerate = true
							break
						}
					}
					if canGenerate {
						name := strings.TrimPrefix(m.Name, "models/")
						models = append(models, name)
					}
				}
				if len(models) > 0 {
					sort.Strings(models)
					return models, nil
				}
			}
		}
	}

	// 2. Build candidate URLs based on endpoint type
	var candidateURLs []string
	isOllama := strings.Contains(baseURL, "11434") || strings.Contains(baseURL, "ollama") || strings.Contains(baseURL, "localhost")

	if strings.HasSuffix(baseURL, "/v1") {
		candidateURLs = append(candidateURLs, baseURL+"/models")
		if isOllama {
			rootBase := strings.TrimSuffix(baseURL, "/v1")
			candidateURLs = append(candidateURLs, rootBase+"/api/tags")
		}
	} else {
		candidateURLs = append(candidateURLs, baseURL+"/v1/models")
		candidateURLs = append(candidateURLs, baseURL+"/models")
		if isOllama {
			candidateURLs = append(candidateURLs, baseURL+"/api/tags")
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error
	var authErr error

	for _, u := range candidateURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		// If unauthorized (401 or 403), this endpoint EXISTS but rejected authentication
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			var errBody struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			_ = json.Unmarshal(body, &errBody)
			detail := errBody.Error.Message
			if detail == "" {
				detail = strings.TrimSpace(string(body))
			}
			if apiKey == "" {
				authErr = fmt.Errorf("API key required for %s: %s", baseURL, detail)
			} else {
				authErr = fmt.Errorf("authentication failed (HTTP %d) at %s: %s", resp.StatusCode, u, detail)
			}
			break
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, u)
			continue
		}

		// Try OpenAI format: { "data": [ { "id": "..." } ] }
		var openAIResp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &openAIResp); err == nil && len(openAIResp.Data) > 0 {
			var models []string
			for _, m := range openAIResp.Data {
				if m.ID != "" {
					models = append(models, m.ID)
				}
			}
			if len(models) > 0 {
				sort.Strings(models)
				return models, nil
			}
		}

		// Try Ollama format: { "models": [ { "name": "..." } ] }
		var ollamaResp struct {
			Models []struct {
				Name  string `json:"name"`
				Model string `json:"model"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &ollamaResp); err == nil && len(ollamaResp.Models) > 0 {
			var models []string
			for _, m := range ollamaResp.Models {
				name := m.Name
				if name == "" {
					name = m.Model
				}
				if name != "" {
					models = append(models, name)
				}
			}
			if len(models) > 0 {
				sort.Strings(models)
				return models, nil
			}
		}
	}

	if authErr != nil {
		return nil, authErr
	}
	if lastErr != nil {
		return nil, fmt.Errorf("could not fetch models: %w", lastErr)
	}
	return nil, fmt.Errorf("no models found at endpoint %s", baseURL)
}


