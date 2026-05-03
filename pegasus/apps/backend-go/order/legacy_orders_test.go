package order

import "testing"

func TestLegacyOrderPatchTarget(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		wantID string
		wantOK bool
	}{
		{name: "status path", path: "/v1/orders/ord-1/status", wantID: "ord-1", wantOK: true},
		{name: "state path", path: "/v1/orders/ord-1/state", wantID: "ord-1", wantOK: true},
		{name: "detail path", path: "/v1/orders/ord-1", wantID: "", wantOK: false},
		{name: "missing id", path: "/v1/orders//state", wantID: "", wantOK: false},
		{name: "unknown suffix", path: "/v1/orders/ord-1/items", wantID: "", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotID, gotOK := legacyOrderPatchTarget(test.path)
			if gotID != test.wantID || gotOK != test.wantOK {
				t.Fatalf("legacyOrderPatchTarget(%q) = (%q, %v), want (%q, %v)", test.path, gotID, gotOK, test.wantID, test.wantOK)
			}
		})
	}
}
