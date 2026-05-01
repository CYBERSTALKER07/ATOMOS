package notifications

import (
	"testing"

	"backend-go/auth"
)

func TestNotificationRecipientID_UsesSupplierScopeForSupplier(t *testing.T) {
	claims := &auth.PegasusClaims{
		UserID:     "user-1",
		SupplierID: "supplier-1",
		Role:       "SUPPLIER",
	}

	if got := notificationRecipientID(claims); got != "supplier-1" {
		t.Fatalf("recipientID = %q, want %q", got, "supplier-1")
	}
}

func TestNotificationRecipientID_UsesSupplierScopeForPayloader(t *testing.T) {
	claims := &auth.PegasusClaims{
		UserID:     "worker-1",
		SupplierID: "supplier-1",
		Role:       "PAYLOADER",
	}

	if got := notificationRecipientID(claims); got != "supplier-1" {
		t.Fatalf("recipientID = %q, want %q", got, "supplier-1")
	}
}

func TestNotificationRecipientID_UsesUserScopeForRetailer(t *testing.T) {
	claims := &auth.PegasusClaims{
		UserID: "retailer-1",
		Role:   "RETAILER",
	}

	if got := notificationRecipientID(claims); got != "retailer-1" {
		t.Fatalf("recipientID = %q, want %q", got, "retailer-1")
	}
}

func TestShouldUseNotificationInboxCache_DefaultQuery(t *testing.T) {
	if !shouldUseNotificationInboxCache(false, defaultNotificationInboxLimit, defaultNotificationInboxOffset) {
		t.Fatal("expected cache to be enabled for default inbox query")
	}
}

func TestShouldUseNotificationInboxCache_NonDefaultQuery(t *testing.T) {
	tests := []struct {
		name       string
		unreadOnly bool
		limit      int64
		offset     int64
		want       bool
	}{
		{name: "unread only", unreadOnly: true, limit: defaultNotificationInboxLimit, offset: defaultNotificationInboxOffset, want: false},
		{name: "custom limit", unreadOnly: false, limit: 25, offset: defaultNotificationInboxOffset, want: false},
		{name: "custom offset", unreadOnly: false, limit: defaultNotificationInboxLimit, offset: 10, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldUseNotificationInboxCache(test.unreadOnly, test.limit, test.offset); got != test.want {
				t.Fatalf("shouldUseNotificationInboxCache(%v,%d,%d)=%v, want %v", test.unreadOnly, test.limit, test.offset, got, test.want)
			}
		})
	}
}

func TestNotificationIDSuffix_HandlesShortIDs(t *testing.T) {
	if got := notificationIDSuffix("abc"); got != "abc" {
		t.Fatalf("notificationIDSuffix(short) = %q, want %q", got, "abc")
	}

	if got := notificationIDSuffix("1234567890"); got != "12345678" {
		t.Fatalf("notificationIDSuffix(long) = %q, want %q", got, "12345678")
	}
}
