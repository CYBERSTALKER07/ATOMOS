package retailer

import (
	"encoding/json"
	"net/http"
	"strings"
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

	var req map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

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
		ret.SupplierID = s.supplierID
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
	s.writeMobileAuthResponse(w, ret)
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
