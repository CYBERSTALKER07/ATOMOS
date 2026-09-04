package payload

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCanonicalBoardState(t *testing.T) {
	if got := CanonicalBoardState(" loading "); got != payloadManifestStateLoading {
		t.Fatalf("got %q", got)
	}
	if got := CanonicalBoardState("COMPLETED"); got != "" {
		t.Fatalf("COMPLETED must not be a board column, got %q", got)
	}
	if got := CanonicalBoardState(""); got != "" {
		t.Fatalf("empty must stay empty, got %q", got)
	}
}

func TestGroupBoardColumns_FourStatesNotTruckCount(t *testing.T) {
	trucks := []payloaderTruckWire{
		{ID: "v-draft", TruckStatus: "DRAFT"},
		{ID: "v-load", TruckStatus: "LOADING"},
		{ID: "v-seal", TruckStatus: "SEALED"},
		{ID: "v-disp", TruckStatus: "DISPATCHED"},
		{ID: "v-done", TruckStatus: "COMPLETED"},
		{ID: "v-none", TruckStatus: ""},
	}
	cols := GroupBoardColumns(trucks)
	if len(cols) != 4 {
		t.Fatalf("board columns=%d want 4", len(cols))
	}
	want := []string{
		payloadManifestStateDraft,
		payloadManifestStateLoading,
		payloadManifestStateSealed,
		payloadManifestStateDispatched,
	}
	for i, st := range want {
		if cols[i].State != st {
			t.Fatalf("col %d state=%q want %q", i, cols[i].State, st)
		}
		if len(cols[i].Trucks) != 1 {
			t.Fatalf("col %s trucks=%d want 1 (not a single trucks count)", st, len(cols[i].Trucks))
		}
		if cols[i].Trucks[0].ID != map[string]string{
			payloadManifestStateDraft:      "v-draft",
			payloadManifestStateLoading:    "v-load",
			payloadManifestStateSealed:     "v-seal",
			payloadManifestStateDispatched: "v-disp",
		}[st] {
			t.Fatalf("col %s unexpected truck %#v", st, cols[i].Trucks[0])
		}
	}
}

func TestHandleTrucks_TruckStatusFromManifestNotInvented(t *testing.T) {
	svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})
	svc.mu.Lock()
	svc.trucks = []TruckRow{
		{VehicleID: "veh_a", PlateNo: "01A111AA"},
		{VehicleID: "veh_b", PlateNo: "01B222BB"},
	}
	svc.manifests = []ManifestRow{
		{
			ManifestID:    "mf_a",
			VehicleID:     "veh_a",
			State:         payloadManifestStateLoading,
			TotalVolumeVU: 12,
			MaxVolumeVU:   40,
			StopCount:     3,
			UpdatedAt:     "2026-08-16T10:00:00Z",
		},
		{
			ManifestID:    "mf_done",
			VehicleID:     "veh_b",
			State:         "COMPLETED",
			TotalVolumeVU: 9,
			MaxVolumeVU:   40,
			UpdatedAt:     "2026-08-16T11:00:00Z",
		},
	}
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/payloader/trucks", nil)
	rr := httptest.NewRecorder()
	svc.HandleTrucks(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var wire []payloaderTruckWire
	if err := json.Unmarshal(rr.Body.Bytes(), &wire); err != nil {
		t.Fatalf("decode: %v", err)
	}
	byID := map[string]payloaderTruckWire{}
	for i := range wire {
		byID[wire[i].ID] = wire[i]
	}
	a := byID["veh_a"]
	if a.TruckStatus != payloadManifestStateLoading {
		t.Fatalf("veh_a truck_status=%q want LOADING", a.TruckStatus)
	}
	if a.UsedVolumeVU != 12 || a.MaxVolumeVU != 40 || a.StopCount != 3 {
		t.Fatalf("veh_a vu=%d/%d stops=%d", a.UsedVolumeVU, a.MaxVolumeVU, a.StopCount)
	}
	b := byID["veh_b"]
	if b.TruckStatus != "" {
		t.Fatalf("veh_b must not invent DRAFT from COMPLETED, got %q", b.TruckStatus)
	}
	cols := GroupBoardColumns(wire)
	if len(cols[1].Trucks) != 1 || cols[1].Trucks[0].ID != "veh_a" {
		t.Fatalf("LOADING column=%#v", cols[1].Trucks)
	}
	if len(cols[0].Trucks)+len(cols[2].Trucks)+len(cols[3].Trucks) != 0 {
		t.Fatalf("other columns must be empty, got %#v", cols)
	}
}
