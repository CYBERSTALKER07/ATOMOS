package order

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

// Claim window policy sources persisted on Orders.ClaimWindowPolicySource.
const (
	ClaimWindowSourceSupplier          = "SUPPLIER"
	ClaimWindowSourceWarehouseOverride = "WAREHOUSE_OVERRIDE"
	ClaimWindowSourceEnv               = "ENV"
	ClaimWindowSourceDefault           = "DEFAULT"
)

const (
	minClaimWindowHours = 1
	maxClaimWindowHours = 168
)

// SupplierReturnPolicy is the commercial SoT for retailer file-claim window.
type SupplierReturnPolicy struct {
	SupplierID                 string
	DefaultWindowHours         int64
	ConcealedDamageWindowHours *int64
	RequirePhoto               bool
	AllowExpiredClaims         bool
	UpdatedAt                  time.Time
	UpdatedByUserID            string
}

// WarehouseReturnPolicy is ops reverse-dock + optional lengthen of retailer window.
type WarehouseReturnPolicy struct {
	WarehouseID               string
	SupplierID                string // empty = warehouse-wide default
	ReverseDockSLAHours       *int64
	RetailerFileWindowHours   *int64
	CanOverrideRetailerWindow bool
	UpdatedAt                 time.Time
	UpdatedByUserID           string
}

// ReturnPolicyStore loads/saves claim window policies (Spanner or memory).
type ReturnPolicyStore interface {
	GetSupplierReturnPolicy(ctx context.Context, supplierID string) (SupplierReturnPolicy, bool, error)
	UpsertSupplierReturnPolicy(ctx context.Context, p SupplierReturnPolicy) error
	GetWarehouseReturnPolicy(ctx context.Context, warehouseID, supplierID string) (WarehouseReturnPolicy, bool, error)
	UpsertWarehouseReturnPolicy(ctx context.Context, p WarehouseReturnPolicy) error
}

// ResolvedClaimWindow is the result of resolve_window at COMPLETED time.
type ResolvedClaimWindow struct {
	Hours  int64
	Source string
}

func claimWindowHoursFromEnv() (hours int64, source string) {
	raw := strings.TrimSpace(os.Getenv("CLAIM_WINDOW_HOURS"))
	if raw == "" {
		return int64(DefaultPostDeliveryAmendWindow.Hours()), ClaimWindowSourceDefault
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < minClaimWindowHours {
		return int64(DefaultPostDeliveryAmendWindow.Hours()), ClaimWindowSourceDefault
	}
	if n > maxClaimWindowHours {
		n = maxClaimWindowHours
	}
	return n, ClaimWindowSourceEnv
}

func clampClaimWindowHours(h int64) int64 {
	if h < minClaimWindowHours {
		return minClaimWindowHours
	}
	if h > maxClaimWindowHours {
		return maxClaimWindowHours
	}
	return h
}

// ResolveClaimWindow computes hours + policy source (supplier base, env fallback, WH lengthen-only).
func ResolveClaimWindow(ctx context.Context, store ReturnPolicyStore, supplierID, warehouseID string) (ResolvedClaimWindow, error) {
	base, source := claimWindowHoursFromEnv()
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)

	if store != nil && supplierID != "" {
		p, ok, err := store.GetSupplierReturnPolicy(ctx, supplierID)
		if err != nil {
			return ResolvedClaimWindow{}, err
		}
		if ok && p.DefaultWindowHours > 0 {
			base = clampClaimWindowHours(p.DefaultWindowHours)
			source = ClaimWindowSourceSupplier
		}
	}

	hours := base
	if store != nil && warehouseID != "" {
		wh, ok, err := store.GetWarehouseReturnPolicy(ctx, warehouseID, supplierID)
		if err != nil {
			return ResolvedClaimWindow{}, err
		}
		if !ok {
			wh, ok, err = store.GetWarehouseReturnPolicy(ctx, warehouseID, "")
			if err != nil {
				return ResolvedClaimWindow{}, err
			}
		}
		if ok && wh.CanOverrideRetailerWindow && wh.RetailerFileWindowHours != nil && *wh.RetailerFileWindowHours > 0 {
			override := clampClaimWindowHours(*wh.RetailerFileWindowHours)
			if override > hours {
				hours = override
				source = ClaimWindowSourceWarehouseOverride
			}
		}
	}

	return ResolvedClaimWindow{Hours: hours, Source: source}, nil
}

// ApplyClaimWindowSnapshot fills immutable claim-window columns when entering COMPLETED.
// Idempotent: skips when ClaimWindowEndsAt already set.
func (s *Service) ApplyClaimWindowSnapshot(ctx context.Context, o *Order, completedAt time.Time) error {
	if o == nil {
		return nil
	}
	if o.ClaimWindowEndsAt != nil && !o.ClaimWindowEndsAt.IsZero() {
		return nil
	}
	if completedAt.IsZero() {
		completedAt = s.now().UTC()
	} else {
		completedAt = completedAt.UTC()
	}
	var store ReturnPolicyStore
	if s != nil {
		store = s.returnPolicies
	}
	resolved, err := ResolveClaimWindow(ctx, store, o.SupplierID, o.WarehouseID)
	if err != nil {
		// Fail open to env/default so COMPLETED is never blocked by policy read errors.
		h, src := claimWindowHoursFromEnv()
		resolved = ResolvedClaimWindow{Hours: h, Source: src}
		if s != nil && s.log != nil {
			s.log.WarnContext(ctx, "claim window policy resolve failed; using env/default",
				"order_id", o.OrderID, "err", err)
		}
	}
	ends := completedAt.Add(time.Duration(resolved.Hours) * time.Hour)
	o.ClaimWindowHours = resolved.Hours
	o.ClaimWindowEndsAt = &ends
	o.ClaimWindowPolicySource = resolved.Source
	return nil
}

// HasClaimWindowSnapshot reports whether the order already has an immutable window.
func (o Order) HasClaimWindowSnapshot() bool {
	return o.ClaimWindowEndsAt != nil && !o.ClaimWindowEndsAt.IsZero()
}
