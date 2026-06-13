package routing

import (
	"fmt"
	"math"
	"strings"
)

// EncodePolyline encodes coordinates using the Google polyline algorithm.
func EncodePolyline(coords []LatLng) string {
	if len(coords) == 0 {
		return ""
	}
	var b strings.Builder
	var prevLat, prevLng int64
	for _, c := range coords {
		lat := int64(math.Round(c.Lat * 1e5))
		lng := int64(math.Round(c.Lng * 1e5))
		encodeSigned(&b, lat-prevLat)
		encodeSigned(&b, lng-prevLng)
		prevLat = lat
		prevLng = lng
	}
	return b.String()
}

func encodeSigned(b *strings.Builder, value int64) {
	value = value << 1
	if value < 0 {
		value = ^value
	}
	for value >= 0x20 {
		b.WriteByte(byte((0x20 | (value & 0x1f)) + 63))
		value >>= 5
	}
	b.WriteByte(byte(value + 63))
}

// DecodePolyline decodes a Google-encoded polyline into coordinates.
func DecodePolyline(encoded string) ([]LatLng, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, nil
	}
	coords := make([]LatLng, 0, 32)
	var idx int
	var lat, lng int64
	for idx < len(encoded) {
		dLat, next, err := decodePolylineComponent(encoded, idx)
		if err != nil {
			return nil, err
		}
		idx = next
		dLng, next, err := decodePolylineComponent(encoded, idx)
		if err != nil {
			return nil, err
		}
		idx = next
		lat += dLat
		lng += dLng
		coords = append(coords, LatLng{
			Lat: float64(lat) / 1e5,
			Lng: float64(lng) / 1e5,
		})
	}
	return coords, nil
}

func decodePolylineComponent(encoded string, idx int) (int64, int, error) {
	var result int64
	var shift uint
	for {
		if idx >= len(encoded) {
			return 0, idx, fmt.Errorf("truncated polyline at index %d", idx)
		}
		b := int64(encoded[idx]) - 63
		idx++
		result |= (b & 0x1f) << shift
		if b < 0x20 {
			break
		}
		shift += 5
	}
	if result&1 != 0 {
		result = ^(result >> 1)
	} else {
		result >>= 1
	}
	return result, idx, nil
}
