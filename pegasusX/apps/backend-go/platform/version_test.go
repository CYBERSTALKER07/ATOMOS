package platform

import "testing"

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal", "1.2.3", "1.2.3", 0},
		{"less patch", "1.2.2", "1.2.3", -1},
		{"greater minor", "1.3.0", "1.2.9", 1},
		{"v prefix", "v2.0.0", "1.9.9", 1},
		{"empty is zero", "", "0.0.1", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareSemver(tt.a, tt.b); got != tt.want {
				t.Fatalf("CompareSemver(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEvaluate_OutdatedAndDeferred(t *testing.T) {
	policies := NewMemoryPolicyRepository()
	_ = policies.UpsertPolicy(t.Context(), PolicyRow{
		Role: "DRIVER", Platform: "ios", Channel: "production",
		MinimumVersion: "2.0.0", RecommendedVersion: "2.1.0", ForceUpdate: true,
	})
	svc := NewService(policies, NoopSessionChecker{}, nil)
	resp, err := svc.Evaluate(t.Context(), "DRIVER", "ios", "production", "1.0.0", "driver-1")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Outdated {
		t.Fatal("expected outdated")
	}
	if !resp.ForceUpdate {
		t.Fatal("expected force_update when not deferred")
	}
}
