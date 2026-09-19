package outbox

import (
	"testing"
)

func TestGetOutboxLagP99Seconds(t *testing.T) {
	// Record 100 samples: 1..100
	for i := 1; i <= 100; i++ {
		RecordPublishLag(float64(i))
	}

	p99 := GetOutboxLagP99Seconds()
	// p99 index for 100 items should be near 99 or 100
	if p99 < 98.0 || p99 > 100.0 {
		t.Fatalf("expected p99 around 99-100, got %f", p99)
	}
}
