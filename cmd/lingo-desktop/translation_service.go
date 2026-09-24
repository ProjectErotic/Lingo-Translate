package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lingo-translate/pkg/app"
	"lingo-translate/pkg/model"
)

type TranslationDonePayload struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type TranslationService struct {
	session     *Session
	mu          sync.Mutex
	cancelFunc  context.CancelFunc
	isTranslating bool
}

func NewTranslationService(session *Session) *TranslationService {
	return &TranslationService{session: session}
}

func (t *TranslationService) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.isTranslating
}

// Start launches a translation pipeline in background goroutine
func (t *TranslationService) Start(opts app.TranslateOptions) error {
	t.mu.Lock()
	if t.isTranslating {
		t.mu.Unlock()
		return fmt.Errorf("a translation is already running")
	}

	ws, err := t.session.GetWorkspace()
	if err != nil {
		t.mu.Unlock()
		return err
	}

	// Enrich options with Tasks and SystemOne from Settings if not explicitly overridden
	if s, sErr := NewSettingsService().GetSettings(); sErr == nil {
		if opts.Provider.APIKey == "" {
			key, base := s.ResolveProviderAuth(opts.Provider.Name)
			opts.Provider.APIKey = key
			if opts.Provider.BaseURL == "" {
				opts.Provider.BaseURL = base
			}
		}
		if opts.FastProvider == nil && s.Tasks.FastTranslation.Provider != "" {
			fastKey, fastBase := s.ResolveProviderAuth(s.Tasks.FastTranslation.Provider)
			opts.FastProvider = &app.ProviderConfig{
				Name:    s.Tasks.FastTranslation.Provider,
				Model:   s.Tasks.FastTranslation.Model,
				APIKey:  fastKey,
				BaseURL: fastBase,
			}
			opts.AutoRouteShort = s.Tasks.AutoRouteShortText
			opts.MaxShortLen = s.Tasks.MaxShortLength
		} else if opts.FastProvider != nil && opts.FastProvider.APIKey == "" {
			fastKey, fastBase := s.ResolveProviderAuth(opts.FastProvider.Name)
			opts.FastProvider.APIKey = fastKey
			if opts.FastProvider.BaseURL == "" {
				opts.FastProvider.BaseURL = fastBase
			}
		}
		if opts.FallbackProvider == nil && s.Tasks.EnableFallback && s.Tasks.FallbackTranslation.Provider != "" {
			fbKey, fbBase := s.ResolveProviderAuth(s.Tasks.FallbackTranslation.Provider)
			opts.FallbackProvider = &app.ProviderConfig{
				Name:    s.Tasks.FallbackTranslation.Provider,
				Model:   s.Tasks.FallbackTranslation.Model,
				APIKey:  fbKey,
				BaseURL: fbBase,
			}
		} else if opts.FallbackProvider != nil && opts.FallbackProvider.APIKey == "" {
			fbKey, fbBase := s.ResolveProviderAuth(opts.FallbackProvider.Name)
			opts.FallbackProvider.APIKey = fbKey
			if opts.FallbackProvider.BaseURL == "" {
				opts.FallbackProvider.BaseURL = fbBase
			}
		}
		if opts.EnableMemoryCache == nil {
			enableCache := s.EnableMemoryCache
			opts.EnableMemoryCache = &enableCache
		}
		if opts.Style == "" && s.TranslationStyle != "" {
			opts.Style = s.TranslationStyle
		}
		if opts.Prompt == "" && s.ContextLore != "" {
			opts.Prompt = s.ContextLore
		}
		if opts.SystemOne == nil && s.SystemOne.Enabled {
			opts.SystemOne = &app.SystemOneOptions{
				Enabled:             s.SystemOne.Enabled,
				Provider:            s.SystemOne.Provider,
				APIKey:              s.SystemOne.APIKey,
				BaseURL:             s.SystemOne.BaseURL,
				ConfidenceThreshold: s.SystemOne.ConfidenceThreshold,
				FilterAmbiguousCode: s.SystemOne.Features.FilterAmbiguousCode,
				AcceptUIDrafts:      s.SystemOne.Features.AcceptUIDrafts,
				VerifyQA:            s.SystemOne.Features.VerifyQA,
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.cancelFunc = cancel
	t.isTranslating = true
	t.mu.Unlock()

	go func() {
		defer func() {
			t.mu.Lock()
			t.isTranslating = false
			t.cancelFunc = nil
			t.mu.Unlock()
		}()

		err := ws.Translate(ctx, opts, func(p model.TranslationProgress) {
			if wailsApp := application.Get(); wailsApp != nil {
				wailsApp.Event.Emit("translation:progress", p)
			}
		})

		payload := TranslationDonePayload{
			Success: err == nil,
		}
		if err != nil {
			payload.Error = err.Error()
		}

		if wailsApp := application.Get(); wailsApp != nil {
			wailsApp.Event.Emit("translation:done", payload)
		}
	}()

	return nil
}

// Cancel terminates the active translation pipeline
func (t *TranslationService) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
}
