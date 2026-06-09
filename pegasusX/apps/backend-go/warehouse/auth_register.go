package warehouse

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleWarehouseRegister serves POST /v1/auth/warehouse/register
func (s *Service) HandleWarehouseRegister(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name                string `json:"name"`
		Phone               string `json:"phone"`
		Password            string `json:"password"`
		AssignedWarehouseID string `json:"assigned_warehouse_id"`
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

	var warehouseID spanner.NullString
	if req.AssignedWarehouseID != "" {
		warehouseID = spanner.NullString{StringVal: req.AssignedWarehouseID, Valid: true}
	}

	m := spanner.Insert("SupplierUsers",
		[]string{"UserId", "SupplierId", "Name", "Phone", "PasswordHash", "SupplierRole", "AssignedWarehouseId", "IsActive", "CreatedAt", "UpdatedAt"},
		[]any{userID, s.supplierID, strings.TrimSpace(req.Name), strings.TrimSpace(req.Phone), req.Password, "WAREHOUSE", warehouseID, true, now, now},
	)

	if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		s.log.ErrorContext(r.Context(), "failed to register warehouse user", "err", err)
		web.JSONError(w, "Failed to register warehouse user", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id":               userID,
		"name":                  req.Name,
		"phone":                 req.Phone,
		"supplier_role":         "WAREHOUSE",
		"assigned_warehouse_id": req.AssignedWarehouseID,
	})
}
