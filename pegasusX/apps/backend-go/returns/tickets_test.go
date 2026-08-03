package returns

import "testing"

func TestShouldOpenReturnTicket(t *testing.T) {
	cases := []struct {
		reason, source string
		want           bool
	}{
		{"DAMAGED", "RETAILER_CLAIM", true},
		{"WRONG_ITEM", "DRIVER_EXCEPTION", true},
		{"MISSING", "RETAILER_CLAIM", false},
		{"OTHER", "CLAIM", false},
		{"", "RETAILER_CLAIM", false},
		{"CONCEALED_DAMAGE", "CLAIM", true},
	}
	for _, tc := range cases {
		got := shouldOpenReturnTicket(tc.reason, tc.source)
		if got != tc.want {
			t.Fatalf("reason=%q source=%q got %v want %v", tc.reason, tc.source, got, tc.want)
		}
	}
}

func TestBuildTicketNotes(t *testing.T) {
	n := buildTicketNotes("RETAILER_CLAIM", "clm_1", "DAMAGED")
	if n != "source=RETAILER_CLAIM | claim_id=clm_1 | DAMAGED" {
		t.Fatalf("got %q", n)
	}
}

func TestNormalizeTicketReason(t *testing.T) {
	if normalizeTicketReason("") != "DAMAGED" {
		t.Fatal("empty should default DAMAGED")
	}
	if normalizeTicketReason(" missing ") != "MISSING" {
		t.Fatal("want MISSING")
	}
}
