package partner

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Default 1C-style chart of accounts when a tenant has no PartnerCoaMaps row.
const (
	DefaultCoaAR       = "62.01"
	DefaultCoaRevenue  = "90.01"
	DefaultCoaBankCash = "51.01"
)

var coaAccountRe = regexp.MustCompile(`^[A-Za-z0-9._\-]{1,32}$`)

// CoaMap is a per-tenant chart of accounts for journals exports.
type CoaMap struct {
	TenantType       string
	TenantID         string
	AccountAR        string // receivables (e.g. 62.01)
	AccountRevenue   string // revenue (e.g. 90.01)
	AccountBankCash  string // bank / cash (e.g. 51.01)
	UpdatedAt        time.Time
	UpdatedBy        string
	UsingDefaults    bool // true when no persisted row (or all fields fell back)
}

// CoaRepository persists PartnerCoaMaps.
type CoaRepository interface {
	Get(ctx context.Context, tenantType, tenantID string) (CoaMap, bool, error)
	Upsert(ctx context.Context, m CoaMap) error
}

// DefaultCoa returns platform defaults, optionally overridden by PARTNER_COA_* env.
func DefaultCoa() CoaMap {
	return CoaMap{
		AccountAR:       firstNonEmpty(os.Getenv("PARTNER_COA_AR"), DefaultCoaAR),
		AccountRevenue:  firstNonEmpty(os.Getenv("PARTNER_COA_REVENUE"), DefaultCoaRevenue),
		AccountBankCash: firstNonEmpty(os.Getenv("PARTNER_COA_BANK"), DefaultCoaBankCash),
		UsingDefaults:   true,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// ResolveCoa fills empty account fields from defaults.
func ResolveCoa(stored CoaMap, found bool) CoaMap {
	def := DefaultCoa()
	if !found {
		return def
	}
	out := stored
	out.UsingDefaults = false
	if strings.TrimSpace(out.AccountAR) == "" {
		out.AccountAR = def.AccountAR
	}
	if strings.TrimSpace(out.AccountRevenue) == "" {
		out.AccountRevenue = def.AccountRevenue
	}
	if strings.TrimSpace(out.AccountBankCash) == "" {
		out.AccountBankCash = def.AccountBankCash
	}
	if out.AccountAR == def.AccountAR && out.AccountRevenue == def.AccountRevenue && out.AccountBankCash == def.AccountBankCash {
		out.UsingDefaults = !found || (stored.AccountAR == "" && stored.AccountRevenue == "" && stored.AccountBankCash == "")
	}
	return out
}

// NormalizeCoa trims fields.
func NormalizeCoa(m *CoaMap) {
	if m == nil {
		return
	}
	m.TenantType = strings.ToUpper(strings.TrimSpace(m.TenantType))
	m.TenantID = strings.TrimSpace(m.TenantID)
	m.AccountAR = strings.TrimSpace(m.AccountAR)
	m.AccountRevenue = strings.TrimSpace(m.AccountRevenue)
	m.AccountBankCash = strings.TrimSpace(m.AccountBankCash)
	m.UpdatedBy = strings.TrimSpace(m.UpdatedBy)
}

// ValidateCoaAccounts checks account code shape (empty allowed — will resolve to default).
func ValidateCoaAccounts(m CoaMap) error {
	for _, pair := range []struct {
		name, val string
	}{
		{"account_ar", m.AccountAR},
		{"account_revenue", m.AccountRevenue},
		{"account_bank_cash", m.AccountBankCash},
	} {
		if pair.val == "" {
			continue
		}
		if !coaAccountRe.MatchString(pair.val) {
			return fmt.Errorf("invalid_%s", pair.name)
		}
	}
	return nil
}

func coaDTO(m CoaMap) map[string]any {
	return map[string]any{
		"account_ar":         m.AccountAR,
		"account_revenue":    m.AccountRevenue,
		"account_bank_cash":  m.AccountBankCash,
		"using_defaults":     m.UsingDefaults,
		"updated_at":         formatCoaTime(m.UpdatedAt),
		"updated_by":         m.UpdatedBy,
	}
}

func formatCoaTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
