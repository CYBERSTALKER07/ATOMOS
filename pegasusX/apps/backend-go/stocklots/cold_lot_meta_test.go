package stocklots

import "testing"

func TestClassifyColdLotState(t *testing.T) {
	cases := []struct {
		name          string
		cold          bool
		perishable    bool
		handlingClass string
		locationID    string
		zone          string
		locType       string
		wantReqZone   string
		wantStaging   bool
		wantWarn      string
	}{
		{
			name:          "migrated cold lot in recv-default staging warns",
			cold:          true,
			handlingClass: "CHILLED",
			locationID:    "recv-default",
			zone:          "RECV",
			locType:       "STAGE",
			wantReqZone:   "CHILL",
			wantStaging:   true,
			wantWarn:      ColdLotWarningCode,
		},
		{
			name:          "cold lot placed in ambient bin warns",
			cold:          true,
			handlingClass: "CHILLED",
			locationID:    "A-01-01",
			zone:          "AMBIENT",
			locType:       "PICK",
			wantReqZone:   "CHILL",
			wantStaging:   false,
			wantWarn:      ColdLotWarningCode,
		},
		{
			name:          "frozen sku in chill bin mismatches required freeze zone",
			cold:          true,
			handlingClass: "FROZEN",
			locationID:    "C-01-02",
			zone:          "CHILL",
			locType:       "PICK",
			wantReqZone:   "FREEZE",
			wantStaging:   false,
			wantWarn:      ColdLotWarningCode,
		},
		{
			name:          "chilled sku placed in chill pick bin is clean",
			cold:          true,
			handlingClass: "CHILLED",
			locationID:    "C-01-02",
			zone:          "CHILL",
			locType:       "PICK",
			wantReqZone:   "CHILL",
			wantStaging:   false,
			wantWarn:      "",
		},
		{
			name:          "dry sku in recv-default staging is staging without cold warning",
			cold:          false,
			perishable:    false,
			handlingClass: "GENERAL",
			locationID:    "recv-default",
			zone:          "RECV",
			locType:       "STAGE",
			wantReqZone:   "",
			wantStaging:   true,
			wantWarn:      "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqZone, staging, warn := ClassifyColdLotState(
				tc.cold, tc.perishable, tc.handlingClass,
				tc.locationID, tc.zone, tc.locType,
			)
			if reqZone != tc.wantReqZone {
				t.Fatalf("reqZone = %q, want %q", reqZone, tc.wantReqZone)
			}
			if staging != tc.wantStaging {
				t.Fatalf("staging = %v, want %v", staging, tc.wantStaging)
			}
			if warn != tc.wantWarn {
				t.Fatalf("warn = %q, want %q", warn, tc.wantWarn)
			}
		})
	}
}
