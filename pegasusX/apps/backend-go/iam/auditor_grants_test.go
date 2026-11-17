package iam

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
)

func TestIAM_GrantAuditorAccess_ValidationSoDAndRevocation(t *testing.T) {
	t.Parallel()
	const jwtSecret = "test-iam-auditor-secret-0123456789abcdef"

	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	auditSpy := &spyAuditRecorder{}
	svc.SetAuditRecorder(auditSpy)
	svc.SetJWTSecret(jwtSecret, "atomos-test")

	now := time.Now().UTC()
	ctx := context.Background()
	tid := "sup-aud-unit-1"

	// 1. Default 72h grant issues time-boxed AUDITOR JWT and records AuditEvent.
	res, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID:  tid,
		UserID:    "aud-1",
		Reason:    "Q4 external audit",
		GrantedBy: "owner-1",
		ActorRole: auth.SupplierRoleOwner,
		ActorType: auth.ActorTypeHuman,
	})
	if err != nil {
		t.Fatalf("GrantAuditorAccess default 72h: %v", err)
	}
	delta := res.ExpiresAt.Sub(now)
	if delta < 71*time.Hour || delta > 73*time.Hour {
		t.Fatalf("default TTL delta = %v, want ~72h", delta)
	}
	claims, err := auth.ParseTokenWithKeyring(res.Token, auth.NewKeyring(jwtSecret))
	if err != nil {
		t.Fatalf("ParseTokenWithKeyring: %v", err)
	}
	if claims.SupplierRole != auth.SupplierRoleAuditor || claims.Subject != "aud-1" || claims.SupplierID != tid {
		t.Fatalf("unexpected auditor JWT claims: %+v", claims)
	}
	if claims.AuditorExpiresAt == nil || !claims.AuditorExpiresAt.Equal(res.ExpiresAt) {
		t.Fatalf("claims.AuditorExpiresAt = %v, want %v", claims.AuditorExpiresAt, res.ExpiresAt)
	}
	if len(claims.SupplierPerms) != 1 || claims.SupplierPerms[0] != auth.PermAuditRead {
		t.Fatalf("claims.SupplierPerms = %v, want [audit.read]", claims.SupplierPerms)
	}

	// 2. Self-grant rejected -> ErrSodViolation.
	if _, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID:  tid,
		UserID:    "owner-1",
		GrantedBy: "owner-1",
		ActorRole: auth.SupplierRoleOwner,
		ActorType: auth.ActorTypeHuman,
	}); !errors.Is(err, ErrSodViolation) {
		t.Fatalf("self-grant got %v, want ErrSodViolation", err)
	}

	// 3. Past expires_at rejected -> ErrInvalidIAMInput.
	past := now.Add(-10 * time.Minute)
	if _, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID:  tid,
		UserID:    "aud-2",
		ExpiresAt: &past,
		GrantedBy: "owner-1",
	}); !errors.Is(err, ErrInvalidIAMInput) {
		t.Fatalf("past expires_at got %v, want ErrInvalidIAMInput", err)
	}

	// 4. AGENT caller or target rejected -> ErrAgentApprovalForbidden.
	if _, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID:  tid,
		UserID:    "aud-2",
		GrantedBy: "agent-caller",
		ActorType: auth.ActorTypeAgent,
	}); !errors.Is(err, ErrAgentApprovalForbidden) {
		t.Fatalf("AGENT caller got %v, want ErrAgentApprovalForbidden", err)
	}
	if _, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID:        tid,
		UserID:          "bot-1",
		TargetActorType: auth.ActorTypeAgent,
		GrantedBy:       "owner-1",
	}); !errors.Is(err, ErrAgentApprovalForbidden) {
		t.Fatalf("AGENT target got %v, want ErrAgentApprovalForbidden", err)
	}

	// 5. User already holding AGENT role cannot receive AUDITOR grant.
	if _, err := svc.AssignRole(ctx, AssignRoleInput{
		TenantID: tid, UserID: "svc-agent", RoleName: auth.SupplierRoleAgent, AssignedBy: "owner-1",
	}); err != nil {
		t.Fatalf("assign AGENT: %v", err)
	}
	if _, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID: tid, UserID: "svc-agent", GrantedBy: "owner-1",
	}); !errors.Is(err, ErrAgentApprovalForbidden) {
		t.Fatalf("target holding AGENT got %v, want ErrAgentApprovalForbidden", err)
	}

	// 6. User holding ACCOUNTANT cannot receive AUDITOR grant -> ErrSodRoleConflict.
	if _, err := svc.AssignRole(ctx, AssignRoleInput{
		TenantID: tid, UserID: "acct-1", RoleName: auth.SupplierRoleAccountant, AssignedBy: "owner-1",
	}); err != nil {
		t.Fatalf("assign ACCOUNTANT: %v", err)
	}
	if _, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID: tid, UserID: "acct-1", GrantedBy: "owner-1",
	}); !errors.Is(err, ErrSodRoleConflict) {
		t.Fatalf("target holding ACCOUNTANT got %v, want ErrSodRoleConflict", err)
	}

	// 7. Revoke invalidates previously issued JTI and removes active grant; re-granting works cleanly.
	if err := svc.RevokeAuditorGrant(ctx, RevokeAuditorGrantInput{
		TenantID:  tid,
		UserID:    "aud-1",
		RevokedBy: "owner-1",
	}); err != nil {
		t.Fatalf("RevokeAuditorGrant: %v", err)
	}
	revoked, rerr := auth.GetRevocationStore().IsRevoked(ctx, claims.JTI)
	if rerr != nil || !revoked {
		t.Fatalf("expected old JTI revoked (revoked=%v, err=%v)", revoked, rerr)
	}
	if err := svc.RevokeAuditorGrant(ctx, RevokeAuditorGrantInput{
		TenantID: tid, UserID: "aud-1", RevokedBy: "owner-1",
	}); !errors.Is(err, ErrAuditorGrantNotFound) {
		t.Fatalf("second revoke got %v, want ErrAuditorGrantNotFound", err)
	}

	regrant, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID:  tid,
		UserID:    "aud-1",
		TTLHours:  24,
		GrantedBy: "owner-1",
	})
	if err != nil {
		t.Fatalf("re-grant after revoke: %v", err)
	}
	reClaims, err := auth.ParseTokenWithKeyring(regrant.Token, auth.NewKeyring(jwtSecret))
	if err != nil {
		t.Fatalf("ParseTokenWithKeyring on re-grant: %v", err)
	}
	if rev2, _ := auth.GetRevocationStore().IsRevoked(ctx, reClaims.JTI); rev2 {
		t.Fatalf("expected new JTI active after re-grant, got revoked")
	}

	// 8. Verify AuditEvents captured grant + revoke + grant.
	auditSpy.mu.Lock()
	var actions []string
	for _, ev := range auditSpy.events {
		actions = append(actions, ev.Action)
	}
	auditSpy.mu.Unlock()
	joinedActions := strings.Join(actions, ",")
	if !strings.Contains(joinedActions, "iam.auditor_access.granted") || !strings.Contains(joinedActions, "iam.auditor_access.revoked") {
		t.Fatalf("expected grant & revoke audit actions, got %v", actions)
	}
}

func TestSpannerIAM_AuditorGrantLifecycleAndOutbox(t *testing.T) {
	client := openIAMSpannerTestClient(t)
	ctx := context.Background()
	tid := "sup-aud-spn-" + uuid.NewString()[:8]

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("SupplierUserRoles", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid + "~"}}),
			spanner.Delete("RolePermissions", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid + "~"}}),
			spanner.Delete("SodPolicies", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid + "~"}}),
			spanner.Delete("SodViolations", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid + "~"}}),
			spanner.Delete("OutboxEvents", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid + "~"}}),
		})
	})

	svc := NewService(client)
	svc.SetJWTSecret("spanner-auditor-secret-0123456789abcdef", "atomos-spanner")

	shortExp := time.Now().UTC().Add(150 * time.Millisecond)
	res, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID:  tid,
		UserID:    "aud-spn-1",
		ExpiresAt: &shortExp,
		Reason:    "Spanner integration check",
		GrantedBy: "owner-spn-1",
		ActorRole: auth.SupplierRoleOwner,
		TraceID:   "tr-aud-spn-1",
	})
	if err != nil {
		t.Fatalf("Spanner GrantAuditorAccess: %v", err)
	}
	if res.Token == "" || !res.ExpiresAt.Equal(shortExp) {
		t.Fatalf("unexpected result: %+v", res)
	}

	// Verify SupplierUserRoles.ExpiresAt was stored in Spanner.
	roles, err := svc.Repo().ListUserRoles(ctx, tid, "aud-spn-1")
	if err != nil || len(roles) != 1 || roles[0].ExpiresAt == nil {
		t.Fatalf("ListUserRoles = (%+v, %v), want 1 role with ExpiresAt", roles, err)
	}
	if roles[0].ExpiresAt.Sub(shortExp).Abs() > time.Millisecond {
		t.Fatalf("Spanner ExpiresAt = %v, want %v", roles[0].ExpiresAt, shortExp)
	}

	// Verify OutboxEvents recorded iam.auditor_access.granted.
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()
	foundGrantEvent := false
	for {
		row, rerr := iter.Next()
		if rerr == iterator.Done {
			break
		}
		if rerr != nil {
			t.Fatalf("query OutboxEvents: %v", rerr)
		}
		var topic string
		var raw []byte
		if serr := row.Columns(&topic, &raw); serr != nil {
			t.Fatalf("scan OutboxEvents: %v", serr)
		}
		body := string(raw)
		if topic == events.TopicLedger && strings.Contains(body, `"iam.auditor_access.granted"`) && strings.Contains(body, "aud-spn-1") {
			foundGrantEvent = true
		}
	}
	if !foundGrantEvent {
		t.Fatalf("expected iam.auditor_access.granted on ledger topic in OutboxEvents for %s", tid)
	}

	// Wait past shortExp -> ListAuditorGrants reports EXPIRED, and assigning ACCOUNTANT succeeds because expired AUDITOR does not block.
	time.Sleep(180 * time.Millisecond)
	grants, err := svc.ListAuditorGrants(ctx, tid)
	if err != nil || len(grants) != 1 || grants[0].Status != "EXPIRED" {
		t.Fatalf("ListAuditorGrants after expiry = (%+v, %v), want 1 EXPIRED", grants, err)
	}
	if _, err := svc.AssignRole(ctx, AssignRoleInput{
		TenantID: tid, UserID: "aud-spn-1", RoleName: auth.SupplierRoleAccountant, AssignedBy: "owner-spn-1",
	}); err != nil {
		t.Fatalf("expected assigning operational role after AUDITOR expiry to succeed, got %v", err)
	}
	if _, err := auth.ParseTokenWithKeyring(res.Token, auth.NewKeyring("spanner-auditor-secret-0123456789abcdef")); !errors.Is(err, auth.ErrAuditorAccessExpired) {
		t.Fatalf("ParseTokenWithKeyring after expiry got %v, want ErrAuditorAccessExpired", err)
	}

	// Grant another user, list, then revoke cleanly on Spanner.
	if _, err := svc.GrantAuditorAccess(ctx, GrantAuditorAccessInput{
		TenantID:  tid,
		UserID:    "aud-spn-2",
		TTLHours:  24,
		GrantedBy: "owner-spn-1",
	}); err != nil {
		t.Fatalf("second grant: %v", err)
	}
	if err := svc.RevokeAuditorGrant(ctx, RevokeAuditorGrantInput{
		TenantID:  tid,
		UserID:    "aud-spn-2",
		RevokedBy: "owner-spn-1",
		TraceID:   fmt.Sprintf("tr-rev-%s", tid),
	}); err != nil {
		t.Fatalf("Spanner RevokeAuditorGrant: %v", err)
	}
	afterRevoke, err := svc.Repo().ListUserRoles(ctx, tid, "aud-spn-2")
	if err != nil || len(afterRevoke) != 0 {
		t.Fatalf("expected 0 roles after revoke, got (%+v, %v)", afterRevoke, err)
	}
}
