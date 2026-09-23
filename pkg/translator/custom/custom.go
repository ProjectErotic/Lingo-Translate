package custom

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lingo-translate/pkg/translator"
	"lingo-translate/pkg/translator/openai"
)

// Definition represents a declarative external translation provider configuration (JSON file)
type Definition struct {
	Name            string            `json:"name"`                   // Unique identifier, e.g. "openrouter", "deepseek", "local_llm"
	DisplayName     string            `json:"display_name,omitempty"` // User-friendly label, e.g. "OpenRouter", "DeepSeek"
	Description     string            `json:"description,omitempty"`  // Brief description of provider
	Type            string            `json:"type,omitempty"`         // "openai" (default), "anthropic", etc.
	BaseURL         string            `json:"base_url"`               // API Base URL, e.g. "https://api.openai.com/v1"
	DefaultModel    string            `json:"default_model"`          // e.g. "gpt-4o", "deepseek-chat"
	AvailableModels []string          `json:"available_models,omitempty"` // Pre-configured choices of models
	APIKeyEnv       string            `json:"api_key_env,omitempty"`  // Environment variable name for API key, e.g. "OPENROUTER_API_KEY"
	APIKey          string            `json:"api_key,omitempty"`      // Optional direct API key fallback
	Headers         map[string]string `json:"headers,omitempty"`      // Optional custom headers
	TimeoutSec      int               `json:"timeout_sec,omitempty"`  // Request timeout in seconds (default: 90)
}

// GetSearchDirs returns the list of directories to search for provider definitions in priority order
func GetSearchDirs() []string {
	var dirs []string

	if customEnv := os.Getenv("NST_PROVIDERS_DIR"); customEnv != "" {
		dirs = append(dirs, customEnv)
	}

	// 1. User home directory ~/.nst/providers and ~/.nst (secure, completely outside git)
	if homeDir, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(homeDir, ".nst", "providers"))
		dirs = append(dirs, filepath.Join(homeDir, ".nst"))
	}

	// 2. User config directory ~/.config/lingo/providers
	if cfgDir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(cfgDir, "lingo", "providers"))
	}

	// 3. Current working directory ./providers
	dirs = append(dirs, "providers")

	return dirs
}

// GetUserConfigDir returns the default user directory for storing custom provider configs (~/.nst/providers)
func GetUserConfigDir() (string, error) {
	if homeDir, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(homeDir, ".nst", "providers")
		if err := os.MkdirAll(dir, 0700); err == nil {
			return dir, nil
		}
	}

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cfgDir, "lingo", "providers")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// List searches all provider directories and returns all available custom provider definitions
func List() ([]Definition, error) {
	seen := make(map[string]bool)
	var list []Definition

	for _, dir := range GetSearchDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
				continue
			}

			filePath := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			var def Definition
			if err := json.Unmarshal(data, &def); err != nil {
				continue
			}

			normalizedName := strings.ToLower(strings.TrimSpace(def.Name))
			if normalizedName == "" {
				base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				def.Name = base
				normalizedName = strings.ToLower(base)
			}

			if !seen[normalizedName] {
				seen[normalizedName] = true
				if def.DisplayName == "" {
					def.DisplayName = def.Name
				}
				list = append(list, def)
			}
		}
	}

	return list, nil
}

// Find searches for a custom provider definition by its name (case-insensitive)
func Find(name string) (*Definition, error) {
	providers, err := List()
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(strings.TrimSpace(name))
	for _, p := range providers {
		if strings.ToLower(p.Name) == target {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("custom provider %q not found", name)
}

// Save writes a provider definition to the user's config directory (~/.config/lingo/providers/<name>.json)
func Save(def Definition) error {
	if strings.TrimSpace(def.Name) == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	dir, err := GetUserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get user config directory: %w", err)
	}

	filePath := filepath.Join(dir, strings.ToLower(def.Name)+".json")
	data, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize provider config: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// Delete removes a provider definition from the user's config directory
func Delete(name string) error {
	dir, err := GetUserConfigDir()
	if err != nil {
		return err
	}
	filePath := filepath.Join(dir, strings.ToLower(name)+".json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("custom provider file %s does not exist", filePath)
	}
	return os.Remove(filePath)
}

// CustomTranslator wraps underlying engine implementations and exposes custom provider identity
type CustomTranslator struct {
	def   Definition
	inner translator.Translator
}

func (c *CustomTranslator) Name() string {
	return c.def.Name
}

func (c *CustomTranslator) Translate(ctx context.Context, texts []string, opts translator.Options) ([]translator.Result, error) {
	results, err := c.inner.Translate(ctx, texts, opts)
	if err != nil {
		return nil, err
	}
	// Stamp provider name to match the custom definition
	for i := range results {
		results[i].Translator = c.def.Name
	}
	return results, nil
}

// NewTranslator creates a running translator client based on the Definition and runtime overrides
func NewTranslator(def Definition, overrideAPIKey, overrideModel, overrideBaseURL string) (translator.Translator, error) {
	// 1. Resolve API Key: runtime override -> env var -> direct config
	apiKey := strings.TrimSpace(overrideAPIKey)
	if apiKey == "" && def.APIKeyEnv != "" {
		apiKey = strings.TrimSpace(os.Getenv(def.APIKeyEnv))
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(def.APIKey)
	}

	// 2. Resolve BaseURL
	baseURL := strings.TrimSpace(overrideBaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(def.BaseURL)
	}

	// 3. Resolve Model
	model := strings.TrimSpace(overrideModel)
	if model == "" {
		model = strings.TrimSpace(def.DefaultModel)
	}

	// 4. Timeout
	timeout := 90 * time.Second
	if def.TimeoutSec > 0 {
		timeout = time.Duration(def.TimeoutSec) * time.Second
	}

	client := openai.New(openai.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Timeout: timeout,
		Headers: def.Headers,
	})

	return &CustomTranslator{
		def:   def,
		inner: client,
	}, nil
}
