package promotion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestCreateCampaignRejectsNonPositiveBudgetBeforeStore(t *testing.T) {
	_, err := CreateCampaign(context.Background(), nil, "sup-1", "Weekend", 0)
	if err == nil || !strings.Contains(err.Error(), "budget must be greater than 0") {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateCampaignRejectsNilStore(t *testing.T) {
	_, err := CreateCampaign(context.Background(), nil, "sup-1", "Weekend", 100)
	if err == nil || err.Error() != "campaign_store_unavailable" {
		t.Fatalf("err = %v", err)
	}
}

func TestHandleCreateCampaignUnavailableWithoutStore(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/campaigns", strings.NewReader(`{"name":"Weekend","budget_limit_minor":100}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-1", Subject: "user-1"}))
	rr := httptest.NewRecorder()
	svc.HandleCreateCampaign(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body %s", rr.Code, rr.Body.String())
	}
}

func TestHandleEnrollRetailerIgnoresBodyRetailer(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/promotions/enroll", strings.NewReader(`{"campaign_id":"camp-1","retailer_id":"someone-else"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()
	svc.HandleEnrollRetailer(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body %s", rr.Code, rr.Body.String())
	}
}
