package predictivepush

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		expected float64
	}{
		{
			name:     "New York to London",
			lat1:     40.7128,
			lon1:     -74.0060,
			lat2:     51.5074,
			lon2:     -0.1278,
			expected: 5570.2, // Approx distance in km
		},
		{
			name:     "Same point",
			lat1:     40.7128,
			lon1:     -74.0060,
			lat2:     40.7128,
			lon2:     -74.0060,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := haversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(dist-tt.expected) > 5.0 { // Allow 5km margin of error for different earth radius constants
				t.Errorf("haversineDistance() = %v, want %v", dist, tt.expected)
			}
		})
	}
}
