package supplier

import "testing"

func TestPortalStatusFromManifestState(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  string
	}{
		{name: "draft", state: "DRAFT", want: "DRAFT"},
		{name: "loading", state: "LOADING", want: "LOADING"},
		{name: "sealed maps loading column", state: "SEALED", want: "LOADING"},
		{name: "dispatched", state: "DISPATCHED", want: "DISPATCHED"},
		{name: "completed maps dispatched column", state: "COMPLETED", want: "DISPATCHED"},
		{name: "empty defaults draft", state: "", want: "DRAFT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := portalStatusFromManifestState(tt.state); got != tt.want {
				t.Fatalf("portalStatusFromManifestState(%q) = %q want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestVolumeVuToInt64(t *testing.T) {
	if got := volumeVuToInt64(142.5); got != 143 {
		t.Fatalf("volumeVuToInt64(142.5) = %d want 143", got)
	}
	if got := volumeVuToInt64(0); got != 0 {
		t.Fatalf("volumeVuToInt64(0) = %d want 0", got)
	}
}
