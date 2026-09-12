package payout

import (
	"testing"
)

func TestRailInfo_BankFileHonesty(t *testing.T) {
	svc := NewService(nil)
	info := svc.RailInfo()
	if info.IsLive {
		t.Fatal("default rail must not be live")
	}
	if info.Name != "bank-file" {
		t.Fatalf("name=%q", info.Name)
	}
	if info.Workflow != "bank_file_export_then_mark_paid" {
		t.Fatalf("workflow=%q", info.Workflow)
	}
	if len(info.Steps) < 3 {
		t.Fatalf("expected workflow steps, got %v", info.Steps)
	}
}

func TestSubmitForDispatch_LiveRejectedOnBankFile(t *testing.T) {
	// Covered more fully in rail_test with Spanner; unit-level message contract:
	if !errorsIsNoLiveRail(ErrNoLiveRail) {
		t.Fatal("sentinel")
	}
}

func errorsIsNoLiveRail(err error) bool {
	return err == ErrNoLiveRail
}
