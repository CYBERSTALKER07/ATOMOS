package retailer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestTheatreKillP1Gone(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})

	tests := []struct {
		name       string
		method     string
		path       string
		handle     func(http.ResponseWriter, *http.Request)
		wantError  string
		wantStatus int
	}{
		{
			name:       "cards GET",
			method:     http.MethodGet,
			path:       "/v1/retailer/cards",
			handle:     svc.HandleRetailerCards,
			wantError:  "saved_cards_not_product",
			wantStatus: http.StatusGone,
		},
		{
			name:       "cards initiate POST",
			method:     http.MethodPost,
			path:       "/v1/retailer/card/initiate",
			handle:     svc.HandleRetailerCardMutation,
			wantError:  "saved_cards_not_product",
			wantStatus: http.StatusGone,
		},
		{
			name:       "AI alias GET",
			method:     http.MethodGet,
			path:       "/v1/ai/predictions",
			handle:     svc.HandleAIPredictionsAlias,
			wantError:  "use_retailer_ai_predictions",
			wantStatus: http.StatusGone,
		},
		{
			name:       "AI correct PATCH",
			method:     http.MethodPatch,
			path:       "/v1/ai/predictions/correct",
			handle:     svc.HandleCorrectPrediction,
			wantError:  "prediction_correct_unwired",
			wantStatus: http.StatusGone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			tt.handle(rr, req)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v body=%s", err, rr.Body.String())
			}
			if body["error"] != tt.wantError {
				t.Fatalf("error=%q want=%q body=%s", body["error"], tt.wantError, rr.Body.String())
			}
			if body["status"] == "ok" {
				t.Fatal("theatre {status:ok} must not appear")
			}
		})
	}
}

func TestTheatreKillP1MethodNotAllowed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/cards", nil)
	rr := httptest.NewRecorder()
	svc.HandleRetailerCards(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleUserNotifications_NilStoreUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodGet, "/v1/user/notifications", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer, RetailerOrgID: "org-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleUserNotifications(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "inbox_unavailable" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["notifications"]; ok {
		t.Fatal("nil store must not return notifications[]")
	}
}

func TestHandleMarkNotificationsRead_NilStoreUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/notifications/read",
		strings.NewReader(`{"mark_all":true}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer, RetailerOrgID: "org-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleMarkNotificationsRead(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "inbox_unavailable" || payload["status"] == "ok" {
		t.Fatalf("payload=%v", payload)
	}
}
