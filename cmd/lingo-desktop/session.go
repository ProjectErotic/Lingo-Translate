package main

import (
	"fmt"
	"sync"

	"lingo-translate/pkg/app"
)

// Session holds the shared open workspace reference across all desktop services
type Session struct {
	mu sync.RWMutex
	ws *app.Workspace
}

func NewSession() *Session {
	return &Session{}
}

func (s *Session) SetWorkspace(ws *app.Workspace) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws != nil && s.ws != ws {
		_ = s.ws.Close()
	}
	s.ws = ws
}

func (s *Session) GetWorkspace() (*app.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ws == nil {
		return nil, fmt.Errorf("no workspace is currently open")
	}
	return s.ws, nil
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws != nil {
		err := s.ws.Close()
		s.ws = nil
		return err
	}
	return nil
}
