package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lingo-translate/pkg/model"
	_ "modernc.org/sqlite"
)

// Storage provides SQLite-backed project and cache persistence without CGO
type Storage struct {
	db *sql.DB
}

// Open opens or creates a workspace SQLite file (.nst)
func Open(filePath string) (*Storage, error) {
	if dir := filepath.Dir(filePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for workspace: %w", err)
		}
	}

	db, err := sql.Open("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite Memory Footprint Optimization
	_, _ = db.Exec(`
		PRAGMA cache_size = -2000;
		PRAGMA temp_store = MEMORY;
		PRAGMA mmap_size = 0;
		PRAGMA synchronous = NORMAL;
	`)

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS project (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		engine TEXT NOT NULL,
		source_path TEXT NOT NULL,
		source_lang TEXT NOT NULL,
		target_lang TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS entries (
		id TEXT PRIMARY KEY,
		source TEXT NOT NULL,
		target TEXT NOT NULL DEFAULT '',
		file_path TEXT NOT NULL,
		key_path TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'untranslated',
		context TEXT NOT NULL DEFAULT '',
		translator TEXT NOT NULL DEFAULT '',
		updated_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_entries_status ON entries(status);
	CREATE INDEX IF NOT EXISTS idx_entries_file ON entries(file_path);
	CREATE INDEX IF NOT EXISTS idx_entries_file_key ON entries(file_path, key_path);

	CREATE TABLE IF NOT EXISTS tm_cache (
		hash TEXT PRIMARY KEY,
		source TEXT NOT NULL,
		target TEXT NOT NULL,
		source_lang TEXT NOT NULL,
		target_lang TEXT NOT NULL,
		service TEXT NOT NULL,
		updated_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_tm_lang ON tm_cache(source_lang, target_lang);

	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetMetadata saves or updates a key-value pair in the metadata table
func (s *Storage) SetMetadata(key, value string) error {
	query := `INSERT INTO metadata (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value;`
	_, err := s.db.Exec(query, key, value)
	return err
}

// GetMetadata retrieves a value by key from the metadata table
func (s *Storage) GetMetadata(key string) (string, bool, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM metadata WHERE key = ?;`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

// GetAllMetadata retrieves all key-value pairs from the metadata table
func (s *Storage) GetAllMetadata() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value FROM metadata ORDER BY key ASC;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		res[k] = v
	}
	return res, rows.Err()
}

func hashText(src, srcLang, tgtLang string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s", src, srcLang, tgtLang)))
	return hex.EncodeToString(sum[:])
}

// SaveProject inserts or replaces project metadata
func (s *Storage) SaveProject(p *model.Project) error {
	query := `
	INSERT INTO project (id, name, engine, source_path, source_lang, target_lang, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		engine = excluded.engine,
		source_path = excluded.source_path,
		source_lang = excluded.source_lang,
		target_lang = excluded.target_lang,
		updated_at = excluded.updated_at
	`
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	_, err := s.db.Exec(query, p.ID, p.Name, p.Engine, p.SourcePath, p.SourceLang, p.TargetLang, p.CreatedAt, p.UpdatedAt)
	return err
}

// GetProject retrieves project metadata
func (s *Storage) GetProject() (*model.Project, error) {
	row := s.db.QueryRow(`SELECT id, name, engine, source_path, source_lang, target_lang, created_at, updated_at FROM project LIMIT 1`)
	var p model.Project
	err := row.Scan(&p.ID, &p.Name, &p.Engine, &p.SourcePath, &p.SourceLang, &p.TargetLang, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// SaveEntries batch-inserts text entries into database using transaction
func (s *Storage) SaveEntries(entries []model.TextEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO entries (id, source, target, file_path, key_path, status, context, translator, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source = excluded.source,
			file_path = excluded.file_path,
			key_path = excluded.key_path,
			context = excluded.context
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, e := range entries {
		if e.UpdatedAt.IsZero() {
			e.UpdatedAt = now
		}
		if e.Status == "" {
			e.Status = model.StatusUntranslated
		}
		_, err := stmt.Exec(e.ID, e.Source, e.Target, e.FilePath, e.KeyPath, string(e.Status), e.Context, e.Translator, e.UpdatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateEntriesTargetBatch updates target, status, and translator for entries in a single transaction
func (s *Storage) UpdateEntriesTargetBatch(entries []model.TextEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		UPDATE entries 
		SET target = ?, status = ?, translator = ?, updated_at = ? 
		WHERE id = ?
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, e := range entries {
		if e.Target != "" && e.Status != model.StatusUntranslated {
			_, err := stmt.Exec(e.Target, string(e.Status), e.Translator, now, e.ID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// GetEntries returns all entries or filtered by status
func (s *Storage) GetEntries(statusFilter string) ([]model.TextEntry, error) {
	var query string
	var rows *sql.Rows
	var err error

	if statusFilter == "" || statusFilter == "all" {
		query = `SELECT id, source, target, file_path, key_path, status, context, translator, updated_at FROM entries ORDER BY file_path, key_path`
		rows, err = s.db.Query(query)
	} else {
		query = `SELECT id, source, target, file_path, key_path, status, context, translator, updated_at FROM entries WHERE status = ? ORDER BY file_path, key_path`
		rows, err = s.db.Query(query, statusFilter)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.TextEntry
	for rows.Next() {
		var e model.TextEntry
		var statusStr string
		var updatedAt sql.NullTime
		err := rows.Scan(&e.ID, &e.Source, &e.Target, &e.FilePath, &e.KeyPath, &statusStr, &e.Context, &e.Translator, &updatedAt)
		if err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			e.UpdatedAt = updatedAt.Time
		}
		e.Status = model.TranslationStatus(statusStr)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// CountEntries returns total number of entries matching status filter
func (s *Storage) CountEntries(statusFilter string) (int, error) {
	var query string
	var row *sql.Row
	if statusFilter == "" || statusFilter == "all" {
		query = `SELECT COUNT(*) FROM entries`
		row = s.db.QueryRow(query)
	} else {
		query = `SELECT COUNT(*) FROM entries WHERE status = ?`
		row = s.db.QueryRow(query, statusFilter)
	}
	var count int
	err := row.Scan(&count)
	return count, err
}

// GetEntriesPaged returns paginated entries to keep heap allocation low
func (s *Storage) GetEntriesPaged(statusFilter string, limit, offset int) ([]model.TextEntry, error) {
	var query string
	var rows *sql.Rows
	var err error

	if statusFilter == "" || statusFilter == "all" {
		query = `SELECT id, source, target, file_path, key_path, status, context, translator, updated_at FROM entries ORDER BY file_path, key_path LIMIT ? OFFSET ?`
		rows, err = s.db.Query(query, limit, offset)
	} else {
		query = `SELECT id, source, target, file_path, key_path, status, context, translator, updated_at FROM entries WHERE status = ? ORDER BY file_path, key_path LIMIT ? OFFSET ?`
		rows, err = s.db.Query(query, statusFilter, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.TextEntry
	for rows.Next() {
		var e model.TextEntry
		var statusStr string
		var updatedAt sql.NullTime
		err := rows.Scan(&e.ID, &e.Source, &e.Target, &e.FilePath, &e.KeyPath, &statusStr, &e.Context, &e.Translator, &updatedAt)
		if err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			e.UpdatedAt = updatedAt.Time
		}
		e.Status = model.TranslationStatus(statusStr)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// FileSummary contains count of total and translated entries per file
type FileSummary struct {
	Path       string `json:"path"`
	Total      int    `json:"total"`
	Translated int    `json:"translated"`
}

// EntryQuery specifies filtering and pagination for text entries
type EntryQuery struct {
	File   string `json:"file"`
	Status string `json:"status"`
	Search string `json:"search"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// WorkspaceStats contains overall translation progress metrics
type WorkspaceStats struct {
	Total      int     `json:"total"`
	Translated int     `json:"translated"`
	Pending    int     `json:"pending"`
	Percent    float64 `json:"percent"`
}

// Stats returns aggregated workspace statistics
func (s *Storage) Stats() (WorkspaceStats, error) {
	row := s.db.QueryRow(`
		SELECT 
			COUNT(*),
			COALESCE(SUM(CASE WHEN status != 'untranslated' AND target != '' THEN 1 ELSE 0 END), 0)
		FROM entries
	`)
	var total, translated int
	if err := row.Scan(&total, &translated); err != nil {
		return WorkspaceStats{}, err
	}
	pending := total - translated
	var pct float64
	if total > 0 {
		pct = float64(translated) / float64(total) * 100
	}
	return WorkspaceStats{
		Total:      total,
		Translated: translated,
		Pending:    pending,
		Percent:    pct,
	}, nil
}

// ListFiles returns summary stats for each file in the workspace
func (s *Storage) ListFiles() ([]FileSummary, error) {
	query := `
		SELECT 
			file_path,
			COUNT(*),
			COALESCE(SUM(CASE WHEN status != 'untranslated' AND target != '' THEN 1 ELSE 0 END), 0)
		FROM entries
		GROUP BY file_path
		ORDER BY file_path
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []FileSummary
	for rows.Next() {
		var fs FileSummary
		if err := rows.Scan(&fs.Path, &fs.Total, &fs.Translated); err != nil {
			return nil, err
		}
		summaries = append(summaries, fs)
	}
	if summaries == nil {
		summaries = []FileSummary{}
	}
	return summaries, rows.Err()
}

// QueryEntries returns filtered, searched, and paginated entries along with total count
func (s *Storage) QueryEntries(q EntryQuery) ([]model.TextEntry, int, error) {
	var whereClauses []string
	var args []interface{}

	if q.File != "" && q.File != "all" {
		whereClauses = append(whereClauses, "file_path = ?")
		args = append(args, q.File)
	}

	if q.Status != "" && q.Status != "all" {
		if q.Status == "translated" {
			whereClauses = append(whereClauses, "(status != 'untranslated' AND target != '')")
		} else if q.Status == "untranslated" {
			whereClauses = append(whereClauses, "(status = 'untranslated' OR target = '')")
		} else {
			whereClauses = append(whereClauses, "status = ?")
			args = append(args, q.Status)
		}
	}

	if q.Search != "" {
		whereClauses = append(whereClauses, "(source LIKE ? OR target LIKE ? OR key_path LIKE ?)")
		pattern := "%" + q.Search + "%"
		args = append(args, pattern, pattern, pattern)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE "
		for i, clause := range whereClauses {
			if i > 0 {
				whereSQL += " AND "
			}
			whereSQL += clause
		}
	}

	// 1. Get total matching count
	countQuery := "SELECT COUNT(*) FROM entries" + whereSQL
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 2. Query paginated entries
	querySQL := "SELECT id, source, target, file_path, key_path, status, context, translator, updated_at FROM entries" + whereSQL + " ORDER BY file_path, key_path"
	queryArgs := make([]interface{}, len(args))
	copy(queryArgs, args)

	if q.Limit > 0 {
		querySQL += " LIMIT ? OFFSET ?"
		queryArgs = append(queryArgs, q.Limit, q.Offset)
	}

	rows, err := s.db.Query(querySQL, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []model.TextEntry
	for rows.Next() {
		var e model.TextEntry
		var statusStr string
		var updatedAt sql.NullTime
		err := rows.Scan(&e.ID, &e.Source, &e.Target, &e.FilePath, &e.KeyPath, &statusStr, &e.Context, &e.Translator, &updatedAt)
		if err != nil {
			return nil, 0, err
		}
		if updatedAt.Valid {
			e.UpdatedAt = updatedAt.Time
		}
		e.Status = model.TranslationStatus(statusStr)
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []model.TextEntry{}
	}
	return entries, total, rows.Err()
}

// UpdateEntryTarget updates translated text and status of a single entry
func (s *Storage) UpdateEntryTarget(id, target string, status model.TranslationStatus, translator string) error {
	query := `UPDATE entries SET target = ?, status = ?, translator = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, target, string(status), translator, time.Now(), id)
	return err
}

// GetCache looks up Translation Memory cache
func (s *Storage) GetCache(source, srcLang, tgtLang string) (string, bool, error) {
	h := hashText(source, srcLang, tgtLang)
	var target string
	err := s.db.QueryRow(`SELECT target FROM tm_cache WHERE hash = ?`, h).Scan(&target)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	return target, true, nil
}

// SetCache saves a translated pair to Translation Memory
func (s *Storage) SetCache(source, target, srcLang, tgtLang, service string) error {
	h := hashText(source, srcLang, tgtLang)
	query := `
	INSERT INTO tm_cache (hash, source, target, source_lang, target_lang, service, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(hash) DO UPDATE SET
		target = excluded.target,
		service = excluded.service,
		updated_at = excluded.updated_at
	`
	_, err := s.db.Exec(query, h, source, target, srcLang, tgtLang, service, time.Now())
	return err
}

// GetPrimaryTranslator queries the most used AI model/service for translated entries
func (s *Storage) GetPrimaryTranslator() (string, error) {
	row := s.db.QueryRow(`
		SELECT translator, COUNT(*) as cnt 
		FROM entries 
		WHERE status != 'untranslated' AND translator != '' AND translator != 'manual' AND translator != 'tm_cache'
		GROUP BY translator 
		ORDER BY cnt DESC 
		LIMIT 1
	`)
	var trans string
	var cnt int
	if err := row.Scan(&trans, &cnt); err == nil && trans != "" {
		return trans, nil
	}

	// Fallback to any non-empty translator
	row2 := s.db.QueryRow(`SELECT translator FROM entries WHERE translator != '' LIMIT 1`)
	if err2 := row2.Scan(&trans); err2 == nil {
		return trans, nil
	}

	return "", nil
}

