package bqexport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// SpoolRecord is the JSONL line format persisted by JSONLSpoolSink in non-production environments.
type SpoolRecord struct {
	Table     string         `json:"table"`
	InsertID  string         `json:"insert_id"`
	WrittenAt string         `json:"written_at"`
	Data      map[string]any `json:"data"`
}

// JSONLSpoolSink appends idempotent JSONL records per table to a local directory.
// Refuses initialization and writes when auth.IsProduction() is true.
type JSONLSpoolSink struct {
	dir  string
	mu   sync.Mutex
	seen map[string]struct{}
}

// NewJSONLSpoolSink creates a local JSONL spool sink rooted at dir.
// Fails closed when PEGASUSX_ENV=production.
func NewJSONLSpoolSink(dir string) (*JSONLSpoolSink, error) {
	if auth.IsProduction() {
		return nil, ErrLocalSpoolDisallowedInProduction
	}
	clean := strings.TrimSpace(dir)
	if clean == "" {
		return nil, fmt.Errorf("%w: empty local JSONL spool directory", ErrBigQueryConfigRequired)
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return nil, fmt.Errorf("resolve jsonl spool dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir jsonl spool dir: %w", err)
	}
	s := &JSONLSpoolSink{
		dir:  abs,
		seen: make(map[string]struct{}),
	}
	if err := s.hydrateSeenIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

// Dir returns the absolute root directory of the JSONL spool sink.
func (s *JSONLSpoolSink) Dir() string {
	if s == nil {
		return ""
	}
	return s.dir
}

// WriteBatch appends rows grouped by destination table to `<dir>/<dataset>.<table>.jsonl`,
// deduplicating identical (table, insert_id) pairs across retries and restarts.
func (s *JSONLSpoolSink) WriteBatch(ctx context.Context, rows []Row) error {
	if auth.IsProduction() {
		return ErrLocalSpoolDisallowedInProduction
	}
	if s == nil || s.dir == "" {
		return fmt.Errorf("%w: uninitialized jsonl spool sink", ErrBigQueryConfigRequired)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	byFile := make(map[string][]SpoolRecord)
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)

	for _, r := range rows {
		ds, tbl, err := splitDatasetTable(r.Table)
		if err != nil {
			return err
		}
		tableName := ds + "." + tbl
		insertID := strings.TrimSpace(r.InsertID)
		if insertID != "" {
			seenKey := tableName + "\x00" + insertID
			if _, ok := s.seen[seenKey]; ok {
				continue
			}
			s.seen[seenKey] = struct{}{}
		}
		byFile[tableName] = append(byFile[tableName], SpoolRecord{
			Table:     tableName,
			InsertID:  insertID,
			WrittenAt: nowStr,
			Data:      r.Data,
		})
	}

	for tableName, recs := range byFile {
		target := filepath.Join(s.dir, tableName+".jsonl")
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget, s.dir+string(filepath.Separator)) {
			return fmt.Errorf("%w: path escapes spool root", ErrInvalidRowTable)
		}
		f, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open spool file %s: %w", cleanTarget, err)
		}
		enc := json.NewEncoder(f)
		enc.SetEscapeHTML(false)
		var encodeErr error
		for _, rec := range recs {
			if err := enc.Encode(rec); err != nil {
				encodeErr = err
				break
			}
		}
		closeErr := f.Close()
		if encodeErr != nil {
			return fmt.Errorf("write spool record %s: %w", cleanTarget, encodeErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close spool file %s: %w", cleanTarget, closeErr)
		}
	}
	return nil
}

// Health verifies the local JSONL spool directory is writable and not running in production.
func (s *JSONLSpoolSink) Health(_ context.Context) (HealthStatus, error) {
	status := HealthStatus{
		Adapter:   "jsonl_spool",
		SpoolDir:  s.Dir(),
		CheckedAt: time.Now().UTC(),
	}
	if auth.IsProduction() {
		status.Healthy = false
		status.Detail = ErrLocalSpoolDisallowedInProduction.Error()
		return status, ErrLocalSpoolDisallowedInProduction
	}
	if s == nil || s.dir == "" {
		status.Healthy = false
		status.Detail = "spool dir unset"
		return status, ErrBigQueryConfigRequired
	}
	probe, err := os.CreateTemp(s.dir, ".healthcheck-*")
	if err != nil {
		status.Healthy = false
		status.Detail = err.Error()
		return status, err
	}
	probePath := probe.Name()
	_ = probe.Close()
	_ = os.Remove(probePath)
	status.Healthy = true
	status.Detail = "spool directory writable"
	return status, nil
}

func (s *JSONLSpoolSink) hydrateSeenIndex() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("read jsonl spool dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("open existing spool %s: %w", entry.Name(), err)
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
		for scanner.Scan() {
			var rec struct {
				Table    string `json:"table"`
				InsertID string `json:"insert_id"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &rec); err == nil {
				if rec.Table != "" && rec.InsertID != "" {
					s.seen[rec.Table+"\x00"+rec.InsertID] = struct{}{}
				}
			}
		}
		scanErr := scanner.Err()
		_ = f.Close()
		if scanErr != nil {
			return fmt.Errorf("scan existing spool %s: %w", entry.Name(), scanErr)
		}
	}
	return nil
}
