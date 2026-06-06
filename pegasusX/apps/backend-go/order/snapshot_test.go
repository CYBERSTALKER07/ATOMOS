package order

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func TestSnapshotReceivingWindowsOnOrder(t *testing.T) {
	tests := []struct {
		name         string
		retailerOpen string
		retailerClose string
		wantOpen     string
		wantClose    string
		wantErr      bool
	}{
		{
			name:         "canonicalizes padded windows",
			retailerOpen: "9:00",
			retailerClose: "18:30",
			wantOpen:     "09:00",
			wantClose:    "18:30",
		},
		{
			name:         "empty retailer windows",
			retailerOpen: "",
			retailerClose: "",
			wantOpen:     "",
			wantClose:    "",
		},
		{
			name:         "rejects invalid open",
			retailerOpen: "25:00",
			retailerClose: "18:00",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Order{OrderID: "ord-1", RetailerID: "ret-1"}
			err := SnapshotReceivingWindowsOnOrder(o, tt.retailerOpen, tt.retailerClose)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if o.ReceivingWindowOpen != tt.wantOpen {
				t.Fatalf("open = %q, want %q", o.ReceivingWindowOpen, tt.wantOpen)
			}
			if o.ReceivingWindowClose != tt.wantClose {
				t.Fatalf("close = %q, want %q", o.ReceivingWindowClose, tt.wantClose)
			}
		})
	}
}

func TestSnapshotReceivingWindowsOnOrderNilOrder(t *testing.T) {
	if err := SnapshotReceivingWindowsOnOrder(nil, "09:00", "18:00"); err == nil {
		t.Fatal("expected error for nil order")
	}
}

func TestSnapshotReceivingWindowsRejectsInvalidClose(t *testing.T) {
	o := &Order{}
	err := SnapshotReceivingWindowsOnOrder(o, "09:00", "99:00")
	if err == nil {
		t.Fatal("expected validation error")
	}
	_ = proximity.ErrInvalidReceivingWindow
}
