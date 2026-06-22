package geolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

const (
	nominatimURL = "https://nominatim.openstreetmap.org"
	nominatimUA  = "PegasusX/1.0 (logistics platform)"

	ttlAutocomplete = 24 * time.Hour
	ttlForward      = 7 * 24 * time.Hour
	ttlReverse      = 7 * 24 * time.Hour
	ttlPlace        = 7 * 24 * time.Hour
)

// Service resolves addresses via Google Maps when configured, otherwise Nominatim.
type Service struct {
	googleAPIKey string
	httpClient   *http.Client
	cache        *cache.Cache
}

// NewService constructs a geolocation resolver. Pass cache for Redis-backed geocode caching.
func NewService(googleAPIKey string, c *cache.Cache) *Service {
	key := strings.TrimSpace(googleAPIKey)
	return &Service{
		googleAPIKey: key,
		httpClient:   &http.Client{Timeout: 8 * time.Second},
		cache:        c,
	}
}

// ResolvedLocation is the canonical address + coordinate bundle stored in Spanner.
type ResolvedLocation struct {
	Address   string  `json:"address"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	PlaceID   string  `json:"place_id,omitempty"`
	Formatted string  `json:"formatted_address,omitempty"`
}

// AutocompletePrediction is one Places autocomplete suggestion.
type AutocompletePrediction struct {
	PlaceID     string `json:"place_id"`
	Description string `json:"description"`
}

// ReverseGeocode resolves coordinates to a display address.
func (s *Service) ReverseGeocode(ctx context.Context, lat, lng float64) (ResolvedLocation, error) {
	lat = roundCoord(lat)
	lng = roundCoord(lng)
	key := reverseCacheKey(lat, lng)
	var out ResolvedLocation
	if err := s.loadJSON(ctx, key, ttlReverse, func(ctx context.Context) (any, error) {
		if s != nil && s.googleAPIKey != "" {
			return s.reverseGoogle(ctx, lat, lng)
		}
		return s.reverseNominatim(ctx, lat, lng)
	}, &out); err != nil {
		return ResolvedLocation{}, err
	}
	return out, nil
}

// ForwardGeocode resolves a free-text address to coordinates.
func (s *Service) ForwardGeocode(ctx context.Context, address string) (ResolvedLocation, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return ResolvedLocation{}, fmt.Errorf("address_required")
	}
	key := forwardCacheKey(address)
	var out ResolvedLocation
	if err := s.loadJSON(ctx, key, ttlForward, func(ctx context.Context) (any, error) {
		if s != nil && s.googleAPIKey != "" {
			return s.forwardGoogle(ctx, address)
		}
		return s.forwardNominatim(ctx, address)
	}, &out); err != nil {
		return ResolvedLocation{}, err
	}
	return out, nil
}

// Autocomplete returns address suggestions for partial user input.
func (s *Service) Autocomplete(ctx context.Context, input string) ([]AutocompletePrediction, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	key := autocompleteCacheKey(input)
	var out []AutocompletePrediction
	if err := s.loadJSON(ctx, key, ttlAutocomplete, func(ctx context.Context) (any, error) {
		return s.autocompleteUncached(ctx, input)
	}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResolvePlaceID loads coordinates for a Google place_id.
func (s *Service) ResolvePlaceID(ctx context.Context, placeID string) (ResolvedLocation, error) {
	placeID = strings.TrimSpace(placeID)
	if placeID == "" {
		return ResolvedLocation{}, fmt.Errorf("place_id_required")
	}
	if s == nil || s.googleAPIKey == "" {
		return ResolvedLocation{}, fmt.Errorf("google_maps_not_configured")
	}
	key := placeCacheKey(placeID)
	var out ResolvedLocation
	if err := s.loadJSON(ctx, key, ttlPlace, func(ctx context.Context) (any, error) {
		return s.resolvePlaceIDUncached(ctx, placeID)
	}, &out); err != nil {
		return ResolvedLocation{}, err
	}
	return out, nil
}

func (s *Service) loadJSON(ctx context.Context, key string, ttl time.Duration, loader func(context.Context) (any, error), dest any) error {
	if s == nil || s.cache == nil {
		v, err := loader(ctx)
		if err != nil {
			return err
		}
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, dest)
	}
	raw, err := s.cache.GetOrLoad(ctx, key, ttl, func(ctx context.Context) ([]byte, error) {
		v, err := loader(ctx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(v)
	})
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func (s *Service) autocompleteUncached(ctx context.Context, input string) ([]AutocompletePrediction, error) {
	if s == nil || s.googleAPIKey == "" {
		loc, err := s.ForwardGeocode(ctx, input)
		if err != nil {
			return nil, nil
		}
		return []AutocompletePrediction{{
			PlaceID:     loc.PlaceID,
			Description: loc.Address,
		}}, nil
	}
	endpoint := "https://maps.googleapis.com/maps/api/place/autocomplete/json"
	q := url.Values{}
	q.Set("input", input)
	q.Set("key", s.googleAPIKey)
	q.Set("types", "address")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Status      string `json:"status"`
		Predictions []struct {
			PlaceID     string `json:"place_id"`
			Description string `json:"description"`
		} `json:"predictions"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Status != "OK" && parsed.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("places_autocomplete_%s", strings.ToLower(parsed.Status))
	}
	out := make([]AutocompletePrediction, 0, len(parsed.Predictions))
	for _, p := range parsed.Predictions {
		out = append(out, AutocompletePrediction{
			PlaceID:     p.PlaceID,
			Description: p.Description,
		})
	}
	return out, nil
}

func (s *Service) resolvePlaceIDUncached(ctx context.Context, placeID string) (ResolvedLocation, error) {
	endpoint := "https://maps.googleapis.com/maps/api/place/details/json"
	q := url.Values{}
	q.Set("place_id", placeID)
	q.Set("fields", "formatted_address,geometry")
	q.Set("key", s.googleAPIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return ResolvedLocation{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ResolvedLocation{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Status string `json:"status"`
		Result struct {
			FormattedAddress string `json:"formatted_address"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ResolvedLocation{}, err
	}
	if parsed.Status != "OK" {
		return ResolvedLocation{}, fmt.Errorf("place_details_%s", strings.ToLower(parsed.Status))
	}
	addr := strings.TrimSpace(parsed.Result.FormattedAddress)
	return ResolvedLocation{
		Address:   addr,
		Formatted: addr,
		Lat:       parsed.Result.Geometry.Location.Lat,
		Lng:       parsed.Result.Geometry.Location.Lng,
		PlaceID:   placeID,
	}, nil
}

func (s *Service) reverseGoogle(ctx context.Context, lat, lng float64) (ResolvedLocation, error) {
	endpoint := "https://maps.googleapis.com/maps/api/geocode/json"
	q := url.Values{}
	q.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))
	q.Set("key", s.googleAPIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return ResolvedLocation{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ResolvedLocation{}, err
	}
	defer resp.Body.Close()
	var data struct {
		Status  string `json:"status"`
		Results []struct {
			FormattedAddress string `json:"formatted_address"`
			PlaceID          string `json:"place_id"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ResolvedLocation{}, err
	}
	if data.Status != "OK" || len(data.Results) == 0 {
		return ResolvedLocation{Lat: lat, Lng: lng, Address: fmt.Sprintf("%f, %f", lat, lng)}, nil
	}
	best := data.Results[0]
	return ResolvedLocation{
		Address:   best.FormattedAddress,
		Formatted: best.FormattedAddress,
		Lat:       lat,
		Lng:       lng,
		PlaceID:   best.PlaceID,
	}, nil
}

func (s *Service) forwardGoogle(ctx context.Context, address string) (ResolvedLocation, error) {
	endpoint := "https://maps.googleapis.com/maps/api/geocode/json"
	q := url.Values{}
	q.Set("address", address)
	q.Set("key", s.googleAPIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return ResolvedLocation{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ResolvedLocation{}, err
	}
	defer resp.Body.Close()
	var data struct {
		Status  string `json:"status"`
		Results []struct {
			FormattedAddress string `json:"formatted_address"`
			PlaceID          string `json:"place_id"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ResolvedLocation{}, err
	}
	if data.Status != "OK" || len(data.Results) == 0 {
		return ResolvedLocation{}, fmt.Errorf("geocode_not_found")
	}
	best := data.Results[0]
	return ResolvedLocation{
		Address:   best.FormattedAddress,
		Formatted: best.FormattedAddress,
		Lat:       best.Geometry.Location.Lat,
		Lng:       best.Geometry.Location.Lng,
		PlaceID:   best.PlaceID,
	}, nil
}

func (s *Service) reverseNominatim(ctx context.Context, lat, lng float64) (ResolvedLocation, error) {
	q := url.Values{}
	q.Set("format", "jsonv2")
	q.Set("lat", strconv.FormatFloat(lat, 'f', 6, 64))
	q.Set("lon", strconv.FormatFloat(lng, 'f', 6, 64))
	q.Set("addressdetails", "1")
	q.Set("zoom", "18")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nominatimURL+"/reverse?"+q.Encode(), nil)
	if err != nil {
		return ResolvedLocation{}, err
	}
	req.Header.Set("User-Agent", nominatimUA)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ResolvedLocation{}, err
	}
	defer resp.Body.Close()
	var result struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ResolvedLocation{}, err
	}
	addr := strings.TrimSpace(result.DisplayName)
	if addr == "" {
		addr = fmt.Sprintf("%f, %f", lat, lng)
	}
	return ResolvedLocation{Address: addr, Formatted: addr, Lat: lat, Lng: lng}, nil
}

func (s *Service) forwardNominatim(ctx context.Context, address string) (ResolvedLocation, error) {
	q := url.Values{}
	q.Set("q", address)
	q.Set("format", "json")
	q.Set("limit", "1")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nominatimURL+"/search?"+q.Encode(), nil)
	if err != nil {
		return ResolvedLocation{}, err
	}
	req.Header.Set("User-Agent", nominatimUA)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ResolvedLocation{}, err
	}
	defer resp.Body.Close()
	var rows []struct {
		DisplayName string `json:"display_name"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return ResolvedLocation{}, err
	}
	if len(rows) == 0 {
		return ResolvedLocation{}, fmt.Errorf("geocode_not_found")
	}
	lat, _ := strconv.ParseFloat(rows[0].Lat, 64)
	lng, _ := strconv.ParseFloat(rows[0].Lon, 64)
	return ResolvedLocation{
		Address:   rows[0].DisplayName,
		Formatted: rows[0].DisplayName,
		Lat:       lat,
		Lng:       lng,
	}, nil
}
