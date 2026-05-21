// Package retailer owns the retailer-domain handlers, services and repository
// boundaries. In pegasusX every retailer is scoped to the single seeded
// supplier, so the registration handler does NOT accept a supplier_id from the
// body — it resolves the seeded supplier id from the application context.
package retailer

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

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Repository is the storage seam. Production binds this to a Spanner-backed
// implementation that runs every mutation inside a ReadWriteTransaction and
// uses the supplied TxnBuffer to write the OutboxEvents row atomically.
type Repository interface {
	// CreateRetailer inserts the row + outbox event inside one RW transaction.
	// emit is invoked with a TxnBuffer scoped to the same transaction so the
	// caller can append an outbox event atomically.
	CreateRetailer(ctx context.Context, r Retailer, emit func(outbox.TxnBuffer) error) error
	// FindByPhone returns the retailer for a phone number if present.
	FindByPhone(ctx context.Context, phone string) (Retailer, bool, error)
	// GetRetailer returns the retailer by id.
	GetRetailer(ctx context.Context, retailerID string) (Retailer, bool, error)
	// UpdateRetailer mutates a retailer row and optionally emits outbox payload.
	UpdateRetailer(ctx context.Context, r Retailer, emit func(outbox.TxnBuffer) error) error
	// ListRetailersBySupplier lists retailer rows by supplier scope.
	ListRetailersBySupplier(ctx context.Context, supplierID string) ([]Retailer, error)
}

// Retailer is the persisted aggregate.
type Retailer struct {
	RetailerID  string
	SupplierID  string
	Phone       string
	Name        string
	CountryCode string
	Lat         float64
	Lng         float64
	H3Cell      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Service wires repository, cache, idempotency and outbox dependencies.
type Service struct {
	repo        Repository
	cache       *cache.Cache
	idem        idempotency.Store
	proximity   *RetailerProximityService
	supplierID  string
	countryCode string
	log         *slog.Logger
	now         func() time.Time
	newID       func() string

	mu                sync.RWMutex
	favoriteSuppliers map[string]map[string]bool
	cartByRetailer    map[string]map[string]any
	familyByRetailer  map[string][]FamilyMember
}

// ServiceConfig is the constructor input.
type ServiceConfig struct {
	Repo        Repository
	Cache       *cache.Cache
	Idem        idempotency.Store
	Proximity   *RetailerProximityService
	SupplierID  string
	CountryCode string
	Log         *slog.Logger
	Now         func() time.Time
	NewID       func() string
}

// NewService constructs a Service with sensible defaults for Now/NewID.
func NewService(c ServiceConfig) *Service {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.NewID == nil {
		c.NewID = defaultRetailerID
	}
	return &Service{
		repo:              c.Repo,
		cache:             c.Cache,
		idem:              c.Idem,
		proximity:         c.Proximity,
		supplierID:        c.SupplierID,
		countryCode:       c.CountryCode,
		log:               c.Log,
		now:               c.Now,
		newID:             c.NewID,
		favoriteSuppliers: make(map[string]map[string]bool),
		cartByRetailer:    make(map[string]map[string]any),
		familyByRetailer:  make(map[string][]FamilyMember),
	}
}

// RegisterRequest is the wire shape for POST /v1/auth/retailer/register.
type RegisterRequest struct {
	Phone  string  `json:"phone"`
	Name   string  `json:"name,omitempty"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
	H3Cell string  `json:"h3_cell"`
}

// RegisterResponse is what callers get back.
type RegisterResponse struct {
	RetailerID string `json:"retailer_id"`
	Phone      string `json:"phone"`
	H3Cell     string `json:"h3_cell"`
	CreatedAt  string `json:"created_at"`
}

// Validate enforces input invariants. Returns a human-readable error suitable
// for direct JSON surfacing.
func (r RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Phone) == "" {
		return errors.New("phone required")
	}
	if strings.TrimSpace(r.H3Cell) != "" && len(r.H3Cell) != 15 {
		return fmt.Errorf("h3_cell must be 15-char hex, got %d", len(r.H3Cell))
	}
	if r.Lat == 0 && r.Lng == 0 {
		return errors.New("lat/lng required")
	}
	return nil
}

// Register performs the registration mutation: dedupe by phone, write retailer
// row + RETAILER_REGISTERED outbox event in the same RW transaction, then
// invalidate any cached retailer-by-phone entry.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return RegisterResponse{}, err
	}

	// Dedupe: phone is uniquely indexed. A retry MAY hit a row that already
	// exists; treat that as idempotent success.
	if existing, found, err := s.repo.FindByPhone(ctx, req.Phone); err != nil {
		return RegisterResponse{}, fmt.Errorf("lookup retailer: %w", err)
	} else if found {
		return RegisterResponse{
			RetailerID: existing.RetailerID,
			Phone:      existing.Phone,
			H3Cell:     existing.H3Cell,
			CreatedAt:  existing.CreatedAt.Format(time.RFC3339Nano),
		}, nil
	}

	h3Cell := strings.TrimSpace(req.H3Cell)
	if s.proximity != nil {
		cell, err := s.proximity.CellForCoordinate(req.Lat, req.Lng)
		if err != nil {
			return RegisterResponse{}, fmt.Errorf("derive retailer h3_cell: %w", err)
		}
		h3Cell = cell
	}
	if h3Cell == "" {
		return RegisterResponse{}, errors.New("h3_cell required")
	}

	r := Retailer{
		RetailerID:  s.newID(),
		SupplierID:  s.supplierID,
		Phone:       req.Phone,
		Name:        req.Name,
		CountryCode: s.countryCode,
		Lat:         req.Lat,
		Lng:         req.Lng,
		H3Cell:      h3Cell,
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
	}

	err := s.repo.CreateRetailer(ctx, r, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateRetailer, r.RetailerID, events.TopicMain, retailerRegisteredEvent{
			Type:        events.EventRetailerRegistered,
			RetailerID:  r.RetailerID,
			Phone:       r.Phone,
			Name:        r.Name,
			SupplierID:  s.supplierID,
			Lat:         r.Lat,
			Lng:         r.Lng,
			H3Cell:      r.H3Cell,
			CountryCode: r.CountryCode,
			Timestamp:   r.CreatedAt.Format(time.RFC3339Nano),
		})
	})
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("persist retailer: %w", err)
	}

	// Post-commit cache invalidation. Pre-commit invalidation races with
	// rollback — TTL is the safety net but never the correctness mechanism.
	s.cache.Invalidate(ctx, retailerByPhoneKey(r.Phone))

	s.log.Info("retailer registered",
		"retailer_id", r.RetailerID,
		"supplier_id", s.supplierID,
		"h3_cell", r.H3Cell,
	)
	return RegisterResponse{
		RetailerID: r.RetailerID,
		Phone:      r.Phone,
		H3Cell:     r.H3Cell,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339Nano),
	}, nil
}

type retailerRegisteredEvent struct {
	Type        string  `json:"type"`
	RetailerID  string  `json:"retailer_id"`
	Phone       string  `json:"phone"`
	Name        string  `json:"name,omitempty"`
	SupplierID  string  `json:"supplier_id"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	H3Cell      string  `json:"h3_cell"`
	CountryCode string  `json:"country_code"`
	Timestamp   string  `json:"timestamp"`
}

func retailerByPhoneKey(phone string) string {
	return "retailer:phone:" + phone
}

// HandleRegister is the HTTP entry-point for POST /v1/auth/retailer/register.
// Wired by retailerroutes.RegisterRoutes.
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

	// Optional Idempotency-Key flow.
	if key := r.Header.Get("Idempotency-Key"); key != "" && s.idem != nil {
		hash := sha256Hex(body)
		rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, hash)
		switch {
		case errors.Is(err, idempotency.ErrConflict):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "idempotency_key_payload_mismatch",
			})
			return
		case err != nil:
			s.log.Warn("idempotency guard failed", "err", err)
		case hit:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(rec.StatusCode)
			_, _ = w.Write(rec.Response)
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
		s.log.Warn("retailer registration failed", "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
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

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func defaultRetailerID() string {
	// Scaffold: timestamp-based id. Production swaps for uuid.NewV7.
	return fmt.Sprintf("ret_%d", time.Now().UnixNano())
}
