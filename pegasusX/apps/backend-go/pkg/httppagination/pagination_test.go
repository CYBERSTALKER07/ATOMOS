package httppagination

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLimitOffset(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		defaultLimit int
		maxLimit     int
		wantLimit    int
		wantOffset   int
	}{
		{
			name:         "defaults when empty",
			url:          "/api/v1/items",
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    20,
			wantOffset:   0,
		},
		{
			name:         "custom valid limit and offset",
			url:          "/api/v1/items?limit=50&offset=150",
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    50,
			wantOffset:   150,
		},
		{
			name:         "clamped to maxLimit",
			url:          "/api/v1/items?limit=500&offset=10",
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    100,
			wantOffset:   10,
		},
		{
			name:         "invalid string fallbacks to default",
			url:          "/api/v1/items?limit=not-a-number&offset=bad",
			defaultLimit: 25,
			maxLimit:     50,
			wantLimit:    25,
			wantOffset:   0,
		},
		{
			name:         "negative values fallback to default",
			url:          "/api/v1/items?limit=-10&offset=-5",
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    20,
			wantOffset:   0,
		},
		{
			name:         "whitespace trimmed properly",
			url:          "/api/v1/items?limit=%2030%20&offset=%2040%20",
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    30,
			wantOffset:   40,
		},
		{
			name:         "maxLimit 0 means no clamping",
			url:          "/api/v1/items?limit=250&offset=0",
			defaultLimit: 20,
			maxLimit:     0,
			wantLimit:    250,
			wantOffset:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			gotLimit, gotOffset := ParseLimitOffset(req, tt.defaultLimit, tt.maxLimit)
			if gotLimit != tt.wantLimit {
				t.Errorf("ParseLimitOffset() limit = %v, want %v", gotLimit, tt.wantLimit)
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("ParseLimitOffset() offset = %v, want %v", gotOffset, tt.wantOffset)
			}
		})
	}
}
