package auth

// PackMapCenter returns the shipped pack camera. Empty/planned packs do not
// invent Tashkent (or any other city).
func PackMapCenter(pack MarketPack) (lat, lng float64, ok bool) {
	if pack.Status != MarketPackShipped {
		return 0, 0, false
	}
	if pack.MapCenterLat == 0 && pack.MapCenterLng == 0 {
		return 0, 0, false
	}
	return pack.MapCenterLat, pack.MapCenterLng, true
}
