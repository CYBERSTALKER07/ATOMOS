package retailer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func newTelegramTestRouter(svc *Service) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/v1/retailer/telegram/link-request", svc.HandleTelegramLinkRequest)
	r.Get("/v1/retailer/telegram/status", svc.HandleTelegramStatus)
	r.Post("/v1/retailer/telegram/verify-link", svc.HandleTelegramVerifyLink)
	r.Get("/v1/retailer/telegram/chat/{chatId}/context", svc.HandleTelegramChatContext)
	r.Patch("/v1/retailer/telegram/chat/{chatId}/locale", svc.HandleTelegramUpdateLocale)
	r.Post("/v1/retailer/telegram/chat/{chatId}/preorder/{orderId}/confirm", svc.HandleTelegramPreorderConfirm)
	r.Post("/v1/retailer/telegram/chat/{chatId}/preorder/{orderId}/reject", svc.HandleTelegramPreorderReject)
	return r
}

func TestTelegramOTPLinkFlowAndFailClosedSecret(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	repo := &testRetailerRepo{
		retailer: Retailer{RetailerID: "ret-tg-1", Name: "Baraka Market", Phone: "+998901234567"},
		found:    true,
		tracking: []TrackingOrder{
			{OrderID: "ord-tg-1", SupplierID: "sup-1", Status: "EN_ROUTE", TotalMinor: 50000000, Currency: "UZS"},
		},
	}
	ordersFake := &testOrderLifecycle{
		predictions: []order.RetailerAIPrediction{
			{
				OrderID:            "ord-ai-tg-1",
				SupplierID:         "sup-1",
				Source:             order.OrderSourceAIPreorder,
				ConfirmationStatus: order.ConfirmationStatusPending,
				TotalMinor:         25000000,
				Currency:           "UZS",
			},
		},
		response: order.RetailerOrderLifecycleResponse{
			OrderID:            "ord-ai-tg-1",
			Status:             "CONFIRMED",
			ConfirmationStatus: order.ConfirmationStatusConfirmed,
		},
	}

	svc := NewService(ServiceConfig{
		Repo:   repo,
		Orders: ordersFake,
		Now:    func() time.Time { return now },
	})
	router := newTelegramTestRouter(svc)

	// 1) Fail-closed check when secret is unconfigured -> 503.
	unconfReq := httptest.NewRequest(http.MethodGet, "/v1/retailer/telegram/chat/chat-999/context", nil)
	unconfRR := httptest.NewRecorder()
	router.ServeHTTP(unconfRR, unconfReq)
	if unconfRR.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured webhook secret code = %d, want 503", unconfRR.Code)
	}

	svc.SetBotWebhookSecret("test-secret-xyz")

	// 2) Bad secret -> 401.
	badSecReq := httptest.NewRequest(http.MethodGet, "/v1/retailer/telegram/chat/chat-999/context", nil)
	badSecReq.Header.Set("X-Bot-Webhook-Secret", "wrong-secret")
	badSecRR := httptest.NewRecorder()
	router.ServeHTTP(badSecRR, badSecReq)
	if badSecRR.Code != http.StatusUnauthorized {
		t.Fatalf("wrong webhook secret code = %d, want 401", badSecRR.Code)
	}

	// 3) Generate OTP via retailer JWT.
	linkReq := httptest.NewRequest(http.MethodPost, "/v1/retailer/telegram/link-request", bytes.NewBufferString(`{"locale":"uz"}`))
	linkReq = linkReq.WithContext(auth.WithClaims(linkReq.Context(), auth.Claims{
		Role:           auth.RoleRetailer,
		Subject:        "ret-tg-1",
		RetailerOrgID:  "ret-tg-1",
		RetailerUserID: "usr-tg-1",
		PhoneNumber:    "+998901234567",
	}))
	linkRR := httptest.NewRecorder()
	router.ServeHTTP(linkRR, linkReq)
	if linkRR.Code != http.StatusOK {
		t.Fatalf("link-request status = %d: %s", linkRR.Code, linkRR.Body.String())
	}
	var linkResp struct {
		OK          bool   `json:"ok"`
		OTPCode     string `json:"otp_code"`
		ExpiresAt   string `json:"expires_at"`
		BotDeepLink string `json:"bot_deep_link"`
	}
	if err := json.Unmarshal(linkRR.Body.Bytes(), &linkResp); err != nil {
		t.Fatalf("decode link-request: %v", err)
	}
	if !linkResp.OK || len(linkResp.OTPCode) != 6 || linkResp.BotDeepLink == "" {
		t.Fatalf("unexpected link-request response: %+v", linkResp)
	}

	// 4) Bot verifies OTP -> binds chat_id.
	verifyBody, _ := json.Marshal(map[string]string{
		"telegram_chat_id": "chat-555",
		"otp_code":         linkResp.OTPCode,
	})
	verReq := httptest.NewRequest(http.MethodPost, "/v1/retailer/telegram/verify-link", bytes.NewReader(verifyBody))
	verReq.Header.Set("X-Bot-Webhook-Secret", "test-secret-xyz")
	verRR := httptest.NewRecorder()
	router.ServeHTTP(verRR, verReq)
	if verRR.Code != http.StatusOK {
		t.Fatalf("verify-link status = %d: %s", verRR.Code, verRR.Body.String())
	}
	var verResp struct {
		Linked       bool   `json:"linked"`
		RetailerID   string `json:"retailer_id"`
		RetailerName string `json:"retailer_name"`
		Locale       string `json:"locale"`
	}
	if err := json.Unmarshal(verRR.Body.Bytes(), &verResp); err != nil {
		t.Fatalf("decode verify-link: %v", err)
	}
	if !verResp.Linked || verResp.RetailerID != "ret-tg-1" || verResp.RetailerName != "Baraka Market" {
		t.Fatalf("unexpected verify-link response: %+v", verResp)
	}

	// 5) OTP cannot be reused after bind.
	reuseReq := httptest.NewRequest(http.MethodPost, "/v1/retailer/telegram/verify-link", bytes.NewReader(verifyBody))
	reuseReq.Header.Set("X-Bot-Webhook-Secret", "test-secret-xyz")
	reuseRR := httptest.NewRecorder()
	router.ServeHTTP(reuseRR, reuseReq)
	if reuseRR.Code != http.StatusUnauthorized {
		t.Fatalf("reused OTP status = %d, want 401", reuseRR.Code)
	}

	// 6) Status check from retailer JWT reflects linked state.
	stReq := httptest.NewRequest(http.MethodGet, "/v1/retailer/telegram/status", nil)
	stReq = stReq.WithContext(auth.WithClaims(stReq.Context(), auth.Claims{
		Role:           auth.RoleRetailer,
		Subject:        "ret-tg-1",
		RetailerOrgID:  "ret-tg-1",
		RetailerUserID: "usr-tg-1",
	}))
	stRR := httptest.NewRecorder()
	router.ServeHTTP(stRR, stReq)
	if stRR.Code != http.StatusOK {
		t.Fatalf("status endpoint code = %d", stRR.Code)
	}
	var stResp map[string]any
	if err := json.Unmarshal(stRR.Body.Bytes(), &stResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if stResp["linked"] != true || stResp["telegram_chat_id"] != "chat-555" {
		t.Fatalf("unexpected status response: %+v", stResp)
	}

	// 7) Update locale via bot webhook secret -> ru.
	locReq := httptest.NewRequest(http.MethodPatch, "/v1/retailer/telegram/chat/chat-555/locale", bytes.NewBufferString(`{"locale":"ru"}`))
	locReq.Header.Set("X-Bot-Webhook-Secret", "test-secret-xyz")
	locRR := httptest.NewRecorder()
	router.ServeHTTP(locRR, locReq)
	if locRR.Code != http.StatusOK {
		t.Fatalf("locale update code = %d: %s", locRR.Code, locRR.Body.String())
	}

	// 8) Bot fetches context & sees updated locale and orders.
	ctxReq := httptest.NewRequest(http.MethodGet, "/v1/retailer/telegram/chat/chat-555/context", nil)
	ctxReq.Header.Set("X-Bot-Webhook-Secret", "test-secret-xyz")
	ctxRR := httptest.NewRecorder()
	router.ServeHTTP(ctxRR, ctxReq)
	if ctxRR.Code != http.StatusOK {
		t.Fatalf("chat context code = %d: %s", ctxRR.Code, ctxRR.Body.String())
	}
	var ctxResp struct {
		Linked       bool                       `json:"linked"`
		RetailerName string                     `json:"retailer_name"`
		Locale       string                     `json:"locale"`
		Orders       []TelegramChatOrderSummary `json:"orders"`
	}
	if err := json.Unmarshal(ctxRR.Body.Bytes(), &ctxResp); err != nil {
		t.Fatalf("decode context: %v", err)
	}
	if !ctxResp.Linked || ctxResp.Locale != "ru" || ctxResp.RetailerName != "Baraka Market" || len(ctxResp.Orders) < 2 {
		t.Fatalf("unexpected context response: %+v", ctxResp)
	}

	// 9) Bot confirms preorder cleanly without user JWT.
	confReq := httptest.NewRequest(http.MethodPost, "/v1/retailer/telegram/chat/chat-555/preorder/ord-ai-tg-1/confirm", bytes.NewBufferString(`{}`))
	confReq.Header.Set("X-Bot-Webhook-Secret", "test-secret-xyz")
	confReq.Header.Set("Idempotency-Key", "tg-confirm-1")
	confRR := httptest.NewRecorder()
	router.ServeHTTP(confRR, confReq)
	if confRR.Code != http.StatusOK {
		t.Fatalf("preorder confirm code = %d: %s", confRR.Code, confRR.Body.String())
	}
	if ordersFake.confirmAIReq.OrderID != "ord-ai-tg-1" || ordersFake.retailerID != "ret-tg-1" {
		t.Fatalf("unexpected order lifecycle confirm args: retailer=%q req=%+v", ordersFake.retailerID, ordersFake.confirmAIReq)
	}
}

func TestTelegramVerifyLinkByNormalizedUZPhoneShare(t *testing.T) {
	t.Parallel()

	repo := &testRetailerRepo{}
	svc := NewService(ServiceConfig{
		Repo:             repo,
		BotWebhookSecret: "bot-sec",
	})
	svc.mu.Lock()
	svc.ownerByRetailer["ret-phone-1"] = RetailerUser{
		UserID:     "owner-1",
		RetailerID: "ret-phone-1",
		Phone:      "+998907778899",
		Name:       "Aziz",
		IsOwner:    true,
		IsActive:   true,
	}
	svc.mu.Unlock()

	router := newTelegramTestRouter(svc)
	// Contact share phone with formatting differences (spaces/dashes/998 without +).
	verReq := httptest.NewRequest(http.MethodPost, "/v1/retailer/telegram/verify-link", bytes.NewBufferString(`{"telegram_chat_id":"chat-phone-7","phone_number":"998 90-777-88-99","locale":"en"}`))
	verReq.Header.Set("X-Bot-Webhook-Secret", "bot-sec")
	verRR := httptest.NewRecorder()
	router.ServeHTTP(verRR, verReq)
	if verRR.Code != http.StatusOK {
		t.Fatalf("phone verify status = %d: %s", verRR.Code, verRR.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(verRR.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["linked"] != true || resp["retailer_id"] != "ret-phone-1" || resp["phone_number"] != "+998907778899" || resp["locale"] != "en" {
		t.Fatalf("unexpected phone verify response: %+v", resp)
	}

	// Reject action dispatches cleanly when AI returns ErrInvalidStatusTransition -> Confirm/RejectPreorder.
	ordersFake := &testOrderLifecycle{
		response: order.RetailerOrderLifecycleResponse{
			OrderID:            "ord-manual-pre",
			Status:             "CANCELLED",
			ConfirmationStatus: order.ConfirmationStatusRejected,
		},
	}
	svc.orders = ordersFake
	rejReq := httptest.NewRequest(http.MethodPost, "/v1/retailer/telegram/chat/chat-phone-7/preorder/ord-manual-pre/reject", bytes.NewBufferString(`{"reason":"price_changed"}`))
	rejReq.Header.Set("X-Bot-Webhook-Secret", "bot-sec")
	rejRR := httptest.NewRecorder()
	router.ServeHTTP(rejRR, rejReq)
	if rejRR.Code != http.StatusOK {
		t.Fatalf("reject status = %d: %s", rejRR.Code, rejRR.Body.String())
	}
	_ = context.Background()
}
