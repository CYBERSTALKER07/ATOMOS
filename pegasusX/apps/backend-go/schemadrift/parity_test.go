package schemadrift

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestMigrationTableParity_spannerDDL(t *testing.T) {
	mig, ddl, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	if err := AssertMigrationTableParity(mig, ddl); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredProductTables_nonEmpty(t *testing.T) {
	if len(RequiredProductTables) < 14 {
		t.Fatalf("expected >=14 required tables, got %d", len(RequiredProductTables))
	}
	seen := map[string]bool{}
	for _, tname := range RequiredProductTables {
		if tname == "" || seen[tname] {
			t.Fatalf("bad/duplicate table %q", tname)
		}
		seen[tname] = true
	}
}

func TestRequiredProductTables_M02GLTablesRegisteredAndConverge(t *testing.T) {
	glTables := []string{"GlAccounts", "JournalEntries", "JournalPostings", "GlBalances", "PostingRules"}
	req := make(map[string]bool, len(RequiredProductTables))
	for _, tbl := range RequiredProductTables {
		req[tbl] = true
	}

	migDir, ddlPath, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	ddlTables, err := SpannerDDLTables(ddlPath)
	if err != nil {
		t.Fatal(err)
	}

	m02Path := filepath.Join(migDir, "20261116_m02_append_only_gl.ddl")
	m02Raw, err := os.ReadFile(m02Path)
	if err != nil {
		t.Fatalf("read %s: %v", m02Path, err)
	}
	m02Tables := make(map[string]bool)
	for _, name := range extractCreateTables(m02Raw) {
		m02Tables[name] = true
	}

	for _, tbl := range glTables {
		if !req[tbl] {
			t.Errorf("%s missing from RequiredProductTables", tbl)
		}
		if _, ok := ddlTables[tbl]; !ok {
			t.Errorf("%s missing from spanner.ddl", tbl)
		}
		if !m02Tables[tbl] {
			t.Errorf("%s missing from 20261116_m02_append_only_gl.ddl", tbl)
		}
	}

	m02Stmts := splitDDLForTest(string(m02Raw))
	if len(m02Stmts) == 0 {
		t.Fatal("expected statements in 20261116_m02_append_only_gl.ddl")
	}
	cat := ParseDatabaseDDL(m02Stmts)
	for _, tbl := range glTables {
		if !cat.hasTable(tbl) {
			t.Errorf("ParseDatabaseDDL did not register table %s", tbl)
		}
	}
	if planned := PlanDDLStatements(cat, m02Stmts); len(planned) != 0 {
		t.Fatalf("expected greenfield convergence to skip all M0.2 statements, got %d remaining: %#v", len(planned), planned)
	}
}

func TestRequiredProductTables_M04AuditRBACSodTablesRegisteredAndConverge(t *testing.T) {
	m04TablesList := []string{
		"AuditEvents",
		"AuditChainAnchors",
		"AuditLegalHolds",
		"RolePermissions",
		"SupplierUserRoles",
		"SodPolicies",
		"SodViolations",
	}
	req := make(map[string]bool, len(RequiredProductTables))
	for _, tbl := range RequiredProductTables {
		req[tbl] = true
	}

	migDir, ddlPath, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	ddlTables, err := SpannerDDLTables(ddlPath)
	if err != nil {
		t.Fatal(err)
	}

	m04Path := filepath.Join(migDir, "20261116_m04_audit_v2_rbac_sod.ddl")
	m04Raw, err := os.ReadFile(m04Path)
	if err != nil {
		t.Fatalf("read %s: %v", m04Path, err)
	}
	m04Tables := make(map[string]bool)
	for _, name := range extractCreateTables(m04Raw) {
		m04Tables[name] = true
	}

	for _, tbl := range m04TablesList {
		if !req[tbl] {
			t.Errorf("%s missing from RequiredProductTables", tbl)
		}
		if _, ok := ddlTables[tbl]; !ok {
			t.Errorf("%s missing from spanner.ddl", tbl)
		}
		if !m04Tables[tbl] {
			t.Errorf("%s missing from 20261116_m04_audit_v2_rbac_sod.ddl", tbl)
		}
	}

	m04Stmts := splitDDLForTest(string(m04Raw))
	if len(m04Stmts) == 0 {
		t.Fatal("expected statements in 20261116_m04_audit_v2_rbac_sod.ddl")
	}
	cat := ParseDatabaseDDL(m04Stmts)
	for _, tbl := range m04TablesList {
		if !cat.hasTable(tbl) {
			t.Errorf("ParseDatabaseDDL did not register table %s", tbl)
		}
	}
	if planned := PlanDDLStatements(cat, m04Stmts); len(planned) != 0 {
		t.Fatalf("expected greenfield convergence to skip all M0.4 statements, got %d remaining: %#v", len(planned), planned)
	}
}

func TestRequiredProductTables_M05BigQuerySinkCheckpointsRegisteredAndConverge(t *testing.T) {
	m05TablesList := []string{"BigQuerySinkCheckpoints"}
	req := make(map[string]bool, len(RequiredProductTables))
	for _, tbl := range RequiredProductTables {
		req[tbl] = true
	}

	migDir, ddlPath, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	ddlTables, err := SpannerDDLTables(ddlPath)
	if err != nil {
		t.Fatal(err)
	}

	m05Path := filepath.Join(migDir, "20261117_m05_ontology_data_platform.ddl")
	m05Raw, err := os.ReadFile(m05Path)
	if err != nil {
		t.Fatalf("read %s: %v", m05Path, err)
	}
	m05Tables := make(map[string]bool)
	for _, name := range extractCreateTables(m05Raw) {
		m05Tables[name] = true
	}

	for _, tbl := range m05TablesList {
		if !req[tbl] {
			t.Errorf("%s missing from RequiredProductTables", tbl)
		}
		if _, ok := ddlTables[tbl]; !ok {
			t.Errorf("%s missing from spanner.ddl", tbl)
		}
		if !m05Tables[tbl] {
			t.Errorf("%s missing from 20261117_m05_ontology_data_platform.ddl", tbl)
		}
	}

	m05Stmts := splitDDLForTest(string(m05Raw))
	if len(m05Stmts) == 0 {
		t.Fatal("expected statements in 20261117_m05_ontology_data_platform.ddl")
	}
	cat := ParseDatabaseDDL(m05Stmts)
	for _, tbl := range m05TablesList {
		if !cat.hasTable(tbl) {
			t.Errorf("ParseDatabaseDDL did not register table %s", tbl)
		}
	}
	if planned := PlanDDLStatements(cat, m05Stmts); len(planned) != 0 {
		t.Fatalf("expected greenfield convergence to skip all M0.5 statements, got %d remaining: %#v", len(planned), planned)
	}
}

func TestRequiredProductTables_Wave3AControlAccountingRegisteredAndConverge(t *testing.T) {
	wave3aTablesList := []string{
		"Periods",
		"ApprovalPolicies",
		"ApprovalRequests",
		"ApprovalDecisions",
		"ApprovalDelegations",
		"InventoryCostLayers",
		"InventoryMovements",
		"EsfDocuments",
		"EsfStatusHistory",
	}
	if len(RequiredProductTables) < 59 {
		t.Fatalf("len(RequiredProductTables) = %d, want >= 59", len(RequiredProductTables))
	}
	req := make(map[string]bool, len(RequiredProductTables))
	for _, tbl := range RequiredProductTables {
		req[tbl] = true
	}

	migDir, ddlPath, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	ddlRaw, err := os.ReadFile(ddlPath)
	if err != nil {
		t.Fatalf("read %s: %v", ddlPath, err)
	}
	ddlTables, err := SpannerDDLTables(ddlPath)
	if err != nil {
		t.Fatal(err)
	}

	w3aPath := filepath.Join(migDir, "20261117_p1_wave3a_control_accounting.ddl")
	w3aRaw, err := os.ReadFile(w3aPath)
	if err != nil {
		t.Fatalf("read %s: %v", w3aPath, err)
	}
	if strings.Contains(strings.ToUpper(string(w3aRaw)), "FLOAT64") {
		t.Fatalf("%s must not contain FLOAT64 columns", w3aPath)
	}
	w3aTables := make(map[string]bool)
	for _, name := range extractCreateTables(w3aRaw) {
		w3aTables[name] = true
	}

	for _, tbl := range wave3aTablesList {
		if !req[tbl] {
			t.Errorf("%s missing from RequiredProductTables", tbl)
		}
		if _, ok := ddlTables[tbl]; !ok {
			t.Errorf("%s missing from spanner.ddl", tbl)
		}
		if !w3aTables[tbl] {
			t.Errorf("%s missing from 20261117_p1_wave3a_control_accounting.ddl", tbl)
		}
	}

	ddlCat := ParseDatabaseDDL(splitDDLForTest(string(ddlRaw)))
	if !ddlCat.hasColumn("JournalEntries", "OriginalPeriodId") {
		t.Errorf("JournalEntries.OriginalPeriodId missing from spanner.ddl")
	}

	w3aStmts := splitDDLForTest(string(w3aRaw))
	if len(w3aStmts) == 0 {
		t.Fatal("expected statements in 20261117_p1_wave3a_control_accounting.ddl")
	}
	cat := ParseDatabaseDDL(w3aStmts)
	for _, tbl := range wave3aTablesList {
		if !cat.hasTable(tbl) {
			t.Errorf("ParseDatabaseDDL did not register table %s", tbl)
		}
	}
	if !cat.hasColumn("JournalEntries", "OriginalPeriodId") {
		t.Errorf("ParseDatabaseDDL did not register JournalEntries.OriginalPeriodId")
	}
	if planned := PlanDDLStatements(ddlCat, w3aStmts); len(planned) != 0 {
		t.Fatalf("expected greenfield convergence against spanner.ddl to skip all Wave 3A statements, got %d remaining: %#v", len(planned), planned)
	}
	if planned := PlanDDLStatements(cat, w3aStmts); len(planned) != 0 {
		t.Fatalf("expected idempotent Wave 3A self-convergence to skip all statements, got %d remaining: %#v", len(planned), planned)
	}
}

func splitDDLForTest(raw string) []string {
	var cleaned strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleaned.WriteString(line)
		cleaned.WriteString("\n")
	}
	var stmts []string
	for _, part := range strings.Split(cleaned.String(), ";") {
		if s := strings.TrimSpace(part); s != "" {
			stmts = append(stmts, s)
		}
	}
	return stmts
}

func TestCreateTableRe_skipsIF(t *testing.T) {
	raw := []byte("CREATE TABLE IF NOT EXISTS PlatformTenants (\n  X INT64\n);\nCREATE TABLE Foo (Y INT64);")
	got := extractCreateTables(raw)
	if len(got) != 2 || got[0] != "PlatformTenants" || got[1] != "Foo" {
		t.Fatalf("got %#v", got)
	}
}

func TestLiveSpanner_Wave3AControlAccountingSchemaParity(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner Wave 3A schema parity test")
	}

	project := strings.TrimSpace(os.Getenv("SPANNER_PROJECT"))
	if project == "" {
		project = "pegasusx-local"
	}
	instance := strings.TrimSpace(os.Getenv("SPANNER_INSTANCE"))
	if instance == "" {
		instance = "pegasusx-instance"
	}
	database := strings.TrimSpace(os.Getenv("SPANNER_DATABASE"))
	if database == "" {
		database = "pegasusx-db"
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("new spanner client (%s): %v", dbPath, err)
	}
	defer client.Close()

	if err := AssertLiveSchema(ctx, client); err != nil {
		t.Fatalf("AssertLiveSchema failed: %v", err)
	}

	ro := client.ReadOnlyTransaction()
	defer ro.Close()

	// Verify JournalEntries.OriginalPeriodId exists and is nullable STRING(16).
	colIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT IS_NULLABLE, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME = 'JournalEntries' AND COLUMN_NAME = 'OriginalPeriodId'`,
	})
	defer colIter.Stop()
	colRow, err := colIter.Next()
	if err != nil {
		t.Fatalf("JournalEntries.OriginalPeriodId lookup failed: %v", err)
	}
	var isNullable, spannerType string
	if err := colRow.Columns(&isNullable, &spannerType); err != nil {
		t.Fatalf("decode OriginalPeriodId metadata: %v", err)
	}
	if isNullable != "YES" || spannerType != "STRING(16)" {
		t.Fatalf("JournalEntries.OriginalPeriodId = (%s, %s), want (YES, STRING(16))", isNullable, spannerType)
	}

	// Verify zero FLOAT64 columns across all 9 Wave 3A tables.
	floatIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME IN (
		        'Periods', 'ApprovalPolicies', 'ApprovalRequests',
		        'ApprovalDecisions', 'ApprovalDelegations',
		        'InventoryCostLayers', 'InventoryMovements',
		        'EsfDocuments', 'EsfStatusHistory'
		      ) AND UPPER(SPANNER_TYPE) LIKE '%FLOAT%'`,
	})
	defer floatIter.Stop()
	if row, err := floatIter.Next(); err != iterator.Done {
		if err != nil {
			t.Fatalf("float check query failed: %v", err)
		}
		var tbl, col, typ string
		_ = row.Columns(&tbl, &col, &typ)
		t.Fatalf("forbidden float column in Wave 3A table: %s.%s (%s)", tbl, col, typ)
	}

	// Verify required Wave 3A indexes exist.
	wantIndexes := map[string]bool{
		"Idx_ApprovalRequests_ByTenantStatus": true,
		"Idx_InventoryCostLayers_FEFO":        true,
		"UQ_InventoryMovements_Idempotency":   true,
		"Idx_EsfDocuments_ByOrder":            true,
		"Idx_EsfDocuments_ByStatus":           true,
	}
	idxIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT INDEX_NAME FROM INFORMATION_SCHEMA.INDEXES
		      WHERE INDEX_NAME IN (
		        'Idx_ApprovalRequests_ByTenantStatus',
		        'Idx_InventoryCostLayers_FEFO',
		        'UQ_InventoryMovements_Idempotency',
		        'Idx_EsfDocuments_ByOrder',
		        'Idx_EsfDocuments_ByStatus'
		      )`,
	})
	defer idxIter.Stop()
	for {
		row, err := idxIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("index check query failed: %v", err)
		}
		var idxName string
		if err := row.Columns(&idxName); err != nil {
			t.Fatalf("decode index name: %v", err)
		}
		delete(wantIndexes, idxName)
	}
	if len(wantIndexes) != 0 {
		t.Fatalf("missing Wave 3A indexes on live Spanner: %v", wantIndexes)
	}

	// Verify allow_commit_timestamp=TRUE on commit timestamp columns.
	wantCommitCols := map[string]bool{
		"Periods.SoftClosedAt":           true,
		"Periods.ClosedAt":               true,
		"Periods.ReopenedAt":             true,
		"Periods.CreatedAt":              true,
		"Periods.UpdatedAt":              true,
		"ApprovalPolicies.CreatedAt":     true,
		"ApprovalPolicies.UpdatedAt":     true,
		"ApprovalRequests.ResolvedAt":    true,
		"ApprovalRequests.CreatedAt":     true,
		"ApprovalRequests.UpdatedAt":     true,
		"ApprovalDecisions.CreatedAt":    true,
		"ApprovalDelegations.CreatedAt":  true,
		"InventoryCostLayers.ReceivedAt": true,
		"InventoryCostLayers.CreatedAt":  true,
		"InventoryCostLayers.UpdatedAt":  true,
		"InventoryMovements.CreatedAt":   true,
		"EsfDocuments.IssuedAt":          true,
		"EsfDocuments.AcceptedAt":        true,
		"EsfDocuments.CreatedAt":         true,
		"EsfDocuments.UpdatedAt":         true,
		"EsfStatusHistory.CreatedAt":     true,
	}
	optIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, OPTION_VALUE
		      FROM INFORMATION_SCHEMA.COLUMN_OPTIONS
		      WHERE OPTION_NAME = 'allow_commit_timestamp'
		        AND TABLE_NAME IN (
		          'Periods', 'ApprovalPolicies', 'ApprovalRequests',
		          'ApprovalDecisions', 'ApprovalDelegations',
		          'InventoryCostLayers', 'InventoryMovements',
		          'EsfDocuments', 'EsfStatusHistory'
		        )`,
	})
	defer optIter.Stop()
	for {
		row, err := optIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("column options query failed: %v", err)
		}
		var tbl, col, val string
		if err := row.Columns(&tbl, &col, &val); err != nil {
			t.Fatalf("decode column option: %v", err)
		}
		if strings.EqualFold(val, "TRUE") {
			delete(wantCommitCols, tbl+"."+col)
		}
	}
	if len(wantCommitCols) != 0 {
		t.Fatalf("missing allow_commit_timestamp=TRUE on commit timestamp columns: %v", wantCommitCols)
	}
}

func TestRequiredProductTables_Wave3BIntelProposalsRoiRegisteredAndConverge(t *testing.T) {
	wave3bTablesList := []string{
		"ForecastRuns",
		"ForecastValues",
		"ForecastExplanations",
		"CreditScoreHistory",
		"Proposals",
		"RoiBaselines",
		"RoiEvents",
	}
	if len(RequiredProductTables) < 66 {
		t.Fatalf("len(RequiredProductTables) = %d, want >= 66", len(RequiredProductTables))
	}
	req := make(map[string]bool, len(RequiredProductTables))
	for _, tbl := range RequiredProductTables {
		req[tbl] = true
	}

	migDir, ddlPath, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	ddlRaw, err := os.ReadFile(ddlPath)
	if err != nil {
		t.Fatalf("read %s: %v", ddlPath, err)
	}
	ddlTables, err := SpannerDDLTables(ddlPath)
	if err != nil {
		t.Fatal(err)
	}

	w3bPath := filepath.Join(migDir, "20261117_p1_wave3b_intel_proposals_roi.ddl")
	w3bRaw, err := os.ReadFile(w3bPath)
	if err != nil {
		t.Fatalf("read %s: %v", w3bPath, err)
	}
	if strings.Contains(strings.ToUpper(string(w3bRaw)), "FLOAT64") {
		t.Fatalf("%s must not contain FLOAT64 columns", w3bPath)
	}
	w3bTables := make(map[string]bool)
	for _, name := range extractCreateTables(w3bRaw) {
		w3bTables[name] = true
	}

	for _, tbl := range wave3bTablesList {
		if !req[tbl] {
			t.Errorf("%s missing from RequiredProductTables", tbl)
		}
		if _, ok := ddlTables[tbl]; !ok {
			t.Errorf("%s missing from spanner.ddl", tbl)
		}
		if !w3bTables[tbl] {
			t.Errorf("%s missing from 20261117_p1_wave3b_intel_proposals_roi.ddl", tbl)
		}
	}

	ddlCat := ParseDatabaseDDL(splitDDLForTest(string(ddlRaw)))
	if !ddlCat.hasColumn("RetailerPaymentTerms", "ConsentVerified") {
		t.Errorf("RetailerPaymentTerms.ConsentVerified missing from spanner.ddl")
	}
	w3bStmts := splitDDLForTest(string(w3bRaw))
	if len(w3bStmts) == 0 {
		t.Fatal("expected statements in 20261117_p1_wave3b_intel_proposals_roi.ddl")
	}
	cat := ParseDatabaseDDL(w3bStmts)
	for _, tbl := range wave3bTablesList {
		if !cat.hasTable(tbl) {
			t.Errorf("ParseDatabaseDDL did not register table %s", tbl)
		}
	}
	if !cat.hasColumn("RetailerPaymentTerms", "ConsentVerified") {
		t.Errorf("ParseDatabaseDDL did not register RetailerPaymentTerms.ConsentVerified")
	}
	if planned := PlanDDLStatements(ddlCat, w3bStmts); len(planned) != 0 {
		t.Fatalf("expected greenfield convergence against spanner.ddl to skip all Wave 3B statements, got %d remaining: %#v", len(planned), planned)
	}
	if planned := PlanDDLStatements(cat, w3bStmts); len(planned) != 0 {
		t.Fatalf("expected idempotent Wave 3B self-convergence to skip all statements, got %d remaining: %#v", len(planned), planned)
	}
}

func TestLiveSpanner_Wave3BIntelProposalsRoiSchemaParity(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner Wave 3B schema parity test")
	}

	project := strings.TrimSpace(os.Getenv("SPANNER_PROJECT"))
	if project == "" {
		project = "pegasusx-local"
	}
	instance := strings.TrimSpace(os.Getenv("SPANNER_INSTANCE"))
	if instance == "" {
		instance = "pegasusx-instance"
	}
	database := strings.TrimSpace(os.Getenv("SPANNER_DATABASE"))
	if database == "" {
		database = "pegasusx-db"
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("new spanner client (%s): %v", dbPath, err)
	}
	defer client.Close()

	if err := AssertLiveSchema(ctx, client); err != nil {
		t.Fatalf("AssertLiveSchema failed: %v", err)
	}

	ro := client.ReadOnlyTransaction()
	defer ro.Close()

	// Verify RetailerPaymentTerms.ConsentVerified exists and is non-null BOOL.
	colIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT IS_NULLABLE, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME = 'RetailerPaymentTerms' AND COLUMN_NAME = 'ConsentVerified'`,
	})
	defer colIter.Stop()
	colRow, err := colIter.Next()
	if err != nil {
		t.Fatalf("RetailerPaymentTerms.ConsentVerified lookup failed: %v", err)
	}
	var isNullable, spannerType string
	if err := colRow.Columns(&isNullable, &spannerType); err != nil {
		t.Fatalf("decode ConsentVerified metadata: %v", err)
	}
	if isNullable != "NO" || spannerType != "BOOL" {
		t.Fatalf("RetailerPaymentTerms.ConsentVerified = (%s, %s), want (NO, BOOL)", isNullable, spannerType)
	}

	// Verify zero FLOAT columns across all 7 Wave 3B tables.
	floatIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME IN (
		        'ForecastRuns', 'ForecastValues', 'ForecastExplanations',
		        'CreditScoreHistory', 'Proposals', 'RoiBaselines', 'RoiEvents'
		      ) AND UPPER(SPANNER_TYPE) LIKE '%FLOAT%'`,
	})
	defer floatIter.Stop()
	if row, err := floatIter.Next(); err != iterator.Done {
		if err != nil {
			t.Fatalf("float check query failed: %v", err)
		}
		var tbl, col, typ string
		_ = row.Columns(&tbl, &col, &typ)
		t.Fatalf("forbidden float column in Wave 3B table: %s.%s (%s)", tbl, col, typ)
	}

	// Verify required Wave 3B indexes exist.
	wantIndexes := map[string]bool{
		"Idx_ForecastRuns_ByEntityCreated":            true,
		"Idx_ForecastValues_ByEntitySkuLocationDate":  true,
		"Idx_ForecastExplanations_ByEntitySkuCreated": true,
		"Idx_CreditScoreHistory_ByRetailerComputed":   true,
		"Idx_CreditScoreHistory_BySupplierRetailer":   true,
		"UQ_Proposals_Idempotency":                    true,
		"Idx_Proposals_ByStatusAgent":                 true,
		"Idx_Proposals_ByTarget":                      true,
		"Idx_RoiBaselines_ByTenantCreated":            true,
		"UQ_RoiEvents_Idempotency":                    true,
		"Idx_RoiEvents_ByCategoryOccurred":            true,
	}
	idxIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT INDEX_NAME FROM INFORMATION_SCHEMA.INDEXES
		      WHERE INDEX_NAME IN (
		        'Idx_ForecastRuns_ByEntityCreated',
		        'Idx_ForecastValues_ByEntitySkuLocationDate',
		        'Idx_ForecastExplanations_ByEntitySkuCreated',
		        'Idx_CreditScoreHistory_ByRetailerComputed',
		        'Idx_CreditScoreHistory_BySupplierRetailer',
		        'UQ_Proposals_Idempotency',
		        'Idx_Proposals_ByStatusAgent',
		        'Idx_Proposals_ByTarget',
		        'Idx_RoiBaselines_ByTenantCreated',
		        'UQ_RoiEvents_Idempotency',
		        'Idx_RoiEvents_ByCategoryOccurred'
		      )`,
	})
	defer idxIter.Stop()
	for {
		row, err := idxIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("index check query failed: %v", err)
		}
		var idxName string
		if err := row.Columns(&idxName); err != nil {
			t.Fatalf("decode index name: %v", err)
		}
		delete(wantIndexes, idxName)
	}
	if len(wantIndexes) != 0 {
		t.Fatalf("missing Wave 3B indexes on live Spanner: %v", wantIndexes)
	}

	// Verify allow_commit_timestamp=TRUE on commit timestamp columns.
	wantCommitCols := map[string]bool{
		"ForecastRuns.StartedAt":          true,
		"ForecastRuns.CompletedAt":        true,
		"ForecastRuns.CreatedAt":          true,
		"ForecastRuns.UpdatedAt":          true,
		"ForecastValues.CreatedAt":        true,
		"ForecastExplanations.ApprovedAt": true,
		"ForecastExplanations.CreatedAt":  true,
		"ForecastExplanations.UpdatedAt":  true,
		"CreditScoreHistory.ComputedAt":   true,
		"CreditScoreHistory.CreatedAt":    true,
		"Proposals.DecidedAt":             true,
		"Proposals.ExecutedAt":            true,
		"Proposals.CreatedAt":             true,
		"Proposals.UpdatedAt":             true,
		"RoiBaselines.CreatedAt":          true,
		"RoiBaselines.UpdatedAt":          true,
		"RoiEvents.CreatedAt":             true,
	}
	optIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, OPTION_VALUE
		      FROM INFORMATION_SCHEMA.COLUMN_OPTIONS
		      WHERE OPTION_NAME = 'allow_commit_timestamp'
		        AND TABLE_NAME IN (
		          'ForecastRuns', 'ForecastValues', 'ForecastExplanations',
		          'CreditScoreHistory', 'Proposals', 'RoiBaselines', 'RoiEvents'
		        )`,
	})
	defer optIter.Stop()
	for {
		row, err := optIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("column options query failed: %v", err)
		}
		var tbl, col, val string
		if err := row.Columns(&tbl, &col, &val); err != nil {
			t.Fatalf("decode column option: %v", err)
		}
		if strings.EqualFold(val, "TRUE") {
			delete(wantCommitCols, tbl+"."+col)
		}
	}
	if len(wantCommitCols) != 0 {
		t.Fatalf("missing allow_commit_timestamp=TRUE on Wave 3B commit timestamp columns: %v", wantCommitCols)
	}
}

func TestRequiredProductTables_Wave3CConnectorsAuditorPackRegisteredAndConverge(t *testing.T) {
	wave3cTablesList := []string{
		"ExternalCredentials",
		"ConnectorSyncRuns",
		"ConnectorEntityMaps",
		"ExportPacks",
	}
	if len(RequiredProductTables) < 70 {
		t.Fatalf("len(RequiredProductTables) = %d, want >= 70", len(RequiredProductTables))
	}
	req := make(map[string]bool, len(RequiredProductTables))
	for _, tbl := range RequiredProductTables {
		req[tbl] = true
	}

	migDir, ddlPath, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	ddlRaw, err := os.ReadFile(ddlPath)
	if err != nil {
		t.Fatalf("read %s: %v", ddlPath, err)
	}
	ddlTables, err := SpannerDDLTables(ddlPath)
	if err != nil {
		t.Fatal(err)
	}

	w3cPath := filepath.Join(migDir, "20261117_p1_wave3c_connectors_auditor_pack.ddl")
	w3cRaw, err := os.ReadFile(w3cPath)
	if err != nil {
		t.Fatalf("read %s: %v", w3cPath, err)
	}
	if strings.Contains(strings.ToUpper(string(w3cRaw)), "FLOAT64") {
		t.Fatalf("%s must not contain FLOAT64 columns", w3cPath)
	}
	w3cTables := make(map[string]bool)
	for _, name := range extractCreateTables(w3cRaw) {
		w3cTables[name] = true
	}

	for _, tbl := range wave3cTablesList {
		if !req[tbl] {
			t.Errorf("%s missing from RequiredProductTables", tbl)
		}
		if _, ok := ddlTables[tbl]; !ok {
			t.Errorf("%s missing from spanner.ddl", tbl)
		}
		if !w3cTables[tbl] {
			t.Errorf("%s missing from 20261117_p1_wave3c_connectors_auditor_pack.ddl", tbl)
		}
	}

	ddlCat := ParseDatabaseDDL(splitDDLForTest(string(ddlRaw)))
	if !ddlCat.hasColumn("SupplierUserRoles", "ExpiresAt") {
		t.Errorf("SupplierUserRoles.ExpiresAt missing from spanner.ddl")
	}
	w3cStmts := splitDDLForTest(string(w3cRaw))
	if len(w3cStmts) == 0 {
		t.Fatal("expected statements in 20261117_p1_wave3c_connectors_auditor_pack.ddl")
	}
	cat := ParseDatabaseDDL(w3cStmts)
	for _, tbl := range wave3cTablesList {
		if !cat.hasTable(tbl) {
			t.Errorf("ParseDatabaseDDL did not register table %s", tbl)
		}
	}
	if !cat.hasColumn("SupplierUserRoles", "ExpiresAt") {
		t.Errorf("ParseDatabaseDDL did not register SupplierUserRoles.ExpiresAt")
	}
	if planned := PlanDDLStatements(ddlCat, w3cStmts); len(planned) != 0 {
		t.Fatalf("expected greenfield convergence against spanner.ddl to skip all Wave 3C statements, got %d remaining: %#v", len(planned), planned)
	}
	if planned := PlanDDLStatements(cat, w3cStmts); len(planned) != 0 {
		t.Fatalf("expected idempotent Wave 3C self-convergence to skip all statements, got %d remaining: %#v", len(planned), planned)
	}
}

func TestLiveSpanner_Wave3CConnectorsAuditorPackSchemaParity(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner Wave 3C schema parity test")
	}

	project := strings.TrimSpace(os.Getenv("SPANNER_PROJECT"))
	if project == "" {
		project = "pegasusx-local"
	}
	instance := strings.TrimSpace(os.Getenv("SPANNER_INSTANCE"))
	if instance == "" {
		instance = "pegasusx-instance"
	}
	database := strings.TrimSpace(os.Getenv("SPANNER_DATABASE"))
	if database == "" {
		database = "pegasusx-db"
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("new spanner client (%s): %v", dbPath, err)
	}
	defer client.Close()

	if err := AssertLiveSchema(ctx, client); err != nil {
		t.Fatalf("AssertLiveSchema failed: %v", err)
	}

	ro := client.ReadOnlyTransaction()
	defer ro.Close()

	// Verify SupplierUserRoles.ExpiresAt exists and is nullable TIMESTAMP.
	colIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT IS_NULLABLE, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME = 'SupplierUserRoles' AND COLUMN_NAME = 'ExpiresAt'`,
	})
	defer colIter.Stop()
	colRow, err := colIter.Next()
	if err != nil {
		t.Fatalf("SupplierUserRoles.ExpiresAt lookup failed: %v", err)
	}
	var isNullable, spannerType string
	if err := colRow.Columns(&isNullable, &spannerType); err != nil {
		t.Fatalf("decode ExpiresAt metadata: %v", err)
	}
	if isNullable != "YES" || spannerType != "TIMESTAMP" {
		t.Fatalf("SupplierUserRoles.ExpiresAt = (%s, %s), want (YES, TIMESTAMP)", isNullable, spannerType)
	}

	// Verify zero FLOAT columns across all Wave 3C tables and SupplierUserRoles.
	floatIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME IN (
		        'ExternalCredentials', 'ConnectorSyncRuns',
		        'ConnectorEntityMaps', 'ExportPacks', 'SupplierUserRoles'
		      ) AND UPPER(SPANNER_TYPE) LIKE '%FLOAT%'`,
	})
	defer floatIter.Stop()
	if row, err := floatIter.Next(); err != iterator.Done {
		if err != nil {
			t.Fatalf("float check query failed: %v", err)
		}
		var tbl, col, typ string
		_ = row.Columns(&tbl, &col, &typ)
		t.Fatalf("forbidden float column in Wave 3C table: %s.%s (%s)", tbl, col, typ)
	}

	// Verify required Wave 3C indexes exist.
	wantIndexes := map[string]bool{
		"Idx_ExternalCredentials_ByProvider":      true,
		"UQ_ConnectorSyncRuns_Idempotency":        true,
		"Idx_ConnectorSyncRuns_ByProviderStarted": true,
		"Idx_ConnectorEntityMaps_ByInternal":      true,
		"Idx_ExportPacks_ByCreated":               true,
	}
	idxIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT INDEX_NAME FROM INFORMATION_SCHEMA.INDEXES
		      WHERE INDEX_NAME IN (
		        'Idx_ExternalCredentials_ByProvider',
		        'UQ_ConnectorSyncRuns_Idempotency',
		        'Idx_ConnectorSyncRuns_ByProviderStarted',
		        'Idx_ConnectorEntityMaps_ByInternal',
		        'Idx_ExportPacks_ByCreated'
		      )`,
	})
	defer idxIter.Stop()
	for {
		row, err := idxIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("index check query failed: %v", err)
		}
		var idxName string
		if err := row.Columns(&idxName); err != nil {
			t.Fatalf("decode index name: %v", err)
		}
		delete(wantIndexes, idxName)
	}
	if len(wantIndexes) != 0 {
		t.Fatalf("missing Wave 3C indexes on live Spanner: %v", wantIndexes)
	}

	// Verify allow_commit_timestamp=TRUE on commit timestamp columns.
	wantCommitCols := map[string]bool{
		"ExternalCredentials.CreatedAt":    true,
		"ExternalCredentials.UpdatedAt":    true,
		"ConnectorSyncRuns.CreatedAt":      true,
		"ConnectorEntityMaps.LastSyncedAt": true,
		"ExportPacks.CreatedAt":            true,
	}
	optIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, OPTION_VALUE
		      FROM INFORMATION_SCHEMA.COLUMN_OPTIONS
		      WHERE OPTION_NAME = 'allow_commit_timestamp'
		        AND TABLE_NAME IN (
		          'ExternalCredentials', 'ConnectorSyncRuns',
		          'ConnectorEntityMaps', 'ExportPacks'
		        )`,
	})
	defer optIter.Stop()
	for {
		row, err := optIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("column options query failed: %v", err)
		}
		var tbl, col, val string
		if err := row.Columns(&tbl, &col, &val); err != nil {
			t.Fatalf("decode column option: %v", err)
		}
		if strings.EqualFold(val, "TRUE") {
			delete(wantCommitCols, tbl+"."+col)
		}
	}
	if len(wantCommitCols) != 0 {
		t.Fatalf("missing allow_commit_timestamp=TRUE on Wave 3C commit timestamp columns: %v", wantCommitCols)
	}
}

func TestRequiredProductTables_Wave4AEntitiesP2PBankRegisteredAndConverge(t *testing.T) {
	wave4aTablesList := []string{
		"LegalEntities",
		"IntercompanyTransfers",
		"FxRevaluationRuns",
		"ProcessTemplates",
		"ProcessInstances",
		"PurchaseOrders",
		"PurchaseOrderLines",
		"GoodsReceipts",
		"VendorInvoices",
	}
	if len(RequiredProductTables) < 79 {
		t.Fatalf("len(RequiredProductTables) = %d, want >= 79", len(RequiredProductTables))
	}
	req := make(map[string]bool, len(RequiredProductTables))
	for _, tbl := range RequiredProductTables {
		req[tbl] = true
	}

	migDir, ddlPath, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	ddlRaw, err := os.ReadFile(ddlPath)
	if err != nil {
		t.Fatalf("read %s: %v", ddlPath, err)
	}
	ddlTables, err := SpannerDDLTables(ddlPath)
	if err != nil {
		t.Fatal(err)
	}

	w4aPath := filepath.Join(migDir, "20261117_p2_wave4a_entities_p2p_bank.ddl")
	w4aRaw, err := os.ReadFile(w4aPath)
	if err != nil {
		t.Fatalf("read %s: %v", w4aPath, err)
	}
	if strings.Contains(strings.ToUpper(string(w4aRaw)), "FLOAT64") {
		t.Fatalf("%s must not contain FLOAT64 columns", w4aPath)
	}
	w4aTables := make(map[string]bool)
	for _, name := range extractCreateTables(w4aRaw) {
		w4aTables[name] = true
	}

	for _, tbl := range wave4aTablesList {
		if !req[tbl] {
			t.Errorf("%s missing from RequiredProductTables", tbl)
		}
		if _, ok := ddlTables[tbl]; !ok {
			t.Errorf("%s missing from spanner.ddl", tbl)
		}
		if !w4aTables[tbl] {
			t.Errorf("%s missing from 20261117_p2_wave4a_entities_p2p_bank.ddl", tbl)
		}
	}

	ddlCat := ParseDatabaseDDL(splitDDLForTest(string(ddlRaw)))
	if !ddlCat.hasColumn("JournalEntries", "LegalEntityId") {
		t.Errorf("JournalEntries.LegalEntityId missing from spanner.ddl")
	}
	w4aStmts := splitDDLForTest(string(w4aRaw))
	if len(w4aStmts) == 0 {
		t.Fatal("expected statements in 20261117_p2_wave4a_entities_p2p_bank.ddl")
	}
	cat := ParseDatabaseDDL(w4aStmts)
	for _, tbl := range wave4aTablesList {
		if !cat.hasTable(tbl) {
			t.Errorf("ParseDatabaseDDL did not register table %s", tbl)
		}
	}
	if !cat.hasColumn("JournalEntries", "LegalEntityId") {
		t.Errorf("ParseDatabaseDDL did not register JournalEntries.LegalEntityId")
	}
	if planned := PlanDDLStatements(ddlCat, w4aStmts); len(planned) != 0 {
		t.Fatalf("expected greenfield convergence against spanner.ddl to skip all Wave 4A statements, got %d remaining: %#v", len(planned), planned)
	}
	if planned := PlanDDLStatements(cat, w4aStmts); len(planned) != 0 {
		t.Fatalf("expected idempotent Wave 4A self-convergence to skip all statements, got %d remaining: %#v", len(planned), planned)
	}
}

func TestLiveSpanner_Wave4AEntitiesP2PBankSchemaParity(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner Wave 4A schema parity test")
	}

	project := strings.TrimSpace(os.Getenv("SPANNER_PROJECT"))
	if project == "" {
		project = "pegasusx-local"
	}
	instance := strings.TrimSpace(os.Getenv("SPANNER_INSTANCE"))
	if instance == "" {
		instance = "pegasusx-instance"
	}
	database := strings.TrimSpace(os.Getenv("SPANNER_DATABASE"))
	if database == "" {
		database = "pegasusx-db"
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("new spanner client (%s): %v", dbPath, err)
	}
	defer client.Close()

	if err := AssertLiveSchema(ctx, client); err != nil {
		t.Fatalf("AssertLiveSchema failed: %v", err)
	}

	ro := client.ReadOnlyTransaction()
	defer ro.Close()

	// Verify JournalEntries.LegalEntityId exists and is nullable STRING(64).
	colIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT IS_NULLABLE, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME = 'JournalEntries' AND COLUMN_NAME = 'LegalEntityId'`,
	})
	defer colIter.Stop()
	colRow, err := colIter.Next()
	if err != nil {
		t.Fatalf("JournalEntries.LegalEntityId lookup failed: %v", err)
	}
	var isNullable, spannerType string
	if err := colRow.Columns(&isNullable, &spannerType); err != nil {
		t.Fatalf("decode LegalEntityId metadata: %v", err)
	}
	if isNullable != "YES" || spannerType != "STRING(64)" {
		t.Fatalf("JournalEntries.LegalEntityId = (%s, %s), want (YES, STRING(64))", isNullable, spannerType)
	}

	// Verify zero FLOAT columns across all 9 Wave 4A tables and JournalEntries.
	floatIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME IN (
		        'LegalEntities', 'IntercompanyTransfers', 'FxRevaluationRuns',
		        'ProcessTemplates', 'ProcessInstances',
		        'PurchaseOrders', 'PurchaseOrderLines', 'GoodsReceipts', 'VendorInvoices',
		        'JournalEntries'
		      ) AND UPPER(SPANNER_TYPE) LIKE '%FLOAT%'`,
	})
	defer floatIter.Stop()
	if row, err := floatIter.Next(); err != iterator.Done {
		if err != nil {
			t.Fatalf("float check query failed: %v", err)
		}
		var tbl, col, typ string
		_ = row.Columns(&tbl, &col, &typ)
		t.Fatalf("forbidden float column in Wave 4A table: %s.%s (%s)", tbl, col, typ)
	}

	// Verify PurchaseOrderLines is interleaved in PurchaseOrders with ON DELETE CASCADE.
	parentIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT PARENT_TABLE_NAME, ON_DELETE_ACTION
		      FROM INFORMATION_SCHEMA.TABLES
		      WHERE TABLE_NAME = 'PurchaseOrderLines'`,
	})
	defer parentIter.Stop()
	parentRow, err := parentIter.Next()
	if err != nil {
		t.Fatalf("PurchaseOrderLines parent lookup failed: %v", err)
	}
	var parentTbl, onDelete string
	if err := parentRow.Columns(&parentTbl, &onDelete); err != nil {
		t.Fatalf("decode PurchaseOrderLines parent metadata: %v", err)
	}
	if parentTbl != "PurchaseOrders" || onDelete != "CASCADE" {
		t.Fatalf("PurchaseOrderLines parent = (%s, %s), want (PurchaseOrders, CASCADE)", parentTbl, onDelete)
	}

	// Verify required Wave 4A indexes exist.
	wantIndexes := map[string]bool{
		"Idx_LegalEntities_ByTin":              true,
		"UQ_IntercompanyTransfers_Idempotency": true,
		"UQ_FxRevaluationRuns_Idempotency":     true,
		"Idx_JournalEntries_ByEntityPeriod":    true,
		"Idx_ProcessInstances_ByObject":        true,
		"UQ_PurchaseOrders_Idempotency":        true,
		"Idx_GoodsReceipts_ByPo":               true,
		"UQ_GoodsReceipts_Idempotency":         true,
		"Idx_VendorInvoices_ByPo":              true,
		"UQ_VendorInvoices_Idempotency":        true,
	}
	idxIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT INDEX_NAME FROM INFORMATION_SCHEMA.INDEXES
		      WHERE INDEX_NAME IN (
		        'Idx_LegalEntities_ByTin',
		        'UQ_IntercompanyTransfers_Idempotency',
		        'UQ_FxRevaluationRuns_Idempotency',
		        'Idx_JournalEntries_ByEntityPeriod',
		        'Idx_ProcessInstances_ByObject',
		        'UQ_PurchaseOrders_Idempotency',
		        'Idx_GoodsReceipts_ByPo',
		        'UQ_GoodsReceipts_Idempotency',
		        'Idx_VendorInvoices_ByPo',
		        'UQ_VendorInvoices_Idempotency'
		      )`,
	})
	defer idxIter.Stop()
	for {
		row, err := idxIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("index check query failed: %v", err)
		}
		var idxName string
		if err := row.Columns(&idxName); err != nil {
			t.Fatalf("decode index name: %v", err)
		}
		delete(wantIndexes, idxName)
	}
	if len(wantIndexes) != 0 {
		t.Fatalf("missing Wave 4A indexes on live Spanner: %v", wantIndexes)
	}

	// Verify allow_commit_timestamp=TRUE on commit timestamp columns.
	wantCommitCols := map[string]bool{
		"LegalEntities.CreatedAt":         true,
		"LegalEntities.UpdatedAt":         true,
		"IntercompanyTransfers.CreatedAt": true,
		"FxRevaluationRuns.CreatedAt":     true,
		"ProcessTemplates.CreatedAt":      true,
		"ProcessTemplates.UpdatedAt":      true,
		"ProcessInstances.UpdatedAt":      true,
		"PurchaseOrders.CreatedAt":        true,
		"PurchaseOrders.UpdatedAt":        true,
		"GoodsReceipts.CreatedAt":         true,
		"VendorInvoices.CreatedAt":        true,
		"VendorInvoices.UpdatedAt":        true,
	}
	optIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, OPTION_VALUE
		      FROM INFORMATION_SCHEMA.COLUMN_OPTIONS
		      WHERE OPTION_NAME = 'allow_commit_timestamp'
		        AND TABLE_NAME IN (
		          'LegalEntities', 'IntercompanyTransfers', 'FxRevaluationRuns',
		          'ProcessTemplates', 'ProcessInstances',
		          'PurchaseOrders', 'GoodsReceipts', 'VendorInvoices'
		        )`,
	})
	defer optIter.Stop()
	for {
		row, err := optIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("column options query failed: %v", err)
		}
		var tbl, col, val string
		if err := row.Columns(&tbl, &col, &val); err != nil {
			t.Fatalf("decode column option: %v", err)
		}
		if strings.EqualFold(val, "TRUE") {
			delete(wantCommitCols, tbl+"."+col)
		}
	}
	if len(wantCommitCols) != 0 {
		t.Fatalf("missing allow_commit_timestamp=TRUE on Wave 4A commit timestamp columns: %v", wantCommitCols)
	}
}

func TestRequiredProductTables_Wave4BAnomaliesHaccpScimRegisteredAndConverge(t *testing.T) {
	wave4bTablesList := []string{
		"AnomalyCases",
		"WhatIfSimulations",
		"CustomRoles",
		"ScimIdentities",
	}
	if len(RequiredProductTables) != 83 {
		t.Fatalf("len(RequiredProductTables) = %d, want 83", len(RequiredProductTables))
	}
	req := make(map[string]bool, len(RequiredProductTables))
	for _, tbl := range RequiredProductTables {
		req[tbl] = true
	}

	migDir, ddlPath, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	ddlRaw, err := os.ReadFile(ddlPath)
	if err != nil {
		t.Fatalf("read %s: %v", ddlPath, err)
	}
	ddlTables, err := SpannerDDLTables(ddlPath)
	if err != nil {
		t.Fatal(err)
	}

	w4bPath := filepath.Join(migDir, "20261117_p2_wave4b_anomalies_haccp_scim.ddl")
	w4bRaw, err := os.ReadFile(w4bPath)
	if err != nil {
		t.Fatalf("read %s: %v", w4bPath, err)
	}
	if strings.Contains(strings.ToUpper(string(w4bRaw)), "FLOAT64") {
		t.Fatalf("%s must not contain FLOAT64 columns", w4bPath)
	}
	w4bTables := make(map[string]bool)
	for _, name := range extractCreateTables(w4bRaw) {
		w4bTables[name] = true
	}

	for _, tbl := range wave4bTablesList {
		if !req[tbl] {
			t.Errorf("%s missing from RequiredProductTables", tbl)
		}
		if _, ok := ddlTables[tbl]; !ok {
			t.Errorf("%s missing from spanner.ddl", tbl)
		}
		if !w4bTables[tbl] {
			t.Errorf("%s missing from 20261117_p2_wave4b_anomalies_haccp_scim.ddl", tbl)
		}
	}

	ddlCat := ParseDatabaseDDL(splitDDLForTest(string(ddlRaw)))
	if !ddlCat.hasColumn("Proposals", "AutonomyLevel") {
		t.Errorf("Proposals.AutonomyLevel missing from spanner.ddl")
	}
	w4bStmts := splitDDLForTest(string(w4bRaw))
	if len(w4bStmts) == 0 {
		t.Fatal("expected statements in 20261117_p2_wave4b_anomalies_haccp_scim.ddl")
	}
	cat := ParseDatabaseDDL(w4bStmts)
	for _, tbl := range wave4bTablesList {
		if !cat.hasTable(tbl) {
			t.Errorf("ParseDatabaseDDL did not register table %s", tbl)
		}
	}
	if !cat.hasColumn("Proposals", "AutonomyLevel") {
		t.Errorf("ParseDatabaseDDL did not register Proposals.AutonomyLevel")
	}
	if planned := PlanDDLStatements(ddlCat, w4bStmts); len(planned) != 0 {
		t.Fatalf("expected greenfield convergence against spanner.ddl to skip all Wave 4B statements, got %d remaining: %#v", len(planned), planned)
	}
	if planned := PlanDDLStatements(cat, w4bStmts); len(planned) != 0 {
		t.Fatalf("expected idempotent Wave 4B self-convergence to skip all statements, got %d remaining: %#v", len(planned), planned)
	}
}

func TestLiveSpanner_Wave4BAnomaliesHaccpScimSchemaParity(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner Wave 4B schema parity test")
	}

	project := strings.TrimSpace(os.Getenv("SPANNER_PROJECT"))
	if project == "" {
		project = "pegasusx-local"
	}
	instance := strings.TrimSpace(os.Getenv("SPANNER_INSTANCE"))
	if instance == "" {
		instance = "pegasusx-instance"
	}
	database := strings.TrimSpace(os.Getenv("SPANNER_DATABASE"))
	if database == "" {
		database = "pegasusx-db"
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("new spanner client (%s): %v", dbPath, err)
	}
	defer client.Close()

	if err := AssertLiveSchema(ctx, client); err != nil {
		t.Fatalf("AssertLiveSchema failed: %v", err)
	}

	ro := client.ReadOnlyTransaction()
	defer ro.Close()

	// Verify Proposals.AutonomyLevel exists and is nullable STRING(8).
	colIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT IS_NULLABLE, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME = 'Proposals' AND COLUMN_NAME = 'AutonomyLevel'`,
	})
	defer colIter.Stop()
	colRow, err := colIter.Next()
	if err != nil {
		t.Fatalf("Proposals.AutonomyLevel lookup failed: %v", err)
	}
	var isNullable, spannerType string
	if err := colRow.Columns(&isNullable, &spannerType); err != nil {
		t.Fatalf("decode AutonomyLevel metadata: %v", err)
	}
	if isNullable != "YES" || spannerType != "STRING(8)" {
		t.Fatalf("Proposals.AutonomyLevel = (%s, %s), want (YES, STRING(8))", isNullable, spannerType)
	}

	// Verify zero FLOAT columns across all 4 Wave 4B tables and Proposals.
	floatIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, SPANNER_TYPE
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME IN (
		        'AnomalyCases', 'WhatIfSimulations', 'CustomRoles', 'ScimIdentities', 'Proposals'
		      ) AND UPPER(SPANNER_TYPE) LIKE '%FLOAT%'`,
	})
	defer floatIter.Stop()
	if row, err := floatIter.Next(); err != iterator.Done {
		if err != nil {
			t.Fatalf("float check query failed: %v", err)
		}
		var tbl, col, typ string
		_ = row.Columns(&tbl, &col, &typ)
		t.Fatalf("forbidden float column in Wave 4B table: %s.%s (%s)", tbl, col, typ)
	}

	// Verify required Wave 4B indexes exist.
	wantIndexes := map[string]bool{
		"Idx_AnomalyCases_ByStatusSeverity": true,
		"Idx_WhatIfSimulations_ByCreated":   true,
		"Idx_ScimIdentities_ByUserName":     true,
	}
	idxIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT INDEX_NAME FROM INFORMATION_SCHEMA.INDEXES
		      WHERE INDEX_NAME IN (
		        'Idx_AnomalyCases_ByStatusSeverity',
		        'Idx_WhatIfSimulations_ByCreated',
		        'Idx_ScimIdentities_ByUserName'
		      )`,
	})
	defer idxIter.Stop()
	for {
		row, err := idxIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("index check query failed: %v", err)
		}
		var idxName string
		if err := row.Columns(&idxName); err != nil {
			t.Fatalf("decode index name: %v", err)
		}
		delete(wantIndexes, idxName)
	}
	if len(wantIndexes) != 0 {
		t.Fatalf("missing Wave 4B indexes on live Spanner: %v", wantIndexes)
	}

	// Verify allow_commit_timestamp=TRUE on commit timestamp columns.
	wantCommitCols := map[string]bool{
		"AnomalyCases.UpdatedAt":      true,
		"WhatIfSimulations.CreatedAt": true,
		"WhatIfSimulations.UpdatedAt": true,
		"CustomRoles.CreatedAt":       true,
		"CustomRoles.UpdatedAt":       true,
		"ScimIdentities.CreatedAt":    true,
		"ScimIdentities.UpdatedAt":    true,
	}
	optIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME, OPTION_VALUE
		      FROM INFORMATION_SCHEMA.COLUMN_OPTIONS
		      WHERE OPTION_NAME = 'allow_commit_timestamp'
		        AND TABLE_NAME IN (
		          'AnomalyCases', 'WhatIfSimulations', 'CustomRoles', 'ScimIdentities'
		        )`,
	})
	defer optIter.Stop()
	for {
		row, err := optIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("column options query failed: %v", err)
		}
		var tbl, col, val string
		if err := row.Columns(&tbl, &col, &val); err != nil {
			t.Fatalf("decode column option: %v", err)
		}
		if strings.EqualFold(val, "TRUE") {
			delete(wantCommitCols, tbl+"."+col)
		}
	}
	if len(wantCommitCols) != 0 {
		t.Fatalf("missing allow_commit_timestamp=TRUE on Wave 4B commit timestamp columns: %v", wantCommitCols)
	}
}
