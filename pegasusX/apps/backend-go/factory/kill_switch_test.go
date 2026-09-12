package factory

import "testing"

func TestClassifyKillSwitch_LeavesManual(t *testing.T) {
	cancel, keep := ClassifyKillSwitch([]killSwitchRow{
		{TransferID: "sys-1", Source: TransferSourceThreshold, State: TransferStateCreated},
		{TransferID: "sys-2", Source: TransferSourcePredicted, State: TransferStateApproved},
		{TransferID: "man-1", Source: TransferSourceManual, State: TransferStateCreated},
		{TransferID: "sys-load", Source: TransferSourceThreshold, State: TransferStateLoading},
	})
	if len(cancel) != 2 {
		t.Fatalf("cancel=%v", cancel)
	}
	if len(keep) != 2 {
		t.Fatalf("keep=%v want manual + loading", keep)
	}
	foundManual := false
	for _, id := range keep {
		if id == "man-1" {
			foundManual = true
		}
	}
	if !foundManual {
		t.Fatal("MANUAL_EMERGENCY must survive kill switch")
	}
}

func TestTransferSLALevel(t *testing.T) {
	if transferSLALevel(10, 24) != "" {
		t.Fatal("under 1x")
	}
	if transferSLALevel(24, 24) != "WARNING" {
		t.Fatal("1x")
	}
	if transferSLALevel(36, 24) != "CRITICAL" {
		t.Fatal("1.5x")
	}
	if transferSLALevel(48, 24) != "AUTO_REROUTE" {
		t.Fatal("2x")
	}
}
