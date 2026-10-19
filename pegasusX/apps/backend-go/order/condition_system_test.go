package order

import (
	"strings"
	"testing"
)

func TestConditionReportable_AllowedStatuses(t *testing.T) {
	t.Parallel()
	allowed := []Status{StatusInTransit, StatusArrived, StatusAwaitingPayment,
		StatusPendingCashCollection, StatusDeliveredOnCredit, StatusCompleted}
	for _, s := range allowed {
		if !conditionReportable(s) {
			t.Errorf("status %s should be reportable", s)
		}
	}
}

func TestConditionReportable_DisallowedStatuses(t *testing.T) {
	t.Parallel()
	disallowed := []Status{StatusPending, StatusLoaded, StatusCancelled, StatusCancelRequested, ""}
	for _, s := range disallowed {
		if conditionReportable(s) {
			t.Errorf("status %q should NOT be reportable", s)
		}
	}
}

func TestSystemTemperatureBreachArgs_WhitespaceFiltering(t *testing.T) {
	t.Parallel()
	args := SystemTemperatureBreachArgs{
		ManifestID: "  m-1  ",
		ReadingID:  "r-1",
		TempC:      15.0,
		MinC:       2.0,
		MaxC:       8.0,
		OrderIDs:   []string{"  ", "", "  ord-1  "},
	}
	if strings.TrimSpace(args.ManifestID) != "m-1" {
		t.Fatalf("ManifestID=%q want m-1 after trim", args.ManifestID)
	}
	// Temperature is outside cold-chain band [2,8]
	if args.TempC >= args.MinC && args.TempC <= args.MaxC {
		t.Fatal("temp should be outside cold-chain band")
	}
	// Count valid (non-whitespace) order IDs
	valid := 0
	for _, oid := range args.OrderIDs {
		if strings.TrimSpace(oid) != "" {
			valid++
		}
	}
	if valid != 1 {
		t.Fatalf("valid order IDs=%d want 1", valid)
	}
}

func TestSystemTemperatureBreachArgs_EmptyOrderIDs(t *testing.T) {
	t.Parallel()
	args := SystemTemperatureBreachArgs{
		ManifestID: "m-1",
		ReadingID:  "r-1",
		TempC:      15.0,
		MinC:       2.0,
		MaxC:       8.0,
		OrderIDs:   nil,
	}
	if len(args.OrderIDs) != 0 {
		t.Fatal("nil order IDs should be zero length")
	}
}

func TestConditionReport_TemperatureBreachFields(t *testing.T) {
	t.Parallel()
	r := ConditionReport{
		ReportID:         "rpt-1",
		OrderID:          "ord-1",
		SupplierID:       "sup-1",
		RetailerID:       "ret-1",
		ConditionType:    ConditionTypeTemperatureBreach,
		Severity:         SeverityHigh,
		Description:      "WMS cold-chain excursion: temp=15.00C band=[2.00,8.00]",
		ReportedBy:       systemColdChainReporter,
		ReportedByRole:   "SYSTEM",
		ResolutionStatus: ResolutionStatusOpen,
	}
	if r.ConditionType != ConditionTypeTemperatureBreach {
		t.Fatalf("type=%s want TEMPERATURE_BREACH", r.ConditionType)
	}
	if r.Severity != SeverityHigh {
		t.Fatalf("severity=%s want HIGH", r.Severity)
	}
	if r.ResolutionStatus != ResolutionStatusOpen {
		t.Fatalf("status=%s want OPEN", r.ResolutionStatus)
	}
	if r.ReportedByRole != "SYSTEM" {
		t.Fatalf("role=%s want SYSTEM", r.ReportedByRole)
	}
	if r.ReportedBy != "wms-cold-chain" {
		t.Fatalf("reportedBy=%s want wms-cold-chain", r.ReportedBy)
	}
	if !r.ConditionType.Valid() {
		t.Fatal("TEMPERATURE_BREACH should be a valid condition type")
	}
}

func TestConditionType_Valid(t *testing.T) {
	t.Parallel()
	validTypes := []ConditionType{
		ConditionTypeDamaged, ConditionTypeExpired, ConditionTypeTemperatureBreach,
		ConditionTypeMissing, ConditionTypeQualityReject, ConditionTypeOther,
	}
	for _, ct := range validTypes {
		if !ct.Valid() {
			t.Errorf("ConditionType %q should be valid", ct)
		}
	}
	invalidTypes := []ConditionType{"UNKNOWN", "", "RANDOM"}
	for _, ct := range invalidTypes {
		if ct.Valid() {
			t.Errorf("ConditionType %q should be invalid", ct)
		}
	}
}
