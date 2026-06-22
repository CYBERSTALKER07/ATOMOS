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

func normalizeAutocompleteInput(input string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(input)), " "))
}

func normalizeForwardAddress(address string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(address)), " "))
}

func reverseCacheKey(lat, lng float64) string {
	return cachePrefixReverse + fmt.Sprintf("%.5f,%.5f", lat, lng)
}

func autocompleteCacheKey(input string) string {
	return cachePrefixAutocomplete + normalizeAutocompleteInput(input)
}

func forwardCacheKey(address string) string {
	return cachePrefixForward + normalizeForwardAddress(address)
}

func placeCacheKey(placeID string) string {
	return cachePrefixPlace + strings.TrimSpace(placeID)
}

func roundCoord(v float64) float64 {
	s := strconv.FormatFloat(v, 'f', 5, 64)
	out, _ := strconv.ParseFloat(s, 64)
	return out
}
