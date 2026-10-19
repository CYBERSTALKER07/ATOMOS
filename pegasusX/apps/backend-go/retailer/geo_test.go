package retailer

import (
	"context"
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/uber/h3-go/v4"
)

func TestStampRetailerOptionalCoords_InheritsPack(t *testing.T) {
	t.Parallel()
	country, cell, err := stampRetailerOptionalCoords(context.Background(), 41.3111, 69.2797, "")
	if err != nil {
		t.Fatal(err)
	}
	if country != "UZ" {
		t.Fatalf("country=%q", country)
	}
	h3cell := h3.Cell(h3.IndexFromString(cell))
	if h3cell.Resolution() != 7 {
		t.Fatalf("res=%d", h3cell.Resolution())
	}
}

func TestStampRetailerOptionalCoords_RejectsUS(t *testing.T) {
	t.Parallel()
	_, _, err := stampRetailerOptionalCoords(context.Background(), 41.3111, 69.2797, "US")
	if !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
}
