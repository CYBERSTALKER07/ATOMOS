package globalproducts

import (
	"strings"
	"unicode"
)

// NormalizeBrandToken lowercases and strips non-alphanumerics for fuzzy keys.
func NormalizeBrandToken(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// BuildNormalizedKey builds brand|name|pack|uom for fuzzy indexing.
func BuildNormalizedKey(brand, name string, packQty int64, uomCode string) string {
	if packQty <= 0 {
		packQty = 1
	}
	uom := strings.ToUpper(strings.TrimSpace(uomCode))
	if uom == "" {
		uom = "EACH"
	}
	return NormalizeBrandToken(brand) + "|" + NormalizeBrandToken(name) + "|" +
		itoa(packQty) + "|" + uom
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var neg bool
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// FuzzyScore returns 0..1 similarity for brand+name+pack+uom.
func FuzzyScore(aBrand, aName string, aPack int64, aUom, bBrand, bName string, bPack int64, bUom string) float64 {
	if aPack <= 0 {
		aPack = 1
	}
	if bPack <= 0 {
		bPack = 1
	}
	if aPack != bPack {
		return 0
	}
	if strings.ToUpper(strings.TrimSpace(aUom)) != strings.ToUpper(strings.TrimSpace(bUom)) &&
		strings.TrimSpace(aUom) != "" && strings.TrimSpace(bUom) != "" {
		return 0
	}
	ab := NormalizeBrandToken(aBrand)
	bb := NormalizeBrandToken(bBrand)
	an := NormalizeBrandToken(aName)
	bn := NormalizeBrandToken(bName)
	if ab == "" && an == "" {
		return 0
	}
	score := 0.0
	if ab != "" && ab == bb {
		score += 0.45
	}
	if an != "" && an == bn {
		score += 0.45
	} else if an != "" && bn != "" && (strings.Contains(an, bn) || strings.Contains(bn, an)) {
		score += 0.25
	}
	if aPack == bPack {
		score += 0.1
	}
	return score
}

const fuzzyAutoLinkThreshold = 0.8
const fuzzyQueueThreshold = 0.55

// DecideFuzzy picks auto-link vs queue vs none from scored candidates.
// candidates must be sorted by score descending.
func DecideFuzzy(candidates []scoredCandidate) (auto *scoredCandidate, queue []scoredCandidate) {
	var strong []scoredCandidate
	for _, c := range candidates {
		if c.Score >= fuzzyQueueThreshold {
			strong = append(strong, c)
		}
	}
	if len(strong) == 0 {
		return nil, nil
	}
	if len(strong) == 1 && strong[0].Score >= fuzzyAutoLinkThreshold {
		return &strong[0], nil
	}
	if len(strong) == 1 && strong[0].Score < fuzzyAutoLinkThreshold {
		return nil, strong
	}
	// Multiple strong candidates → human review.
	return nil, strong
}

type scoredCandidate struct {
	GlobalProductID string
	Score           float64
}
