package spannerx

import (
	"testing"
	"time"
)

func TestForLastWrite_Clamping(t *testing.T) {
	tests := []struct {
		name          string
		lastWriteAt   time.Time
		wantStaleness time.Duration
	}{
		{
			name:          "zero time falls back to default",
			lastWriteAt:   time.Time{},
			wantStaleness: DefaultStaleness,
		},
		{
			name:          "very recent write uses default (below min freshness)",
			lastWriteAt:   time.Now().Add(-500 * time.Millisecond),
			wantStaleness: DefaultStaleness,
		},
		{
			name:          "age within window is preserved",
			lastWriteAt:   time.Now().Add(-10 * time.Second),
			wantStaleness: 10 * time.Second,
		},
		{
			name:          "age above max is clamped to MaxStaleness",
			lastWriteAt:   time.Now().Add(-2 * time.Minute),
			wantStaleness: MaxStaleness,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ForLastWrite(nil, tt.lastWriteAt)
			// allow 500ms tolerance for test execution latency
			diff := r.staleness - tt.wantStaleness
			if diff < 0 {
				diff = -diff
			}
			if diff > 500*time.Millisecond {
				t.Errorf("staleness = %v, want ~%v (diff %v)", r.staleness, tt.wantStaleness, diff)
			}
		})
	}
}
