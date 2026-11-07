// Package staffinvite mints HMAC invites for factory/warehouse register (GS-T5).
// No Spanner table — same grain as retailer trading-partner invites (GS-T4).
package staffinvite

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleFactory   = "factory"
	RoleWarehouse = "warehouse"
)

var (
	ErrInviteRequired      = errors.New("staff_invite_required")
	ErrInviteInvalid       = errors.New("invite_invalid")
	ErrInviteExpired       = errors.New("invite_expired")
	ErrInviteSecretMissing = errors.New("invite_secret_missing")
	ErrInviteRoleMismatch  = errors.New("invite_role_mismatch")
	ErrNodeRequired        = errors.New("node_required")
	ErrNodeNotOwned        = errors.New("node_not_owned")
	ErrSeedStaffForbidden  = errors.New("seed_staff_forbidden")
	ErrPasswordRequired    = errors.New("password_required")
)

// Invite is a time-limited staff register grant for one supplier node.
type Invite struct {
	Role       string
	SupplierID string
	NodeID     string
	ExpiresAt  time.Time
}

// NodeOwnedFunc reports whether nodeID belongs to supplierID for role.
type NodeOwnedFunc func(ctx context.Context, supplierID, role, nodeID string) (bool, error)

// DemoScaffoldAllowed is true only in the isolated sandbox (PEGASUSX_ENV=sandbox|ssmr).
func DemoScaffoldAllowed() bool {
	return auth.IsSandbox()
}

// NormalizeRole maps factory/warehouse role strings to invite roles.
func NormalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleFactory, "factory_admin", "factory_staff", string(auth.RoleFactory), string(auth.RoleFactoryAdmin):
		return RoleFactory
	case RoleWarehouse, "warehouse_admin", "warehouse_staff", "payloader", string(auth.RoleWarehouse), string(auth.RoleWarehouseAdmin), string(auth.RolePayload):
		return RoleWarehouse
	default:
		return strings.ToLower(strings.TrimSpace(role))
	}
}

// HashPassword bcrypts a register/login secret. Empty is rejected.
func HashPassword(password string) (string, error) {
	secret := strings.TrimSpace(password)
	if secret == "" {
		return "", ErrPasswordRequired
	}
	h, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// GuardSeed rejects attaching staff to the seed tenant outside ssmr.
func GuardSeed(seedSupplierID, supplierID string) error {
	seed := strings.TrimSpace(seedSupplierID)
	sid := strings.TrimSpace(supplierID)
	if seed == "" || sid == "" || sid != seed {
		return nil
	}
	if DemoScaffoldAllowed() {
		return nil
	}
	return ErrSeedStaffForbidden
}

// Mint signs a time-limited staff invite.
func Mint(secret, role, supplierID, nodeID string, ttl time.Duration, now time.Time) (string, time.Time, error) {
	role = NormalizeRole(role)
	sid := strings.TrimSpace(supplierID)
	nid := strings.TrimSpace(nodeID)
	if strings.TrimSpace(secret) == "" {
		return "", time.Time{}, ErrInviteSecretMissing
	}
	if role != RoleFactory && role != RoleWarehouse {
		return "", time.Time{}, ErrInviteRoleMismatch
	}
	if sid == "" {
		return "", time.Time{}, ErrInviteRequired
	}
	if nid == "" {
		return "", time.Time{}, ErrNodeRequired
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	exp := now.Add(ttl)
	payload := role + "|" + sid + "|" + nid + "|" + strconv.FormatInt(exp.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return token, exp, nil
}

// Parse verifies an invite token.
func Parse(secret, token string, now time.Time) (Invite, error) {
	if strings.TrimSpace(secret) == "" {
		return Invite{}, ErrInviteSecretMissing
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return Invite{}, ErrInviteInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Invite{}, ErrInviteInvalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Invite{}, ErrInviteInvalid
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(raw)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return Invite{}, ErrInviteInvalid
	}
	fields := strings.Split(string(raw), "|")
	if len(fields) != 4 {
		return Invite{}, ErrInviteInvalid
	}
	role := NormalizeRole(fields[0])
	sid := strings.TrimSpace(fields[1])
	nid := strings.TrimSpace(fields[2])
	expUnix, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil || sid == "" || nid == "" || (role != RoleFactory && role != RoleWarehouse) {
		return Invite{}, ErrInviteInvalid
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	exp := time.Unix(expUnix, 0).UTC()
	if !now.Before(exp) {
		return Invite{}, ErrInviteExpired
	}
	return Invite{Role: role, SupplierID: sid, NodeID: nid, ExpiresAt: exp}, nil
}

// Scope is the supplier+node a public staff register may write.
type Scope struct {
	Role       string
	SupplierID string
	NodeID     string
}

// ResolveOpts is the public-register grant: invite XOR supplier ADMIN + node.
type ResolveOpts struct {
	Secret         string
	InviteToken    string
	WantRole       string
	RequestedNode  string
	SeedSupplierID string
	Admin          *auth.Claims
	Now            time.Time
	NodeOwned      NodeOwnedFunc
	Ctx            context.Context
}

// ResolveRegister accepts an invite or an ADMIN JWT that owns the node.
func ResolveRegister(opts ResolveOpts) (Scope, error) {
	want := NormalizeRole(opts.WantRole)
	if tok := strings.TrimSpace(opts.InviteToken); tok != "" {
		inv, err := Parse(opts.Secret, tok, opts.Now)
		if err != nil {
			return Scope{}, err
		}
		if inv.Role != want {
			return Scope{}, ErrInviteRoleMismatch
		}
		if req := strings.TrimSpace(opts.RequestedNode); req != "" && req != inv.NodeID {
			return Scope{}, ErrNodeNotOwned
		}
		if err := GuardSeed(opts.SeedSupplierID, inv.SupplierID); err != nil {
			return Scope{}, err
		}
		return Scope{Role: inv.Role, SupplierID: inv.SupplierID, NodeID: inv.NodeID}, nil
	}
	if opts.Admin != nil && opts.Admin.Role == auth.RoleAdmin {
		sid := strings.TrimSpace(opts.Admin.SupplierID)
		nid := strings.TrimSpace(opts.RequestedNode)
		if sid == "" {
			return Scope{}, ErrInviteRequired
		}
		if nid == "" {
			return Scope{}, ErrNodeRequired
		}
		if err := GuardSeed(opts.SeedSupplierID, sid); err != nil {
			return Scope{}, err
		}
		if opts.NodeOwned != nil {
			ctx := opts.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			ok, err := opts.NodeOwned(ctx, sid, want, nid)
			if err != nil {
				return Scope{}, err
			}
			if !ok {
				return Scope{}, ErrNodeNotOwned
			}
		}
		return Scope{Role: want, SupplierID: sid, NodeID: nid}, nil
	}
	return Scope{}, ErrInviteRequired
}

// ParseBearer reads Authorization: Bearer from a request header map.
func ParseBearer(authorization, secret string) (*auth.Claims, bool) {
	h := strings.TrimSpace(authorization)
	if len(h) < 8 || !strings.EqualFold(h[:7], "bearer ") {
		return nil, false
	}
	tok := strings.TrimSpace(h[7:])
	if tok == "" || strings.TrimSpace(secret) == "" {
		return nil, false
	}
	claims, err := auth.Parse(tok, secret)
	if err != nil {
		return nil, false
	}
	return &claims, true
}

// SpannerNodeOwned checks Factories / Warehouses.SupplierId.
func SpannerNodeOwned(client *spanner.Client) NodeOwnedFunc {
	if client == nil {
		return nil
	}
	return func(ctx context.Context, supplierID, role, nodeID string) (bool, error) {
		sid := strings.TrimSpace(supplierID)
		nid := strings.TrimSpace(nodeID)
		if sid == "" || nid == "" {
			return false, nil
		}
		table, key := "Factories", "FactoryId"
		if NormalizeRole(role) == RoleWarehouse {
			table, key = "Warehouses", "WarehouseId"
		}
		row, err := client.Single().ReadRow(ctx, table, spanner.Key{nid}, []string{key, "SupplierId"})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				return false, nil
			}
			return false, err
		}
		var id, owner string
		if err := row.Columns(&id, &owner); err != nil {
			return false, err
		}
		return strings.TrimSpace(owner) == sid, nil
	}
}
