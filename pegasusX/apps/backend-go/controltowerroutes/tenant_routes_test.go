package controltowerroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
)

type memoryMockRepo struct {
	playbooks map[string]controltower.Playbook
	runs      map[string]controltower.PlaybookRun
}

func (m *memoryMockRepo) ListActivePlaybooks(ctx context.Context, supplierID string) ([]controltower.Playbook, error) {
	var out []controltower.Playbook
	for _, p := range m.playbooks {
		if p.SupplierID == supplierID && p.IsActive {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *memoryMockRepo) GetPlaybook(ctx context.Context, playbookID string) (controltower.Playbook, error) {
	p, ok := m.playbooks[playbookID]
	if !ok {
		return controltower.Playbook{}, context.Canceled
	}
	return p, nil
}

func (m *memoryMockRepo) CreatePlaybook(ctx context.Context, pb controltower.Playbook) error {
	m.playbooks[pb.PlaybookID] = pb
	return nil
}

func (m *memoryMockRepo) UpdatePlaybook(ctx context.Context, playbookID string, fields map[string]any) error {
	p, ok := m.playbooks[playbookID]
	if !ok {
		return context.Canceled
	}
	if name, ok := fields["Name"].(string); ok {
		p.Name = name
	}
	if active, ok := fields["IsActive"].(bool); ok {
		p.IsActive = active
	}
	m.playbooks[playbookID] = p
	return nil
}

func (m *memoryMockRepo) DeactivatePlaybook(ctx context.Context, playbookID string) error {
	p, ok := m.playbooks[playbookID]
	if !ok {
		return context.Canceled
	}
	p.IsActive = false
	m.playbooks[playbookID] = p
	return nil
}

func (m *memoryMockRepo) ListOpenExceptions(ctx context.Context, supplierID string) ([]controltower.Exception, error) {
	return nil, nil
}

func (m *memoryMockRepo) ListSupplierIDsWithOpenExceptions(ctx context.Context) ([]string, error) {
	return []string{"sup-1"}, nil
}

func (m *memoryMockRepo) HasBlockingRun(ctx context.Context, exceptionID string) (bool, error) {
	return false, nil
}

func (m *memoryMockRepo) CreateRun(ctx context.Context, run controltower.PlaybookRun) error {
	m.runs[run.RunID] = run
	return nil
}

func (m *memoryMockRepo) GetRun(ctx context.Context, runID string) (controltower.PlaybookRun, error) {
	r, ok := m.runs[runID]
	if !ok {
		return controltower.PlaybookRun{}, context.Canceled
	}
	return r, nil
}

func (m *memoryMockRepo) UpdateRun(ctx context.Context, run controltower.PlaybookRun) error {
	m.runs[run.RunID] = run
	return nil
}

func (m *memoryMockRepo) ListRuns(ctx context.Context, supplierID, status string, limit int) ([]controltower.PlaybookRun, error) {
	var out []controltower.PlaybookRun
	for _, r := range m.runs {
		if r.SupplierID == supplierID && (status == "" || r.Status == status) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memoryMockRepo) ListRunsForException(ctx context.Context, supplierID, exceptionID string) ([]controltower.PlaybookRun, error) {
	var out []controltower.PlaybookRun
	for _, r := range m.runs {
		if r.SupplierID == supplierID && r.ExceptionID == exceptionID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memoryMockRepo) UpdateExceptionStatus(ctx context.Context, exceptionID, status string) error {
	return nil
}

func (m *memoryMockRepo) UpdateExceptionAssignee(ctx context.Context, exceptionID, role string) error {
	return nil
}

func (m *memoryMockRepo) RecordIntervention(ctx context.Context, manifestID, supplierID, operatorID, commandType, reasonCode, notes string) error {
	return nil
}

func issueSupplierToken(t *testing.T, supplierID string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "admin-user",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: "controltower-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func setupTestHandlers() (*controltower.Handlers, *memoryMockRepo) {
	repo := &memoryMockRepo{
		playbooks: map[string]controltower.Playbook{
			"pb-sup1": {PlaybookID: "pb-sup1", SupplierID: "sup-1", Name: "Sup1 Playbook", IsActive: true},
			"pb-sup2": {PlaybookID: "pb-sup2", SupplierID: "sup-2", Name: "Sup2 Playbook", IsActive: true},
			"pb-plat": {PlaybookID: "pb-plat", SupplierID: "", Name: "Platform Global Playbook", IsActive: true},
		},
		runs: map[string]controltower.PlaybookRun{
			"run-sup1": {
				RunID: "run-sup1", PlaybookID: "pb-sup1", ExceptionID: "exc-1",
				SupplierID: "sup-1", Status: controltower.RunStatusSuggested, Mode: controltower.RunModeSuggested,
			},
			"run-sup2": {
				RunID: "run-sup2", PlaybookID: "pb-sup2", ExceptionID: "exc-1",
				SupplierID: "sup-2", Status: controltower.RunStatusSuggested, Mode: controltower.RunModeSuggested,
			},
		},
	}
	engine := controltower.NewEngine(repo, controltower.NewActionExecutor(repo, controltower.ExecutorDeps{}), nil, controltower.Config{Enabled: true}, nil)
	svc := controltower.NewService(repo, engine, controltower.Config{Enabled: true})
	h := controltower.NewHandlers(svc, nil) // supplierID will fall back to claims.SupplierID
	return h, repo
}

func TestControlTowerRoutes_TenantIsolation_PlaybookMutations(t *testing.T) {
	h, _ := setupTestHandlers()
	router := controlTowerRouter(h)

	tokenSup1 := issueSupplierToken(t, "sup-1")

	// 1. PATCH /v1/control-tower/playbooks/pb-sup2 with sup-1 token -> 403 Forbidden
	patchBody, _ := json.Marshal(map[string]string{"name": "Hacked Name"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/control-tower/playbooks/pb-sup2", bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant PATCH playbook want 403, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// 2. PATCH /v1/control-tower/playbooks/pb-plat (platform) with sup-1 token -> 403 Forbidden
	req = httptest.NewRequest(http.MethodPatch, "/v1/control-tower/playbooks/pb-plat", bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("platform PATCH playbook want 403, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// 3. POST /v1/control-tower/playbooks/pb-sup2/deactivate with sup-1 token -> 403 Forbidden
	req = httptest.NewRequest(http.MethodPost, "/v1/control-tower/playbooks/pb-sup2/deactivate", nil)
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant deactivate playbook want 403, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// 4. POST /v1/control-tower/playbooks/pb-plat/deactivate with sup-1 token -> 403 Forbidden
	req = httptest.NewRequest(http.MethodPost, "/v1/control-tower/playbooks/pb-plat/deactivate", nil)
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("platform deactivate playbook want 403, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// 5. Legitimate PATCH /v1/control-tower/playbooks/pb-sup1 with sup-1 token -> 200 OK
	legitBody, _ := json.Marshal(map[string]string{"name": "Legit Updated Name"})
	req = httptest.NewRequest(http.MethodPatch, "/v1/control-tower/playbooks/pb-sup1", bytes.NewReader(legitBody))
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("legitimate PATCH playbook want 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// 6. Legitimate POST /v1/control-tower/playbooks/pb-sup1/deactivate with sup-1 token -> 200 OK
	req = httptest.NewRequest(http.MethodPost, "/v1/control-tower/playbooks/pb-sup1/deactivate", nil)
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("legitimate deactivate playbook want 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestControlTowerRoutes_TenantIsolation_RunActionsAndListing(t *testing.T) {
	h, _ := setupTestHandlers()
	router := controlTowerRouter(h)

	tokenSup1 := issueSupplierToken(t, "sup-1")

	// 1. POST /v1/control-tower/runs/run-sup2/approve with sup-1 token -> 403 Forbidden
	req := httptest.NewRequest(http.MethodPost, "/v1/control-tower/runs/run-sup2/approve", nil)
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant approve want 403, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// 2. POST /v1/control-tower/runs/run-sup2/skip with sup-1 token -> 403 Forbidden
	req = httptest.NewRequest(http.MethodPost, "/v1/control-tower/runs/run-sup2/skip", nil)
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant skip want 403, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// 3. GET /v1/control-tower/runs?exception_id=exc-1 with sup-1 token -> 200 OK, only contains run-sup1
	req = httptest.NewRequest(http.MethodGet, "/v1/control-tower/runs?exception_id=exc-1", nil)
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("runs by exception want 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	var res struct {
		Runs []controltower.PlaybookRun `json:"runs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Runs) != 1 || res.Runs[0].RunID != "run-sup1" {
		t.Fatalf("expected exactly [run-sup1], got %+v", res.Runs)
	}

	// 4. Legitimate skip on own run -> 200 OK
	req = httptest.NewRequest(http.MethodPost, "/v1/control-tower/runs/run-sup1/skip", nil)
	req.Header.Set("Authorization", "Bearer "+tokenSup1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("legitimate skip want 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}
