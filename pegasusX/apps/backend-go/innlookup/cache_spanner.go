package innlookup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// MemoryStore is an in-memory CacheStore, TenantDirectory, and JoinRequestStore for unit tests.
type MemoryStore struct {
	mu           sync.RWMutex
	cache        map[string]LookupResult
	tenants      map[string]TenantMatch
	joinRequests map[string]JoinRequest
	memberships  map[string]map[string]string // targetTenantID -> phone -> role
	events       []outbox.Event
}

// NewMemoryStore initializes a thread-safe in-memory INN cache and join request store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		cache:        make(map[string]LookupResult),
		tenants:      make(map[string]TenantMatch),
		joinRequests: make(map[string]JoinRequest),
		memberships:  make(map[string]map[string]string),
	}
}

func cacheKey(homeCell, normalizedINN string) string {
	return strings.TrimSpace(homeCell) + "|" + strings.TrimSpace(normalizedINN)
}

func tenantKey(homeCell, tenantType, normalizedINN string) string {
	return strings.TrimSpace(homeCell) + "|" + strings.ToUpper(strings.TrimSpace(tenantType)) + "|" + strings.TrimSpace(normalizedINN)
}

// RegisterTenant registers a test tenant in MemoryStore for duplicate-INN verification.
func (m *MemoryStore) RegisterTenant(match TenantMatch) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[tenantKey(match.HomeCell, match.TenantType, match.NormalizedINN)] = match
}

// MembershipRole returns the provisioned role for (tenantID, phone) in MemoryStore tests.
func (m *MemoryStore) MembershipRole(tenantID, phone string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	role, ok := m.memberships[tenantID][phone]
	return role, ok
}

// GetCachedINN returns a non-expired cache entry from MemoryStore.
func (m *MemoryStore) GetCachedINN(_ context.Context, homeCell, normalizedINN string, now time.Time) (LookupResult, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res, ok := m.cache[cacheKey(homeCell, normalizedINN)]
	if !ok || !res.ExpiresAt.After(now) {
		return LookupResult{}, false, nil
	}
	return res, true, nil
}

// PutCachedINN persists FOUND or NOT_FOUND into MemoryStore.
func (m *MemoryStore) PutCachedINN(_ context.Context, homeCell string, res LookupResult) error {
	if res.Status != StatusFound && res.Status != StatusNotFound {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[cacheKey(homeCell, res.NormalizedINN)] = res
	return nil
}

// FindTenantByNormalizedINN looks up an existing tenant owning normalizedINN in homeCell.
func (m *MemoryStore) FindTenantByNormalizedINN(_ context.Context, homeCell, tenantType, normalizedINN string) (TenantMatch, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	match, ok := m.tenants[tenantKey(homeCell, tenantType, normalizedINN)]
	return match, ok, nil
}

// CreateJoinRequest stores or returns an idempotent PENDING JoinRequest in MemoryStore.
func sanitizeJoinRequestCopy(req JoinRequest) JoinRequest {
	req.RequesterPasswordHash = ""
	return req
}

func (m *MemoryStore) CreateJoinRequest(ctx context.Context, req JoinRequest, emit func(outbox.TxnBuffer) error) (JoinRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.joinRequests {
		if existing.Status == JoinStatusPending &&
			existing.TargetTenantType == req.TargetTenantType &&
			existing.TargetTenantID == req.TargetTenantID &&
			existing.RequesterPhone == req.RequesterPhone {
			return sanitizeJoinRequestCopy(existing), nil
		}
	}
	if strings.TrimSpace(req.RequestID) == "" {
		req.RequestID = "jr_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	if req.Status == "" {
		req.Status = JoinStatusPending
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	buf := &memoryTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return JoinRequest{}, err
		}
	}
	m.joinRequests[req.RequestID] = req
	m.events = append(m.events, buf.events...)
	return sanitizeJoinRequestCopy(req), nil
}

// ListJoinRequests returns matching join requests ordered newest-first.
func (m *MemoryStore) ListJoinRequests(_ context.Context, targetTenantType, targetTenantID, status string, limit int) ([]JoinRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]JoinRequest, 0)
	upperType := strings.ToUpper(strings.TrimSpace(targetTenantType))
	upperStatus := strings.ToUpper(strings.TrimSpace(status))
	for _, req := range m.joinRequests {
		if upperType != "" && req.TargetTenantType != upperType {
			continue
		}
		if req.TargetTenantID != targetTenantID {
			continue
		}
		if upperStatus != "" && req.Status != upperStatus {
			continue
		}
		out = append(out, sanitizeJoinRequestCopy(req))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// GetJoinRequest reads one join request scoped to the owning tenant.
func (m *MemoryStore) GetJoinRequest(_ context.Context, targetTenantType, targetTenantID, requestID string) (JoinRequest, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.joinRequests[requestID]
	if !ok || req.TargetTenantID != targetTenantID || req.TargetTenantType != strings.ToUpper(strings.TrimSpace(targetTenantType)) {
		return JoinRequest{}, false, nil
	}
	return sanitizeJoinRequestCopy(req), true, nil
}

// DecideJoinRequest approves or rejects a PENDING join request and records supplier membership on approval.
func (m *MemoryStore) DecideJoinRequest(ctx context.Context, targetTenantType, targetTenantID, requestID, decisionStatus, decidedBy, reason string, decidedAt time.Time, emit func(outbox.TxnBuffer) error) (JoinRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	req, ok := m.joinRequests[requestID]
	if !ok || req.TargetTenantID != targetTenantID || req.TargetTenantType != strings.ToUpper(strings.TrimSpace(targetTenantType)) {
		return JoinRequest{}, ErrJoinRequestNotFound
	}
	if req.Status != JoinStatusPending {
		return sanitizeJoinRequestCopy(req), ErrJoinRequestDecided
	}
	req.Status = decisionStatus
	req.DecidedBy = decidedBy
	decidedCopy := decidedAt.UTC()
	req.DecidedAt = &decidedCopy
	req.DecisionReason = reason
	req.RequesterPasswordHash = ""

	buf := &memoryTxnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return JoinRequest{}, err
		}
	}
	if decisionStatus == JoinStatusApproved {
		if m.memberships[targetTenantID] == nil {
			m.memberships[targetTenantID] = make(map[string]string)
		}
		m.memberships[targetTenantID][req.RequesterPhone] = req.RoleSought
	}
	m.joinRequests[requestID] = req
	m.events = append(m.events, buf.events...)
	return sanitizeJoinRequestCopy(req), nil
}

type memoryTxnBuffer struct {
	events []outbox.Event
}

func (b *memoryTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// SpannerStore persists InnRegistryCache, duplicate-INN lookups, and JoinRequests in Cloud Spanner.
type SpannerStore struct {
	client *spanner.Client
}

// NewSpannerStore constructs a Spanner-backed innlookup store.
func NewSpannerStore(client *spanner.Client) *SpannerStore {
	return &SpannerStore{client: client}
}

// GetCachedINN fetches a non-expired InnRegistryCache entry.
func (s *SpannerStore) GetCachedINN(ctx context.Context, homeCell, normalizedINN string, now time.Time) (LookupResult, bool, error) {
	if s == nil || s.client == nil {
		return LookupResult{}, false, fmt.Errorf("spanner inn store: nil client")
	}
	row, err := s.client.Single().ReadRow(ctx, "InnRegistryCache", spanner.Key{homeCell, normalizedINN}, []string{
		"status", "legal_name", "address", "raw_json", "fetched_at", "expires_at",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return LookupResult{}, false, nil
		}
		return LookupResult{}, false, fmt.Errorf("read InnRegistryCache: %w", err)
	}
	var status string
	var legalName, address spanner.NullString
	var rawJSON spanner.NullJSON
	var fetchedAt, expiresAt time.Time
	if err := row.Columns(&status, &legalName, &address, &rawJSON, &fetchedAt, &expiresAt); err != nil {
		return LookupResult{}, false, fmt.Errorf("scan InnRegistryCache: %w", err)
	}
	if !expiresAt.After(now) {
		return LookupResult{}, false, nil
	}
	var rawBytes json.RawMessage
	if rawJSON.Valid {
		if encoded, merr := json.Marshal(rawJSON.Value); merr == nil {
			rawBytes = encoded
		}
	}
	_, entityType, _ := NormalizeTaxID(normalizedINN)
	return LookupResult{
		Status:        status,
		NormalizedINN: normalizedINN,
		EntityType:    entityType,
		LegalName:     legalName.StringVal,
		Address:       address.StringVal,
		RawJSON:       rawBytes,
		Cached:        true,
		FetchedAt:     fetchedAt,
		ExpiresAt:     expiresAt,
	}, true, nil
}

// PutCachedINN writes FOUND or NOT_FOUND entries into InnRegistryCache.
func (s *SpannerStore) PutCachedINN(ctx context.Context, homeCell string, res LookupResult) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("spanner inn store: nil client")
	}
	if res.Status != StatusFound && res.Status != StatusNotFound {
		return nil
	}
	var rawJSON spanner.NullJSON
	if len(res.RawJSON) > 0 && json.Valid(res.RawJSON) {
		var val any
		if err := json.Unmarshal(res.RawJSON, &val); err == nil {
			rawJSON = spanner.NullJSON{Value: val, Valid: true}
		}
	}
	_, err := s.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("InnRegistryCache", map[string]any{
			"home_cell":  homeCell,
			"inn":        res.NormalizedINN,
			"status":     res.Status,
			"legal_name": nullString(res.LegalName),
			"address":    nullString(res.Address),
			"raw_json":   rawJSON,
			"fetched_at": res.FetchedAt.UTC(),
			"expires_at": res.ExpiresAt.UTC(),
		}),
	})
	return err
}

// FindTenantByNormalizedINN queries Suppliers or Retailers within homeCell by NormalizedTaxId.
func (s *SpannerStore) FindTenantByNormalizedINN(ctx context.Context, homeCell, tenantType, normalizedINN string) (TenantMatch, bool, error) {
	if s == nil || s.client == nil {
		return TenantMatch{}, false, fmt.Errorf("spanner inn store: nil client")
	}
	upperType := strings.ToUpper(strings.TrimSpace(tenantType))
	var sql string
	switch upperType {
	case TenantTypeSupplier:
		sql = `SELECT SupplierId, COALESCE(Name, '')
		       FROM Suppliers
		       WHERE HomeCell = @cell AND NormalizedTaxId = @inn
		       LIMIT 1`
	case TenantTypeRetailer:
		sql = `SELECT RetailerId, COALESCE(Name, '')
		       FROM Retailers
		       WHERE HomeCell = @cell AND NormalizedTaxId = @inn
		       LIMIT 1`
	default:
		return TenantMatch{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: sql,
		Params: map[string]any{
			"cell": strings.TrimSpace(homeCell),
			"inn":  strings.TrimSpace(normalizedINN),
		},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return TenantMatch{}, false, nil
	}
	if err != nil {
		return TenantMatch{}, false, fmt.Errorf("find %s by normalized inn: %w", upperType, err)
	}
	var id, name string
	if err := row.Columns(&id, &name); err != nil {
		return TenantMatch{}, false, err
	}
	return TenantMatch{
		HomeCell:      homeCell,
		TenantType:    upperType,
		TenantID:      id,
		LegalName:     name,
		NormalizedINN: normalizedINN,
	}, true, nil
}

// CreateJoinRequest inserts a JoinRequests row if no identical PENDING request exists for requester phone.
func (s *SpannerStore) CreateJoinRequest(ctx context.Context, req JoinRequest, emit func(outbox.TxnBuffer) error) (JoinRequest, error) {
	if s == nil || s.client == nil {
		return JoinRequest{}, fmt.Errorf("spanner inn store: nil client")
	}
	var saved JoinRequest
	err := spannerutils.RunReadWriteTransaction(ctx, s.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		checkStmt := spanner.Statement{
			SQL: `SELECT request_id, home_cell, target_tenant_type, target_tenant_id, normalized_inn,
			             requester_phone, COALESCE(requester_name, ''), COALESCE(requester_email, ''),
			             role_sought, status, created_at
			      FROM JoinRequests@{FORCE_INDEX=Idx_JoinRequests_ByTargetStatus}
			      WHERE target_tenant_type = @ttype AND target_tenant_id = @tid AND status = 'PENDING' AND requester_phone = @phone
			      LIMIT 1`,
			Params: map[string]any{
				"ttype": req.TargetTenantType,
				"tid":   req.TargetTenantID,
				"phone": req.RequesterPhone,
			},
		}
		iter := txn.Query(ctx, checkStmt)
		row, err := iter.Next()
		iter.Stop()
		if err == nil {
			var existing JoinRequest
			if serr := row.Columns(
				&existing.RequestID,
				&existing.HomeCell,
				&existing.TargetTenantType,
				&existing.TargetTenantID,
				&existing.NormalizedINN,
				&existing.RequesterPhone,
				&existing.RequesterName,
				&existing.RequesterEmail,
				&existing.RoleSought,
				&existing.Status,
				&existing.CreatedAt,
			); serr != nil {
				return serr
			}
			saved = existing
			return nil
		}
		if err != iterator.Done {
			return err
		}

		if strings.TrimSpace(req.RequestID) == "" {
			req.RequestID = "jr_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		}
		if req.Status == "" {
			req.Status = JoinStatusPending
		}
		if req.CreatedAt.IsZero() {
			req.CreatedAt = time.Now().UTC()
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("JoinRequests", map[string]any{
				"request_id":              req.RequestID,
				"home_cell":               req.HomeCell,
				"target_tenant_type":      req.TargetTenantType,
				"target_tenant_id":        req.TargetTenantID,
				"normalized_inn":          req.NormalizedINN,
				"requester_phone":         req.RequesterPhone,
				"requester_name":          nullString(req.RequesterName),
				"requester_email":         nullString(req.RequesterEmail),
				"role_sought":             req.RoleSought,
				"requester_password_hash": nullString(req.RequesterPasswordHash),
				"status":                  req.Status,
				"decided_by":              spanner.NullString{},
				"decided_at":              spanner.NullTime{},
				"decision_reason":         spanner.NullString{},
				"created_at":              req.CreatedAt.UTC(),
			}),
		}); err != nil {
			return err
		}
		if emit != nil {
			buf := outbox.NewSpannerTxnBuffer(txn)
			if err := emit(buf); err != nil {
				return err
			}
			if err := buf.Flush(ctx); err != nil {
				return err
			}
		}
		saved = sanitizeJoinRequestCopy(req)
		return nil
	})
	if err != nil {
		return JoinRequest{}, err
	}
	return saved, nil
}

// ListJoinRequests returns JoinRequests for targetTenantID ordered newest first.
func (s *SpannerStore) ListJoinRequests(ctx context.Context, targetTenantType, targetTenantID, status string, limit int) ([]JoinRequest, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("spanner inn store: nil client")
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	upperType := strings.ToUpper(strings.TrimSpace(targetTenantType))
	upperStatus := strings.ToUpper(strings.TrimSpace(status))
	sql := `SELECT request_id, home_cell, target_tenant_type, target_tenant_id, normalized_inn,
	               requester_phone, COALESCE(requester_name, ''), COALESCE(requester_email, ''),
	               role_sought, status, decided_by, decided_at, decision_reason, created_at
	        FROM JoinRequests@{FORCE_INDEX=Idx_JoinRequests_ByTargetStatus}
	        WHERE target_tenant_type = @ttype AND target_tenant_id = @tid`
	params := map[string]any{
		"ttype": upperType,
		"tid":   targetTenantID,
		"lim":   int64(limit),
	}
	if upperStatus != "" {
		sql += ` AND status = @status`
		params["status"] = upperStatus
	}
	sql += ` ORDER BY created_at DESC LIMIT @lim`

	iter := s.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]JoinRequest, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list join requests: %w", err)
		}
		req, err := scanJoinRequest(row)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, nil
}

// GetJoinRequest reads a single JoinRequest enforcing target tenant scope.
func (s *SpannerStore) GetJoinRequest(ctx context.Context, targetTenantType, targetTenantID, requestID string) (JoinRequest, bool, error) {
	if s == nil || s.client == nil {
		return JoinRequest{}, false, fmt.Errorf("spanner inn store: nil client")
	}
	row, err := s.client.Single().ReadRow(ctx, "JoinRequests", spanner.Key{requestID}, joinRequestCols)
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return JoinRequest{}, false, nil
		}
		return JoinRequest{}, false, err
	}
	req, err := scanJoinRequest(row)
	if err != nil {
		return JoinRequest{}, false, err
	}
	if req.TargetTenantID != targetTenantID || req.TargetTenantType != strings.ToUpper(strings.TrimSpace(targetTenantType)) {
		return JoinRequest{}, false, nil
	}
	return req, true, nil
}

// DecideJoinRequest atomically approves or rejects a JoinRequest and creates SupplierUsers/RetailerUsers on approval.
func (s *SpannerStore) DecideJoinRequest(ctx context.Context, targetTenantType, targetTenantID, requestID, decisionStatus, decidedBy, reason string, decidedAt time.Time, emit func(outbox.TxnBuffer) error) (JoinRequest, error) {
	if s == nil || s.client == nil {
		return JoinRequest{}, fmt.Errorf("spanner inn store: nil client")
	}
	upperType := strings.ToUpper(strings.TrimSpace(targetTenantType))
	var updated JoinRequest
	err := spannerutils.RunReadWriteTransaction(ctx, s.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		colsWithPwd := append(append([]string(nil), joinRequestCols...), "requester_password_hash")
		row, err := txn.ReadRow(ctx, "JoinRequests", spanner.Key{requestID}, colsWithPwd)
		if err != nil {
			if errors.Is(err, spanner.ErrRowNotFound) {
				return ErrJoinRequestNotFound
			}
			return err
		}
		req, err := scanJoinRequestWithPassword(row)
		if err != nil {
			return err
		}
		if req.TargetTenantID != targetTenantID || req.TargetTenantType != upperType {
			return ErrJoinRequestNotFound
		}
		if req.Status != JoinStatusPending {
			return ErrJoinRequestDecided
		}

		now := decidedAt.UTC()
		req.Status = decisionStatus
		req.DecidedBy = decidedBy
		req.DecidedAt = &now
		req.DecisionReason = reason

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("JoinRequests", map[string]any{
				"request_id":              requestID,
				"status":                  decisionStatus,
				"decided_by":              nullString(decidedBy),
				"decided_at":              now,
				"decision_reason":         nullString(reason),
				"requester_password_hash": nil,
			}),
		}

		if decisionStatus == JoinStatusApproved && upperType == TenantTypeSupplier {
			if err := appendSupplierApprovalMutations(ctx, txn, targetTenantID, req, now, &mutations); err != nil {
				return err
			}
		} else if decisionStatus == JoinStatusApproved && upperType == TenantTypeRetailer {
			if err := appendRetailerApprovalMutations(ctx, txn, targetTenantID, req, now, &mutations); err != nil {
				return err
			}
		}

		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}
		if emit != nil {
			buf := outbox.NewSpannerTxnBuffer(txn)
			if err := emit(buf); err != nil {
				return err
			}
			if err := buf.Flush(ctx); err != nil {
				return err
			}
		}
		updated = sanitizeJoinRequestCopy(req)
		return nil
	})
	if err != nil {
		return JoinRequest{}, err
	}
	return updated, nil
}

func appendSupplierApprovalMutations(ctx context.Context, txn *spanner.ReadWriteTransaction, targetTenantID string, req JoinRequest, now time.Time, mutations *[]*spanner.Mutation) error {
	existingIter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT UserId FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone}
		      WHERE Phone = @phone AND SupplierId = @sid LIMIT 1`,
		Params: map[string]any{"phone": req.RequesterPhone, "sid": targetTenantID},
	})
	_, existErr := existingIter.Next()
	existingIter.Stop()
	if existErr != nil && existErr != iterator.Done {
		return existErr
	}
	if existErr == nil {
		return nil
	}
	role, rerr := NormalizeRoleSought(req.RoleSought)
	if rerr != nil || role == string(auth.RoleRetailer) {
		role = string(auth.RoleAdmin)
	}
	name := strings.TrimSpace(req.RequesterName)
	if name == "" {
		name = req.RequesterPhone
	}
	*mutations = append(*mutations, spanner.InsertMap("SupplierUsers", map[string]any{
		"UserId":              uuid.NewString(),
		"SupplierId":          targetTenantID,
		"Email":               nullString(req.RequesterEmail),
		"Phone":               req.RequesterPhone,
		"Name":                name,
		"PasswordHash":        strings.TrimSpace(req.RequesterPasswordHash),
		"SupplierRole":        role,
		"AssignedWarehouseId": nil,
		"AssignedFactoryId":   nil,
		"IsActive":            true,
		"FirebaseUid":         "",
		"CreatedAt":           now,
		"UpdatedAt":           now,
	}))
	return nil
}

func normalizeRetailerJoinRole(raw string) string {
	role := strings.ToUpper(strings.TrimSpace(raw))
	switch role {
	case "", string(auth.RoleRetailer):
		return "MANAGER"
	case "ADMIN", "MANAGER", "BUYER", "RECEIVER", "CASHIER", "STOCK_CLERK", "SECTION_LEAD", "VIEWER":
		return role
	default:
		return "MANAGER"
	}
}

func appendRetailerApprovalMutations(ctx context.Context, txn *spanner.ReadWriteTransaction, targetTenantID string, req JoinRequest, now time.Time, mutations *[]*spanner.Mutation) error {
	existingIter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT UserId, RetailerRole FROM RetailerUsers@{FORCE_INDEX=Idx_RetailerUsers_ByPhone}
		      WHERE Phone = @phone AND RetailerId = @rid LIMIT 1`,
		Params: map[string]any{"phone": req.RequesterPhone, "rid": targetTenantID},
	})
	existingRow, existErr := existingIter.Next()
	existingIter.Stop()
	if existErr != nil && existErr != iterator.Done {
		return existErr
	}
	if existErr == nil {
		var existingUID, existingRole string
		if err := existingRow.Columns(&existingUID, &existingRole); err != nil {
			return err
		}
		_, mErr := txn.ReadRow(ctx, "RetailerUserMemberships", spanner.Key{existingUID, targetTenantID}, []string{"UserId"})
		if spanner.ErrCode(mErr) == codes.NotFound || errors.Is(mErr, spanner.ErrRowNotFound) {
			*mutations = append(*mutations, spanner.InsertMap("RetailerUserMemberships", map[string]any{
				"UserId":          existingUID,
				"RetailerId":      targetTenantID,
				"RetailerRole":    normalizeRetailerJoinRole(existingRole),
				"IsActive":        true,
				"LocationIdsJson": nil,
				"CreatedAt":       now,
				"UpdatedAt":       now,
			}))
		} else if mErr != nil {
			return mErr
		}
		return nil
	}

	pwdHash := strings.TrimSpace(req.RequesterPasswordHash)
	firebaseUID := ""
	name := strings.TrimSpace(req.RequesterName)

	// Carry forward login credentials from any active RetailerUsers row owned by the requester phone.
	credIter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT IFNULL(Name, ''), IFNULL(PasswordHash, ''), IFNULL(FirebaseUid, '')
		      FROM RetailerUsers@{FORCE_INDEX=Idx_RetailerUsers_ByPhone}
		      WHERE Phone = @phone AND IsActive = TRUE
		      LIMIT 10`,
		Params: map[string]any{"phone": req.RequesterPhone},
	})
	for {
		cRow, cErr := credIter.Next()
		if cErr == iterator.Done {
			break
		}
		if cErr != nil {
			credIter.Stop()
			return cErr
		}
		var cName, cPwd, cFB string
		if err := cRow.Columns(&cName, &cPwd, &cFB); err != nil {
			credIter.Stop()
			return err
		}
		if name == "" && strings.TrimSpace(cName) != "" {
			name = strings.TrimSpace(cName)
		}
		if pwdHash == "" && strings.TrimSpace(cPwd) != "" {
			pwdHash = strings.TrimSpace(cPwd)
		}
		if firebaseUID == "" && strings.TrimSpace(cFB) != "" {
			firebaseUID = strings.TrimSpace(cFB)
		}
	}
	credIter.Stop()

	if name == "" {
		name = req.RequesterPhone
	}
	retailerRole := normalizeRetailerJoinRole(req.RoleSought)
	newUserID := uuid.NewString()
	*mutations = append(*mutations,
		spanner.InsertMap("RetailerUsers", map[string]any{
			"UserId":       newUserID,
			"RetailerId":   targetTenantID,
			"Phone":        req.RequesterPhone,
			"Name":         name,
			"PasswordHash": nullString(pwdHash),
			"FirebaseUid":  nullString(firebaseUID),
			"RetailerRole": retailerRole,
			"IsOwner":      false,
			"IsActive":     true,
			"CreatedAt":    now,
			"UpdatedAt":    now,
		}),
		spanner.InsertMap("RetailerUserMemberships", map[string]any{
			"UserId":          newUserID,
			"RetailerId":      targetTenantID,
			"RetailerRole":    retailerRole,
			"IsActive":        true,
			"LocationIdsJson": nil,
			"CreatedAt":       now,
			"UpdatedAt":       now,
		}),
	)
	return nil
}

var joinRequestCols = []string{
	"request_id",
	"home_cell",
	"target_tenant_type",
	"target_tenant_id",
	"normalized_inn",
	"requester_phone",
	"requester_name",
	"requester_email",
	"role_sought",
	"status",
	"decided_by",
	"decided_at",
	"decision_reason",
	"created_at",
}

func scanJoinRequest(row *spanner.Row) (JoinRequest, error) {
	var req JoinRequest
	var name, email, decidedBy, reason spanner.NullString
	var decidedAt spanner.NullTime
	if err := row.Columns(
		&req.RequestID,
		&req.HomeCell,
		&req.TargetTenantType,
		&req.TargetTenantID,
		&req.NormalizedINN,
		&req.RequesterPhone,
		&name,
		&email,
		&req.RoleSought,
		&req.Status,
		&decidedBy,
		&decidedAt,
		&reason,
		&req.CreatedAt,
	); err != nil {
		return JoinRequest{}, fmt.Errorf("scan JoinRequests row: %w", err)
	}
	req.RequesterName = name.StringVal
	req.RequesterEmail = email.StringVal
	req.DecidedBy = decidedBy.StringVal
	req.DecisionReason = reason.StringVal
	if decidedAt.Valid {
		t := decidedAt.Time
		req.DecidedAt = &t
	}
	return req, nil
}

func scanJoinRequestWithPassword(row *spanner.Row) (JoinRequest, error) {
	var req JoinRequest
	var name, email, decidedBy, reason, pwdHash spanner.NullString
	var decidedAt spanner.NullTime
	if err := row.Columns(
		&req.RequestID,
		&req.HomeCell,
		&req.TargetTenantType,
		&req.TargetTenantID,
		&req.NormalizedINN,
		&req.RequesterPhone,
		&name,
		&email,
		&req.RoleSought,
		&req.Status,
		&decidedBy,
		&decidedAt,
		&reason,
		&req.CreatedAt,
		&pwdHash,
	); err != nil {
		return JoinRequest{}, fmt.Errorf("scan JoinRequests row: %w", err)
	}
	req.RequesterName = name.StringVal
	req.RequesterEmail = email.StringVal
	req.DecidedBy = decidedBy.StringVal
	req.DecisionReason = reason.StringVal
	req.RequesterPasswordHash = pwdHash.StringVal
	if decidedAt.Valid {
		t := decidedAt.Time
		req.DecidedAt = &t
	}
	return req, nil
}

func nullString(s string) spanner.NullString {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: trimmed, Valid: true}
}
