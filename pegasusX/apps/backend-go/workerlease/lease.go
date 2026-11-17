package workerlease

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

const (
	// WorkerLeasesTable is the Cloud Spanner table storing distributed worker leases.
	WorkerLeasesTable = "WorkerLeases"

	// MaxLeaseNameLen matches STRING(128) on WorkerLeases.LeaseName.
	MaxLeaseNameLen = 128

	// MaxHolderIDLen matches STRING(128) on WorkerLeases.HolderId.
	MaxHolderIDLen = 128

	// DefaultTTL is the default lease duration when Config.TTL is zero.
	DefaultTTL = 30 * time.Second

	// DefaultRetryInterval is the default standby poll delay when another pod holds the lease.
	DefaultRetryInterval = 5 * time.Second

	// DefaultReleaseTimeout bounds lease release on graceful shutdown.
	DefaultReleaseTimeout = 3 * time.Second
)

var (
	// ErrSpannerClientNil indicates a Spanner-backed lease manager has a nil client.
	ErrSpannerClientNil = errors.New("workerlease: spanner client is nil")
	// ErrManagerNil indicates no lease manager was provided.
	ErrManagerNil = errors.New("workerlease: manager is nil")
	// ErrInvalidLeaseName indicates an empty or oversized lease name.
	ErrInvalidLeaseName = errors.New("workerlease: invalid lease name")
	// ErrInvalidHolderID indicates an empty or oversized holder id.
	ErrInvalidHolderID = errors.New("workerlease: invalid holder id")
	// ErrInvalidTTL indicates a non-positive lease TTL.
	ErrInvalidTTL = errors.New("workerlease: ttl must be positive")
	// ErrLeaseLost indicates an active holder failed to renew its lease before expiry.
	ErrLeaseLost = errors.New("workerlease: lease lost")
)

// Lease mirrors a row in Cloud Spanner table WorkerLeases.
type Lease struct {
	LeaseName string    `spanner:"LeaseName" json:"lease_name"`
	HolderID  string    `spanner:"HolderId" json:"holder_id"`
	ExpiresAt time.Time `spanner:"ExpiresAt" json:"expires_at"`
	UpdatedAt time.Time `spanner:"UpdatedAt" json:"updated_at"`
}

// Manager provides fenced distributed lease operations across worker pods.
type Manager interface {
	TryAcquireOrRenew(ctx context.Context, leaseName, holderID string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, leaseName, holderID string) error
	IsHeld(ctx context.Context, leaseName string) (bool, error)
}

// DefaultHolderID builds a collision-free holder identifier bounded to MaxHolderIDLen.
func DefaultHolderID(prefix string) string {
	cleanPrefix := strings.TrimSpace(prefix)
	if cleanPrefix == "" {
		cleanPrefix = "worker"
	}
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "unknown-host"
	}
	suffix := uuid.NewString()
	maxPrefixHost := MaxHolderIDLen - len(suffix) - 2
	prefixHost := cleanPrefix + ":" + strings.TrimSpace(host)
	if len(prefixHost) > maxPrefixHost && maxPrefixHost > 0 {
		prefixHost = prefixHost[:maxPrefixHost]
	}
	id := prefixHost + ":" + suffix
	if len(id) > MaxHolderIDLen {
		return id[:MaxHolderIDLen]
	}
	return id
}

func validateLeaseAndHolder(leaseName, holderID string) (string, string, error) {
	ln := strings.TrimSpace(leaseName)
	if ln == "" || len(ln) > MaxLeaseNameLen {
		return "", "", ErrInvalidLeaseName
	}
	hid := strings.TrimSpace(holderID)
	if hid == "" || len(hid) > MaxHolderIDLen {
		return "", "", ErrInvalidHolderID
	}
	return ln, hid, nil
}

func validateLeaseName(leaseName string) (string, error) {
	ln := strings.TrimSpace(leaseName)
	if ln == "" || len(ln) > MaxLeaseNameLen {
		return "", ErrInvalidLeaseName
	}
	return ln, nil
}

// SpannerManager implements Manager backed by Cloud Spanner table WorkerLeases.
type SpannerManager struct {
	client *spanner.Client
	now    func() time.Time
}

// NewSpannerManager creates a Spanner-backed distributed lease manager.
func NewSpannerManager(client *spanner.Client) *SpannerManager {
	return &SpannerManager{
		client: client,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides the clock source used for expiry checks (primarily for testing).
func (m *SpannerManager) WithClock(now func() time.Time) *SpannerManager {
	if now != nil {
		m.now = func() time.Time { return now().UTC() }
	}
	return m
}

// TryAcquireOrRenew atomically acquires leaseName if unheld/expired, or renews it if already held by holderID.
func (m *SpannerManager) TryAcquireOrRenew(ctx context.Context, leaseName, holderID string, ttl time.Duration) (bool, error) {
	if m == nil || m.client == nil {
		return false, ErrSpannerClientNil
	}
	ln, hid, err := validateLeaseAndHolder(leaseName, holderID)
	if err != nil {
		return false, err
	}
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}

	var acquired bool
	_, err = m.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		acquired = false
		now := m.now().UTC()

		row, readErr := txn.ReadRow(ctx, WorkerLeasesTable, spanner.Key{ln}, []string{"HolderId", "ExpiresAt"})
		if readErr != nil {
			if spanner.ErrCode(readErr) != codes.NotFound {
				return fmt.Errorf("read worker lease %s: %w", ln, readErr)
			}
		} else {
			var existingHolder string
			var expiresAt time.Time
			if scanErr := row.Columns(&existingHolder, &expiresAt); scanErr != nil {
				return fmt.Errorf("scan worker lease %s: %w", ln, scanErr)
			}
			if existingHolder != hid && expiresAt.After(now) {
				acquired = false
				return nil
			}
		}

		mut := spanner.InsertOrUpdateMap(WorkerLeasesTable, map[string]any{
			"LeaseName": ln,
			"HolderId":  hid,
			"ExpiresAt": now.Add(ttl),
			"UpdatedAt": spanner.CommitTimestamp,
		})
		if writeErr := txn.BufferWrite([]*spanner.Mutation{mut}); writeErr != nil {
			return fmt.Errorf("write worker lease %s: %w", ln, writeErr)
		}
		acquired = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("try acquire worker lease %s: %w", ln, err)
	}
	return acquired, nil
}

// Release performs a fenced delete of leaseName only if currently owned by holderID.
func (m *SpannerManager) Release(ctx context.Context, leaseName, holderID string) error {
	if m == nil || m.client == nil {
		return ErrSpannerClientNil
	}
	ln, hid, err := validateLeaseAndHolder(leaseName, holderID)
	if err != nil {
		return err
	}

	_, err = m.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, WorkerLeasesTable, spanner.Key{ln}, []string{"HolderId"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return nil
			}
			return fmt.Errorf("read worker lease %s for release: %w", ln, readErr)
		}
		var existingHolder string
		if scanErr := row.Columns(&existingHolder); scanErr != nil {
			return fmt.Errorf("scan worker lease %s for release: %w", ln, scanErr)
		}
		if existingHolder != hid {
			// Fenced: never release a lease held by another pod.
			return nil
		}
		if writeErr := txn.BufferWrite([]*spanner.Mutation{spanner.Delete(WorkerLeasesTable, spanner.Key{ln})}); writeErr != nil {
			return fmt.Errorf("delete worker lease %s: %w", ln, writeErr)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("release worker lease %s: %w", ln, err)
	}
	return nil
}

// IsHeld reports whether any holder currently holds an unexpired lease for leaseName.
func (m *SpannerManager) IsHeld(ctx context.Context, leaseName string) (bool, error) {
	if m == nil || m.client == nil {
		return false, ErrSpannerClientNil
	}
	ln, err := validateLeaseName(leaseName)
	if err != nil {
		return false, err
	}

	row, err := m.client.Single().ReadRow(ctx, WorkerLeasesTable, spanner.Key{ln}, []string{"ExpiresAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("check worker lease %s: %w", ln, err)
	}
	var expiresAt time.Time
	if err := row.Columns(&expiresAt); err != nil {
		return false, fmt.Errorf("scan worker lease %s expiry: %w", ln, err)
	}
	return expiresAt.After(m.now().UTC()), nil
}

// GetLease returns the raw WorkerLeases row if present.
func (m *SpannerManager) GetLease(ctx context.Context, leaseName string) (*Lease, bool, error) {
	if m == nil || m.client == nil {
		return nil, false, ErrSpannerClientNil
	}
	ln, err := validateLeaseName(leaseName)
	if err != nil {
		return nil, false, err
	}

	row, err := m.client.Single().ReadRow(ctx, WorkerLeasesTable, spanner.Key{ln}, []string{"LeaseName", "HolderId", "ExpiresAt", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get worker lease %s: %w", ln, err)
	}
	var out Lease
	if err := row.ToStruct(&out); err != nil {
		return nil, false, fmt.Errorf("decode worker lease %s: %w", ln, err)
	}
	return &out, true, nil
}

// MemoryManager is an in-process Manager for unit tests and single-process TestingMode runs.
type MemoryManager struct {
	mu     sync.Mutex
	leases map[string]Lease
	now    func() time.Time
}

// NewMemoryManager constructs an in-memory lease manager.
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		leases: make(map[string]Lease),
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides the clock source on MemoryManager for deterministic tests.
func (m *MemoryManager) WithClock(now func() time.Time) *MemoryManager {
	m.mu.Lock()
	defer m.mu.Unlock()
	if now != nil {
		m.now = func() time.Time { return now().UTC() }
	}
	return m
}

// TryAcquireOrRenew acquires or renews leaseName for holderID with exact Spanner expiry semantics.
func (m *MemoryManager) TryAcquireOrRenew(ctx context.Context, leaseName, holderID string, ttl time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	ln, hid, err := validateLeaseAndHolder(leaseName, holderID)
	if err != nil {
		return false, err
	}
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now().UTC()
	if cur, ok := m.leases[ln]; ok {
		if cur.HolderID != hid && cur.ExpiresAt.After(now) {
			return false, nil
		}
	}
	m.leases[ln] = Lease{
		LeaseName: ln,
		HolderID:  hid,
		ExpiresAt: now.Add(ttl),
		UpdatedAt: now,
	}
	return true, nil
}

// Release performs a fenced release of leaseName only if owned by holderID.
func (m *MemoryManager) Release(ctx context.Context, leaseName, holderID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ln, hid, err := validateLeaseAndHolder(leaseName, holderID)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cur, ok := m.leases[ln]; ok && cur.HolderID == hid {
		delete(m.leases, ln)
	}
	return nil
}

// IsHeld reports whether an unexpired lease exists for leaseName.
func (m *MemoryManager) IsHeld(ctx context.Context, leaseName string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	ln, err := validateLeaseName(leaseName)
	if err != nil {
		return false, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cur, ok := m.leases[ln]
	if !ok {
		return false, nil
	}
	return cur.ExpiresAt.After(m.now().UTC()), nil
}

// Config parameterizes fenced lease acquisition and renewal loops.
type Config struct {
	LeaseName     string
	HolderID      string
	TTL           time.Duration
	RenewInterval time.Duration
	RetryInterval time.Duration
	Now           func() time.Time
	Log           *slog.Logger
}

func normalizeConfig(cfg Config) (Config, error) {
	ln, err := validateLeaseName(cfg.LeaseName)
	if err != nil {
		return cfg, err
	}
	cfg.LeaseName = ln

	if strings.TrimSpace(cfg.HolderID) == "" {
		cfg.HolderID = DefaultHolderID(ln)
	} else {
		_, hid, err := validateLeaseAndHolder(ln, cfg.HolderID)
		if err != nil {
			return cfg, err
		}
		cfg.HolderID = hid
	}

	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	if cfg.RenewInterval <= 0 || cfg.RenewInterval >= cfg.TTL {
		cfg.RenewInterval = cfg.TTL / 3
		if cfg.RenewInterval <= 0 {
			cfg.RenewInterval = time.Millisecond
		}
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = DefaultRetryInterval
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	return cfg, nil
}

// WithLease acquires the named lease once and runs fn inside a fenced context that is
// cancelled immediately if the lease is lost or cannot be renewed before expiry.
// Returns acquired=false, err=nil when another pod currently holds the lease.
func WithLease(ctx context.Context, mgr Manager, cfg Config, fn func(leaseCtx context.Context) error) (bool, error) {
	if mgr == nil {
		return false, ErrManagerNil
	}
	if fn == nil {
		return false, errors.New("workerlease: fn is nil")
	}
	cfg, err := normalizeConfig(cfg)
	if err != nil {
		return false, err
	}

	acquired, err := mgr.TryAcquireOrRenew(ctx, cfg.LeaseName, cfg.HolderID, cfg.TTL)
	if err != nil || !acquired {
		return acquired, err
	}

	leaseExpiresAt := cfg.Now().UTC().Add(cfg.TTL)
	safetyMargin := cfg.TTL / 6
	if safetyMargin <= 0 {
		safetyMargin = time.Millisecond
	}

	leaseCtx, leaseCancel := context.WithCancel(ctx)
	defer leaseCancel()

	fnDone := make(chan error, 1)
	go func() {
		fnDone <- fn(leaseCtx)
	}()

	ticker := time.NewTicker(cfg.RenewInterval)
	defer ticker.Stop()

	var (
		fnErr     error
		fnExited  bool
		leaseLost bool
		renewErr  error
	)

loop:
	for {
		select {
		case fnErr = <-fnDone:
			fnExited = true
			leaseCancel()
			break loop
		case <-ctx.Done():
			leaseCancel()
			break loop
		case <-ticker.C:
			ok, err := mgr.TryAcquireOrRenew(leaseCtx, cfg.LeaseName, cfg.HolderID, cfg.TTL)
			if err == nil && ok {
				leaseExpiresAt = cfg.Now().UTC().Add(cfg.TTL)
				continue
			}
			now := cfg.Now().UTC()
			if err == nil && !ok {
				leaseLost = true
				renewErr = ErrLeaseLost
				cfg.Log.WarnContext(ctx, "worker lease lost to another holder; cancelling fenced context",
					"lease", cfg.LeaseName, "holder_id", cfg.HolderID)
				leaseCancel()
				break loop
			}
			// Transient error: allow retries while safely inside lease window, otherwise fence.
			if !now.Add(safetyMargin).Before(leaseExpiresAt) {
				leaseLost = true
				renewErr = fmt.Errorf("%w: %v", ErrLeaseLost, err)
				cfg.Log.WarnContext(ctx, "worker lease renewal failed near expiry; fencing worker",
					"lease", cfg.LeaseName, "holder_id", cfg.HolderID, "err", err)
				leaseCancel()
				break loop
			}
			cfg.Log.WarnContext(ctx, "worker lease renewal error; will retry before expiry",
				"lease", cfg.LeaseName, "holder_id", cfg.HolderID, "err", err)
		}
	}

	if !fnExited {
		fnErr = <-fnDone
	}

	var releaseErr error
	if !leaseLost {
		releaseCtx, cancelRelease := context.WithTimeout(context.WithoutCancel(ctx), DefaultReleaseTimeout)
		if relErr := mgr.Release(releaseCtx, cfg.LeaseName, cfg.HolderID); relErr != nil {
			cfg.Log.WarnContext(ctx, "failed to release worker lease on exit",
				"lease", cfg.LeaseName, "holder_id", cfg.HolderID, "err", relErr)
			releaseErr = relErr
		}
		cancelRelease()
	}

	if leaseLost {
		return true, errors.Join(renewErr, fnErr)
	}
	if ctx.Err() != nil && fnErr == nil && releaseErr == nil {
		return true, nil
	}
	return true, errors.Join(fnErr, releaseErr)
}

// RunWithLease continuously runs fn under distributed lease ownership until ctx is cancelled.
// Standby pods poll every RetryInterval and automatically take over if the active leader drops its lease.
func RunWithLease(ctx context.Context, mgr Manager, cfg Config, fn func(leaseCtx context.Context) error) error {
	if mgr == nil {
		return ErrManagerNil
	}
	if fn == nil {
		return errors.New("workerlease: fn is nil")
	}
	cfg, err := normalizeConfig(cfg)
	if err != nil {
		return err
	}

	for {
		if ctx.Err() != nil {
			return nil
		}
		acquired, runErr := WithLease(ctx, mgr, cfg, fn)
		if ctx.Err() != nil {
			return nil
		}
		if runErr != nil {
			cfg.Log.WarnContext(ctx, "leased worker cycle ended with error; backing off",
				"lease", cfg.LeaseName, "holder_id", cfg.HolderID, "acquired", acquired, "err", runErr)
		}
		timer := time.NewTimer(cfg.RetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

// StartLeasedLoop spawns a background goroutine (optionally tracked by wg) that acquires
// cfg.LeaseName and runs fn(leaseCtx). Unlike RunWithLease, if fn returns on its own before
// leaseCtx is cancelled (for example when a worker feature flag is disabled), the loop exits
// cleanly rather than restarting an intentionally disabled worker every RetryInterval.
func StartLeasedLoop(
	ctx context.Context,
	wg *sync.WaitGroup,
	mgr Manager,
	cfg Config,
	fn func(leaseCtx context.Context),
) {
	if mgr == nil || fn == nil {
		return
	}
	normCfg, err := normalizeConfig(cfg)
	if err != nil {
		if cfg.Log != nil {
			cfg.Log.ErrorContext(ctx, "invalid workerlease config; refusing to start unleased worker",
				"lease", cfg.LeaseName, "err", err)
		}
		return
	}
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		for {
			if ctx.Err() != nil {
				return
			}
			exitedVoluntarily := false
			acquired, runErr := WithLease(ctx, mgr, normCfg, func(leaseCtx context.Context) error {
				fn(leaseCtx)
				if leaseCtx.Err() == nil && ctx.Err() == nil {
					exitedVoluntarily = true
				}
				return nil
			})
			if ctx.Err() != nil || exitedVoluntarily {
				return
			}
			if runErr != nil {
				normCfg.Log.WarnContext(ctx, "leased worker loop lost lease or errored; retrying",
					"lease", normCfg.LeaseName, "holder_id", normCfg.HolderID, "acquired", acquired, "err", runErr)
			}
			timer := time.NewTimer(normCfg.RetryInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
}
