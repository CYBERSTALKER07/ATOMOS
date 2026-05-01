package proximity

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ─── Reverse Geocoding (Nominatim / OSM) ──────────────────────────────────────
//
// Converts lat/lng coordinates to a structured street address using the free
// OpenStreetMap Nominatim API. No API key required.
//
// Usage policy: ≤1 req/sec with a descriptive User-Agent header.
// https://operations.osmfoundation.org/policies/nominatim/

const nominatimURL = "https://nominatim.openstreetmap.org/reverse"
const nominatimUA = "Pegasus-Leviathan/1.0 (logistics platform)"

// GeocodedAddress holds the parsed result of a reverse geocode call.
type GeocodedAddress struct {
	Street      string `json:"street"`
	HouseNumber string `json:"house_number"`
	City        string `json:"city"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	DisplayName string `json:"display_name"`
}

// ReverseGeocode resolves lat/lng to a structured address via Nominatim.
// Returns a zero-value GeocodedAddress on failure (non-blocking, best-effort).
func ReverseGeocode(lat, lng float64) (GeocodedAddress, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", nominatimURL, nil)
	if err != nil {
		return GeocodedAddress{}, fmt.Errorf("build request: %w", err)
	}

	q := req.URL.Query()
	q.Set("format", "jsonv2")
	q.Set("lat", strconv.FormatFloat(lat, 'f', 6, 64))
	q.Set("lon", strconv.FormatFloat(lng, 'f', 6, 64))
	q.Set("addressdetails", "1")
	q.Set("zoom", "18")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", nominatimUA)
	req.Header.Set("Accept-Language", "en")

	resp, err := client.Do(req)
	if err != nil {
		return GeocodedAddress{}, fmt.Errorf("nominatim request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GeocodedAddress{}, fmt.Errorf("nominatim status %d", resp.StatusCode)
	}

	var result struct {
		DisplayName string `json:"display_name"`
		Address     struct {
			Road        string `json:"road"`
			HouseNumber string `json:"house_number"`
			City        string `json:"city"`
			Town        string `json:"town"`
			Village     string `json:"village"`
			Country     string `json:"country"`
			CountryCode string `json:"country_code"`
		} `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return GeocodedAddress{}, fmt.Errorf("nominatim decode: %w", err)
	}

	city := result.Address.City
	if city == "" {
		city = result.Address.Town
	}
	if city == "" {
		city = result.Address.Village
	}

	return GeocodedAddress{
		Street:      result.Address.Road,
		HouseNumber: result.Address.HouseNumber,
		City:        city,
		Country:     result.Address.Country,
		CountryCode: result.Address.CountryCode,
		DisplayName: result.DisplayName,
	}, nil
}

// HandleReverseGeocode is an HTTP handler for GET /v1/supplier/geocode/reverse?lat=X&lng=Y
func HandleReverseGeocode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		latStr := r.URL.Query().Get("lat")
		lngStr := r.URL.Query().Get("lng")
		if latStr == "" || lngStr == "" {
			http.Error(w, `{"error":"lat and lng query parameters required"}`, http.StatusBadRequest)
			return
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid lat value"}`, http.StatusBadRequest)
			return
		}
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid lng value"}`, http.StatusBadRequest)
			return
		}

		addr, err := ReverseGeocode(lat, lng)
		if err != nil {
			// Best-effort — return empty fields, not an error
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(GeocodedAddress{})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(addr)
	}
}
