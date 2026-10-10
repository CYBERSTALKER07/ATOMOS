// Package tenantreg implements GS-T1 self-serve tenant mint.
// POST /v1/platform/tenants/register always creates a new SupplierId.
// It never reuses the seed-overwrite register path.
package tenantreg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrRepoUnavailable    = errors.New("tenant_register_unavailable")
	ErrLegalNameRequired  = errors.New("legal_name_required")
	ErrPhoneRequired      = errors.New("phone_required")
	ErrPasswordRequired   = errors.New("password_required")
	ErrMarketCodeRequired = errors.New("market_code_required")
	ErrUnknownMarket      = errors.New("unknown_market")
	ErrMarketNotShipped   = errors.New("market_not_shipped")
	ErrPhoneTaken         = errors.New("phone_already_registered")
	ErrSeedCollision      = errors.New("seed_id_collision")
)

// Registry is the persist seam. Narrower than supplier.Repository.
type Registry interface {
	GetAuthByPhone(ctx context.Context, phone string) (supplier.SupplierAuthRecord, bool, error)
	GetProfile(ctx context.Context, supplierID string) (supplier.Profile, bool, error)
	UpdateProfile(ctx context.Context, p supplier.Profile, emit func(outbox.TxnBuffer) error) error
}

// Request is the public mint payload.
type Request struct {
	LegalName   string `json:"legal_name"`
	ContactName string `json:"contact_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone"`
	Password    string `json:"password"`
	MarketCode  string `json:"market_code"`
}

// Response is the minted tenant + session token.
type Response struct {
	SupplierID   string `json:"supplier_id"`
	LegalName    string `json:"legal_name"`
	MarketCode   string `json:"market_code"`
	HomeCell     string `json:"home_cell"`
	Source       string `json:"source"`
	IsRegistered bool   `json:"is_registered"`
	IsConfigured bool   `json:"is_configured"`
	NextStep     string `json:"next_step"`
	Token        string `json:"token,omitempty"`
}

// Config wires GS-T1.
type Config struct {
	Repo           Registry
	SeedSupplierID string
	JWTSecret      string
	JWTIssuer      string
	JWTTTL         time.Duration
	CookieSecure   bool
	Idem           idempotency.Store
	Log            *slog.Logger
	Now            func() time.Time
	OnRegistered   func(ctx context.Context, supplierID, legalName string) error
	NewID          func() string
}

// Service mints non-seed suppliers.
type Service struct {
	repo           Registry
	seedSupplierID string
	jwtSecret      string
	jwtIssuer      string
	jwtTTL         time.Duration
	cookieSecure   bool
	idemStore      idempotency.Store
	log            *slog.Logger
	now            func() time.Time
	OnRegistered   func(ctx context.Context, supplierID, legalName string) error
	newID          func() string
}

// NewService returns a configured mint service.
func NewService(c Config) *Service {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.JWTTTL <= 0 {
		c.JWTTTL = 24 * time.Hour
	}
	if c.NewID == nil {
		c.NewID = uuid.NewString
	}
	return &Service{
		repo:           c.Repo,
		seedSupplierID: strings.TrimSpace(c.SeedSupplierID),
		jwtSecret:      c.JWTSecret,
		jwtIssuer:      c.JWTIssuer,
		jwtTTL:         c.JWTTTL,
		cookieSecure:   c.CookieSecure,
		idemStore:      c.Idem,
		log:            c.Log,
		now:            c.Now,
		OnRegistered:   c.OnRegistered,
		newID:          c.NewID,
	}
}

// Register mints a new SupplierId with a shipped pack. Never mutates seed.
func (s *Service) Register(ctx context.Context, req Request) (Response, error) {
	if s == nil || s.repo == nil {
		return Response{}, ErrRepoUnavailable
	}
	legal := strings.TrimSpace(req.LegalName)
	if legal == "" {
		return Response{}, ErrLegalNameRequired
	}
	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		return Response{}, ErrPhoneRequired
	}
	if strings.TrimSpace(req.Password) == "" {
		return Response{}, ErrPasswordRequired
	}
	code := auth.NormalizeMarketCode(req.MarketCode)
	if code == "" {
		return Response{}, ErrMarketCodeRequired
	}
	pack, ok := auth.ResolveMarketPack(code)
	if !ok {
		return Response{}, fmt.Errorf("%w: %s", ErrUnknownMarket, code)
	}
	if pack.Status != auth.MarketPackShipped {
		return Response{}, fmt.Errorf("%w: %s", ErrMarketNotShipped, pack.Code)
	}

	if _, found, err := s.repo.GetAuthByPhone(ctx, phone); err != nil {
		return Response{}, fmt.Errorf("lookup phone: %w", err)
	} else if found {
		return Response{}, ErrPhoneTaken
	}

	supplierID, err := s.mintID(ctx)
	if err != nil {
		return Response{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Response{}, fmt.Errorf("hash password: %w", err)
	}

	contact := strings.TrimSpace(req.ContactName)
	if contact == "" {
		contact = legal
	}
	now := s.now()
	profile := supplier.Profile{
		SupplierID:       supplierID,
		LegalName:        legal,
		ContactName:      contact,
		Email:            strings.TrimSpace(req.Email),
		Phone:            phone,
		AuthPasswordHash: string(passwordHash),
		Country:          pack.Code,
		Currency:         pack.CurrencyCode,
		MarketCode:       pack.Code,
		HomeCell:         pack.HomeCell,
		IsRegistered:     false,
		IsConfigured:     false,
		RegisteredAt:     now,
		UpdatedAt:        now,
	}

	if err := s.repo.UpdateProfile(ctx, profile, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateSupplier, supplierID, events.TopicMain, events.SupplierEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventSupplierProfileUpdated, Timestamp: now.Format(time.RFC3339Nano)},
			SupplierID:   supplierID,
			LegalName:    profile.LegalName,
			ContactName:  profile.ContactName,
			Email:        profile.Email,
			Phone:        profile.Phone,
			Country:      profile.Country,
			IsRegistered: profile.IsRegistered,
			IsConfigured: profile.IsConfigured,
			Action:       "TENANT_REGISTERED",
		})
	}); err != nil {
		return Response{}, fmt.Errorf("persist tenant: %w", err)
	}

	if s.OnRegistered != nil {
		if hookErr := s.OnRegistered(ctx, supplierID, legal); hookErr != nil {
			s.log.Warn("platform tenant mint failed", "supplier_id", supplierID, "err", hookErr)
		}
	}

	return Response{
		SupplierID:   supplierID,
		LegalName:    legal,
		MarketCode:   pack.Code,
		HomeCell:     pack.HomeCell,
		Source:       auth.MarketSourceProfile,
		IsRegistered: profile.IsRegistered,
		IsConfigured: profile.IsConfigured,
		NextStep:     "/setup/business",
	}, nil
}

func (s *Service) mintID(ctx context.Context) (string, error) {
	for i := 0; i < 3; i++ {
		id := strings.TrimSpace(s.newID())
		if id == "" {
			continue
		}
		if s.seedSupplierID != "" && id == s.seedSupplierID {
			continue
		}
		_, found, err := s.repo.GetProfile(ctx, id)
		if err != nil {
			return "", fmt.Errorf("check minted id: %w", err)
		}
		if found {
			continue
		}
		return id, nil
	}
	return "", ErrSeedCollision
}

// IssueToken stamps a supplier ADMIN JWT for the minted tenant.
func (s *Service) IssueToken(supplierID, marketCode, homeCell string) (string, error) {
	if strings.TrimSpace(s.jwtSecret) == "" {
		return "", errors.New("jwt_not_configured")
	}
	return auth.Issue(auth.Claims{
		Subject:    supplierID,
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
		MarketCode: marketCode,
		HomeCell:   homeCell,
	}, auth.IssueOptions{
		Secret: s.jwtSecret,
		Issuer: s.jwtIssuer,
		TTL:    s.jwtTTL,
		Now:    s.now,
	})
}
