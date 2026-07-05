package predictivepush

import (
	"context"
	"testing"
	"time"
)

func TestWriteDemandBaselines_NoClient(t *testing.T) {
	a := &Allocator{}
	if err := a.writeDemandBaselines(context.Background(), []*DemandEvent{
		{SupplierId: "s1", ProductId: "p1", RetailerId: "r1", Quantity: 10, Confidence: 0.8},
	}, time.Now()); err != nil {
		t.Fatal(err)
	}
}
