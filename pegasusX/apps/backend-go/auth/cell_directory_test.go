package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIURLForHomeCell_Catalog(t *testing.T) {
	t.Setenv("HOME_CELL", "cell-uz")
	t.Setenv("PUBLIC_BASE_URL", "")
	if got := APIURLForHomeCell("cell-uz"); got != "https://api.pegasusx.app" {
		t.Fatalf("uz=%q", got)
	}
	if got := APIURLForHomeCell("cell-eu"); got != "https://api-eu.pegasusx.app" {
		t.Fatalf("eu=%q", got)
	}
	if got := APIURLForHomeCell("no-such-cell"); got != "" {
		t.Fatalf("unknown=%q", got)
	}
}

func TestAPIURLForHomeCell_PublicBaseOverridesCurrentCell(t *testing.T) {
	t.Setenv("HOME_CELL", "cell-uz")
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8180")
	if got := APIURLForHomeCell("cell-uz"); got != "http://localhost:8180" {
		t.Fatalf("local uz=%q", got)
	}
	if got := APIURLForHomeCell("cell-eu"); got != "https://api-eu.pegasusx.app" {
		t.Fatalf("eu must not take UZ PUBLIC_BASE_URL, got %q", got)
	}
	if got := WSURLForHomeCell("cell-uz"); got != "ws://localhost:8180" {
		t.Fatalf("ws=%q", got)
	}
}

func TestHandleListCells(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleListCells(rr, httptest.NewRequest(http.MethodGet, "/v1/platform/cells", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
	var body struct {
		Items []CellDirectoryEntry `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) < 2 {
		t.Fatalf("items=%d", len(body.Items))
	}
	var uz, eu *CellDirectoryEntry
	for i := range body.Items {
		switch body.Items[i].ID {
		case "cell-uz":
			uz = &body.Items[i]
		case "cell-eu":
			eu = &body.Items[i]
		}
	}
	if uz == nil || !uz.Live || uz.Status != CellStatusShipped {
		t.Fatalf("uz=%+v", uz)
	}
	if eu == nil || eu.Live || eu.Status != CellStatusPlanned || eu.APIHostname != "api-eu.pegasusx.app" {
		t.Fatalf("eu=%+v", eu)
	}
}

func TestHandleSession_IncludesAPIURL(t *testing.T) {
	t.Setenv("HOME_CELL", "cell-uz")
	t.Setenv("PUBLIC_BASE_URL", "")
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject: "admin-1", Role: RoleAdmin, SupplierID: "sup-1",
		HomeCell: "cell-uz", MarketCode: "UZ",
	}))
	rr := httptest.NewRecorder()
	HandleSession(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["api_url"] != "https://api.pegasusx.app" {
		t.Fatalf("api_url=%v", body["api_url"])
	}
	if body["ws_url"] != "wss://api.pegasusx.app" {
		t.Fatalf("ws_url=%v", body["ws_url"])
	}
}
