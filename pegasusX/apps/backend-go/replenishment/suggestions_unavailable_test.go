package replenishment

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleListUnavailableWithoutStore(t *testing.T) {
	api := &SuggestionsAPI{}
	req := httptest.NewRequest(http.MethodGet, "/v1/replenishment/suggestions", nil)
	rr := httptest.NewRecorder()
	api.HandleList(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body %s", rr.Code, rr.Body.String())
	}
}
