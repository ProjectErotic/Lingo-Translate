package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lingo-translate/pkg/app"
	"lingo-translate/pkg/plugins/chanomhub"
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
	PrimaryTranslation TaskBinding `json:"primary_translation"`   // Narrative / Complex dialogue
	FastTranslation    TaskBinding `json:"fast_translation"`      // UI, items, skills, bulk short text
	AutoRouteShortText bool        `json:"auto_route_short_text"` // Route lines shorter than MaxShortLength to FastTranslation
	MaxShortLength     int         `json:"max_short_length"`       // Character length threshold (default: 60)
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
	Theme              string            `json:"theme"`                // "dark"

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
		DefaultModel:       "gemini-2.5-flash",
		DefaultSourceLang:  "Japanese",
		DefaultTargetLang:  "Thai",
		DefaultBatchSize:   10,
		DefaultConcurrency: 4,
		Theme:              "dark",
		Tasks: TasksConfig{
			PrimaryTranslation: TaskBinding{
				Provider: "gemini",
				Model:    "gemini-2.5-flash",
			},
			FastTranslation: TaskBinding{
				Provider: "google",
				Model:    "google-translate",
			},
			AutoRouteShortText: false,
			MaxShortLength:     60,
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
	if settings.Tasks.MaxShortLength <= 0 {
		settings.Tasks.MaxShortLength = 60
	}
	if settings.SystemOne.Provider == "" {
		settings.SystemOne.Provider = "typesafe_jev"
	}
	if settings.SystemOne.ConfidenceThreshold <= 0 {
		settings.SystemOne.ConfidenceThreshold = 0.85
	}

	return settings, nil
}

// GetProviders returns list of all built-in and plugin translation providers
func (s *SettingsService) GetProviders() []app.ProviderInfo {
	return app.ListAvailableProviders()
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

// RequestUchsKey requests a new Virtual Key from Chanomhub and saves it into settings
func (s *SettingsService) RequestUchsKey() (string, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return "", err
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

// OpenUchsPortal opens the UCHS Account Portal in the default web browser via Chanomhub SSO
func (s *SettingsService) OpenUchsPortal() error {
	settings, err := s.GetSettings()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	return chanomhub.OpenUchsPortal(ctx, settings.ChanomhubToken, "")
}

