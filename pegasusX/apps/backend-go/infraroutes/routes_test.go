package infraroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/redis/go-redis/v9"
)

func TestMetricsAndHealthStayPublicForScrape(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{})
	for _, path := range []string{"/healthz", "/ready", "/metrics", "/v1/health"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
			t.Fatalf("%s status=%d (scrape/probe must stay open)", path, rr.Code)
		}
	}
}

func TestRedisDebugRequiresPlatformAdmin(t *testing.T) {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("infra-test-secret"))
	RegisterRoutes(r, Deps{
		RedisPoolStats: func() *redis.PoolStats { return &redis.PoolStats{TotalConns: 1} },
	})

	anon := httptest.NewRequest(http.MethodGet, "/debug/infra/redis", nil)
	anonRec := httptest.NewRecorder()
	r.ServeHTTP(anonRec, anon)
	if anonRec.Code != http.StatusUnauthorized {
		t.Fatalf("anon status=%d want 401", anonRec.Code)
	}

	supTok, err := auth.Issue(auth.Claims{Subject: "a", Role: auth.RoleAdmin, SupplierID: "s1"}, auth.IssueOptions{
		Secret: "infra-test-secret", TTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	sup := httptest.NewRequest(http.MethodGet, "/debug/infra/redis", nil)
	sup.Header.Set("Authorization", "Bearer "+supTok)
	supRec := httptest.NewRecorder()
	r.ServeHTTP(supRec, sup)
	if supRec.Code != http.StatusForbidden {
		t.Fatalf("supplier ADMIN status=%d want 403", supRec.Code)
	}

	platTok, err := auth.Issue(auth.Claims{Subject: "p", Role: auth.RolePlatformAdmin}, auth.IssueOptions{
		Secret: "infra-test-secret", TTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	plat := httptest.NewRequest(http.MethodGet, "/debug/infra/redis", nil)
	plat.Header.Set("Authorization", "Bearer "+platTok)
	platRec := httptest.NewRecorder()
	r.ServeHTTP(platRec, plat)
	if platRec.Code != http.StatusOK {
		t.Fatalf("platform status=%d body=%s want 200", platRec.Code, platRec.Body.String())
	}
}
