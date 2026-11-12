package payload

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSequenceLIFOReversesStops(t *testing.T) {
	got := SequenceLIFO(VehicleRoute{VehicleID: "veh-1", LocationIDs: []string{"a", "b", "c"}})
	if len(got) != 3 || got[0] != "c" || got[1] != "b" || got[2] != "a" {
		t.Fatalf("sequence = %#v", got)
	}
}

func TestHandleGetLIFOSequence(t *testing.T) {
	rr := httptest.NewRecorder()
	(&Service{}).HandleGetLIFOSequence(rr, httptest.NewRequest(http.MethodGet, "/v1/payload/sequence/veh-1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", rr.Code, rr.Body.String())
	}
}
