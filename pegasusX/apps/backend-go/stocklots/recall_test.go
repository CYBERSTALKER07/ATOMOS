package stocklots

import (
	"context"
	"testing"
)

func TestInitiateRecallValidation(t *testing.T) {
	ctx := context.Background()

	// Missing supplier_id
	_, err := InitiateRecallInTxn(ctx, nil, nil, InitiateRecallRequest{
		ProductID:    "prod-1",
		RecallReason: "Contamination detected",
	})
	if err == nil {
		t.Fatal("expected error for missing supplier_id")
	}

	// Missing product_id
	_, err = InitiateRecallInTxn(ctx, nil, nil, InitiateRecallRequest{
		SupplierID:   "sup-1",
		RecallReason: "Contamination detected",
	})
	if err == nil {
		t.Fatal("expected error for missing product_id")
	}

	// Missing recall_reason
	_, err = InitiateRecallInTxn(ctx, nil, nil, InitiateRecallRequest{
		SupplierID: "sup-1",
		ProductID:  "prod-1",
	})
	if err == nil {
		t.Fatal("expected error for missing recall_reason")
	}
}

func TestQuarantineLotValidation(t *testing.T) {
	ctx := context.Background()

	// Missing lot_id
	_, err := QuarantineLotInTxn(ctx, nil, nil, "", "wh-1", "QUALITY_HOLD", "user-1", "")
	if err == nil {
		t.Fatal("expected error for missing lot_id")
	}
}

func TestReleaseLotValidation(t *testing.T) {
	ctx := context.Background()

	// Missing lot_id
	_, err := ReleaseLotInTxn(ctx, nil, nil, "", "wh-1", "QUALITY_RELEASE", "user-1", "")
	if err == nil {
		t.Fatal("expected error for missing lot_id")
	}
}
