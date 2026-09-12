package payment

import (
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// PSP catalog honesty (GS-K1). Keys are Layer B; this list is not a live charge.
const (
	PSPStatusLive    = "live"
	PSPStatusUnkeyed = "unkeyed"
	PSPStatusPlanned = "planned"
)

// PSPAdapter is a registered payment rail and the markets that may list it.
type PSPAdapter struct {
	Code          string
	Markets       []string
	Status        string
	NationalCards bool
}

// PSPListing is the GET-catalog wire row.
type PSPListing struct {
	Code          string `json:"code"`
	Status        string `json:"status"`
	Selectable    bool   `json:"selectable"`
	NationalCards bool   `json:"national_cards,omitempty"`
}

// Registry is the in-tree PSP set. A market sees a row only when both
// Markets contains the pack code and pack.PSPAdapters contains the code.
func Registry() []PSPAdapter {
	return []PSPAdapter{
		{Code: "CASH", Markets: []string{"UZ", "EU", "US", "CA", "AU", "GB", "KZ", "PK"}, Status: PSPStatusLive},
		{Code: "GLOBAL_PAY", Markets: []string{"UZ"}, Status: PSPStatusLive, NationalCards: true},
		{Code: "PAYME", Markets: []string{"UZ"}, Status: PSPStatusUnkeyed, NationalCards: true},
		{Code: "CLICK", Markets: []string{"UZ"}, Status: PSPStatusUnkeyed, NationalCards: true},
		{Code: "STRIPE", Markets: []string{"EU", "US", "CA", "AU", "GB"}, Status: PSPStatusPlanned, NationalCards: true},
		{Code: "ADYEN", Markets: []string{"EU", "US", "CA", "AU", "GB"}, Status: PSPStatusPlanned, NationalCards: true},
	}
}

// AvailablePSPs is pack.PSPAdapters ∩ registry rows whose Markets include the pack.
func AvailablePSPs(pack auth.MarketPack) []PSPListing {
	code := auth.NormalizeMarketCode(pack.Code)
	out := make([]PSPListing, 0, len(pack.PSPAdapters))
	for _, adapter := range Registry() {
		if !marketAllows(adapter.Markets, code) {
			continue
		}
		if !auth.PackAllowsPSP(pack, adapter.Code) {
			continue
		}
		out = append(out, PSPListing{
			Code:          adapter.Code,
			Status:        adapter.Status,
			Selectable:    adapter.Status != PSPStatusPlanned,
			NationalCards: adapter.NationalCards,
		})
	}
	return out
}

// LivePackGateways is the empty-config default: live catalog rows only.
func LivePackGateways(pack auth.MarketPack) []string {
	out := make([]string, 0, 2)
	for _, listing := range AvailablePSPs(pack) {
		if listing.Status == PSPStatusLive {
			out = append(out, listing.Code)
		}
	}
	return out
}

// LookupPSP returns the registry row for a canonical gateway code.
func LookupPSP(gateway string) (PSPAdapter, bool) {
	want := auth.CanonicalPSP(gateway)
	if want == "" {
		return PSPAdapter{}, false
	}
	for _, adapter := range Registry() {
		if adapter.Code == want {
			return adapter, true
		}
	}
	return PSPAdapter{}, false
}

func marketAllows(markets []string, packCode string) bool {
	packCode = strings.ToUpper(strings.TrimSpace(packCode))
	if packCode == "" {
		return false
	}
	for _, market := range markets {
		if strings.ToUpper(strings.TrimSpace(market)) == packCode {
			return true
		}
	}
	return false
}
