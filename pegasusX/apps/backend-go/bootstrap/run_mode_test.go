package bootstrap

import "testing"

func TestNormalizeRunMode(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", RunModeAll},
		{"ALL", RunModeAll},
		{"api", RunModeAPI},
		{"worker", RunModeWorker},
		{"unknown", RunModeAll},
	}
	for _, tc := range tests {
		if got := NormalizeRunMode(tc.in); got != tc.want {
			t.Fatalf("NormalizeRunMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestConfigRunProfiles(t *testing.T) {
	api := &Config{RunMode: RunModeAPI}
	if !api.RunsAPI() || api.RunsWorkers() {
		t.Fatalf("api mode should run API only")
	}
	worker := &Config{RunMode: RunModeWorker}
	if worker.RunsAPI() || !worker.RunsWorkers() {
		t.Fatalf("worker mode should run workers only")
	}
}
