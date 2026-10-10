package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type stubNotifRepo struct {
	listErr   error
	unreadErr error
}

func (s stubNotifRepo) Create(context.Context, Notification) error { return nil }
func (s stubNotifRepo) ListByRecipient(context.Context, string, int, int) ([]Notification, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []Notification{}, nil
}
func (s stubNotifRepo) MarkRead(context.Context, []string) error  { return nil }
func (s stubNotifRepo) MarkAllRead(context.Context, string) error { return nil }
func (s stubNotifRepo) UnreadCount(context.Context, string) (int64, error) {
	if s.unreadErr != nil {
		return 0, s.unreadErr
	}
	return 0, nil
}
func (s stubNotifRepo) GetPreference(context.Context, string, string, string) (*NotificationPreference, error) {
	return nil, nil
}
func (s stubNotifRepo) UpsertPreference(context.Context, NotificationPreference) error { return nil }

func inboxReq(method, path, body string, claims auth.Claims) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	return req.WithContext(auth.WithClaims(req.Context(), claims))
}

func TestHandleList_NilServiceUnavailable(t *testing.T) {
	h := &InboxHandlers{}
	rr := httptest.NewRecorder()
	h.HandleList(rr, inboxReq(http.MethodGet, "/v1/user/notifications", "", auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
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

func TestHandleList_ListErrorFailed(t *testing.T) {
	h := &InboxHandlers{Service: NewService(stubNotifRepo{listErr: errors.New("spanner_unavailable")}, nil, nil)}
	rr := httptest.NewRecorder()
	h.HandleList(rr, inboxReq(http.MethodGet, "/v1/user/notifications", "", auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "inbox_list_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["notifications"]; ok {
		t.Fatal("list error must not return notifications[]")
	}
}

func TestHandleList_UnreadErrorFailed(t *testing.T) {
	h := &InboxHandlers{Service: NewService(stubNotifRepo{unreadErr: errors.New("spanner_unavailable")}, nil, nil)}
	rr := httptest.NewRecorder()
	h.HandleList(rr, inboxReq(http.MethodGet, "/v1/user/notifications", "", auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "inbox_unread_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if payload["unread_count"] != nil {
		t.Fatal("unread error must not return unread_count 0")
	}
}

func TestHandleMarkRead_NilServiceUnavailable(t *testing.T) {
	h := &InboxHandlers{}
	rr := httptest.NewRecorder()
	h.HandleMarkRead(rr, inboxReq(http.MethodPost, "/v1/user/notifications/read", `{"mark_all":true}`, auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "inbox_unavailable" {
		t.Fatalf("payload=%v", payload)
	}
	if payload["status"] == "ok" {
		t.Fatal("nil store must not return status ok")
	}
}

func TestApplyMarkRead_NilStoreUnavailable(t *testing.T) {
	err := ApplyMarkRead(context.Background(), nil, "ret-1", MarkReadRequest{})
	if !errors.Is(err, errInboxUnavailable) {
		t.Fatalf("err=%v", err)
	}
}
