package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleUnassignedSkusStockErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.stockBalancesQuery = func(context.Context, string, string) ([]StockBalanceDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/sections/unassigned-skus", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-sec",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}))
	rr := httptest.NewRecorder()
	svc.HandleUnassignedSkus(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "unassigned_skus_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["skus"]; ok {
		t.Fatal("failed unassigned SKUs must not return skus[]")
	}
}

func TestHandleSectionByIDSkusErrorFailed(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	svc.sectionSkusQuery = func(context.Context, string) ([]string, error) {
		return nil, errors.New("spanner_unavailable")
	}
	req := sectionByIDRequest(http.MethodGet, secID, owner)
	rr := httptest.NewRecorder()
	svc.HandleSectionByID(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "section_detail_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["skus"]; ok {
		t.Fatal("failed section detail must not return skus[]")
	}
	if _, ok := payload["staff_ids"]; ok {
		t.Fatal("failed section detail must not return staff_ids")
	}
}

func TestHandleSectionByIDStaffErrorFailed(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	svc.sectionStaffQuery = func(context.Context, string) ([]string, error) {
		return nil, errors.New("spanner_unavailable")
	}
	req := sectionByIDRequest(http.MethodGet, secID, owner)
	rr := httptest.NewRecorder()
	svc.HandleSectionByID(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "section_detail_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["skus"]; ok {
		t.Fatal("failed section detail must not return skus[]")
	}
}

func TestHandleSectionByIDGetHonestEmpty(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	req := sectionByIDRequest(http.MethodGet, secID, owner)
	rr := httptest.NewRecorder()
	svc.HandleSectionByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	skus, ok := payload["skus"].([]any)
	if !ok || len(skus) != 0 {
		t.Fatalf("honest empty skus=%v", payload["skus"])
	}
	staff, ok := payload["staff_ids"].([]any)
	if !ok || len(staff) != 0 {
		t.Fatalf("honest empty staff=%v", payload["staff_ids"])
	}
}

func TestHandleSectionSkusGetErrorFailed(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	svc.sectionSkusQuery = func(context.Context, string) ([]string, error) {
		return nil, errors.New("spanner_unavailable")
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sectionID", secID)
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/sections/"+secID+"/skus", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	req = req.WithContext(contextWithChi(req, rctx))
	rr := httptest.NewRecorder()
	svc.HandleSectionSkus(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "section_skus_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["skus"]; ok {
		t.Fatal("failed section SKUs GET must not return skus[]")
	}
}

func TestHandleSectionStaffGetErrorFailed(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	svc.sectionStaffQuery = func(context.Context, string) ([]string, error) {
		return nil, errors.New("spanner_unavailable")
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sectionID", secID)
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/sections/"+secID+"/staff", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	req = req.WithContext(contextWithChi(req, rctx))
	rr := httptest.NewRecorder()
	svc.HandleSectionStaff(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "section_staff_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["user_ids"]; ok {
		t.Fatal("failed section staff GET must not return user_ids")
	}
}

func newSectionFixture(t *testing.T) (*Service, string, auth.Claims) {
	t.Helper()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "sec-" + string(rune('A'+n%26))
		},
	})
	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-sec-detail",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-sec-detail")
	if err != nil {
		t.Fatal(err)
	}
	body := `{"name":"Dairy","location_id":"` + primary.LocationID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/sections", strings.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleSections(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var sec SectionDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &sec); err != nil {
		t.Fatal(err)
	}
	if sec.SectionID == "" {
		t.Fatal("missing section id")
	}
	return svc, sec.SectionID, owner
}

func sectionByIDRequest(method, sectionID string, owner auth.Claims) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sectionID", sectionID)
	req := httptest.NewRequest(method, "/v1/retailer/sections/"+sectionID, nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	return req.WithContext(contextWithChi(req, rctx))
}

func TestHandleSectionSkusPutReplaceErrorFailed(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	svc.replaceSectionSkusFn = func(context.Context, SectionDTO, []string) error {
		return errors.New("spanner_unavailable")
	}
	rr := httptest.NewRecorder()
	svc.HandleSectionSkus(rr, sectionNestedPutRequest(secID, "/skus", `{"skus":["SKU-A"]}`, owner))
	assertSectionPutFailed(t, rr, "section_skus_failed", "skus")
}

func TestHandleSectionSkusPutAddErrorFailed(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	svc.addSectionSkusFn = func(context.Context, SectionDTO, []string) error {
		return errors.New("spanner_unavailable")
	}
	rr := httptest.NewRecorder()
	svc.HandleSectionSkus(rr, sectionNestedPutRequest(secID, "/skus", `{"add":["SKU-A"]}`, owner))
	assertSectionPutFailed(t, rr, "section_skus_failed", "skus")
}

func TestHandleSectionSkusPutListAfterWriteFailed(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	svc.sectionSkusQuery = func(context.Context, string) ([]string, error) {
		return nil, errors.New("spanner_unavailable")
	}
	rr := httptest.NewRecorder()
	svc.HandleSectionSkus(rr, sectionNestedPutRequest(secID, "/skus", `{"skus":["SKU-A"]}`, owner))
	assertSectionPutFailed(t, rr, "section_skus_failed", "skus")
}

func TestHandleSectionStaffPutReplaceErrorFailed(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	svc.replaceSectionStaffFn = func(context.Context, SectionDTO, []string) error {
		return errors.New("spanner_unavailable")
	}
	rr := httptest.NewRecorder()
	svc.HandleSectionStaff(rr, sectionNestedPutRequest(secID, "/staff", `{"user_ids":["u1"]}`, owner))
	assertSectionPutFailed(t, rr, "section_staff_failed", "user_ids")
}

func TestHandleSectionSkusPutHonestOK(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	rr := httptest.NewRecorder()
	svc.HandleSectionSkus(rr, sectionNestedPutRequest(secID, "/skus", `{"skus":["SKU-A"]}`, owner))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	skus, ok := payload["skus"].([]any)
	if !ok || len(skus) != 1 || skus[0] != "SKU-A" {
		t.Fatalf("payload=%v", payload)
	}
}

func sectionNestedPutRequest(sectionID, suffix, body string, owner auth.Claims) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sectionID", sectionID)
	req := httptest.NewRequest(http.MethodPut, "/v1/retailer/sections/"+sectionID+suffix, strings.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	return req.WithContext(contextWithChi(req, rctx))
}

func assertSectionPutFailed(t *testing.T, rr *httptest.ResponseRecorder, wantErr, forbiddenKey string) {
	t.Helper()
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != wantErr {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload[forbiddenKey]; ok {
		t.Fatalf("failed PUT must not return %s", forbiddenKey)
	}
}

func TestHandleMySectionsStaffErrorFailed(t *testing.T) {
	t.Parallel()
	svc, _, owner := newSectionFixture(t)
	svc.sectionStaffQuery = func(context.Context, string) ([]string, error) {
		return nil, errors.New("spanner_unavailable")
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/me/sections", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleMySections(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "my_sections_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["items"]; ok {
		t.Fatal("failed me/sections must not return items[]")
	}
}

func TestHandleMySectionsHonestEmpty(t *testing.T) {
	t.Parallel()
	svc, _, owner := newSectionFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/me/sections", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleMySections(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Items []SectionDTO `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 0 {
		t.Fatalf("unassigned owner must not invent membership items=%+v", payload.Items)
	}
}

func TestHandleMySectionsAssignedOK(t *testing.T) {
	t.Parallel()
	svc, secID, owner := newSectionFixture(t)
	rrPut := httptest.NewRecorder()
	svc.HandleSectionStaff(rrPut, sectionNestedPutRequest(secID, "/staff", `{"user_ids":["o"]}`, owner))
	if rrPut.Code != http.StatusOK {
		t.Fatalf("staff put status=%d body=%s", rrPut.Code, rrPut.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/me/sections", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleMySections(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Items []SectionDTO `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 1 || payload.Items[0].SectionID != secID {
		t.Fatalf("items=%+v want %s", payload.Items, secID)
	}
}
