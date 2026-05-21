// Package supplier owns the single-tenant supplier-portal handlers.
//
// pegasusX runs as a single-supplier tenant: the Suppliers row is seeded at
// bootstrap. Register and ConfigureBilling MUTATE that row — they do not
// create new suppliers. The handlers also issue the supplier-portal session
// JWT cookie consumed by the Next.js middleware (`is_configured` claim).
package supplier

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"golang.org/x/crypto/bcrypt"
)

// Repository is the persistence seam for the seeded supplier aggregate.
// Production binds this to a Spanner-backed implementation that runs every
// mutation inside a ReadWriteTransaction and uses the TxnBuffer to atomically
// write the OutboxEvents row.
type Repository interface {
	GetProfile(ctx context.Context, supplierID string) (Profile, bool, error)
	UpdateProfile(ctx context.Context, p Profile, emit func(outbox.TxnBuffer) error) error
	GetAuthByPhone(ctx context.Context, phone string) (SupplierAuthRecord, bool, error)
	GetTopology(ctx context.Context, supplierID string) (SupplierTopology, error)
	ReplaceTopology(ctx context.Context, supplierID string, topology SupplierTopology, emit func(outbox.TxnBuffer) error) error
}

// SupplierAuthRecord is the credential snapshot used by login flow.
type SupplierAuthRecord struct {
	UserID       string
	SupplierID   string
	Phone        string
	PasswordHash string
	IsConfigured bool
}

// Profile is the persisted supplier-onboarding aggregate. Mirrors the
// 4-step wizard payload plus the post-registration billing fields.
type Profile struct {
	SupplierID        string
	LegalName         string
	ContactName       string
	Email             string
	Phone             string
	AuthUserID        string
	AuthPasswordHash  string
	Country           string
	Currency          string
	WarehouseName     string
	WarehouseAddress  string
	WarehouseLat      float64
	WarehouseLng      float64
	BillingSameAsWh   bool
	BillingAddress    string
	BillingLat        float64
	BillingLng        float64
	TaxID             string
	CompanyRegNumber  string
	FleetVehicleCount int
	FleetMaxVU        int
	FactoryCount      int
	Categories        []string
	IsRegistered      bool
	IsConfigured      bool
	BankName          string
	AccountHolder     string
	AccountNumber     string
	SwiftBic          string
	IBAN              string
	SelectedGateways  []string
	RegisteredAt      time.Time
	ConfiguredAt      time.Time
	UpdatedAt         time.Time
}

// WarehouseNode is one supplier-owned warehouse topology node.
type WarehouseNode struct {
	WarehouseID      string
	Name             string
	Lat              float64
	Lng              float64
	CoverageRadiusKm float64
	IsActive         bool
	IsOnShift        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// FactoryNode is one supplier-owned factory topology node.
type FactoryNode struct {
	FactoryID string
	Name      string
	Lat       float64
	Lng       float64
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SupplierTopology is the warehouse/factory graph for one supplier.
type SupplierTopology struct {
	Warehouses []WarehouseNode
	Factories  []FactoryNode
}

// Service wires repository, cache, idempotency, and JWT issuance.
type Service struct {
	repo         Repository
	cache        *cache.Cache
	idem         idempotency.Store
	supplierID   string
	country      string
	currency     string
	jwtSecret    string
	jwtIssuer    string
	jwtTTL       time.Duration
	cookieSecure bool
	log          *slog.Logger
	now          func() time.Time

	mu             sync.RWMutex
	inventory      map[string]InventoryItem
	inventoryAudit []InventoryAuditEntry
	orders         map[string]SupplierOrder
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo         Repository
	Cache        *cache.Cache
	Idem         idempotency.Store
	SupplierID   string
	Country      string
	Currency     string
	JWTSecret    string
	JWTIssuer    string
	JWTTTL       time.Duration
	CookieSecure bool
	Log          *slog.Logger
	Now          func() time.Time
}

// NewService returns a configured Service.
func NewService(c ServiceConfig) *Service {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.JWTTTL <= 0 {
		c.JWTTTL = 24 * time.Hour
	}
	return &Service{
		repo:         c.Repo,
		cache:        c.Cache,
		idem:         c.Idem,
		supplierID:   c.SupplierID,
		country:      c.Country,
		currency:     c.Currency,
		jwtSecret:    c.JWTSecret,
		jwtIssuer:    c.JWTIssuer,
		jwtTTL:       c.JWTTTL,
		cookieSecure: c.CookieSecure,
		log:          c.Log,
		now:          c.Now,
		inventory:    make(map[string]InventoryItem),
		orders:       make(map[string]SupplierOrder),
	}
}

// ── Register ───────────────────────────────────────────────────────────────

// AccountStep mirrors wizard step 1.
type AccountStep struct {
	LegalName   string `json:"legalName"`
	ContactName string `json:"contactName"`
	Email       string `json:"email"`
	Country     string `json:"country"`
	Phone       string `json:"phone"`
	Password    string `json:"password"`
}

// AddressStep is a reused shape for warehouse/billing addresses.
type AddressStep struct {
	Name    string  `json:"name,omitempty"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

// LocationStep mirrors wizard step 2.
type LocationStep struct {
	Warehouse       AddressStep `json:"warehouse"`
	BillingSameAsWh bool        `json:"sameAsWarehouse"`
	Billing         AddressStep `json:"billing"`
}

// BusinessStep mirrors wizard step 3.
type BusinessStep struct {
	TaxID             string `json:"taxId"`
	CompanyRegNumber  string `json:"companyRegNumber"`
	FleetVehicleCount int    `json:"fleetVehicleCount"`
	FleetMaxVU        int    `json:"fleetMaxVU"`
	FactoryCount      int    `json:"factoryCount"`
}

// RegisterRequest is the supplier-portal wizard payload (4 steps + phone).
type RegisterRequest struct {
	Account    AccountStep  `json:"account"`
	Location   LocationStep `json:"location"`
	Business   BusinessStep `json:"business"`
	Categories []string     `json:"categories"`
	Phone      string       `json:"phone"`
}

// RegisterResponse mirrors what the wizard expects.
type RegisterResponse struct {
	SupplierID   string `json:"supplier_id"`
	LegalName    string `json:"legal_name"`
	IsConfigured bool   `json:"is_configured"`
	NextStep     string `json:"next_step"`
}

// LoginRequest is the wire shape for POST /v1/auth/supplier/login.
type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// LoginResponse is the login confirmation payload.
type LoginResponse struct {
	SupplierID   string `json:"supplier_id"`
	IsConfigured bool   `json:"is_configured"`
	NextStep     string `json:"next_step"`
}

// Validate enforces wizard invariants.
func (r RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Account.LegalName) == "" {
		return errors.New("account.legalName required")
	}
	if strings.TrimSpace(r.Account.ContactName) == "" {
		return errors.New("account.contactName required")
	}
	if strings.TrimSpace(r.Account.Email) == "" || !strings.Contains(r.Account.Email, "@") {
		return errors.New("account.email invalid")
	}
	if strings.TrimSpace(r.Phone) == "" {
		return errors.New("phone required")
	}
	if len(r.Account.Password) < 8 {
		return errors.New("account.password must be at least 8 chars")
	}
	if strings.TrimSpace(r.Account.Country) == "" {
		return errors.New("account.country required")
	}
	if strings.TrimSpace(r.Location.Warehouse.Address) == "" {
		return errors.New("location.warehouse.address required")
	}
	if strings.TrimSpace(r.Location.Warehouse.Name) == "" {
		return errors.New("location.warehouse.name required")
	}
	if r.Location.Warehouse.Lat == 0 && r.Location.Warehouse.Lng == 0 {
		return errors.New("location.warehouse lat/lng required")
	}
	if !r.Location.BillingSameAsWh && strings.TrimSpace(r.Location.Billing.Address) == "" {
		return errors.New("location.billing.address required when sameAsWarehouse=false")
	}
	if strings.TrimSpace(r.Business.TaxID) == "" {
		return errors.New("business.taxId required")
	}
	if r.Business.FleetVehicleCount < 0 || r.Business.FleetMaxVU < 0 || r.Business.FactoryCount < 0 {
		return errors.New("business: counts must be non-negative")
	}
	if len(r.Categories) == 0 {
		return errors.New("categories required")
	}
	return nil
}

// Validate enforces login payload invariants.
func (r LoginRequest) Validate() error {
	if strings.TrimSpace(r.Phone) == "" {
		return errors.New("phone required")
	}
	if strings.TrimSpace(r.Password) == "" {
		return errors.New("password required")
	}
	return nil
}

// Register persists the wizard payload onto the seeded supplier row, emits a
// SUPPLIER_UPDATED outbox event atomically, invalidates the supplier cache
// post-commit, and returns the response shape the wizard expects.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return RegisterResponse{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Account.Password), bcrypt.DefaultCost)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("hash supplier password: %w", err)
	}

	current, found, err := s.repo.GetProfile(ctx, s.supplierID)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("load supplier: %w", err)
	}
	if !found {
		current = Profile{SupplierID: s.supplierID, Country: s.country, Currency: s.currency}
	}

	now := s.now()
	current.LegalName = req.Account.LegalName
	current.ContactName = req.Account.ContactName
	current.Email = req.Account.Email
	current.Phone = req.Phone
	current.AuthUserID = rootSupplierUserID(s.supplierID)
	current.AuthPasswordHash = string(passwordHash)
	if strings.TrimSpace(req.Account.Country) != "" {
		current.Country = strings.ToUpper(req.Account.Country)
	}
	current.WarehouseAddress = req.Location.Warehouse.Address
	current.WarehouseName = req.Location.Warehouse.Name
	current.WarehouseLat = req.Location.Warehouse.Lat
	current.WarehouseLng = req.Location.Warehouse.Lng
	current.BillingSameAsWh = req.Location.BillingSameAsWh
	if req.Location.BillingSameAsWh {
		current.BillingAddress = req.Location.Warehouse.Address
		current.BillingLat = req.Location.Warehouse.Lat
		current.BillingLng = req.Location.Warehouse.Lng
	} else {
		current.BillingAddress = req.Location.Billing.Address
		current.BillingLat = req.Location.Billing.Lat
		current.BillingLng = req.Location.Billing.Lng
	}
	current.TaxID = req.Business.TaxID
	current.CompanyRegNumber = req.Business.CompanyRegNumber
	current.FleetVehicleCount = req.Business.FleetVehicleCount
	current.FleetMaxVU = req.Business.FleetMaxVU
	current.FactoryCount = req.Business.FactoryCount
	current.Categories = append([]string(nil), req.Categories...)
	current.IsRegistered = true
	current.RegisteredAt = now
	current.UpdatedAt = now

	err = s.repo.UpdateProfile(ctx, current, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateSupplier, s.supplierID, events.TopicMain, supplierUpdatedEvent{
			Type:         events.EventSupplierUpdated,
			SupplierID:   s.supplierID,
			LegalName:    current.LegalName,
			ContactName:  current.ContactName,
			Email:        current.Email,
			Phone:        current.Phone,
			Country:      current.Country,
			Categories:   current.Categories,
			IsRegistered: current.IsRegistered,
			IsConfigured: current.IsConfigured,
			Action:       "REGISTERED",
			Timestamp:    now.Format(time.RFC3339Nano),
		})
	})
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("persist supplier: %w", err)
	}

	s.cache.Invalidate(ctx, supplierCacheKey(s.supplierID))
	s.log.Info("supplier registered",
		"supplier_id", s.supplierID,
		"legal_name", current.LegalName,
		"country", current.Country,
	)
	return RegisterResponse{
		SupplierID:   s.supplierID,
		LegalName:    current.LegalName,
		IsConfigured: current.IsConfigured,
		NextStep:     "/setup/billing",
	}, nil
}

// Login verifies supplier credentials and returns the configured-state snapshot.
func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return LoginResponse{}, err
	}
	rec, found, err := s.repo.GetAuthByPhone(ctx, req.Phone)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("load supplier credentials: %w", err)
	}
	if !found || strings.TrimSpace(rec.PasswordHash) == "" {
		return LoginResponse{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(rec.PasswordHash), []byte(req.Password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}
	nextStep := "/"
	if !rec.IsConfigured {
		nextStep = "/setup/billing"
	}
	return LoginResponse{
		SupplierID:   rec.SupplierID,
		IsConfigured: rec.IsConfigured,
		NextStep:     nextStep,
	}, nil
}

// ── ConfigureBilling ───────────────────────────────────────────────────────

// BillingSetupRequest is the wire shape for POST /v1/supplier/billing/setup.
type BillingSetupRequest struct {
	BankName         string   `json:"bankName"`
	AccountHolder    string   `json:"accountHolder"`
	AccountNumber    string   `json:"accountNumber"`
	SwiftBic         string   `json:"swiftBic"`
	IBAN             string   `json:"iban,omitempty"`
	SelectedGateways []string `json:"selectedGateways"`
}

// BillingSetupResponse mirrors what the wizard expects.
type BillingSetupResponse struct {
	SupplierID       string   `json:"supplier_id"`
	IsConfigured     bool     `json:"is_configured"`
	SelectedGateways []string `json:"selected_gateways"`
}

var allowedGateways = map[string]struct{}{
	"GLOBAL_PAY": {},
	"ADYEN":      {},
	"AIRWALLEX":  {},
	"CASH":       {},
}

const defaultCoverageRadiusKm = 10.0

var ErrInvalidCredentials = errors.New("invalid_credentials")

// Validate enforces billing payload invariants.
func (r BillingSetupRequest) Validate() error {
	if strings.TrimSpace(r.BankName) == "" {
		return errors.New("bankName required")
	}
	if strings.TrimSpace(r.AccountHolder) == "" {
		return errors.New("accountHolder required")
	}
	if strings.TrimSpace(r.AccountNumber) == "" {
		return errors.New("accountNumber required")
	}
	if strings.TrimSpace(r.SwiftBic) == "" {
		return errors.New("swiftBic required")
	}
	if len(r.SelectedGateways) == 0 {
		return errors.New("selectedGateways required")
	}
	for _, g := range r.SelectedGateways {
		if _, ok := allowedGateways[g]; !ok {
			return fmt.Errorf("unknown gateway %q", g)
		}
	}
	return nil
}

// ConfigureBilling marks the supplier configured, persists banking fields, and
// emits SUPPLIER_BILLING_CONFIGURED.
func (s *Service) ConfigureBilling(ctx context.Context, req BillingSetupRequest) (BillingSetupResponse, error) {
	if err := req.Validate(); err != nil {
		return BillingSetupResponse{}, err
	}
	current, found, err := s.repo.GetProfile(ctx, s.supplierID)
	if err != nil {
		return BillingSetupResponse{}, fmt.Errorf("load supplier: %w", err)
	}
	if !found {
		current = Profile{SupplierID: s.supplierID, Country: s.country, Currency: s.currency}
	}
	now := s.now()
	current.BankName = req.BankName
	current.AccountHolder = req.AccountHolder
	current.AccountNumber = req.AccountNumber
	current.SwiftBic = req.SwiftBic
	current.IBAN = req.IBAN
	current.SelectedGateways = append([]string(nil), req.SelectedGateways...)
	current.IsConfigured = true
	current.ConfiguredAt = now
	current.UpdatedAt = now

	err = s.repo.UpdateProfile(ctx, current, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateSupplier, s.supplierID, events.TopicMain, supplierBillingEvent{
			Type:             events.EventSupplierBillingConfigured,
			SupplierID:       s.supplierID,
			BankName:         current.BankName,
			AccountHolder:    current.AccountHolder,
			SelectedGateways: current.SelectedGateways,
			Timestamp:        now.Format(time.RFC3339Nano),
		})
	})
	if err != nil {
		return BillingSetupResponse{}, fmt.Errorf("persist billing: %w", err)
	}

	s.cache.Invalidate(ctx, supplierCacheKey(s.supplierID))
	s.log.Info("supplier billing configured",
		"supplier_id", s.supplierID,
		"gateways", current.SelectedGateways,
	)
	return BillingSetupResponse{
		SupplierID:       s.supplierID,
		IsConfigured:     true,
		SelectedGateways: current.SelectedGateways,
	}, nil
}

// ── HTTP handlers ──────────────────────────────────────────────────────────

// HandleRegister is the HTTP entry-point for POST /v1/auth/supplier/register.
func (s *Service) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	if key := r.Header.Get("Idempotency-Key"); key != "" && s.idem != nil {
		hash := sha256Hex(body)
		rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, hash)
		switch {
		case errors.Is(err, idempotency.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_key_payload_mismatch"})
			return
		case err != nil:
			s.log.Warn("idempotency guard failed", "err", err)
		case hit:
			s.replaySession(w, rec, false)
			return
		}
	}

	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.Register(r.Context(), req)
	if err != nil {
		s.log.Warn("supplier registration failed", "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	if err := s.writeSessionCookie(w, false); err != nil {
		s.log.Warn("session cookie issue failed", "err", err)
	}
	respBytes, _ := json.Marshal(resp)
	if key := r.Header.Get("Idempotency-Key"); key != "" && s.idem != nil {
		_ = s.idem.Save(r.Context(), key, idempotency.Record{
			BodyHash:   sha256Hex(body),
			StatusCode: http.StatusCreated,
			Response:   respBytes,
			StoredAt:   s.now(),
		}, 24*time.Hour)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandleLogin is the HTTP entry-point for POST /v1/auth/supplier/login.
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req LoginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 16*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	resp, err := s.Login(r.Context(), req)
	if err != nil {
		s.log.Warn("supplier login failed", "phone", req.Phone, "err", err)
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}

	if err := s.writeSessionCookie(w, resp.IsConfigured); err != nil {
		s.log.Warn("session cookie issue failed", "err", err)
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleConfigureBilling is the HTTP entry-point for POST /v1/supplier/billing/setup.
// Expected to run behind auth.CookieAuth + auth.RequireRole(ADMIN).
func (s *Service) HandleConfigureBilling(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 32*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	if key := r.Header.Get("Idempotency-Key"); key != "" && s.idem != nil {
		hash := sha256Hex(body)
		rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, hash)
		switch {
		case errors.Is(err, idempotency.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_key_payload_mismatch"})
			return
		case err != nil:
			s.log.Warn("idempotency guard failed", "err", err)
		case hit:
			s.replaySession(w, rec, true)
			return
		}
	}

	var req BillingSetupRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.ConfigureBilling(r.Context(), req)
	if err != nil {
		s.log.Warn("billing setup failed", "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if err := s.writeSessionCookie(w, true); err != nil {
		s.log.Warn("session cookie reissue failed", "err", err)
	}
	respBytes, _ := json.Marshal(resp)
	if key := r.Header.Get("Idempotency-Key"); key != "" && s.idem != nil {
		_ = s.idem.Save(r.Context(), key, idempotency.Record{
			BodyHash:   sha256Hex(body),
			StatusCode: http.StatusOK,
			Response:   respBytes,
			StoredAt:   s.now(),
		}, 24*time.Hour)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// writeSessionCookie issues the supplier-portal session JWT.
func (s *Service) writeSessionCookie(w http.ResponseWriter, isConfigured bool) error {
	token, err := auth.Issue(auth.Claims{
		Subject:      s.supplierID,
		Role:         auth.RoleAdmin,
		SupplierID:   s.supplierID,
		IsConfigured: isConfigured,
	}, auth.IssueOptions{
		Secret: s.jwtSecret,
		Issuer: s.jwtIssuer,
		TTL:    s.jwtTTL,
		Now:    s.now,
	})
	if err != nil {
		return err
	}
	auth.SetSessionCookie(w, token, s.jwtTTL, s.cookieSecure)
	return nil
}

// replaySession returns an idempotency-stored response but also reissues the
// session cookie so the replay-caller still ends up authenticated.
func (s *Service) replaySession(w http.ResponseWriter, rec idempotency.Record, isConfigured bool) {
	_ = s.writeSessionCookie(w, isConfigured)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rec.StatusCode)
	_, _ = w.Write(rec.Response)
}

// ── Outbox payloads + helpers ──────────────────────────────────────────────

type supplierUpdatedEvent struct {
	Type         string   `json:"type"`
	SupplierID   string   `json:"supplier_id"`
	LegalName    string   `json:"legal_name"`
	ContactName  string   `json:"contact_name"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	Country      string   `json:"country"`
	Categories   []string `json:"categories"`
	IsRegistered bool     `json:"is_registered"`
	IsConfigured bool     `json:"is_configured"`
	Action       string   `json:"action"`
	Timestamp    string   `json:"timestamp"`
}

type supplierBillingEvent struct {
	Type             string   `json:"type"`
	SupplierID       string   `json:"supplier_id"`
	BankName         string   `json:"bank_name"`
	AccountHolder    string   `json:"account_holder"`
	SelectedGateways []string `json:"selected_gateways"`
	Timestamp        string   `json:"timestamp"`
}

func supplierCacheKey(id string) string { return "supplier:" + id }

func rootSupplierUserID(supplierID string) string {
	return "root_" + supplierID
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
