package warehouse

import "testing"

func TestInsightWireStatus(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"PENDING", "OPEN"},
		{"pending", "OPEN"},
		{"APPROVED", "APPROVED"},
		{"DISMISSED", "DISMISSED"},
	}
	for _, tt := range tests {
		if got := insightWireStatus(tt.in); got != tt.want {
			t.Fatalf("insightWireStatus(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestInsightWireUrgency(t *testing.T) {
	if got := insightWireUrgency("WARNING"); got != "HIGH" {
		t.Fatalf("got %q want HIGH", got)
	}
	if got := insightWireUrgency("CRITICAL"); got != "CRITICAL" {
		t.Fatalf("got %q want CRITICAL", got)
	}
}

func TestInsightDBStatus(t *testing.T) {
	if got := insightDBStatus(""); got != "PENDING" {
		t.Fatalf("empty status filter = %q", got)
	}
	if got := insightDBStatus("OPEN"); got != "PENDING" {
		t.Fatalf("OPEN status filter = %q", got)
	}
}
