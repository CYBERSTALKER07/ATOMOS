package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPurgeStaleToken_MatchingSessionDeletesToken(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	ctx := context.Background()

	err := repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID:   "retailer-1",
		ActorRole: "RETAILER",
		Platform:  "android",
		Token:     "fcm-token-match",
		DeviceID:  "device-alpha",
		SessionID: "session-alpha",
	})
	if err != nil {
		t.Fatalf("upsert token failed: %v", err)
	}

	// Purge with matching session ID
	if err := repo.PurgeStaleToken(ctx, "fcm-token-match", "session-alpha"); err != nil {
		t.Fatalf("purge stale token failed: %v", err)
	}

	tokens, err := repo.ListTokens(ctx, "retailer-1", "RETAILER")
	if err != nil {
		t.Fatalf("list tokens failed: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("expected token to be purged, got %v", tokens)
	}
}

func TestPurgeStaleToken_ExpiredSessionDoesNotDeleteNewerSession(t *testing.T) {
	// Acceptance Criteria 4:
	// A Go unit/integration test verifies that executing purgeStaleToken on an expired session
	// does *not* delete a newer token registration for the same user.
	repo := NewMemoryDeviceTokenRepository()
	ctx := context.Background()

	// Initial token registration from Session 1
	err := repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID:   "user-retailer-99",
		ActorRole: "RETAILER",
		Platform:  "ios",
		Token:     "fcm-shared-token",
		DeviceID:  "device-iphone-14",
		SessionID: "session-expired-101",
	})
	if err != nil {
		t.Fatalf("upsert initial token failed: %v", err)
	}

	// User logs back in / reinstalls app, generating Session 2 and updating the token
	err = repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID:   "user-retailer-99",
		ActorRole: "RETAILER",
		Platform:  "ios",
		Token:     "fcm-shared-token",
		DeviceID:  "device-iphone-14",
		SessionID: "session-active-202",
	})
	if err != nil {
		t.Fatalf("upsert updated token failed: %v", err)
	}

	// Asynchronous worker for expired session 1 triggers purgeStaleToken
	err = repo.PurgeStaleToken(ctx, "fcm-shared-token", "session-expired-101")
	if err != nil {
		t.Fatalf("purge stale token returned error: %v", err)
	}

	// Verification: The token for the active session MUST NOT be deleted!
	tokens, err := repo.ListTokens(ctx, "user-retailer-99", "RETAILER")
	if err != nil {
		t.Fatalf("list tokens failed: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "fcm-shared-token" {
		t.Fatalf("expected active token 'fcm-shared-token' to remain preserved, got %v", tokens)
	}

	row, ok := repo.tokens["fcm-shared-token"]
	if !ok || row.SessionID != "session-active-202" {
		t.Fatalf("expected stored session ID to be 'session-active-202', got %q (ok=%v)", row.SessionID, ok)
	}
}

func TestPurgeStaleToken_EmptySessionDeletesUnconditionally(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	ctx := context.Background()

	err := repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID:   "driver-1",
		ActorRole: "DRIVER",
		Platform:  "android",
		Token:     "fcm-legacy-token",
		DeviceID:  "dev-1",
		SessionID: "sess-1",
	})
	if err != nil {
		t.Fatalf("upsert token failed: %v", err)
	}

	// Purge with empty session ID (backward compatibility blind delete)
	if err := repo.PurgeStaleToken(ctx, "fcm-legacy-token", ""); err != nil {
		t.Fatalf("purge stale token failed: %v", err)
	}

	tokens, err := repo.ListTokens(ctx, "driver-1", "DRIVER")
	if err != nil {
		t.Fatalf("list tokens failed: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("expected legacy token to be deleted, got %v", tokens)
	}
}

func TestPurgeStaleToken_NonExistentTokenIsNoOp(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	ctx := context.Background()

	if err := repo.PurgeStaleToken(ctx, "non-existent-token", "session-xyz"); err != nil {
		t.Fatalf("expected no error for non-existent token, got %v", err)
	}
}

func TestDeleteTokenBySession(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	ctx := context.Background()

	// Register 2 tokens for user-1 in session-A, 1 token in session-B
	_ = repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID: "user-1", ActorRole: "RETAILER", Platform: "android",
		Token: "tok-1A", SessionID: "session-A",
	})
	_ = repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID: "user-1", ActorRole: "RETAILER", Platform: "ios",
		Token: "tok-2A", SessionID: "session-A",
	})
	_ = repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID: "user-1", ActorRole: "RETAILER", Platform: "android",
		Token: "tok-1B", SessionID: "session-B",
	})
	// Register 1 token for user-2 in session-A (different actor)
	_ = repo.UpsertToken(ctx, DeviceTokenRow{
		ActorID: "user-2", ActorRole: "RETAILER", Platform: "android",
		Token: "tok-user2A", SessionID: "session-A",
	})

	// Delete tokens for user-1 session-A
	if err := repo.DeleteTokenBySession(ctx, "user-1", "session-A"); err != nil {
		t.Fatalf("DeleteTokenBySession failed: %v", err)
	}

	user1Tokens, err := repo.ListTokens(ctx, "user-1", "RETAILER")
	if err != nil {
		t.Fatal(err)
	}
	if len(user1Tokens) != 1 || user1Tokens[0] != "tok-1B" {
		t.Fatalf("expected user-1 to only have 'tok-1B', got %v", user1Tokens)
	}

	user2Tokens, err := repo.ListTokens(ctx, "user-2", "RETAILER")
	if err != nil {
		t.Fatal(err)
	}
	if len(user2Tokens) != 1 || user2Tokens[0] != "tok-user2A" {
		t.Fatalf("expected user-2 to keep 'tok-user2A', got %v", user2Tokens)
	}
}

func TestHandleDeviceToken_ParsesDeviceAndSessionID(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	h := NewHandler(HandlerConfig{DeviceTokens: repo})

	reqBody := `{"token":"fcm-session-token","platform":"android","device_id":"device-uuid-999","session_id":"sess-uuid-888"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/user/device-token", strings.NewReader(reqBody))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-sess-1", Role: auth.RoleRetailer,
	}))

	rr := httptest.NewRecorder()
	h.HandleDeviceToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", resp)
	}

	row, ok := repo.tokens["fcm-session-token"]
	if !ok {
		t.Fatal("token not stored in repository")
	}
	if row.DeviceID != "device-uuid-999" {
		t.Fatalf("expected DeviceID 'device-uuid-999', got %q", row.DeviceID)
	}
	if row.SessionID != "sess-uuid-888" {
		t.Fatalf("expected SessionID 'sess-uuid-888', got %q", row.SessionID)
	}
	if row.ActorID != "ret-sess-1" || row.Platform != "android" {
		t.Fatalf("unexpected row metadata: %+v", row)
	}
}

func TestHandleDeviceToken_SessionPurgeRaceIntegration(t *testing.T) {
	repo := NewMemoryDeviceTokenRepository()
	h := NewHandler(HandlerConfig{DeviceTokens: repo})

	// 1. First registration with Session 1
	req1 := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-race-token","platform":"android","device_id":"pixel-7","session_id":"session-v1"}`))
	req1 = req1.WithContext(auth.WithClaims(req1.Context(), auth.Claims{
		Subject: "user-race", Role: auth.RoleRetailer,
	}))
	rr1 := httptest.NewRecorder()
	h.HandleDeviceToken(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("req1 status=%d", rr1.Code)
	}

	// 2. Second registration with Session 2 (overwrites session association)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/user/device-token",
		strings.NewReader(`{"token":"fcm-race-token","platform":"android","device_id":"pixel-7","session_id":"session-v2"}`))
	req2 = req2.WithContext(auth.WithClaims(req2.Context(), auth.Claims{
		Subject: "user-race", Role: auth.RoleRetailer,
	}))
	rr2 := httptest.NewRecorder()
	h.HandleDeviceToken(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("req2 status=%d", rr2.Code)
	}

	// 3. Stale purge from Session 1
	ctx := context.Background()
	if err := repo.PurgeStaleToken(ctx, "fcm-race-token", "session-v1"); err != nil {
		t.Fatal(err)
	}

	// 4. Token must still exist under user-race
	tokens, err := repo.ListTokens(ctx, "user-race", "RETAILER")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0] != "fcm-race-token" {
		t.Fatalf("expected token to remain intact after old session purge, got %v", tokens)
	}

	// 5. Purge from Session 2 (current session)
	if err := repo.PurgeStaleToken(ctx, "fcm-race-token", "session-v2"); err != nil {
		t.Fatal(err)
	}

	// 6. Token should now be deleted
	tokens, err = repo.ListTokens(ctx, "user-race", "RETAILER")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 0 {
		t.Fatalf("expected token to be purged after active session purge, got %v", tokens)
	}
}
