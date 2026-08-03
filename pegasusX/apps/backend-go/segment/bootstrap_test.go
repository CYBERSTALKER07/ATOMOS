package segment

import "testing"

func TestBootstrapSegment(t *testing.T) {
	cases := []struct {
		name string
		in   RetailerBootstrapInput
		want string
	}{
		{
			name: "claims spike to C",
			in:   RetailerBootstrapInput{ClaimCount: 3, OrderCount: 2},
			want: SegmentC,
		},
		{
			name: "top volume + enough orders to A",
			in:   RetailerBootstrapInput{InTopVolume: true, OrderCount: 20},
			want: SegmentA,
		},
		{
			name: "volume threshold to A",
			in:   RetailerBootstrapInput{OrderCount: 25},
			want: SegmentA,
		},
		{
			name: "mid volume to B",
			in:   RetailerBootstrapInput{OrderCount: 10},
			want: SegmentB,
		},
		{
			name: "high orders without claims not forced to C",
			in:   RetailerBootstrapInput{OrderCount: 100},
			want: SegmentA,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := bootstrapSegment(tc.in); got != tc.want {
				t.Fatalf("bootstrapSegment() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBootstrapSkuClass(t *testing.T) {
	fast := bootstrapSkuClass(SkuBootstrapInput{
		Sku:          "FAST",
		OrderQty:     500,
		VelocityRank: 0.9,
	})
	if fast.VelocityClass != VelocityA {
		t.Fatalf("fast sku class = %s, want A", fast.VelocityClass)
	}
	slow := bootstrapSkuClass(SkuBootstrapInput{
		Sku:          "SLOW",
		OrderQty:     1,
		VelocityRank: 0.1,
	})
	if slow.VelocityClass != VelocityC {
		t.Fatalf("slow sku class = %s, want C", slow.VelocityClass)
	}
}
