package main

import (
	"fmt"

	"lingo-translate/pkg/model"
	"lingo-translate/pkg/storage"
)

type QueryResult struct {
	Entries []model.TextEntry `json:"entries"`
	Total   int               `json:"total"`
}

type EntryService struct {
	session *Session
}

func NewEntryService(session *Session) *EntryService {
	return &EntryService{session: session}
}

// Files returns a summary of all files and their translation progress
func (e *EntryService) Files() ([]storage.FileSummary, error) {
	ws, err := e.session.GetWorkspace()
	if err != nil {
		return nil, err
	}
	return ws.ListFiles()
}

// Query retrieves paginated and filtered entries
func (e *EntryService) Query(q storage.EntryQuery) (QueryResult, error) {
	ws, err := e.session.GetWorkspace()
	if err != nil {
		return QueryResult{Entries: []model.TextEntry{}, Total: 0}, err
	}

	entries, total, err := ws.QueryEntries(q)
	if err != nil {
		return QueryResult{Entries: []model.TextEntry{}, Total: 0}, fmt.Errorf("failed to query entries: %w", err)
	}

	return QueryResult{
		Entries: entries,
		Total:   total,
	}, nil
}

// Update updates the target text of an entry
func (e *EntryService) Update(id, target string) error {
	ws, err := e.session.GetWorkspace()
	if err != nil {
		return err
	}
	return ws.UpdateEntry(id, target, "", "manual")
}

// Stats returns workspace metrics (total, translated, pending, percent)
func (e *EntryService) Stats() (storage.WorkspaceStats, error) {
	ws, err := e.session.GetWorkspace()
	if err != nil {
		return storage.WorkspaceStats{}, err
	}
	return ws.Stats()
}
