package settings
package settings

import "testing"

func TestFeeSnapshotAuthoritativeRead(t *testing.T) {
	tests := []struct {
		name  string
		cache map[string]string
		want  bool
	}{
		{
			name:  "missing defaults false",
			cache: map[string]string{},
			want:  false,
		},
		{
			name:  "true literal",
			cache: map[string]string{"fee_snapshot_authoritative_read": "true"},
			want:  true,
		},
		{
			name:  "enabled one",
			cache: map[string]string{"fee_snapshot_authoritative_read": "1"},
			want:  true,
		},
		{
			name:  "yes keyword",
			cache: map[string]string{"fee_snapshot_authoritative_read": "yes"},
			want:  true,
		},
		{
			name:  "disabled false",
			cache: map[string]string{"fee_snapshot_authoritative_read": "false"},
			want:  false,
		},
		{
			name:  "invalid value falls back false",
			cache: map[string]string{"fee_snapshot_authoritative_read": "not-a-bool"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := &PlatformConfig{cache: tt.cache}
			if got := pc.FeeSnapshotAuthoritativeRead(); got != tt.want {
				t.Fatalf("FeeSnapshotAuthoritativeRead() = %v, want %v", got, tt.want)
			}
		})
	}
}