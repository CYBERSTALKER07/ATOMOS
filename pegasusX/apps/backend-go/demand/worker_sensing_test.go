package demand

import (
	"testing"
	"time"
)

func TestDayOfWeekFactor(t *testing.T) {
	tests := []struct {
		day  time.Weekday
		want float64
	}{
		{time.Sunday, 0.75},
		{time.Monday, 0.95},
		{time.Tuesday, 1.05},
		{time.Wednesday, 1.10},
		{time.Thursday, 1.05},
		{time.Friday, 1.00},
		{time.Saturday, 0.85},
	}
	for _, tt := range tests {
		t.Run(tt.day.String(), func(t *testing.T) {
			got := dayOfWeekFactor(tt.day)
			if got != tt.want {
				t.Errorf("dayOfWeekFactor(%s) = %v, want %v", tt.day, got, tt.want)
			}
		})
	}
}

func TestPaydayFactor(t *testing.T) {
	tests := []struct {
		day  int
		want float64
	}{
		{1, 1.15},
		{2, 1.15},
		{3, 1.0},
		{14, 1.0},
		{15, 1.15},
		{16, 1.15},
		{17, 1.0},
		{28, 1.0},
	}
	for _, tt := range tests {
		t.Run("day_"+string(rune('0'+tt.day/10))+string(rune('0'+tt.day%10)), func(t *testing.T) {
			got := paydayFactor(tt.day)
			if got != tt.want {
				t.Errorf("paydayFactor(%d) = %v, want %v", tt.day, got, tt.want)
			}
		})
	}
}

func TestDayOfWeekFactor_AllDaysCovered(t *testing.T) {
	for d := time.Sunday; d <= time.Saturday; d++ {
		f := dayOfWeekFactor(d)
		if f < 0.5 || f > 1.5 {
			t.Errorf("dayOfWeekFactor(%s) = %v; out of expected range [0.5, 1.5]", d, f)
		}
	}
}

func TestPaydayFactor_NonPaydays(t *testing.T) {
	for d := 3; d <= 14; d++ {
		if paydayFactor(d) != 1.0 {
			t.Errorf("paydayFactor(%d) should be 1.0 for non-payday", d)
		}
	}
	for d := 17; d <= 31; d++ {
		if paydayFactor(d) != 1.0 {
			t.Errorf("paydayFactor(%d) should be 1.0 for non-payday", d)
		}
	}
}
