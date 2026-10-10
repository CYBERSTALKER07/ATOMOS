package retailer

import (
	"context"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newEmulatorClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	host := os.Getenv("SPANNER_EMULATOR_HOST")
	if host == "" {
		return nil
	}
	db := os.Getenv("SPANNER_DATABASE")
	if db == "" {
		db = "projects/test-project/instances/test-instance/databases/test-db"
	}
	client, err := spanner.NewClient(ctx, db, option.WithEndpoint(host), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("failed to create spanner client: %v", err)
	}
	return client
}

func TestSpannerRepository_KycDocuments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := newEmulatorClient(t, ctx)
	if client == nil {
		t.Skip("Spanner not configured")
	}

	repo := &SpannerRepository{client: client, supplierID: "sup1"}

	retID := uuid.New().String()
	docID := uuid.New().String()

	doc := KycDocument{
		DocumentID:   docID,
		RetailerID:   retID,
		Status:       "PENDING",
		DocumentType: "PASSPORT",
		DocumentURL:  "https://example.com/doc.pdf",
	}

	if err := repo.InsertKycDocument(ctx, doc); err != nil {
		t.Fatalf("InsertKycDocument failed: %v", err)
	}

	docs, err := repo.ListKycDocumentsByRetailer(ctx, retID)
	if err != nil {
		t.Fatalf("ListKycDocumentsByRetailer failed: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Status != "PENDING" {
		t.Fatalf("expected PENDING status, got %s", docs[0].Status)
	}

	if err := repo.UpdateKycDocumentStatus(ctx, docID, "APPROVED", "admin1", ""); err != nil {
		t.Fatalf("UpdateKycDocumentStatus failed: %v", err)
	}

	docs, err = repo.ListKycDocumentsByRetailer(ctx, retID)
	if err != nil {
		t.Fatalf("ListKycDocumentsByRetailer failed: %v", err)
	}
	if docs[0].Status != "APPROVED" {
		t.Fatalf("expected APPROVED status, got %s", docs[0].Status)
	}
	if docs[0].ReviewedBy != "admin1" {
		t.Fatalf("expected ReviewedBy=admin1, got %s", docs[0].ReviewedBy)
	}
}
