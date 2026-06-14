package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// BusinessSetupRequest is the wire shape for POST /v1/supplier/business/setup.
type BusinessSetupRequest struct {
	TaxID               string `json:"taxId"`
	RegistrationNumber  string `json:"registrationNumber"`
	HeadquartersAddress string `json:"headquartersAddress"`
	City                string `json:"city"`
	PostalCode          string `json:"postalCode"`
}

// BusinessSetupResponse confirms business onboarding completion.
type BusinessSetupResponse struct {
	SupplierID   string `json:"supplier_id"`
	IsRegistered bool   `json:"is_registered"`
	NextStep     string `json:"next_step"`
}

// registrationComplete reports whether the full pegasus-style register payload
// already captured business + location (skips /setup/business).
func (r RegisterRequest) registrationComplete() bool {
	return strings.TrimSpace(r.Business.TaxID) != "" &&
		strings.TrimSpace(r.Location.Warehouse.Address) != ""
}

func composeHeadquartersAddress(address, city, postal string) string {
	parts := []string{strings.TrimSpace(address)}
	if c := strings.TrimSpace(city); c != "" {
		parts = append(parts, c)
	}
	if p := strings.TrimSpace(postal); p != "" {
		parts = append(parts, p)
	}
	return strings.Join(parts, ", ")
}

// CompleteBusinessSetup persists tax and headquarters fields for the authenticated
// supplier and marks the row registered.
func (s *Service) CompleteBusinessSetup(ctx context.Context, supplierID string, req BusinessSetupRequest) (BusinessSetupResponse, error) {
	if strings.TrimSpace(req.TaxID) == "" {
		return BusinessSetupResponse{}, errors.New("taxId required")
	}
	if strings.TrimSpace(req.HeadquartersAddress) == "" {
		return BusinessSetupResponse{}, errors.New("headquartersAddress required")
	}
	if strings.TrimSpace(req.City) == "" {
		return BusinessSetupResponse{}, errors.New("city required")
	}

	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return BusinessSetupResponse{}, errors.New("supplier scope required")
	}

	current, found, err := s.repo.GetProfile(ctx, supplierID)
	if err != nil {
		return BusinessSetupResponse{}, fmt.Errorf("load supplier profile: %w", err)
	}
	if !found {
		current = Profile{SupplierID: supplierID, Country: s.country, Currency: s.currency}
	}

	now := s.now().UTC()
	current.TaxID = strings.TrimSpace(req.TaxID)
	current.CompanyRegNumber = strings.TrimSpace(req.RegistrationNumber)
	hq := composeHeadquartersAddress(req.HeadquartersAddress, req.City, req.PostalCode)
	if current.BillingAddress == "" {
		current.BillingAddress = hq
	}
	if current.WarehouseAddress == "" {
		current.WarehouseAddress = hq
	}
	current.IsRegistered = true
	if current.RegisteredAt.IsZero() {
		current.RegisteredAt = now
	}
	current.UpdatedAt = now

	if err := s.repo.UpdateProfile(ctx, current, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateSupplier, supplierID, events.TopicMain, events.SupplierEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventSupplierProfileUpdated, Timestamp: now.Format(time.RFC3339Nano)},
			SupplierID:   supplierID,
			LegalName:    current.LegalName,
			ContactName:  current.ContactName,
			Email:        current.Email,
			Phone:        current.Phone,
			Country:      current.Country,
			Categories:   current.Categories,
			IsRegistered: current.IsRegistered,
			IsConfigured: current.IsConfigured,
			Action:       "BUSINESS_SETUP",
		})
	}); err != nil {
		return BusinessSetupResponse{}, fmt.Errorf("persist business setup: %w", err)
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, supplierCacheKey(supplierID))
	}

	return BusinessSetupResponse{
		SupplierID:   supplierID,
		IsRegistered: true,
		NextStep:     "/setup/billing",
	}, nil
}

// HandleSupplierBusinessSetup serves POST /v1/supplier/business/setup.
func (s *Service) HandleSupplierBusinessSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req BusinessSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	supplierID := s.scopedSupplierID(r)
	resp, err := s.CompleteBusinessSetup(r.Context(), supplierID, req)
	if err != nil {
		switch err.Error() {
		case "taxId required", "headquartersAddress required", "city required", "supplier scope required":
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			s.log.WarnContext(r.Context(), "supplier business setup failed", "err", err, "supplier_id", supplierID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "business_setup_failed"})
		}
		return
	}

	if _, err := s.writeSessionCookie(w, resp.SupplierID, resp.IsRegistered, false); err != nil {
		s.log.WarnContext(r.Context(), "business setup session reissue failed", "err", err)
	}
	writeJSON(w, http.StatusOK, resp)
}
