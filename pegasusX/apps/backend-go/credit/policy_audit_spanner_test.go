package credit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

func TestCreditPolicyAudit_AppendOnlyInsertMap_Emulator(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	repo := NewSpannerPolicyRepository(client)
	auditID := fmt.Sprintf("cpa-append-%d", time.Now().UnixNano())
	supID := fmt.Sprintf("sup-cpa-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("CreditPolicyAudit", spanner.Key{auditID}),
		})
	})

	row := CreditPolicyAuditRow{
		AuditID:     auditID,
		SupplierID:  supID,
		Action:      "TERMS_PATCH",
		ActorUserID: "usr-1",
		ActorRole:   "FINANCE_MANAGER",
		BeforeJSON:  `{"limit":100}`,
		AfterJSON:   `{"limit":200}`,
	}
	if err := repo.AppendAudit(ctx, row); err != nil {
		t.Fatalf("first AppendAudit failed: %v", err)
	}
	if err := repo.AppendAudit(ctx, row); spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("expected codes.AlreadyExists on duplicate AuditId (append-only InsertMap), got %v", err)
	}
}
