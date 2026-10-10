package geolocation

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	cachePrefixAutocomplete = "geo:autocomplete:"
	cachePrefixForward      = "geo:forward:"
	cachePrefixReverse      = "geo:reverse:"
	cachePrefixPlace        = "geo:place:"
)

func normalizeCountryCode(cc string) string {
	c := strings.ToLower(strings.TrimSpace(cc))
	if c == "" {
		return "uz"
	}
	return c
}

func normalizeAutocompleteInput(input string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(input)), " "))
}

func normalizeForwardAddress(address string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(address)), " "))
}

func reverseCacheKey(cc string, lat, lng float64) string {
	return fmt.Sprintf("%s%s:%.5f,%.5f", cachePrefixReverse, normalizeCountryCode(cc), lat, lng)
}

func autocompleteCacheKey(cc, input string) string {
	return fmt.Sprintf("%s%s:%s", cachePrefixAutocomplete, normalizeCountryCode(cc), normalizeAutocompleteInput(input))
}

func forwardCacheKey(cc, address string) string {
	return fmt.Sprintf("%s%s:%s", cachePrefixForward, normalizeCountryCode(cc), normalizeForwardAddress(address))
}

func placeCacheKey(cc, placeID string) string {
	return fmt.Sprintf("%s%s:%s", cachePrefixPlace, normalizeCountryCode(cc), strings.TrimSpace(placeID))
}

func roundCoord(v float64) float64 {
	s := strconv.FormatFloat(v, 'f', 5, 64)
	out, _ := strconv.ParseFloat(s, 64)
	return out
}

