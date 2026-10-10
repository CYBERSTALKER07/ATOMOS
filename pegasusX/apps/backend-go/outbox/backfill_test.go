package outbox

import "testing"

func TestSupplierIDFromPayloadKeys(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`{"supplier_id":"a"}`, "a"},
		{`{"SupplierId":"b"}`, "b"},
		{`{"supplierId":"c"}`, "c"},
		{`[]`, ""},
	}
	for _, tc := range cases {
		if got := SupplierIDFromPayload([]byte(tc.raw)); got != tc.want {
			t.Fatalf("raw=%s got=%q want=%q", tc.raw, got, tc.want)
		}
	}
}
