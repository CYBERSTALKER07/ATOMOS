package laborcapacity

import (
	"strings"
	"testing"

	"cloud.google.com/go/civil"
)

func TestBuildUsedCapacityStatement_CivilDateTyping(t *testing.T) {
	from := civil.Date{Year: 2026, Month: 11, Day: 1}
	to := civil.Date{Year: 2026, Month: 11, Day: 30}
	tz := "Asia/Tashkent"

	stmt := buildUsedCapacityStatement(from, to, "", tz)

	// Verify parameter typing: Spanner DATE mapping requires civil.Date
	fromParam, ok := stmt.Params["From"].(civil.Date)
	if !ok {
		t.Fatalf("expected stmt.Params[\"From\"] to be civil.Date, got %T (%v)", stmt.Params["From"], stmt.Params["From"])
	}
	if fromParam != from {
		t.Fatalf("expected fromParam %v, got %v", from, fromParam)
	}

	toParam, ok := stmt.Params["To"].(civil.Date)
	if !ok {
		t.Fatalf("expected stmt.Params[\"To\"] to be civil.Date, got %T (%v)", stmt.Params["To"], stmt.Params["To"])
	}
	if toParam != to {
		t.Fatalf("expected toParam %v, got %v", to, toParam)
	}

	tzParam, ok := stmt.Params["Tz"].(string)
	if !ok || tzParam != tz {
		t.Fatalf("expected tzParam %s, got %v", tz, stmt.Params["Tz"])
	}

	// Verify SQL does not cast to string or compare DATE against STRING
	if strings.Contains(stmt.SQL, "CAST(@From AS STRING)") {
		t.Fatalf("SQL must not contain CAST(@From AS STRING): %s", stmt.SQL)
	}
	if strings.Contains(stmt.SQL, "CAST(@To AS STRING)") {
		t.Fatalf("SQL must not contain CAST(@To AS STRING): %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "DATE(PromisedBy, @Tz) >= @From") {
		t.Fatalf("SQL missing >= @From condition: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "DATE(PromisedBy, @Tz) <= @To") {
		t.Fatalf("SQL missing <= @To condition: %s", stmt.SQL)
	}
}

func TestBuildUsedCapacityStatement_WithFilterZone(t *testing.T) {
	from := civil.Date{Year: 2026, Month: 11, Day: 15}
	to := civil.Date{Year: 2026, Month: 11, Day: 15}
	tz := "UTC"
	filterZone := "881f1d4893fffff"

	stmt := buildUsedCapacityStatement(from, to, filterZone, tz)

	if !strings.Contains(stmt.SQL, "AND ZoneH3 = @Zone") {
		t.Fatalf("expected filter zone clause in SQL: %s", stmt.SQL)
	}
	zoneParam, ok := stmt.Params["Zone"].(string)
	if !ok || zoneParam != filterZone {
		t.Fatalf("expected Zone param %s, got %v", filterZone, stmt.Params["Zone"])
	}
}
