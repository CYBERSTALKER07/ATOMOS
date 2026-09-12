import re

with open("apps/backend-go/fxrates/convert.go", "r") as f:
    content = f.read()

currency_func = """// CurrencyExponent returns the number of minor units for ISO-4217 currencies.
func CurrencyExponent(code string) int {
	switch NormalizeCurrency(code) {
	case "BHD", "IQD", "KWD", "LYD", "OMR", "TND":
		return 3
	case "BIF", "CLP", "DJF", "GNF", "ISK", "JPY", "KMF", "KRW", "PYG", "RWF", "UGX", "VUV", "VND", "XAF", "XOF", "XPF":
		return 0
	case "MGA", "MRU":
		return 1 // non-decimal, 1/5th
	default:
		return 2
	}
}

// applyRate: forward amount*rate/scale; inverse amount*scale/rate. Half away from zero.
func applyRate(amount, rateScaled, scale int64, inverse bool, expFrom, expTo int) (int64, error) {"""

content = content.replace('// applyRate: forward amount*rate/scale; inverse amount*scale/rate. Half away from zero.\nfunc applyRate(amount, rateScaled, scale int64, inverse bool) (int64, error) {', currency_func)

content = content.replace('applyRate(amountMinor, rate.RateScaled, rate.Scale, false)', 'applyRate(amountMinor, rate.RateScaled, rate.Scale, false, CurrencyExponent(from), CurrencyExponent(to))')
content = content.replace('applyRate(amountMinor, inv.RateScaled, inv.Scale, true)', 'applyRate(amountMinor, inv.RateScaled, inv.Scale, true, CurrencyExponent(from), CurrencyExponent(to))')

# Update applyRate body to use math.Pow10
applyRateBody = """
	if rateScaled <= 0 || scale <= 0 {
		return 0, ErrInvalidRate
	}
	a := big.NewInt(amount)
	r := big.NewInt(rateScaled)
	sc := big.NewInt(scale)
	num := new(big.Int)
	den := new(big.Int)
	
	// Exponent scaling: result_minor = (source_minor * rate * 10^expTo) / (scale * 10^expFrom)
	expDiff := expTo - expFrom
	if inverse {
		num.Mul(a, sc)
		den.Set(r)
	} else {
		num.Mul(a, r)
		den.Set(sc)
	}

	if expDiff > 0 {
		num.Mul(num, big.NewInt(int64(math.Pow10(expDiff))))
	} else if expDiff < 0 {
		den.Mul(den, big.NewInt(int64(math.Pow10(-expDiff))))
	}
"""

content = re.sub(r'\tif rateScaled <= 0 \|\| scale <= 0 \{\n\t\treturn 0, ErrInvalidRate\n\t\}\n\ta := big\.NewInt\(amount\)\n\tr := big\.NewInt\(rateScaled\)\n\tsc := big\.NewInt\(scale\)\n\tnum := new\(big\.Int\)\n\tden := new\(big\.Int\)\n\tif inverse \{\n\t\tnum\.Mul\(a, sc\)\n\t\tden\.Set\(r\)\n\t\} else \{\n\t\tnum\.Mul\(a, r\)\n\t\tden\.Set\(sc\)\n\t\}', applyRateBody, content)

with open("apps/backend-go/fxrates/convert.go", "w") as f:
    f.write(content)
