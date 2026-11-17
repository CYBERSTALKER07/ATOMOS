package iam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

func newSCIMTestRouter(svc *Service, tid string) chi.Router {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			role := strings.TrimSpace(req.Header.Get("X-Test-Role"))
			if role == "" {
				role = auth.SupplierRoleOwner
			}
			c := auth.Claims{
				Subject:      "actor-" + strings.ToLower(role),
				Role:         auth.RoleAdmin,
				SupplierID:   tid,
				SupplierRole: auth.Role(role),
				ActorType:    auth.ActorTypeHuman,
			}
			if role == auth.SupplierRoleAgent {
				c.ActorType = auth.ActorTypeAgent
			}
			ctx := auth.WithTenant(auth.WithClaims(req.Context(), c), auth.TenantContext{SupplierID: tid})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Post("/v1/iam/custom-roles", svc.HandleCreateCustomRole)
	r.Get("/scim/v2/Users", svc.HandleListScimUsers)
	r.Post("/scim/v2/Users", svc.HandleCreateScimUser)
	r.Get("/scim/v2/Users/{id}", svc.HandleGetScimUser)
	r.Put("/scim/v2/Users/{id}", svc.HandlePutScimUser)
	r.Patch("/scim/v2/Users/{id}", svc.HandlePatchScimUser)
	r.Delete("/scim/v2/Users/{id}", svc.HandleDeleteScimUser)
	r.Get("/scim/v2/Groups", svc.HandleListScimGroups)
	r.Post("/scim/v2/Groups", svc.HandleCreateScimGroup)
	r.Get("/scim/v2/Groups/{id}", svc.HandleGetScimGroup)
	r.Put("/scim/v2/Groups/{id}", svc.HandlePutScimGroup)
	r.Patch("/scim/v2/Groups/{id}", svc.HandlePatchScimGroup)
	r.Delete("/scim/v2/Groups/{id}", svc.HandleDeleteScimGroup)
	return r
}

func cleanupSCIMAndCustomRoleTenant(t *testing.T, client *spanner.Client, tid string) {
	t.Helper()
	cleanupIAMTenant(t, client, tid)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = client.Apply(ctx, []*spanner.Mutation{
			spanner.Delete("CustomRoles", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ScimIdentities", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
		})
	})
}

func TestSCIM_Users_And_Groups_CRUD_Patch_SoD_And_Deprovision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := NewMemoryRepository()
	auditSpy := &spyAuditRecorder{}
	svc := NewServiceWithRepository(repo)
	svc.SetAuditRecorder(auditSpy)

	const tid = "sup-scim-unit"
	r := newSCIMTestRouter(svc, tid)

	// 1. Seed a custom role first to verify custom role provisioning via SCIM.
	createRoleReq := httptest.NewRequest(http.MethodPost, "/v1/iam/custom-roles", strings.NewReader(`{
		"role_name": "CUSTOM_STOCK_CLERK",
		"description": "Custom stock clerk role",
		"permissions": ["stock.count", "audit.read"]
	}`))
	createRoleRR := httptest.NewRecorder()
	r.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("create custom role = %d body=%s", createRoleRR.Code, createRoleRR.Body.String())
	}

	// 2. AUDITOR and AGENT are blocked on SCIM mutations (403), but AUDITOR can GET.
	for _, blockedRole := range []string{auth.SupplierRoleAuditor, auth.SupplierRoleAgent} {
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{"userName":"blocked@example.com"}`))
		req.Header.Set("X-Test-Role", blockedRole)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("%s POST /scim/v2/Users = %d, want 403 body=%s", blockedRole, rr.Code, rr.Body.String())
		}
	}

	// 3. Create SCIM User with ACCOUNTANT role -> 201 + application/scim+json.
	createUserReq := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
		"externalId": "okta-u-101",
		"userName": "alice.gl",
		"displayName": "Alice GL",
		"emails": [{"value":"alice.gl@example.com","type":"work","primary":true}],
		"active": true,
		"roles": [{"value":"ACCOUNTANT","display":"Accountant"}]
	}`))
	createUserRR := httptest.NewRecorder()
	r.ServeHTTP(createUserRR, createUserReq)
	if createUserRR.Code != http.StatusCreated {
		t.Fatalf("POST /scim/v2/Users = %d body=%s", createUserRR.Code, createUserRR.Body.String())
	}
	if ct := createUserRR.Header().Get("Content-Type"); !strings.Contains(ct, "application/scim+json") {
		t.Fatalf("expected application/scim+json Content-Type, got %q", ct)
	}
	var createdUser ScimUser
	if err := json.Unmarshal(createUserRR.Body.Bytes(), &createdUser); err != nil {
		t.Fatalf("decode created user: %v", err)
	}
	if createdUser.ID == "" || createdUser.UserName != "alice.gl" || !createdUser.Active {
		t.Fatalf("unexpected SCIM user payload: %+v", createdUser)
	}

	// Verify SupplierUserRoles was synced cleanly.
	uRoles, err := repo.ListUserRoles(ctx, tid, createdUser.ID)
	if err != nil || len(uRoles) != 1 || uRoles[0].RoleName != auth.SupplierRoleAccountant {
		t.Fatalf("expected ACCOUNTANT synced to SupplierUserRoles, got (%+v, %v)", uRoles, err)
	}

	// 4. Duplicate userName rejected with 409 Conflict.
	dupReq := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{"userName":"Alice.GL"}`))
	dupRR := httptest.NewRecorder()
	r.ServeHTTP(dupRR, dupReq)
	if dupRR.Code != http.StatusConflict {
		t.Fatalf("duplicate userName POST = %d, want 409 body=%s", dupRR.Code, dupRR.Body.String())
	}

	// 5. Escalation to PLATFORM_ADMIN rejected with 403.
	escReq := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{
		"userName":"evil.admin",
		"roles":[{"value":"PLATFORM_ADMIN"}]
	}`))
	escRR := httptest.NewRecorder()
	r.ServeHTTP(escRR, escReq)
	if escRR.Code != http.StatusForbidden {
		t.Fatalf("PLATFORM_ADMIN SCIM create = %d, want 403 body=%s", escRR.Code, escRR.Body.String())
	}

	// 6. AUDITOR GET list (with filter) & GET by ID -> 200.
	listReq := httptest.NewRequest(http.MethodGet, `/scim/v2/Users?filter=userName%20eq%20%22alice.gl%22`, nil)
	listReq.Header.Set("X-Test-Role", auth.SupplierRoleAuditor)
	listRR := httptest.NewRecorder()
	r.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("AUDITOR GET /scim/v2/Users = %d body=%s", listRR.Code, listRR.Body.String())
	}
	var listResp ScimListResponse
	if err := json.Unmarshal(listRR.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode listResp: %v", err)
	}
	if listResp.TotalResults != 1 || listResp.ItemsPerPage != 1 {
		t.Fatalf("unexpected filtered list counts: %+v", listResp)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/scim/v2/Users/"+createdUser.ID, nil)
	getReq.Header.Set("X-Test-Role", auth.SupplierRoleAuditor)
	getRR := httptest.NewRecorder()
	r.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("GET /scim/v2/Users/{id} = %d body=%s", getRR.Code, getRR.Body.String())
	}

	// 7. PATCH adding conflicting FINANCE_MANAGER alongside ACCOUNTANT triggers SoD 403.
	sodPatchReq := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/"+createdUser.ID, strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op":"add","path":"roles","value":[{"value":"FINANCE_MANAGER"}]}]
	}`))
	sodPatchRR := httptest.NewRecorder()
	r.ServeHTTP(sodPatchRR, sodPatchReq)
	if sodPatchRR.Code != http.StatusForbidden {
		t.Fatalf("SoD conflicting PATCH = %d, want 403 body=%s", sodPatchRR.Code, sodPatchRR.Body.String())
	}

	// 8. PATCH replacing roles with CUSTOM_STOCK_CLERK succeeds and syncs custom role.
	okPatchReq := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/"+createdUser.ID, strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [
			{"op":"replace","path":"displayName","value":"Alice Clerk"},
			{"op":"replace","path":"roles","value":[{"value":"CUSTOM_STOCK_CLERK"}]}
		]
	}`))
	okPatchRR := httptest.NewRecorder()
	r.ServeHTTP(okPatchRR, okPatchReq)
	if okPatchRR.Code != http.StatusOK {
		t.Fatalf("valid PATCH /scim/v2/Users/{id} = %d body=%s", okPatchRR.Code, okPatchRR.Body.String())
	}
	uRoles, err = repo.ListUserRoles(ctx, tid, createdUser.ID)
	if err != nil || len(uRoles) != 1 || uRoles[0].RoleName != "CUSTOM_STOCK_CLERK" {
		t.Fatalf("expected CUSTOM_STOCK_CLERK after PATCH, got (%+v, %v)", uRoles, err)
	}

	// 9. PUT setting active=false suspends user and strips SupplierUserRoles.
	putDeactReq := httptest.NewRequest(http.MethodPut, "/scim/v2/Users/"+createdUser.ID, strings.NewReader(`{
		"userName": "alice.gl",
		"displayName": "Alice Clerk",
		"active": false,
		"roles": [{"value":"CUSTOM_STOCK_CLERK"}]
	}`))
	putDeactRR := httptest.NewRecorder()
	r.ServeHTTP(putDeactRR, putDeactReq)
	if putDeactRR.Code != http.StatusOK {
		t.Fatalf("PUT deactivate user = %d body=%s", putDeactRR.Code, putDeactRR.Body.String())
	}
	uRoles, err = repo.ListUserRoles(ctx, tid, createdUser.ID)
	if err != nil || len(uRoles) != 0 {
		t.Fatalf("expected 0 roles when SCIM user active=false, got (%+v, %v)", uRoles, err)
	}

	// 10. Create active secondary user + SCIM Group assigning ACCOUNTANT role to member.
	u2Req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{
		"id": "scim-u-bob",
		"userName": "bob.fin",
		"active": true
	}`))
	u2RR := httptest.NewRecorder()
	r.ServeHTTP(u2RR, u2Req)
	if u2RR.Code != http.StatusCreated {
		t.Fatalf("POST bob user = %d body=%s", u2RR.Code, u2RR.Body.String())
	}

	grpReq := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
		"externalId": "okta-grp-acct",
		"displayName": "Accounting Team",
		"roles": [{"value":"ACCOUNTANT"}],
		"members": [{"value":"scim-u-bob","display":"Bob Fin"}]
	}`))
	grpRR := httptest.NewRecorder()
	r.ServeHTTP(grpRR, grpReq)
	if grpRR.Code != http.StatusCreated {
		t.Fatalf("POST /scim/v2/Groups = %d body=%s", grpRR.Code, grpRR.Body.String())
	}
	var createdGroup ScimGroup
	if err := json.Unmarshal(grpRR.Body.Bytes(), &createdGroup); err != nil {
		t.Fatalf("decode created group: %v", err)
	}
	bobRoles, err := repo.ListUserRoles(ctx, tid, "scim-u-bob")
	if err != nil || len(bobRoles) != 1 || bobRoles[0].RoleName != auth.SupplierRoleAccountant {
		t.Fatalf("expected Group create to sync ACCOUNTANT to bob, got (%+v, %v)", bobRoles, err)
	}

	// 11. PATCH group remove members, PUT group update, GET group list & detail, DELETE group -> 204.
	grpPatchReq := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/"+createdGroup.ID, strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [
			{"op":"remove","path":"members","value":[{"value":"scim-u-bob"}]},
			{"op":"replace","path":"displayName","value":"Accounting Ops"}
		]
	}`))
	grpPatchRR := httptest.NewRecorder()
	r.ServeHTTP(grpPatchRR, grpPatchReq)
	if grpPatchRR.Code != http.StatusOK {
		t.Fatalf("PATCH /scim/v2/Groups/{id} = %d body=%s", grpPatchRR.Code, grpPatchRR.Body.String())
	}

	grpPutReq := httptest.NewRequest(http.MethodPut, "/scim/v2/Groups/"+createdGroup.ID, strings.NewReader(`{
		"displayName": "Accounting Ops Put",
		"roles": [{"value":"ACCOUNTANT"}],
		"members": [{"value":"scim-u-bob"}]
	}`))
	grpPutRR := httptest.NewRecorder()
	r.ServeHTTP(grpPutRR, grpPutReq)
	if grpPutRR.Code != http.StatusOK {
		t.Fatalf("PUT /scim/v2/Groups/{id} = %d body=%s", grpPutRR.Code, grpPutRR.Body.String())
	}

	grpListReq := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups", nil)
	grpListReq.Header.Set("X-Test-Role", auth.SupplierRoleAuditor)
	grpListRR := httptest.NewRecorder()
	r.ServeHTTP(grpListRR, grpListReq)
	if grpListRR.Code != http.StatusOK {
		t.Fatalf("AUDITOR GET /scim/v2/Groups = %d body=%s", grpListRR.Code, grpListRR.Body.String())
	}

	grpGetReq := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups/"+createdGroup.ID, nil)
	grpGetRR := httptest.NewRecorder()
	r.ServeHTTP(grpGetRR, grpGetReq)
	if grpGetRR.Code != http.StatusOK {
		t.Fatalf("GET /scim/v2/Groups/{id} = %d body=%s", grpGetRR.Code, grpGetRR.Body.String())
	}

	grpDelReq := httptest.NewRequest(http.MethodDelete, "/scim/v2/Groups/"+createdGroup.ID, nil)
	grpDelRR := httptest.NewRecorder()
	r.ServeHTTP(grpDelRR, grpDelReq)
	if grpDelRR.Code != http.StatusNoContent {
		t.Fatalf("DELETE /scim/v2/Groups/{id} = %d, want 204 body=%s", grpDelRR.Code, grpDelRR.Body.String())
	}

	// 12. DELETE SCIM user -> 204 and subsequent GET -> 404.
	delUserReq := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/scim-u-bob", nil)
	delUserRR := httptest.NewRecorder()
	r.ServeHTTP(delUserRR, delUserReq)
	if delUserRR.Code != http.StatusNoContent {
		t.Fatalf("DELETE /scim/v2/Users/{id} = %d, want 204 body=%s", delUserRR.Code, delUserRR.Body.String())
	}
	bobRoles, err = repo.ListUserRoles(ctx, tid, "scim-u-bob")
	if err != nil || len(bobRoles) != 0 {
		t.Fatalf("expected deleted SCIM user roles purged, got (%+v, %v)", bobRoles, err)
	}
	getDeletedRR := httptest.NewRecorder()
	r.ServeHTTP(getDeletedRR, httptest.NewRequest(http.MethodGet, "/scim/v2/Users/scim-u-bob", nil))
	if getDeletedRR.Code != http.StatusNotFound {
		t.Fatalf("GET deleted SCIM user = %d, want 404", getDeletedRR.Code)
	}

	auditSpy.mu.Lock()
	evCount := len(auditSpy.events)
	auditSpy.mu.Unlock()
	if evCount < 8 {
		t.Fatalf("expected >= 8 AuditEvents across SCIM + custom role lifecycle, got %d", evCount)
	}
}

func TestSCIM_LiveSpanner_Users_Groups_RolesSync_And_Outbox(t *testing.T) {
	client := openIAMSpannerTestClient(t)
	ctx := context.Background()
	tid := "sup_scim_" + uuid.NewString()[:8]
	cleanupSCIMAndCustomRoleTenant(t, client, tid)

	auditSpy := &spyAuditRecorder{}
	svc := NewService(client)
	svc.SetAuditRecorder(auditSpy)

	r := newSCIMTestRouter(svc, tid)

	// 1. Create custom role on live Spanner first.
	reqRole := httptest.NewRequest(http.MethodPost, "/v1/iam/custom-roles", strings.NewReader(`{
		"role_name": "WAREHOUSE_CLERK_CUSTOM",
		"description": "Live Spanner SCIM test role",
		"permissions": ["stock.count", "audit.read"]
	}`))
	rrRole := httptest.NewRecorder()
	r.ServeHTTP(rrRole, reqRole)
	if rrRole.Code != http.StatusCreated {
		t.Fatalf("live POST custom-roles = %d body=%s", rrRole.Code, rrRole.Body.String())
	}

	// 2. Provision SCIM User assigned WAREHOUSE_CLERK_CUSTOM.
	reqUser := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{
		"id": "scim-live-u1",
		"externalId": "idp-u-1",
		"userName": "zara.wms",
		"displayName": "Zara WMS",
		"emails": [{"value":"zara@example.com","primary":true}],
		"active": true,
		"roles": [{"value":"WAREHOUSE_CLERK_CUSTOM"}]
	}`))
	rrUser := httptest.NewRecorder()
	r.ServeHTTP(rrUser, reqUser)
	if rrUser.Code != http.StatusCreated {
		t.Fatalf("live POST /scim/v2/Users = %d body=%s", rrUser.Code, rrUser.Body.String())
	}

	userRoles, err := svc.repo.ListUserRoles(ctx, tid, "scim-live-u1")
	if err != nil || len(userRoles) != 1 || userRoles[0].RoleName != "WAREHOUSE_CLERK_CUSTOM" {
		t.Fatalf("live SupplierUserRoles = (%+v, %v), want WAREHOUSE_CLERK_CUSTOM", userRoles, err)
	}

	// 3. Provision SCIM User 2 + Group syncing ACCOUNTANT, then attempt SoD conflict PATCH (FINANCE_MANAGER -> 403).
	reqU2 := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{
		"id": "scim-live-u2",
		"userName": "amir.fin",
		"active": true
	}`))
	rrU2 := httptest.NewRecorder()
	r.ServeHTTP(rrU2, reqU2)
	if rrU2.Code != http.StatusCreated {
		t.Fatalf("live POST user2 = %d body=%s", rrU2.Code, rrU2.Body.String())
	}

	reqGrp := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", strings.NewReader(`{
		"id": "scim-live-g1",
		"displayName": "Finance Clerks",
		"roles": [{"value":"ACCOUNTANT"}],
		"members": [{"value":"scim-live-u2"}]
	}`))
	rrGrp := httptest.NewRecorder()
	r.ServeHTTP(rrGrp, reqGrp)
	if rrGrp.Code != http.StatusCreated {
		t.Fatalf("live POST /scim/v2/Groups = %d body=%s", rrGrp.Code, rrGrp.Body.String())
	}

	sodPatch := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/scim-live-u2", strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op":"add","path":"roles","value":[{"value":"FINANCE_MANAGER"}]}]
	}`))
	rrSod := httptest.NewRecorder()
	r.ServeHTTP(rrSod, sodPatch)
	if rrSod.Code != http.StatusForbidden {
		t.Fatalf("live SoD conflict PATCH = %d, want 403 body=%s", rrSod.Code, rrSod.Body.String())
	}

	// 4. Deactivate u1 via PATCH -> purges SupplierUserRoles.
	deactPatch := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/scim-live-u1", strings.NewReader(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op":"replace","path":"active","value":false}]
	}`))
	rrDeact := httptest.NewRecorder()
	r.ServeHTTP(rrDeact, deactPatch)
	if rrDeact.Code != http.StatusOK {
		t.Fatalf("live PATCH deactivate = %d body=%s", rrDeact.Code, rrDeact.Body.String())
	}
	userRolesAfter, err := svc.repo.ListUserRoles(ctx, tid, "scim-live-u1")
	if err != nil || len(userRolesAfter) != 0 {
		t.Fatalf("expected 0 roles after deactivation, got (%+v, %v)", userRolesAfter, err)
	}

	// 5. Delete group & user -> 204 and verify iam.scim.synced outbox emitted.
	if rr := httptest.NewRecorder(); func() int {
		r.ServeHTTP(rr, httptest.NewRequest(http.MethodDelete, "/scim/v2/Groups/scim-live-g1", nil))
		return rr.Code
	}() != http.StatusNoContent {
		t.Fatalf("live DELETE group failed")
	}
	if rr := httptest.NewRecorder(); func() int {
		r.ServeHTTP(rr, httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/scim-live-u1", nil))
		return rr.Code
	}() != http.StatusNoContent {
		t.Fatalf("live DELETE user1 failed")
	}

	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT Payload FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()
	var scimEvents int
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var raw []byte
		if err := row.Columns(&raw); err != nil {
			t.Fatalf("scan Payload: %v", err)
		}
		if strings.Contains(string(raw), `"iam.scim.synced"`) {
			scimEvents++
		}
	}
	if scimEvents < 4 {
		t.Fatalf("expected >= 4 iam.scim.synced OutboxEvents rows, got %d", scimEvents)
	}
}
