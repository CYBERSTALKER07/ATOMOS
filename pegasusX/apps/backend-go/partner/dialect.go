package partner

import (
	"errors"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// GS-P dialect codes. Sold integrations are not a register gate.
const (
	DialectEDIFACTLite = "edifact_lite_v1"
	DialectPEPPOL      = "peppol"
	DialectOneC        = "onec"
	DialectX12         = "x12"
	DialectAS2         = "as2"
	DialectSAP         = "sap"
)

const (
	DialectShipped  = "shipped"   // in-tree + allowed for listed packs
	DialectPlanned  = "planned"   // catalog only; execute 422
	DialectSoldOnly = "sold_only" // never a default; 422 until a sale wires it
)

// PartnerDialect is the public honesty row for one adapter.
type PartnerDialect struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Packs       []string `json:"packs"`
	ExecuteLive bool     `json:"execute_live"`
	Note        string   `json:"note"`
}

var (
	ErrDialectUnknown    = errors.New("dialect_unknown")
	ErrDialectNotForPack = errors.New("dialect_not_for_pack")
	ErrDialectNotLive    = errors.New("dialect_not_live")
)

// ListPartnerDialects is the GS-P catalog. No secrets. No live PEPPOL/VAN claim.
func ListPartnerDialects() []PartnerDialect {
	return []PartnerDialect{
		{
			Code: DialectEDIFACTLite, Name: "EDIFACT lite", Status: DialectShipped,
			Packs: []string{"UZ"}, ExecuteLive: true,
			Note: "ORDERS/ORDRSP/DESADV/INVOIC. Default UZ EDI pack.",
		},
		{
			Code: DialectOneC, Name: "1C CommerceML-lite", Status: DialectShipped,
			Packs: []string{"UZ", "KZ"}, ExecuteLive: true,
			Note: "CIS only. Not an EU default. Not a certified 1C vendor connector.",
		},
		{
			Code: DialectAS2, Name: "AS2 transport", Status: DialectShipped,
			Packs: []string{"UZ", "EU", "US", "KZ"}, ExecuteLive: true,
			Note: "Per-tenant cert refs. No invented VAN. Receive stays env-gated.",
		},
		{
			Code: DialectPEPPOL, Name: "PEPPOL", Status: DialectPlanned,
			Packs: []string{"EU"}, ExecuteLive: false,
			Note: "EU sale. Execute unimplemented. Fiscal PEPPOL already fail-closes.",
		},
		{
			Code: DialectX12, Name: "X12", Status: DialectSoldOnly,
			Packs: []string{"US"}, ExecuteLive: false,
			Note: "US only if sold.",
		},
		{
			Code: DialectSAP, Name: "SAP IDoc / PI", Status: DialectSoldOnly,
			Packs: []string{"EU", "US"}, ExecuteLive: false,
			Note: "SAP only if sold.",
		},
	}
}

// NormalizeDialect maps profile pack names and aliases.
func NormalizeDialect(code string) string {
	c := strings.ToLower(strings.TrimSpace(code))
	switch c {
	case "", "edifact", "edifact_lite", "edifact-lite", DialectEDIFACTLite:
		return DialectEDIFACTLite
	case "peppol", "peppol_bis":
		return DialectPEPPOL
	case "onec", "1c", "commerceml":
		return DialectOneC
	case "x12", "ansi_x12":
		return DialectX12
	case "as2":
		return DialectAS2
	case "sap", "idoc":
		return DialectSAP
	default:
		return c
	}
}

func dialectByCode(code string) (PartnerDialect, bool) {
	want := NormalizeDialect(code)
	for _, d := range ListPartnerDialects() {
		if d.Code == want {
			return d, true
		}
	}
	return PartnerDialect{}, false
}

// AllowPartnerDialect fail-closes unsold / wrong-pack / not-live execute.
// Register does not call this.
func AllowPartnerDialect(marketCode, dialect string) error {
	d, ok := dialectByCode(dialect)
	if !ok {
		return ErrDialectUnknown
	}
	market := auth.NormalizeMarketCode(marketCode)
	if market == "" {
		market = auth.DefaultMarketCodeFromEnv()
	}
	allowed := false
	for _, p := range d.Packs {
		if auth.NormalizeMarketCode(p) == market {
			allowed = true
			break
		}
	}
	if !allowed {
		return ErrDialectNotForPack
	}
	if !d.ExecuteLive || d.Status != DialectShipped {
		return ErrDialectNotLive
	}
	return nil
}

// DialectsForPack returns catalog rows that list this market (including planned).
func DialectsForPack(marketCode string) []PartnerDialect {
	market := auth.NormalizeMarketCode(marketCode)
	if market == "" {
		market = auth.DefaultMarketCodeFromEnv()
	}
	out := make([]PartnerDialect, 0)
	for _, d := range ListPartnerDialects() {
		for _, p := range d.Packs {
			if auth.NormalizeMarketCode(p) == market {
				out = append(out, d)
				break
			}
		}
	}
	return out
}

func marketCodeForPartner(r *http.Request, tenantType, tenantID string) string {
	if r != nil {
		if claims, ok := auth.FromContext(r.Context()); ok {
			asg := auth.ResolveMarketAssignment(claims)
			if strings.TrimSpace(asg.MarketCode) != "" {
				return asg.MarketCode
			}
		}
	}
	if strings.EqualFold(strings.TrimSpace(tenantType), TenantSupplier) && strings.TrimSpace(tenantID) != "" {
		asg := auth.ResolveMarketAssignment(auth.Claims{SupplierID: tenantID})
		if strings.TrimSpace(asg.MarketCode) != "" {
			return asg.MarketCode
		}
	}
	return auth.DefaultMarketCodeFromEnv()
}

func writeDialectError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrDialectUnknown):
		writePartnerError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrDialectNotForPack), errors.Is(err, ErrDialectNotLive):
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
	}
}

func requireDialect(w http.ResponseWriter, r *http.Request, tenantType, tenantID, dialect string) bool {
	if err := AllowPartnerDialect(marketCodeForPartner(r, tenantType, tenantID), dialect); err != nil {
		writeDialectError(w, err)
		return false
	}
	return true
}

func dialectCodesForPack(market string) []string {
	ds := DialectsForPack(market)
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Code)
	}
	return out
}

// packCurrencyOrEmpty returns the catalog currency for a market. Empty if unknown.
// Does not invent UZS. Planned packs still declare a currency.
func packCurrencyOrEmpty(market string) string {
	if p, ok := auth.ResolveMarketPack(market); ok {
		return strings.TrimSpace(p.CurrencyCode)
	}
	return ""
}

func currencyFromTenantOrEmpty(tenantType, tenantID string) string {
	return packCurrencyOrEmpty(marketCodeForPartner(nil, tenantType, tenantID))
}

// HandleListPartnerDialects GET /v1/platform/partner-dialects — public, no secrets.
func HandleListPartnerDialects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writePartnerError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	pack := auth.NormalizeMarketCode(r.URL.Query().Get("pack"))
	items := ListPartnerDialects()
	if pack != "" {
		items = DialectsForPack(pack)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":                items,
		"register_not_blocked": true,
		"note":                 "GS-P: dialects are per sale. PEPPOL/X12/SAP execute is not live. checkout_reads_this stays false.",
	})
}
