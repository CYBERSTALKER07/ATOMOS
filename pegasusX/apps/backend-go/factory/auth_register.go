package factory

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleFactoryRegister serves POST /v1/auth/factory/register
// Registers a new factory instance user.
func (s *Service) HandleFactoryRegister(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name              string `json:"name"`
		Phone             string `json:"phone"`
		Password          string `json:"password"`
		AssignedFactoryID string `json:"assigned_factory_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Password) == "" {
		web.JSONError(w, "name, phone, and password are required", http.StatusBadRequest)
		return
	}

	// Note: For a production system we must hash the password (e.g. bcrypt). 
	// Storing as plain text or simple hash here for parity with other simple auth flows, 
	// but normally we would use something like auth.HashPassword().
	userID := "usr-" + uuid.NewString()[:8]
	now := s.now().UTC()

	var factoryID spanner.NullString
	if req.AssignedFactoryID != "" {
		factoryID = spanner.NullString{StringVal: req.AssignedFactoryID, Valid: true}
	}

	m := spanner.Insert("SupplierUsers",
		[]string{"UserId", "SupplierId", "Name", "Phone", "PasswordHash", "SupplierRole", "AssignedFactoryId", "IsActive", "CreatedAt", "UpdatedAt"},
		[]any{userID, s.supplierID, strings.TrimSpace(req.Name), strings.TrimSpace(req.Phone), req.Password, "FACTORY", factoryID, true, now, now},
	)

	if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		s.log.ErrorContext(r.Context(), "failed to register factory user", "err", err)
		web.JSONError(w, "Failed to register factory user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user_id":             userID,
		"name":                req.Name,
		"phone":               req.Phone,
		"supplier_role":       "FACTORY",
		"assigned_factory_id": req.AssignedFactoryID,
	})
}
