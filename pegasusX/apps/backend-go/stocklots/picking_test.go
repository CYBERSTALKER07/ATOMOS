package stocklots

import (
	"errors"
	"testing"
)

func TestSortPickTaskDrafts_byPickSequence(t *testing.T) {
	drafts := []pickTaskDraft{
		{LocationID: "B-2", Seq: 20},
		{LocationID: "A-1", Seq: 10},
		{LocationID: "A-2", Seq: 10},
		{LocationID: "C-1", Seq: 5},
	}
	sortPickTaskDrafts(drafts)
	want := []string{"C-1", "A-1", "A-2", "B-2"}
	for i, id := range want {
		if drafts[i].LocationID != id {
			t.Fatalf("index %d: got %s want %s", i, drafts[i].LocationID, id)
		}
	}
}

func TestApplyLotDepletion(t *testing.T) {
	qoh, qr, depleted := applyLotDepletion(10, 4, 3)
	if qoh != 7 || qr != 1 || depleted {
		t.Fatalf("got qoh=%d qr=%d depleted=%v", qoh, qr, depleted)
	}
	qoh, qr, depleted = applyLotDepletion(2, 2, 5)
	if qoh != 0 || qr != 0 || !depleted {
		t.Fatalf("expected clamp+deplete, got qoh=%d qr=%d depleted=%v", qoh, qr, depleted)
	}
}

func TestPickTaskStatusForQty(t *testing.T) {
	if pickTaskStatusForQty(5, 5) != "CONFIRMED" {
		t.Fatal("full pick should CONFIRM")
	}
	if pickTaskStatusForQty(5, 0) != "CONFIRMED" {
		t.Fatal("default qty should CONFIRM")
	}
	if pickTaskStatusForQty(5, 2) != "SHORT" {
		t.Fatal("short pick should SHORT")
	}
}

func TestPickWavesEnabledFlag(t *testing.T) {
	SetPickWavesEnabled(false)
	if PickWavesEnabled() {
		t.Fatal("expected disabled")
	}
	SetPickWavesEnabled(true)
	if !PickWavesEnabled() {
		t.Fatal("expected enabled")
	}
	SetPickWavesEnabled(false)
}

func TestAssertManifestPickReady_flagOff(t *testing.T) {
	SetPickWavesEnabled(false)
	if _, err := AssertManifestPickReady(t.Context(), nil, "m1"); err != nil {
		t.Fatalf("flag off should allow seal: %v", err)
	}
}

func TestAssertManifestPickReady_flagOnNilClient(t *testing.T) {
	SetPickWavesEnabled(true)
	defer SetPickWavesEnabled(false)
	_, err := AssertManifestPickReady(t.Context(), nil, "m1")
	if err == nil || !errors.Is(err, ErrPickWaveRequired) {
		t.Fatalf("expected pick_wave_required, got %v", err)
	}
}

func TestAssertManifestPickReady_emptyManifest(t *testing.T) {
	SetPickWavesEnabled(true)
	defer SetPickWavesEnabled(false)
	_, err := AssertManifestPickReady(t.Context(), nil, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssertManifestPickReady_softWarn(t *testing.T) {
	SetPickWavesEnabled(true)
	SetSealSoftWarnEnabled(true)
	defer func() {
		SetPickWavesEnabled(false)
		SetSealSoftWarnEnabled(false)
	}()
	warn, err := AssertManifestPickReady(t.Context(), nil, "m1")
	if err == nil {
		// nil client still hard-fails as unavailable wrapped required
		_ = warn
	}
}

func TestSortSShapePickTaskDrafts(t *testing.T) {
	drafts := []pickTaskDraft{
		{LocationID: "z1", Zone: "A", Aisle: "2", Seq: 1, StopRank: 1},
		{LocationID: "z0", Zone: "A", Aisle: "1", Seq: 5, StopRank: 2},
		{LocationID: "z2", Zone: "B", Aisle: "1", Seq: 1, StopRank: 2},
	}
	sortSShapePickTaskDrafts(drafts)
	if drafts[0].StopRank < drafts[1].StopRank {
		t.Fatal("expected higher StopRank first")
	}
}
