package schemadrift

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ddlBatchSize = 20

var (
	reCreateTableHead = regexp.MustCompile(`(?is)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s*\(`)
	reCreateIndexHead = regexp.MustCompile(`(?is)^CREATE\s+(?:UNIQUE\s+|NULL_FILTERED\s+)*INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\b`)
	reDropIndexHead   = regexp.MustCompile(`(?is)^DROP\s+INDEX\s+(?:IF\s+EXISTS\s+)?` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?")
	reAddColumn       = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+(.+)$`)
	reDropColumn      = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `$`)
	reAlterColumn     = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+ALTER\s+COLUMN\s+` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+(.+)$`)
	reDropConstraint  = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+DROP\s+CONSTRAINT\s+(?:IF\s+EXISTS\s+)?` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `$`)
	reAddConstraint   = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+ADD\s+CONSTRAINT\s+` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+(.+)$`)
	reConstraintDef   = regexp.MustCompile(`(?is)^CONSTRAINT\s+` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?" + `\s+(.+)$`)
	reLeadingIdent    = regexp.MustCompile(`^` + "`?" + `([A-Za-z_][A-Za-z0-9_]*)` + "`?")
)

// IsBenignDDLConflict reports whether a Spanner DDL error is safe to ignore
// during idempotent schema convergence (object already exists or already dropped).
func IsBenignDDLConflict(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.AlreadyExists:
		return true
	case codes.NotFound:
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "index not found") ||
			strings.Contains(msg, "table not found")
	case codes.FailedPrecondition:
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "duplicate name in schema") ||
			strings.Contains(msg, "duplicate") ||
			strings.Contains(msg, "index not found") ||
			strings.Contains(msg, "table not found")
	case codes.InvalidArgument:
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "duplicate") ||
			strings.Contains(msg, "already has a constraint") ||
			strings.Contains(msg, "index not found") ||
			strings.Contains(msg, "table not found")
	default:
		return false
	}
}

// SchemaCatalog captures live Spanner tables, columns, indexes, and constraints
// parsed from GetDatabaseDdl so redundant UpdateDatabaseDdl RPCs can be skipped.
type SchemaCatalog struct {
	tables      map[string]bool
	columns     map[string]map[string]string
	indexes     map[string]bool
	constraints map[string]map[string]string
}

// ParseDatabaseDDL parses statements returned by GetDatabaseDdl into a SchemaCatalog.
func ParseDatabaseDDL(statements []string) *SchemaCatalog {
	cat := &SchemaCatalog{
		tables:      make(map[string]bool),
		columns:     make(map[string]map[string]string),
		indexes:     make(map[string]bool),
		constraints: make(map[string]map[string]string),
	}
	for _, stmt := range statements {
		cat.applyParsedStatement(stmt)
	}
	return cat
}

func (c *SchemaCatalog) clone() *SchemaCatalog {
	out := &SchemaCatalog{
		tables:      make(map[string]bool, len(c.tables)),
		columns:     make(map[string]map[string]string, len(c.columns)),
		indexes:     make(map[string]bool, len(c.indexes)),
		constraints: make(map[string]map[string]string, len(c.constraints)),
	}
	for k, v := range c.tables {
		out.tables[k] = v
	}
	for k, v := range c.indexes {
		out.indexes[k] = v
	}
	for t, cols := range c.columns {
		cp := make(map[string]string, len(cols))
		for k, v := range cols {
			cp[k] = v
		}
		out.columns[t] = cp
	}
	for t, cons := range c.constraints {
		cp := make(map[string]string, len(cons))
		for k, v := range cons {
			cp[k] = v
		}
		out.constraints[t] = cp
	}
	return out
}

func (c *SchemaCatalog) applyParsedStatement(raw string) {
	stmt := strings.TrimSpace(raw)
	if stmt == "" {
		return
	}
	if m := reCreateTableHead.FindStringSubmatchIndex(stmt); m != nil {
		table := strings.ToLower(stmt[m[2]:m[3]])
		c.tables[table] = true
		openParen := m[1] - 1
		if body, ok := extractBalancedParenBody(stmt, openParen); ok {
			c.parseCreateTableBody(table, body)
		}
		return
	}
	if m := reCreateIndexHead.FindStringSubmatch(stmt); m != nil {
		c.indexes[strings.ToLower(m[1])] = true
		return
	}
	if m := reDropIndexHead.FindStringSubmatch(stmt); m != nil {
		delete(c.indexes, strings.ToLower(m[1]))
		return
	}
	if m := reAddColumn.FindStringSubmatch(stmt); m != nil {
		c.setColumn(m[1], m[2], m[3])
		return
	}
	if m := reDropColumn.FindStringSubmatch(stmt); m != nil {
		if cols := c.columns[strings.ToLower(m[1])]; cols != nil {
			delete(cols, strings.ToLower(m[2]))
		}
		return
	}
	if m := reDropConstraint.FindStringSubmatch(stmt); m != nil {
		if cons := c.constraints[strings.ToLower(m[1])]; cons != nil {
			delete(cons, strings.ToLower(m[2]))
		}
		return
	}
	if m := reAddConstraint.FindStringSubmatch(stmt); m != nil {
		c.setConstraint(m[1], m[2], m[3])
		return
	}
}

func (c *SchemaCatalog) parseCreateTableBody(table, body string) {
	for _, item := range splitTopLevelComma(body) {
		clause := strings.TrimSpace(item)
		if clause == "" {
			continue
		}
		if m := reConstraintDef.FindStringSubmatch(clause); m != nil {
			c.setConstraint(table, m[1], m[2])
			continue
		}
		upper := strings.ToUpper(clause)
		if strings.HasPrefix(upper, "PRIMARY KEY") || strings.HasPrefix(upper, "FOREIGN KEY") || strings.HasPrefix(upper, "CHECK") || strings.HasPrefix(upper, "SYNONYM") {
			continue
		}
		if m := reLeadingIdent.FindStringSubmatch(clause); m != nil {
			col := m[1]
			spec := strings.TrimSpace(clause[len(m[0]):])
			c.setColumn(table, col, spec)
		}
	}
}

func (c *SchemaCatalog) setColumn(table, col, spec string) {
	t := strings.ToLower(table)
	if c.columns[t] == nil {
		c.columns[t] = make(map[string]string)
	}
	c.columns[t][strings.ToLower(col)] = normalizeDDLExpr(spec)
}

func (c *SchemaCatalog) setConstraint(table, name, spec string) {
	t := strings.ToLower(table)
	if c.constraints[t] == nil {
		c.constraints[t] = make(map[string]string)
	}
	c.constraints[t][strings.ToLower(name)] = normalizeDDLExpr(spec)
}

func (c *SchemaCatalog) hasTable(table string) bool {
	return c.tables[strings.ToLower(table)]
}

func (c *SchemaCatalog) hasColumn(table, col string) bool {
	return c.columns[strings.ToLower(table)][strings.ToLower(col)] != ""
}

func (c *SchemaCatalog) columnMatchesAlter(table, col, alterSpec string) bool {
	existing := c.columns[strings.ToLower(table)][strings.ToLower(col)]
	if existing == "" {
		return false
	}
	norm := normalizeDDLExpr(alterSpec)
	if norm == existing {
		return true
	}
	if strings.HasPrefix(norm, "set options") {
		if idx := strings.Index(norm, "("); idx >= 0 {
			opt := strings.TrimSpace(norm[idx:])
			return strings.Contains(existing, opt)
		}
	}
	if strings.HasPrefix(norm, "set default") {
		def := strings.TrimSpace(strings.TrimPrefix(norm, "set "))
		return strings.Contains(existing, def)
	}
	return strings.HasPrefix(existing, norm)
}

// PlanDDLStatements filters out statements already satisfied in catalog or canceled
// by later statements in the same ordered slice (such as an obsolete index that is
// both created and later dropped before ever being persisted).
func PlanDDLStatements(catalog *SchemaCatalog, statements []string) []string {
	if catalog == nil {
		out := make([]string, len(statements))
		copy(out, statements)
		return out
	}

	lastCreateIdx := make(map[string]int)
	lastDropIdx := make(map[string]int)
	for i, raw := range statements {
		stmt := strings.TrimSpace(raw)
		if m := reCreateIndexHead.FindStringSubmatch(stmt); m != nil {
			lastCreateIdx[strings.ToLower(m[1])] = i
		} else if m := reDropIndexHead.FindStringSubmatch(stmt); m != nil {
			lastDropIdx[strings.ToLower(m[1])] = i
		}
	}

	work := catalog.clone()
	skipIdx := make(map[int]bool)
	planned := make([]string, 0, len(statements))

	for i, raw := range statements {
		if skipIdx[i] {
			continue
		}
		stmt := strings.TrimSpace(raw)
		if stmt == "" {
			continue
		}

		switch {
		case reCreateTableHead.MatchString(stmt):
			m := reCreateTableHead.FindStringSubmatch(stmt)
			if work.hasTable(m[1]) {
				continue
			}
			work.applyParsedStatement(stmt)
			planned = append(planned, stmt)

		case reCreateIndexHead.MatchString(stmt):
			m := reCreateIndexHead.FindStringSubmatch(stmt)
			idxName := strings.ToLower(m[1])
			if work.indexes[idxName] {
				continue
			}
			// Skip creating an index if a later migration drops it and never recreates it.
			if dropAt, ok := lastDropIdx[idxName]; ok && dropAt > i && lastCreateIdx[idxName] < dropAt {
				continue
			}
			work.indexes[idxName] = true
			planned = append(planned, stmt)

		case reDropIndexHead.MatchString(stmt):
			m := reDropIndexHead.FindStringSubmatch(stmt)
			idxName := strings.ToLower(m[1])
			if !work.indexes[idxName] {
				continue
			}
			delete(work.indexes, idxName)
			planned = append(planned, stmt)

		case reAddColumn.MatchString(stmt):
			m := reAddColumn.FindStringSubmatch(stmt)
			if work.hasColumn(m[1], m[2]) {
				continue
			}
			work.setColumn(m[1], m[2], m[3])
			planned = append(planned, stmt)

		case reDropColumn.MatchString(stmt):
			m := reDropColumn.FindStringSubmatch(stmt)
			if work.hasTable(m[1]) && !work.hasColumn(m[1], m[2]) {
				continue
			}
			work.applyParsedStatement(stmt)
			planned = append(planned, stmt)

		case reAlterColumn.MatchString(stmt):
			m := reAlterColumn.FindStringSubmatch(stmt)
			if work.columnMatchesAlter(m[1], m[2], m[3]) {
				continue
			}
			planned = append(planned, stmt)

		case reDropConstraint.MatchString(stmt):
			m := reDropConstraint.FindStringSubmatch(stmt)
			t, k := strings.ToLower(m[1]), strings.ToLower(m[2])
			existing := work.constraints[t][k]
			if existing == "" {
				continue
			}
			// If the matching ADD CONSTRAINT immediately re-applies the identical expression, skip both.
			if addAt, addSpec, ok := findMatchingAddConstraint(statements, i+1, t, k); ok && normalizeDDLExpr(addSpec) == existing {
				skipIdx[addAt] = true
				continue
			}
			delete(work.constraints[t], k)
			planned = append(planned, stmt)

		case reAddConstraint.MatchString(stmt):
			m := reAddConstraint.FindStringSubmatch(stmt)
			t, k := strings.ToLower(m[1]), strings.ToLower(m[2])
			if work.constraints[t][k] == normalizeDDLExpr(m[3]) {
				continue
			}
			work.setConstraint(t, k, m[3])
			planned = append(planned, stmt)

		default:
			planned = append(planned, stmt)
		}
	}

	return planned
}

func findMatchingAddConstraint(statements []string, start int, tableLower, nameLower string) (int, string, bool) {
	for j := start; j < len(statements); j++ {
		if m := reAddConstraint.FindStringSubmatch(strings.TrimSpace(statements[j])); m != nil {
			if strings.ToLower(m[1]) == tableLower && strings.ToLower(m[2]) == nameLower {
				return j, m[3], true
			}
		}
		if m := reDropConstraint.FindStringSubmatch(strings.TrimSpace(statements[j])); m != nil {
			if strings.ToLower(m[1]) == tableLower && strings.ToLower(m[2]) == nameLower {
				return -1, "", false
			}
		}
	}
	return -1, "", false
}

// ConvergeDDLStatements applies only unapplied DDL statements in batches to avoid
// Spanner emulator DDL-history memory bloat and unnecessary round-trips.
func ConvergeDDLStatements(
	ctx context.Context,
	admin *database.DatabaseAdminClient,
	databaseName string,
	statements []string,
) error {
	if len(statements) == 0 {
		return nil
	}

	ddlResp, err := admin.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{
		Database: databaseName,
	})
	if err != nil {
		return fmt.Errorf("get database ddl %s: %w", databaseName, err)
	}

	catalog := ParseDatabaseDDL(ddlResp.GetStatements())
	planned := PlanDDLStatements(catalog, statements)
	if len(planned) == 0 {
		slog.Info("spanner schema already converged; skipping ddl update",
			"database", databaseName,
			"candidate_statements", len(statements),
		)
		return nil
	}

	slog.Info("applying converged spanner ddl statements",
		"database", databaseName,
		"candidate_statements", len(statements),
		"planned_statements", len(planned),
	)

	for start := 0; start < len(planned); start += ddlBatchSize {
		end := start + ddlBatchSize
		if end > len(planned) {
			end = len(planned)
		}
		batch := planned[start:end]
		if err := applyDDLBatchOrFallback(ctx, admin, databaseName, batch); err != nil {
			return err
		}
	}
	return nil
}

func applyDDLBatchOrFallback(
	ctx context.Context,
	admin *database.DatabaseAdminClient,
	databaseName string,
	batch []string,
) error {
	if len(batch) > 1 {
		err := execDDLStatements(ctx, admin, databaseName, batch)
		if err == nil {
			return nil
		}
		st, _ := status.FromError(err)
		// Fall back per-statement when a batch trips idempotency or statement ordering rules.
		if !IsBenignDDLConflict(err) && st.Code() != codes.InvalidArgument && st.Code() != codes.FailedPrecondition {
			return err
		}
		slog.Info("ddl batch fell back to per-statement convergence", "batch_size", len(batch), "err", err)
	}

	for _, stmt := range batch {
		if err := execDDLStatements(ctx, admin, databaseName, []string{stmt}); err != nil {
			if IsBenignDDLConflict(err) {
				continue
			}
			return fmt.Errorf("apply statement %q: %w", truncateDDL(stmt), err)
		}
	}
	return nil
}

func execDDLStatements(
	ctx context.Context,
	admin *database.DatabaseAdminClient,
	databaseName string,
	stmts []string,
) error {
	op, err := admin.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   databaseName,
		Statements: stmts,
	})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func extractBalancedParenBody(s string, openIdx int) (string, bool) {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '(' {
		return "", false
	}
	depth := 0
	inQuote := false
	for i := openIdx; i < len(s); i++ {
		ch := s[i]
		if ch == '\'' {
			if inQuote && i+1 < len(s) && s[i+1] == '\'' {
				i++
				continue
			}
			inQuote = !inQuote
			continue
		}
		if inQuote {
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[openIdx+1 : i], true
			}
		}
	}
	return "", false
}

func splitTopLevelComma(s string) []string {
	var parts []string
	depth := 0
	inQuote := false
	start := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\'' {
			if inQuote && i+1 < len(s) && s[i+1] == '\'' {
				i++
				continue
			}
			inQuote = !inQuote
			continue
		}
		if inQuote {
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func normalizeDDLExpr(s string) string {
	fields := strings.Fields(strings.ToLower(s))
	out := strings.Join(fields, " ")
	out = strings.ReplaceAll(out, " (", "(")
	out = strings.ReplaceAll(out, "( ", "(")
	out = strings.ReplaceAll(out, " )", ")")
	out = strings.ReplaceAll(out, " ,", ",")
	out = strings.ReplaceAll(out, ", ", ",")
	out = strings.ReplaceAll(out, " = ", "=")
	return strings.TrimSpace(out)
}

func truncateDDL(stmt string) string {
	const max = 120
	trimmed := strings.TrimSpace(stmt)
	if len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max] + "..."
}
