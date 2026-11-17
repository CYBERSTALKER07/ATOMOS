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
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
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

// JoinRequestPendingError indicates duplicate INN in the target HomeCell and
// returns HTTP 409 join_request_pending instead of minting a duplicate tenant.
type JoinRequestPendingError struct {
	RequestID      string
	TargetTenantID string
	LegalName      string
	NormalizedINN  string
}

func (e *JoinRequestPendingError) Error() string { return innlookup.ErrDuplicateTenant.Error() }
func (e *JoinRequestPendingError) Unwrap() error { return innlookup.ErrDuplicateTenant }

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
	TaxID       string `json:"tax_id,omitempty"`
	RoleSought  string `json:"role_sought,omitempty"`
}

// Response is the minted tenant + session token.
type Response struct {
	SupplierID      string `json:"supplier_id"`
	LegalName       string `json:"legal_name"`
	MarketCode      string `json:"market_code"`
	HomeCell        string `json:"home_cell"`
	Source          string `json:"source"`
	NormalizedTaxID string `json:"normalized_tax_id,omitempty"`
	InnVerifiedAt   string `json:"inn_verified_at,omitempty"`
	IsRegistered    bool   `json:"is_registered"`
	IsConfigured    bool   `json:"is_configured"`
	NextStep        string `json:"next_step"`
	Token           string `json:"token,omitempty"`
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
	INNLookup      *innlookup.Service
	INNRequired    bool
	OnJoinRequest  func(ctx context.Context, supplierID string, jr innlookup.JoinRequest)
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
	innLookup      *innlookup.Service
	innRequired    bool
	onJoinRequest  func(ctx context.Context, supplierID string, jr innlookup.JoinRequest)
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
		innLookup:      c.INNLookup,
		innRequired:    c.INNRequired,
		onJoinRequest:  c.OnJoinRequest,
	}
}

// Register mints a new SupplierId with a shipped pack. Never mutates seed.
func (s *Service) Register(ctx context.Context, req Request) (Response, error) {
	if s == nil || s.repo == nil {
		return Response{}, ErrRepoUnavailable
	}
	legal := strings.TrimSpace(req.LegalName)
	rawTaxID := strings.TrimSpace(req.TaxID)
	if legal == "" && rawTaxID == "" {
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

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Response{}, fmt.Errorf("hash password: %w", err)
	}

	now := s.now()
	var normalizedINN string
	var innVerifiedAt time.Time
	if s.innRequired || rawTaxID != "" {
		norm, _, err := innlookup.NormalizeTaxID(rawTaxID)
		if err != nil {
			return Response{}, err
		}
		roleSought, rerr := innlookup.NormalizeRoleSought(req.RoleSought)
		if rerr != nil {
			return Response{}, rerr
		}
		if s.innLookup == nil {
			return Response{}, innlookup.ErrRegistryUnavailable
		}
		res, lerr := s.innLookup.Lookup(ctx, innlookup.LookupOptions{
			TaxID:      norm,
			HomeCell:   pack.HomeCell,
			MarketCode: pack.Code,
			TenantType: innlookup.TenantTypeSupplier,
		})
		if lerr != nil || res.Status == innlookup.StatusUnavailable {
			if lerr != nil && !errors.Is(lerr, innlookup.ErrRegistryUnavailable) {
				return Response{}, lerr
			}
			return Response{}, innlookup.ErrRegistryUnavailable
		}
		if res.ExistingTenantID != "" {
			reqName := strings.TrimSpace(req.ContactName)
			if reqName == "" {
				reqName = legal
			}
			jr, jerr := s.innLookup.CreateJoinRequest(ctx, innlookup.JoinRequest{
				HomeCell:              pack.HomeCell,
				TargetTenantType:      innlookup.TenantTypeSupplier,
				TargetTenantID:        res.ExistingTenantID,
				NormalizedINN:         res.NormalizedINN,
				RequesterPhone:        phone,
				RequesterName:         reqName,
				RequesterEmail:        strings.TrimSpace(req.Email),
				RoleSought:            roleSought,
				RequesterPasswordHash: string(passwordHash),
				CreatedAt:             now,
			}, func(txn outbox.TxnBuffer) error {
				return outbox.EmitJSON(ctx, txn, events.AggregateSupplier, res.ExistingTenantID, events.TopicMain, map[string]any{
					"type":               "JOIN_REQUEST_CREATED",
					"timestamp":          now.Format(time.RFC3339Nano),
					"supplier_id":        res.ExistingTenantID,
					"target_tenant_type": innlookup.TenantTypeSupplier,
					"target_tenant_id":   res.ExistingTenantID,
					"normalized_inn":     res.NormalizedINN,
					"requester_phone":    phone,
					"requester_name":     reqName,
					"role_sought":        roleSought,
				})
			})
			if jerr != nil {
				return Response{}, fmt.Errorf("create join request: %w", jerr)
			}
			if s.onJoinRequest != nil {
				s.onJoinRequest(ctx, res.ExistingTenantID, jr)
			}
			legalDisplay := strings.TrimSpace(res.ExistingLegalName)
			if legalDisplay == "" {
				legalDisplay = strings.TrimSpace(res.LegalName)
			}
			return Response{}, &JoinRequestPendingError{
				RequestID:      jr.RequestID,
				TargetTenantID: res.ExistingTenantID,
				LegalName:      legalDisplay,
				NormalizedINN:  res.NormalizedINN,
			}
		}
		if res.Status == innlookup.StatusNotFound {
			return Response{}, innlookup.ErrTaxIDNotFound
		}
		if legal == "" {
			legal = strings.TrimSpace(res.LegalName)
		}
		if legal == "" {
			return Response{}, ErrLegalNameRequired
		}
		normalizedINN = res.NormalizedINN
		innVerifiedAt = now
	}

	supplierID, err := s.mintID(ctx)
	if err != nil {
		return Response{}, err
	}

	contact := strings.TrimSpace(req.ContactName)
	if contact == "" {
		contact = legal
	}
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
		TaxID:            rawTaxID,
		NormalizedTaxID:  normalizedINN,
		InnVerifiedAt:    innVerifiedAt,
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
		var hookErr error
		for attempt := 0; attempt < 3; attempt++ {
			if ctx.Err() != nil {
				break
			}
			hookErr = s.OnRegistered(ctx, supplierID, legal)
			if hookErr == nil {
				break
			}
			time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
		}
		if hookErr != nil {
			s.log.Warn("platform tenant mint failed", "supplier_id", supplierID, "err", hookErr)
		}
	}

	innVerifiedStr := ""
	if !profile.InnVerifiedAt.IsZero() {
		innVerifiedStr = profile.InnVerifiedAt.Format(time.RFC3339Nano)
	}
	return Response{
		SupplierID:      supplierID,
		LegalName:       legal,
		MarketCode:      pack.Code,
		HomeCell:        pack.HomeCell,
		Source:          auth.MarketSourceProfile,
		NormalizedTaxID: profile.NormalizedTaxID,
		InnVerifiedAt:   innVerifiedStr,
		IsRegistered:    profile.IsRegistered,
		IsConfigured:    profile.IsConfigured,
		NextStep:        "/setup/business",
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
