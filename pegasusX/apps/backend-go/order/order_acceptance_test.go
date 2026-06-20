package order

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvaluateOrderAcceptance_EnforceClosed(t *testing.T) {
	sched := OperatingSchedule{
		EnforceOrderAcceptance: true,
		Timezone:               "UTC",
		Schedules: map[string]DayWindow{
			"friday": {Open: "09:00", Close: "17:00"},
		},
	}
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC) // Friday
	open, label, _ := EvaluateOrderAcceptance(sched, now)
	require.True(t, open)
	require.Contains(t, label, "09:00")

	now = time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) // Saturday
	open, _, next := EvaluateOrderAcceptance(sched, now)
	require.False(t, open)
	require.NotNil(t, next)
}

func TestEvaluateOrderAcceptance_NotEnforced(t *testing.T) {
	sched := OperatingSchedule{EnforceOrderAcceptance: false}
	open, _, _ := EvaluateOrderAcceptance(sched, time.Now().UTC())
	require.True(t, open)
}
