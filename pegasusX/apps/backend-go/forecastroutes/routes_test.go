package forecastroutes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/forecastv2"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
)

const testSecret = "forecast-routes-secret"

func setupForecastRouter(svc *forecastv2.Service, repl *replenishment.SuggestionsAPI) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth(testSecret))
	RegisterRoutes(r, Deps{
		Service:          svc,
		ReplenishmentAPI: repl,
		JWTSecret:        testSecret,
	})
	return r
}

func issueToken(t *testing.T, sub string, role auth.Role, supplierID, retailerID, supRole, actorType string, perms ...string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:       sub,
		Role:          role,
		SupplierID:    supplierID,
		RetailerOrgID: retailerID,
		SupplierRole:  auth.Role(supRole),
		ActorType:     actorType,
		SupplierPerms: perms,
	}, auth.IssueOptions{Secret: testSecret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestForecastRoutes_AllNilAndAuthGuards(t *testing.T) {
	rNil := setupForecastRouter(nil, nil)
	rrNil := httptest.NewRecorder()
	rNil.ServeHTTP(rrNil, httptest.NewRequest(http.MethodGet, "/v1/planning/forecast", nil))
	if rrNil.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when nil deps, got %d", rrNil.Code)
	}

	repo := forecastv2.NewMemoryRepository()
	svc := forecastv2.NewServiceWithRepository(repo)
	replStore := replenishment.NewMemorySuggestionStore()
	replAPI := replenishment.NewSuggestionsAPI(nil, nil, func(r *http.Request) string {
		if c, ok := auth.FromContext(r.Context()); ok {
			return c.SupplierID
		}
		return ""
	})
	replAPI.SetMemoryStore(replStore)
	r := setupForecastRouter(svc, replAPI)

	// Unauthenticated -> 401.
	rrUnauth := httptest.NewRecorder()
	r.ServeHTTP(rrUnauth, httptest.NewRequest(http.MethodGet, "/v1/planning/forecast", nil))
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rrUnauth.Code)
	}

	// Retailer cannot POST /v1/planning/forecast/run -> 403.
	retTok := issueToken(t, "ret-user", auth.RoleRetailer, "", "ret-1", "", auth.ActorTypeHuman)
	reqRunRet := httptest.NewRequest(http.MethodPost, "/v1/planning/forecast/run", strings.NewReader(`{}`))
	reqRunRet.Header.Set("Authorization", "Bearer "+retTok)
	rrRunRet := httptest.NewRecorder()
	r.ServeHTTP(rrRunRet, reqRunRet)
	if rrRunRet.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for retailer forecast run, got %d", rrRunRet.Code)
	}
}

func TestForecastRoutes_RunGetApproveAndOneTapSuggestion(t *testing.T) {
	repo := forecastv2.NewMemoryRepository()
	asOf := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	y := make([]int64, 30)
	for i := range y {
		y[i] = 12_000
	}
	repo.SeedSeries("sup-1", []forecastv2.SeriesHistoryInput{
		{LocationID: "wh-1", SKU: "sku-1", DailyUnitsMilli: y},
	}, nil)
	repo.SeedRetailerLink("sup-1", "ret-1")
	repo.SeedActualMilli("sup-1", "wh-1", "sku-1", 13_000)

	svc := forecastv2.NewServiceWithRepository(repo)
	svc.SetNow(func() time.Time { return asOf })

	replStore := replenishment.NewMemorySuggestionStore()
	replStore.Seed("sup-1", replenishment.ReorderSuggestion{
		RetailerId:      "ret-1",
		Sku:             "sku-1",
		SuggestedQty:    24,
		SuggestedByDate: civil.Date{Year: 2026, Month: time.March, Day: 23},
		Status:          replenishment.SuggestionStatusOpen,
	})
	replAPI := replenishment.NewSuggestionsAPI(nil, nil, func(r *http.Request) string {
		if c, ok := auth.FromContext(r.Context()); ok {
			return c.SupplierID
		}
		return ""
	})
	replAPI.SetMemoryStore(replStore)

	r := setupForecastRouter(svc, replAPI)
	adminTok := issueToken(t, "owner-1", auth.RoleAdmin, "sup-1", "", auth.SupplierRoleOwner, auth.ActorTypeHuman)
	agentTok := issueToken(t, "agent-1", auth.RoleAdmin, "sup-1", "", auth.SupplierRoleAgent, auth.ActorTypeAgent)
	audTok := issueToken(t, "aud-1", auth.RoleAdmin, "sup-1", "", auth.SupplierRoleAuditor, auth.ActorTypeHuman)
	retTok := issueToken(t, "ret-user", auth.RoleRetailer, "", "ret-1", "", auth.ActorTypeHuman)

	// 1. Float in JSON body rejected cleanly.
	reqFloat := httptest.NewRequest(http.MethodPost, "/v1/planning/forecast/run", strings.NewReader(`{"horizon_days":7.5}`))
	reqFloat.Header.Set("Authorization", "Bearer "+adminTok)
	rrFloat := httptest.NewRecorder()
	r.ServeHTTP(rrFloat, reqFloat)
	if rrFloat.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on float input, got %d body=%s", rrFloat.Code, rrFloat.Body.String())
	}

	// 2. Run forecast -> 201 with run, forecasts, explanations.
	reqRun := httptest.NewRequest(http.MethodPost, "/v1/planning/forecast/run", strings.NewReader(`{"horizon_days":7,"promo_discount_bps":1000}`))
	reqRun.Header.Set("Authorization", "Bearer "+adminTok)
	rrRun := httptest.NewRecorder()
	r.ServeHTTP(rrRun, reqRun)
	if rrRun.Code != http.StatusCreated {
		t.Fatalf("expected 201 on run, got %d body=%s", rrRun.Code, rrRun.Body.String())
	}
	var runOut forecastv2.RunForecastOutput
	if err := json.Unmarshal(rrRun.Body.Bytes(), &runOut); err != nil {
		t.Fatal(err)
	}
	if runOut.Run.RunID == "" || len(runOut.Forecasts) != 14 || len(runOut.Explanations) != 1 {
		t.Fatalf("unexpected run output: %+v", runOut)
	}
	expID := runOut.Explanations[0].ExplanationID

	// 3. GET /v1/planning/forecast as admin and linked retailer -> 200.
	for _, tok := range []string{adminTok, retTok} {
		reqGet := httptest.NewRequest(http.MethodGet, "/v1/planning/forecast?supplier_id=sup-1&sku=sku-1", nil)
		reqGet.Header.Set("Authorization", "Bearer "+tok)
		rrGet := httptest.NewRecorder()
		r.ServeHTTP(rrGet, reqGet)
		if rrGet.Code != http.StatusOK {
			t.Fatalf("GET forecast status=%d body=%s", rrGet.Code, rrGet.Body.String())
		}
	}

	// 4. AGENT and AUDITOR blocked from approving explanation -> 403.
	for _, badTok := range []string{agentTok, audTok} {
		reqBad := httptest.NewRequest(http.MethodPost, "/v1/planning/forecast/explanations/"+expID+"/approve", strings.NewReader(`{}`))
		reqBad.Header.Set("Authorization", "Bearer "+badTok)
		rrBad := httptest.NewRecorder()
		r.ServeHTTP(rrBad, reqBad)
		if rrBad.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for restricted actor, got %d body=%s", rrBad.Code, rrBad.Body.String())
		}
	}

	// 5. Human owner approves explanation -> 200.
	reqAppExp := httptest.NewRequest(http.MethodPost, "/v1/planning/forecast/explanations/"+expID+"/approve", strings.NewReader(`{"adjustment_bps":1000,"comment":"approved"}`))
	reqAppExp.Header.Set("Authorization", "Bearer "+adminTok)
	rrAppExp := httptest.NewRecorder()
	r.ServeHTTP(rrAppExp, reqAppExp)
	if rrAppExp.Code != http.StatusOK {
		t.Fatalf("expected 200 on explanation approve, got %d body=%s", rrAppExp.Code, rrAppExp.Body.String())
	}

	// 6. AGENT blocked from approving suggestion -> 403.
	reqAgentSug := httptest.NewRequest(http.MethodPost, "/v1/replenishment/suggestions/ret-1:sku-1/approve", strings.NewReader(`{}`))
	reqAgentSug.Header.Set("Authorization", "Bearer "+agentTok)
	rrAgentSug := httptest.NewRecorder()
	r.ServeHTTP(rrAgentSug, reqAgentSug)
	if rrAgentSug.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for agent suggestion approve, got %d", rrAgentSug.Code)
	}

	// 7. Retailer one-tap approves suggestion by {sku} -> 200 APPROVED!
	reqRetSug := httptest.NewRequest(http.MethodPost, "/v1/replenishment/suggestions/sku-1/approve", strings.NewReader(`{"qty":30}`))
	reqRetSug.Header.Set("Authorization", "Bearer "+retTok)
	rrRetSug := httptest.NewRecorder()
	r.ServeHTTP(rrRetSug, reqRetSug)
	if rrRetSug.Code != http.StatusOK {
		t.Fatalf("expected 200 on retailer suggestion approve, got %d body=%s", rrRetSug.Code, rrRetSug.Body.String())
	}
	var sugResp replenishment.ApproveSuggestionResponse
	if err := json.Unmarshal(rrRetSug.Body.Bytes(), &sugResp); err != nil {
		t.Fatal(err)
	}
	if sugResp.Status != replenishment.SuggestionStatusApproved || sugResp.ApprovedQty != 30 || sugResp.SuggestionID != "ret-1:sku-1" {
		t.Fatalf("unexpected suggestion approve response: %+v", sugResp)
	}

	// Idempotent re-approve returns already_approved=true.
	reqRetSug2 := httptest.NewRequest(http.MethodPost, "/v1/replenishment/suggestions/ret-1:sku-1/approve", strings.NewReader(`{}`))
	reqRetSug2.Header.Set("Authorization", "Bearer "+retTok)
	rrRetSug2 := httptest.NewRecorder()
	r.ServeHTTP(rrRetSug2, reqRetSug2)
	if rrRetSug2.Code != http.StatusOK {
		t.Fatalf("expected 200 on idempotent re-approve, got %d", rrRetSug2.Code)
	}
}
