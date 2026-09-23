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
