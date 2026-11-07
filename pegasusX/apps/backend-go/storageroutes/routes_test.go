package storageroutes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/storageroutes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func envOr(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func newEmulatorClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		t.Skip("SPANNER_EMULATOR_HOST is unset; skipping spanner test")
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))

	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host), option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		t.Fatalf("spanner client: %v", err)
	}
	return client
}

func TestMountDossiers(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	vault := storage.NewVault(client)

	r := chi.NewRouter()
	
	// Add mock auth middleware for testing
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := auth.Claims{
				Subject:    "test-user",
				Role:       auth.RoleAdmin,
				SupplierID: "test-supplier",
			}
			ctx := auth.WithClaims(req.Context(), claims)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	
	storageroutes.Mount(r, vault)
	
	server := httptest.NewServer(r)
	defer server.Close()

	// 1. Create Dossier
	createReq := storageroutes.CreateDossierRequest{
		TargetID:   "order-test",
		TargetType: "ORDER_POD",
	}
	body, _ := json.Marshal(createReq)
	resp, err := http.Post(server.URL+"/dossiers", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	
	var dossier storage.EvidenceDossier
	err = json.NewDecoder(resp.Body).Decode(&dossier)
	resp.Body.Close()
	require.NoError(t, err)
	require.NotEmpty(t, dossier.DossierID)

	// 2. Add Item
	itemReq := storageroutes.AddItemRequest{
		ItemType:  "IMAGE",
		FileHash:  "testhash",
		MimeType:  "image/jpeg",
		SizeBytes: 1024,
		Extension: "jpg",
	}
	itemBody, _ := json.Marshal(itemReq)
	resp, err = http.Post(server.URL+"/dossiers/"+dossier.DossierID+"/items", "application/json", bytes.NewBuffer(itemBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// 3. Seal Dossier
	resp, err = http.Post(server.URL+"/dossiers/"+dossier.DossierID+"/seal", "application/json", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 4. Get Dossier
	resp, err = http.Get(server.URL + "/dossiers/" + dossier.DossierID)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
