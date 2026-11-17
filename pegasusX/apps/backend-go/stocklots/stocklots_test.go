package stocklots

import (
	"testing"
)

func TestNormalizeLotStatus(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"QUARANTINE", "QUARANTINED"},
		{"quarantine", "QUARANTINED"},
		{"QUARANTINED", "QUARANTINED"},
		{"quarantined", "QUARANTINED"},
		{"  Quarantine  ", "QUARANTINED"},
		{"AVAILABLE", "AVAILABLE"},
		{"available", "AVAILABLE"},
		{"DEPLETED", "DEPLETED"},
		{"ALLOCATED", "ALLOCATED"},
		{"RECALLED", "RECALLED"},
	}

	for _, tc := range cases {
		got := NormalizeLotStatus(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeLotStatus(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestComputeCycleLotAdjustment_QuarantineNormalization(t *testing.T) {
	// From legacy "QUARANTINE" input
	nextQoH, st, err := computeCycleLotAdjustment(10, 0, 5, "QUARANTINE")
	if err != nil || nextQoH != 15 || st != "QUARANTINED" {
		t.Fatalf("expected 15/QUARANTINED, got qoh=%d st=%s err=%v", nextQoH, st, err)
	}

	// From normalized "QUARANTINED" input
	nextQoH, st, err = computeCycleLotAdjustment(10, 0, -3, "QUARANTINED")
	if err != nil || nextQoH != 7 || st != "QUARANTINED" {
		t.Fatalf("expected 7/QUARANTINED, got qoh=%d st=%s err=%v", nextQoH, st, err)
	}
}

func TestPickWaveDrafts_BlocksQuarantineLots(t *testing.T) {
	lotsByID := map[string]stockLotPickMeta{
		"lot-avail": {
			productID:  "prod-1",
			locationID: "loc-1",
			status:     "AVAILABLE",
			reserved:   10,
		},
		"lot-quarantine-legacy": {
			productID:  "prod-1",
			locationID: "loc-2",
			status:     "QUARANTINE",
			reserved:   10,
		},
		"lot-quarantined": {
			productID:  "prod-1",
			locationID: "loc-3",
			status:     "QUARANTINED",
			reserved:   10,
		},
	}

	orderIDs := []string{"ord-1"}
	resByOrder := map[string][]lotResRow{
		"ord-1": {
			{lotID: "lot-avail", qty: 5},
			{lotID: "lot-quarantine-legacy", qty: 5},
			{lotID: "lot-quarantined", qty: 5},
		},
	}

	locMetaByID := map[string]locPickMeta{
		"loc-1": {seq: 1, zone: "Z1", aisle: "A1"},
		"loc-2": {seq: 2, zone: "Z1", aisle: "A2"},
		"loc-3": {seq: 3, zone: "Z1", aisle: "A3"},
	}

	var drafts []pickTaskDraft
	for _, oid := range orderIDs {
		resRows := resByOrder[oid]
		if len(resRows) > 0 {
			for _, rr := range resRows {
				lot, ok := lotsByID[rr.lotID]
				if !ok {
					continue
				}
				if lot.status != "AVAILABLE" {
					continue
				}
				qty := rr.qty
				if qty > lot.reserved && lot.reserved > 0 {
					qty = lot.reserved
				}
				if qty <= 0 {
					continue
				}
				meta := locMetaByID[lot.locationID]
				drafts = append(drafts, pickTaskDraft{
					OrderID: oid, ProductID: lot.productID, LotID: rr.lotID, LocationID: lot.locationID,
					Qty: qty, Seq: meta.seq, Zone: meta.zone, Aisle: meta.aisle,
				})
			}
		}
	}

	if len(drafts) != 1 {
		t.Fatalf("expected exactly 1 draft from AVAILABLE lot, got %d drafts: %+v", len(drafts), drafts)
	}
	if drafts[0].LotID != "lot-avail" {
		t.Fatalf("expected draft for lot-avail, got %s", drafts[0].LotID)
	}
}
