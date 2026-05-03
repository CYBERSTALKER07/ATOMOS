package supplierlogisticsroutes

import "testing"

func TestManifestPathKind(t *testing.T) {
	tests := []struct {
		name string
		path string
		want manifestPathType
	}{
		{name: "root list", path: "/v1/supplier/manifests/", want: manifestPathList},
		{name: "detail", path: "/v1/supplier/manifests/manifest-1", want: manifestPathDetail},
		{name: "start loading", path: "/v1/supplier/manifests/manifest-1/start-loading", want: manifestPathStartLoading},
		{name: "seal", path: "/v1/supplier/manifests/manifest-1/seal", want: manifestPathSeal},
		{name: "inject order", path: "/v1/supplier/manifests/manifest-1/inject-order", want: manifestPathInjectOrder},
		{name: "unknown suffix falls back to list", path: "/v1/supplier/manifests/manifest-1/metrics", want: manifestPathList},
		{name: "missing id falls back to list", path: "/v1/supplier/manifests//seal", want: manifestPathList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := manifestPathKind(tt.path); got != tt.want {
				t.Fatalf("manifestPathKind(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
