package notifications

import (
	"context"
	"testing"
	"time"
)

// mockRepo is a mock Repository for testing ShouldSendNotification
type mockRepo struct {
	Repository // embed to satisfy interface
	pref       *NotificationPreference
	err        error
}

func (m *mockRepo) GetPreference(ctx context.Context, principalID, eventType, channel string) (*NotificationPreference, error) {
	return m.pref, m.err
}

func TestIsQuietHour(t *testing.T) {
	tests := []struct {
		name      string
		nowStr    string
		quietFrom string
		quietTo   string
		expected  bool
	}{
		{
			name:      "Empty quiet hours",
			nowStr:    "12:00:00",
			quietFrom: "",
			quietTo:   "",
			expected:  false,
		},
		{
			name:      "Within same-day quiet hours",
			nowStr:    "10:00:00",
			quietFrom: "09:00",
			quietTo:   "11:00",
			expected:  true,
		},
		{
			name:      "Before same-day quiet hours",
			nowStr:    "08:00:00",
			quietFrom: "09:00",
			quietTo:   "11:00",
			expected:  false,
		},
		{
			name:      "After same-day quiet hours",
			nowStr:    "12:00:00",
			quietFrom: "09:00",
			quietTo:   "11:00",
			expected:  false,
		},
		{
			name:      "Within cross-midnight quiet hours (before midnight)",
			nowStr:    "23:00:00",
			quietFrom: "22:00",
			quietTo:   "08:00",
			expected:  true,
		},
		{
			name:      "Within cross-midnight quiet hours (after midnight)",
			nowStr:    "01:00:00",
			quietFrom: "22:00",
			quietTo:   "08:00",
			expected:  true,
		},
		{
			name:      "Outside cross-midnight quiet hours",
			nowStr:    "12:00:00",
			quietFrom: "22:00",
			quietTo:   "08:00",
			expected:  false,
		},
		{
			name:      "Exact start time same day",
			nowStr:    "09:00:00",
			quietFrom: "09:00",
			quietTo:   "11:00",
			expected:  true,
		},
		{
			name:      "Exact end time same day",
			nowStr:    "11:00:00",
			quietFrom: "09:00",
			quietTo:   "11:00",
			expected:  false,
		},
		{
			name:      "Exact start time cross midnight",
			nowStr:    "22:00:00",
			quietFrom: "22:00",
			quietTo:   "08:00",
			expected:  true,
		},
		{
			name:      "Exact end time cross midnight",
			nowStr:    "08:00:00",
			quietFrom: "22:00",
			quietTo:   "08:00",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now, err := time.Parse("15:04:05", tt.nowStr)
			if err != nil {
				t.Fatalf("Failed to parse nowStr: %v", err)
			}
			result := IsQuietHour(now, tt.quietFrom, tt.quietTo)
			if result != tt.expected {
				t.Errorf("IsQuietHour() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSendNotification(t *testing.T) {
	now, _ := time.Parse("15:04:05", "10:00:00")      // 10 AM
	quietNow, _ := time.Parse("15:04:05", "23:00:00") // 11 PM

	tests := []struct {
		name     string
		pref     *NotificationPreference
		now      time.Time
		expected bool
	}{
		{
			name:     "No preference found (default true)",
			pref:     nil,
			now:      now,
			expected: true,
		},
		{
			name: "Preference disabled",
			pref: &NotificationPreference{
				Enabled: false,
			},
			now:      now,
			expected: false,
		},
		{
			name: "Preference enabled, no quiet hours",
			pref: &NotificationPreference{
				Enabled: true,
			},
			now:      now,
			expected: true,
		},
		{
			name: "Preference enabled, outside quiet hours",
			pref: &NotificationPreference{
				Enabled:   true,
				QuietFrom: "22:00",
				QuietTo:   "08:00",
			},
			now:      now,
			expected: true,
		},
		{
			name: "Preference enabled, inside quiet hours",
			pref: &NotificationPreference{
				Enabled:   true,
				QuietFrom: "22:00",
				QuietTo:   "08:00",
			},
			now:      quietNow,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockRepo{pref: tt.pref}, nil, nil)
			result, err := svc.ShouldSendNotification(context.Background(), "user-1", "ORDER_CREATED", "PUSH", tt.now)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ShouldSendNotification() = %v, want %v", result, tt.expected)
			}
		})
	}
}
