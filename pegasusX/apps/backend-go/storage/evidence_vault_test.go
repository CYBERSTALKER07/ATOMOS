package storage_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
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

func TestEvidenceVault_EndToEnd(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	vault := storage.NewVault(client)

	// 1. Create Dossier
	targetID := "order-123"
	dossier, err := vault.CreateDossier(ctx, targetID, "ORDER_POD")
	require.NoError(t, err)
	require.NotEmpty(t, dossier.DossierID)
	assert.Equal(t, storage.DossierStatusOpen, dossier.Status)

	// 2. Add item 1
	item1 := &storage.EvidenceItem{
		ItemType:       "IMAGE",
		StorageURI:     "gs://bucket/item1.jpg",
		FileHash:       "hash1",
		MimeType:       "image/jpeg",
		SizeBytes:      1024,
		UploaderUserID: "user1",
	}
	added1, err := vault.AddItem(ctx, dossier.DossierID, item1)
	require.NoError(t, err)
	require.NotEmpty(t, added1.ItemID)

	// 3. Add item 2
	item2 := &storage.EvidenceItem{
		ItemType:       "SIGNATURE",
		StorageURI:     "gs://bucket/item2.png",
		FileHash:       "hash2",
		MimeType:       "image/png",
		SizeBytes:      512,
		UploaderUserID: "user1",
	}
	_, err = vault.AddItem(ctx, dossier.DossierID, item2)
	require.NoError(t, err)

	// 4. Seal Dossier
	err = vault.SealDossier(ctx, dossier.DossierID)
	require.NoError(t, err)

	// 5. Try adding another item to SEALED dossier
	item3 := &storage.EvidenceItem{
		ItemType:       "IMAGE",
		StorageURI:     "gs://bucket/item3.jpg",
		FileHash:       "hash3",
		MimeType:       "image/jpeg",
		SizeBytes:      2048,
		UploaderUserID: "user1",
	}
	_, err = vault.AddItem(ctx, dossier.DossierID, item3)
	require.ErrorIs(t, err, storage.ErrDossierSealed)

	// 6. Fetch and verify
	fetchedDossier, items, err := vault.GetDossier(ctx, dossier.DossierID)
	require.NoError(t, err)
	assert.Equal(t, storage.DossierStatusSealed, fetchedDossier.Status)
	assert.True(t, fetchedDossier.SealedAt.Valid)
	assert.True(t, fetchedDossier.SealedHash.Valid)
	assert.Len(t, items, 2)
}
