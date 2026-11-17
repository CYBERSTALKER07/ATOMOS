package retailer

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleRetailerSetup completes retailer onboarding for authenticated clients.
// POST /v1/retailer/setup
func (s *Service) HandleRetailerSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}

	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req map[string]json.RawMessage
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	ret, found, err := s.repo.GetRetailer(r.Context(), retailerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_retailer_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "retailer_not_found"})
		return
	}
	if strings.TrimSpace(ret.SupplierID) == "" {
		ret.SupplierID = s.resolveSupplierScope(r.Context())
	}

	if name := firstNonEmpty(rawString(req, "name"), rawString(req, "store_name"), rawString(req, "owner_name"), rawString(req, "company")); name != "" {
		ret.Name = name
	}
	if phone := firstNonEmpty(rawString(req, "phone"), rawString(req, "phone_number")); phone != "" {
		ret.Phone = phone
	}
	if lat, ok := firstFloat(req, "lat", "latitude"); ok {
		ret.Lat = lat
	}
	if lng, ok := firstFloat(req, "lng", "longitude"); ok {
		ret.Lng = lng
	}
	if country := rawString(req, "country_code"); country != "" {
		ret.CountryCode = country
	}
	deliveryAddr := firstNonEmpty(rawString(req, "delivery_address"), rawString(req, "shippingAddress"), rawString(req, "billingAddress"))
	if deliveryAddr != "" {
		formattedAddr := deliveryAddr
		if city := rawString(req, "city"); city != "" {
			formattedAddr += ", " + city
		}
		if postal := rawString(req, "postalCode"); postal != "" {
			formattedAddr += ", " + postal
		}
		if strings.TrimSpace(ret.DeliveryAddress) == "" || rawString(req, "delivery_address") != "" || rawString(req, "shippingAddress") != "" {
			ret.DeliveryAddress = formattedAddr
		}
	}
	if placeID := rawString(req, "place_id"); placeID != "" {
		ret.PlaceID = placeID
	}
	if ret.Lat == 0 && ret.Lng == 0 && deliveryAddr != "" {
		resolvedCountry, countryErr := resolveRetailerCountry(r.Context(), ret.CountryCode)
		if countryErr != nil {
			if writeMarketLaw(w, countryErr) {
				return
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": countryErr.Error()})
			return
		}
		ret.CountryCode = resolvedCountry
		if pack, ok := auth.ResolveShippedMarketPack(resolvedCountry); ok {
			if centerLat, centerLng, ok := auth.PackMapCenter(pack); ok {
				ret.Lat = centerLat
				ret.Lng = centerLng
			}
		}
	}
	country, cell, stampErr := stampRetailerOptionalCoords(r.Context(), ret.Lat, ret.Lng, ret.CountryCode)
	if stampErr != nil {
		if writeMarketLaw(w, stampErr) {
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": stampErr.Error()})
		return
	}
	ret.CountryCode = country
	if cell != "" {
		ret.H3Cell = cell
	}

	var taxReq taxIdentityRequest
	_ = json.Unmarshal(body, &taxReq)
	if taxStatus, conflict, taxErrCode := s.resolveAndApplyTaxIdentity(r.Context(), &ret, taxReq, rawString(req, "name")); taxStatus != 0 {
		if conflict != nil {
			writeJSON(w, taxStatus, conflict)
		} else {
			writeJSON(w, taxStatus, map[string]string{"error": taxErrCode, "code": taxErrCode})
		}
		return
	}

	// Desktop setup wizard fields — best-effort label when name is still empty.
	if strings.TrimSpace(ret.Name) == "" {
		if label := firstNonEmpty(rawString(req, "shippingAddress"), rawString(req, "billingAddress")); label != "" {
			if city := rawString(req, "city"); city != "" {
				label = label + ", " + city
			}
			ret.Name = label
		}
	}

	ret.UpdatedAt = s.now()
	if err := s.repo.UpdateRetailer(r.Context(), ret, nil); err != nil {
		s.log.ErrorContext(r.Context(), "retailer setup update failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
		return
	}

	s.cache.Invalidate(r.Context(), retailerByPhoneKey(ret.Phone))
	status, respBytes := s.marshalMobileAuthResponse(ret)
	if status != http.StatusOK {
		writeJSONBytes(w, status, respBytes)
		return
	}
	idemCommitted = true
	writeJSONBytes(w, status, respBytes)
	s.saveIdempotency(r.Context(), r, body, status, respBytes)
}

func rawString(req map[string]json.RawMessage, key string) string {
	raw, ok := req[key]
	if !ok || len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstFloat(req map[string]json.RawMessage, keys ...string) (float64, bool) {
	for _, key := range keys {
		raw, ok := req[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var value float64
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		return value, true
	}
	return 0, false
}
