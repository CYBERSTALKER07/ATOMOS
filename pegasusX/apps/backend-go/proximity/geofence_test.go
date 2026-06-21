package proximity

import "testing"

func TestWithinDeliveryApproach(t *testing.T) {
	if !WithinDeliveryApproach(0.499) {
		t.Fatal("499m should be within approach")
	}
	if WithinDeliveryApproach(0.5) {
		t.Fatal("500m boundary is outside (strict less-than km)")
	}
}
