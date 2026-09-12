package platform

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleDeviceToken_UnauthorizedWithoutClaims(t *testing.T) {
	h := NewHandler(HandlerConfig{DeviceTokens: NewMemoryDeviceTokenRepository()})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-1","platform":"ios"}`))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeviceToken_PersistsIOSFCMToken(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	h := NewHandler(HandlerConfig{DeviceTokens: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-ios-token","platform":"IOS"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	tokens, err := repo.ListTokens(req.Context(), "ret-1", "RETAILER")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0] != "fcm-ios-token" {
		t.Fatalf("tokens=%v", tokens)
	}
	row, ok := repo.tokens["fcm-ios-token"]
	if !ok || row.Platform != "ios" {
		t.Fatalf("platform=%q stored=%v", row.Platform, ok)
	}
}

func TestHandleDeviceToken_RejectsEmptyToken(t *testing.T) {
	h := NewHandler(HandlerConfig{DeviceTokens: NewMemoryDeviceTokenRepository()})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"","platform":"ios"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeviceToken_FactoryActorIsHomeNodeID(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	h := NewHandler(HandlerConfig{DeviceTokens: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-fac","platform":"ios"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "staff-1", Role: auth.RoleFactory, HomeNodeID: "fac-1", SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	tokens, err := repo.ListTokens(req.Context(), "fac-1", "FACTORY")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0] != "fcm-fac" {
		t.Fatalf("tokens=%v", tokens)
	}
	staffTokens, err := repo.ListTokens(req.Context(), "staff-1", "FACTORY")
	if err != nil {
		t.Fatal(err)
	}
	if len(staffTokens) != 0 {
		t.Fatalf("keyed by staff subject=%v", staffTokens)
	}
}

func TestHandleDeviceToken_WarehouseActorIsHomeNodeID(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	h := NewHandler(HandlerConfig{DeviceTokens: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-wh","platform":"android"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "staff-1", Role: auth.RoleWarehouse, HomeNodeID: "wh-1", SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	tokens, err := repo.ListTokens(req.Context(), "wh-1", "WAREHOUSE")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0] != "fcm-wh" {
		t.Fatalf("tokens=%v", tokens)
	}
}

func TestHandleDeviceToken_AdminActorIsSupplierID(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	h := NewHandler(HandlerConfig{DeviceTokens: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-admin","platform":"android"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "staff-1", Role: auth.RoleAdmin, SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	tokens, err := repo.ListTokens(req.Context(), "sup-1", "ADMIN")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0] != "fcm-admin" {
		t.Fatalf("tokens=%v", tokens)
	}
	staffTokens, err := repo.ListTokens(req.Context(), "staff-1", "ADMIN")
	if err != nil {
		t.Fatal(err)
	}
	if len(staffTokens) != 0 {
		t.Fatalf("keyed by staff subject=%v", staffTokens)
	}
}

func TestHandleDeviceToken_NilStoreUnavailable(t *testing.T) {
	h := NewHandler(HandlerConfig{})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-nil-store","platform":"android"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "device_token_unavailable" {
		t.Fatalf("payload=%v", payload)
	}
	if payload["status"] == "ok" {
		t.Fatal("nil store must not return status ok")
	}
}

func TestHandleDeviceToken_JSONStatusOK(t *testing.T) {
	h := NewHandler(HandlerConfig{DeviceTokens: NewMemoryDeviceTokenRepository()})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-2","platform":"android"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("payload=%v", payload)
	}
}

func TestHandleDeviceToken_RejectsExpoPlatform(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	h := NewHandler(HandlerConfig{DeviceTokens: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-looking","platform":"expo"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pay-1", Role: auth.RolePayload,
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	tokens, err := repo.ListTokens(req.Context(), "pay-1", "PAYLOAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 0 {
		t.Fatalf("stored expo platform tokens=%v", tokens)
	}
}

func TestHandleDeviceToken_RejectsExponentPushToken(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	h := NewHandler(HandlerConfig{DeviceTokens: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]","platform":"android"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pay-1", Role: auth.RolePayload,
	}))
	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestListTokens_SkipsExpoShapedToken(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	if err := repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID: "pay-1", ActorRole: "PAYLOAD", Platform: "android",
		Token: "ExponentPushToken[legacy]",
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID: "pay-1", ActorRole: "PAYLOAD", Platform: "android",
		Token: "fcm-real",
	}); err != nil {
		t.Fatal(err)
	}
	tokens, err := repo.ListTokens(ctx, "pay-1", "PAYLOAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0] != "fcm-real" {
		t.Fatalf("tokens=%v", tokens)
	}
}
