// Package schemadrift asserts Spanner objects required by live product paths.
package schemadrift

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// createTableRe matches CREATE TABLE [IF NOT EXISTS] Name.
var createTableRe = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][A-Za-z0-9_]*)`)

func extractCreateTables(raw []byte) []string {
	var out []string
	for _, m := range createTableRe.FindAllSubmatch(raw, -1) {
		out = append(out, string(m[1]))
	}
	return out
}

// RequiredProductTables must exist after greenfield apply of schema/spanner.ddl.
// These were historically migration-only (P1-12) and break fresh emulator parity.
var RequiredProductTables = []string{
	"RouteTwins",
	"StopTwins",
	"VehicleInventory",
	"DriverScores",
	"DriverAvailability",
	"ZoneCapacity",
	"RouteETAs",
	"RetailerSellThroughDaily",
	"RetailerAutoOrderBucket",
	"RetailerAutoOrderRuns",
	"RetailerLocalCatalog",
	"RetailerOrgFlags",
	"FlywheelDemandFeed",
	"RetailerStockCountForceAudits",
	"OrderShopClosedLog",
	"PlanningScenarios",
}

// MigrationTables returns CREATE TABLE names from all *.ddl files in dir.
func MigrationTables(migrationsDir string) (map[string][]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("schemadrift: read migrations: %w", err)
	}
	out := make(map[string][]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".ddl") {
			continue
		}
		path := filepath.Join(migrationsDir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for _, name := range extractCreateTables(raw) {
			out[name] = append(out[name], e.Name())
		}
	}
	return out, nil
}

// SpannerDDLTables returns CREATE TABLE names from spanner.ddl.
func SpannerDDLTables(ddlPath string) (map[string]struct{}, error) {
	raw, err := os.ReadFile(ddlPath)
	if err != nil {
		return nil, fmt.Errorf("schemadrift: read spanner.ddl: %w", err)
	}
	out := make(map[string]struct{})
	for _, name := range extractCreateTables(raw) {
		out[name] = struct{}{}
	}
	return out, nil
}

// AssertMigrationTableParity fails when any migration CREATE TABLE is absent
// from schema/spanner.ddl (greenfield / emulator apply source of truth).
func AssertMigrationTableParity(migrationsDir, ddlPath string) error {
	mig, err := MigrationTables(migrationsDir)
	if err != nil {
		return err
	}
	ddl, err := SpannerDDLTables(ddlPath)
	if err != nil {
		return err
	}
	var missing []string
	for table, files := range mig {
		if _, ok := ddl[table]; ok {
			continue
		}
		sort.Strings(files)
		missing = append(missing, fmt.Sprintf("%s (%s)", table, strings.Join(files, ", ")))
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf(
		"schema drift: migration CREATE TABLE absent from spanner.ddl (mirror into apps/backend-go/schema/spanner.ddl): %s",
		strings.Join(missing, "; "),
	)
}

// FindSchemaPaths locates schema/migrations and schema/spanner.ddl from cwd or known roots.
func FindSchemaPaths() (migrationsDir, ddlPath string, err error) {
	candidates := []string{
		"schema",
		"apps/backend-go/schema",
		filepath.Join("..", "schema"),
		filepath.Join("..", "..", "schema"),
	}
	if wd, werr := os.Getwd(); werr == nil {
		candidates = append(candidates,
			filepath.Join(wd, "schema"),
			filepath.Join(wd, "apps", "backend-go", "schema"),
		)
	}
	for _, root := range candidates {
		mig := filepath.Join(root, "migrations")
		ddl := filepath.Join(root, "spanner.ddl")
		if st, e := os.Stat(mig); e == nil && st.IsDir() {
			if _, e2 := os.Stat(ddl); e2 == nil {
				return mig, ddl, nil
			}
		}
	}
	return "", "", fmt.Errorf("schemadrift: schema/migrations + spanner.ddl not found")
}
