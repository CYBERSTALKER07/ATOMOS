package ar

import (
	"time"
)

// Dunning step machine (PLATFORM_AUDIT §8.6):
// DUE_SOON (T−3) → OVERDUE (T+1) → ESCALATED_1 (T+7) → ESCALATED_2 (T+14) → CREDIT_HOLD (T+21) → COLLECTIONS (T+30)
const (
	StepNone        int64 = 0
	StepDueSoon     int64 = 1
	StepOverdue     int64 = 2
	StepEscalated1  int64 = 3
	StepEscalated2  int64 = 4
	StepCreditHold  int64 = 5
	StepCollections int64 = 6
)

// StepName returns a stable wire label for notifications / UI.
func StepName(step int64) string {
	switch step {
	case StepDueSoon:
		return "DUE_SOON"
	case StepOverdue:
		return "OVERDUE"
	case StepEscalated1:
		return "ESCALATED_1"
	case StepEscalated2:
		return "ESCALATED_2"
	case StepCreditHold:
		return "CREDIT_HOLD"
	case StepCollections:
		return "COLLECTIONS"
	default:
		return "NONE"
	}
}

// DesiredDunningStep computes the step for an open invoice at `now`.
// Grace delays OVERDUE+ until after dueAt+graceDays; DUE_SOON still fires in the T−3 window.
func DesiredDunningStep(dueAt time.Time, graceDays int64, now time.Time) int64 {
	if dueAt.IsZero() {
		return StepNone
	}
	now = now.UTC()
	dueAt = dueAt.UTC()

	if now.Before(dueAt) {
		if dueAt.Sub(now) <= 72*time.Hour {
			return StepDueSoon
		}
		return StepNone
	}

	graceEnd := dueAt.AddDate(0, 0, int(graceDays))
	if !now.After(graceEnd) {
		// Past calendar due but inside grace — keep soft reminder.
		return StepDueSoon
	}

	daysPastDue := int(now.Sub(dueAt).Hours() / 24)
	switch {
	case daysPastDue >= 30:
		return StepCollections
	case daysPastDue >= 21:
		return StepCreditHold
	case daysPastDue >= 14:
		return StepEscalated2
	case daysPastDue >= 7:
		return StepEscalated1
	default:
		return StepOverdue
	}
}

// ShouldBumpDelinquency is true when entering OVERDUE for the first time.
func ShouldBumpDelinquency(prevStep, nextStep int64) bool {
	return nextStep >= StepOverdue && prevStep < StepOverdue
}

// ShouldAutoHold is true when entering CREDIT_HOLD or COLLECTIONS.
func ShouldAutoHold(prevStep, nextStep int64) bool {
	return nextStep >= StepCreditHold && prevStep < StepCreditHold
}
