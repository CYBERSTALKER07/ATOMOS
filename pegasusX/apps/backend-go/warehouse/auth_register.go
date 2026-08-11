package warehouse

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
	"golang.org/x/crypto/bcrypt"
)

// HandleWarehouseRegister serves POST /v1/auth/warehouse/register
// Accepts legacy {name, phone, password} and portal {account, id_token} shapes.
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
		IDToken             string `json:"id_token"`
		Account             *struct {
			LegalName   string `json:"legalName"`
			ContactName string `json:"contactName"`
			Email       string `json:"email"`
			Country     string `json:"country"`
			Phone       string `json:"phone"`
		} `json:"account"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	name := strings.TrimSpace(req.Name)
	phone := strings.TrimSpace(req.Phone)
	password := strings.TrimSpace(req.Password)
	warehouseID := strings.TrimSpace(req.AssignedWarehouseID)

	if req.Account != nil {
		if name == "" {
			name = strings.TrimSpace(req.Account.ContactName)
		}
		if name == "" {
			name = strings.TrimSpace(req.Account.LegalName)
		}
		if phone == "" {
			phone = strings.TrimSpace(req.Account.Phone)
		}
	}

	idToken := strings.TrimSpace(req.IDToken)
	if idToken != "" && s.firebaseVerifier != nil {
		fbClaims, err := s.firebaseVerifier.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			s.log.Warn("warehouse register firebase token verification failed", "err", err)
			web.JSONError(w, "invalid_id_token", http.StatusUnauthorized)
			return
		}
		if fbClaims.PhoneNumber != "" {
			phone = fbClaims.PhoneNumber
		}
	}

	if name == "" || phone == "" {
		web.JSONError(w, "name and phone are required", http.StatusBadRequest)
		return
	}

	passwordHash := password
	if passwordHash == "" {
		seed := uuid.NewString()
		hashed, err := bcrypt.GenerateFromPassword([]byte(seed), bcrypt.DefaultCost)
		if err != nil {
			web.JSONError(w, "Failed to register warehouse user", http.StatusInternalServerError)
			return
		}
		passwordHash = string(hashed)
	} else if strings.HasPrefix(passwordHash, "$2a$") || strings.HasPrefix(passwordHash, "$2b$") || strings.HasPrefix(passwordHash, "$2y$") {
		// already hashed
	} else {
		hashed, err := bcrypt.GenerateFromPassword([]byte(passwordHash), bcrypt.DefaultCost)
		if err != nil {
			web.JSONError(w, "Failed to register warehouse user", http.StatusInternalServerError)
			return
		}
		passwordHash = string(hashed)
	}

	userID := "usr-" + uuid.NewString()[:8]
	now := s.now().UTC()

	var assignedWarehouse spanner.NullString
	if warehouseID != "" {
		assignedWarehouse = spanner.NullString{StringVal: warehouseID, Valid: true}
	}

	m := spanner.Insert("SupplierUsers",
		[]string{"UserId", "SupplierId", "Name", "Phone", "PasswordHash", "SupplierRole", "AssignedWarehouseId", "IsActive", "CreatedAt", "UpdatedAt"},
		[]any{userID, s.resolveSupplierScope(r.Context()), name, phone, passwordHash, "WAREHOUSE", assignedWarehouse, true, now, now},
	)

	if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		s.log.ErrorContext(r.Context(), "failed to register warehouse user", "err", err)
		web.JSONError(w, "Failed to register warehouse user", http.StatusInternalServerError)
		return
	}

	if s.jwtSecret == "" {
		web.JSONError(w, "jwt_not_configured", http.StatusServiceUnavailable)
		return
	}

	isConfigured := s.warehouseIsConfigured(r.Context(), warehouseID)
	jwtClaims := auth.Claims{
		Subject:      userID,
		Role:         auth.RoleWarehouse,
		SupplierID:   s.resolveSupplierScope(r.Context()),
		SupplierRole: auth.RoleWarehouse,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   warehouseID,
		IsConfigured: isConfigured,
		PhoneNumber:  phone,
	}

	token, err := auth.Issue(jwtClaims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		web.JSONError(w, "issue_token_failed", http.StatusInternalServerError)
		return
	}
	refresh, err := auth.Issue(jwtClaims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 7 * 24 * time.Hour})
	if err != nil {
		web.JSONError(w, "issue_refresh_failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id":               userID,
		"name":                  name,
		"phone":                 phone,
		"supplier_role":         "WAREHOUSE",
		"assigned_warehouse_id": warehouseID,
		"token":                 token,
		"refresh_token":         refresh,
		"is_configured":         isConfigured,
	})
}
