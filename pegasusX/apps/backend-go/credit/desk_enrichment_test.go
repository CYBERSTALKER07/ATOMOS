package credit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleListSupplierProfilesEnrichesRetailerMetaAndPendingCreditNotes(t *testing.T) {
	t.Parallel()

	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:           "ret-uuid-10",
			SupplierID:           "sup-10",
			CreditLimitMinor:     100000000,
			CurrentBalanceMinor:  25000000,
			AvailableCreditMinor: 75000000,
			Status:               StatusActive,
		},
	}
	svc := NewService(repo)
	svc.SetDeskMetadataLookup(func(_ context.Context, supplierID string, retailerIDs []string) (map[string]SupplierDeskMetadata, int64, error) {
		if supplierID != "sup-10" || len(retailerIDs) != 1 || retailerIDs[0] != "ret-uuid-10" {
			t.Fatalf("unexpected lookup args: supplier=%q retailers=%v", supplierID, retailerIDs)
		}
		return map[string]SupplierDeskMetadata{
			"ret-uuid-10": {
				RetailerName:           "Chorsu Market LLC",
				DisplayName:            "Chorsu Market LLC",
				TIN:                    "301999888",
				Phone:                  "+998901112233",
				PendingCreditNoteCount: 2,
			},
		}, 2, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/credit-profiles", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "admin-10",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-10",
	}))
	rr := httptest.NewRecorder()
	svc.HandleListSupplierProfiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Count                  int64 `json:"count"`
		PendingCreditNoteCount int64 `json:"pending_credit_note_count"`
		Profiles               []struct {
			RetailerID             string `json:"retailer_id"`
			RetailerName           string `json:"retailer_name"`
			DisplayName            string `json:"display_name"`
			TIN                    string `json:"tin"`
			Phone                  string `json:"phone"`
			PendingCreditNoteCount int64  `json:"pending_credit_note_count"`
		} `json:"profiles"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 1 || resp.PendingCreditNoteCount != 2 || len(resp.Profiles) != 1 {
		t.Fatalf("unexpected top-level response: %+v", resp)
	}
	p := resp.Profiles[0]
	if p.RetailerName != "Chorsu Market LLC" || p.DisplayName != "Chorsu Market LLC" || p.TIN != "301999888" || p.Phone != "+998901112233" || p.PendingCreditNoteCount != 2 {
		t.Fatalf("unexpected enriched profile: %+v", p)
	}
}
