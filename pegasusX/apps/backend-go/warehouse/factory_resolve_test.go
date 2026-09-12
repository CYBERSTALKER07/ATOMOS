package warehouse

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func TestWriteFactoryResolveError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		err  error
		code int
		body string
	}{
		{proximity.ErrFactoryUnassigned, http.StatusUnprocessableEntity, "factory_unassigned"},
		{auth.ErrGeographyIncomplete, http.StatusUnprocessableEntity, "geography_incomplete"},
		{auth.ErrCrossMarketDeferred, http.StatusUnprocessableEntity, "cross_market_deferred"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		writeFactoryResolveError(rr, tc.err)
		if rr.Code != tc.code {
			t.Fatalf("%v status=%d", tc.err, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), tc.body) {
			t.Fatalf("body=%s want %s", rr.Body.String(), tc.body)
		}
	}
	rr := httptest.NewRecorder()
	writeFactoryResolveError(rr, errTransferNotFound)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
}
