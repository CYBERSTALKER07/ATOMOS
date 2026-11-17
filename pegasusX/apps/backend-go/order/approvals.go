package order

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	ApprovalStatusPendingOwner = "PENDING_OWNER"
	ApprovalStatusApproving    = "APPROVING"
	ApprovalStatusApproved     = "APPROVED"
	ApprovalStatusRejected     = "REJECTED"
	ApprovalStatusExpired      = "EXPIRED"

	maxApprovedReplayCache = 2000

	ApprovalPayloadCreate          = "CREATE"
	ApprovalPayloadUnifiedCheckout = "UNIFIED_CHECKOUT"

	defaultApprovalTTL    = 24 * time.Hour
	defaultApprovalsLimit = 20
	maxApprovalsPageLimit = 100
)

var (
	ErrApprovalNotFound      = errors.New("approval_request_not_found")
	ErrApprovalForbidden     = errors.New("approval_forbidden")
	ErrApprovalInvalidState  = errors.New("approval_invalid_state")
	ErrApprovalInvalidCursor = errors.New("approval_invalid_cursor")
)

// ApprovalPayloadEnvelope stores the complete canonical request payload for whole-cart replay.
type ApprovalPayloadEnvelope struct {
	Kind            string                  `json:"kind"` // CREATE | UNIFIED_CHECKOUT
	RetailerID      string                  `json:"retailer_id"`
	RequestedByUser string                  `json:"requested_by_user"`
	ClientOrderID   string                  `json:"client_order_id"`
	PaymentMethod   string                  `json:"payment_method"`
	UsageDate       string                  `json:"usage_date"`
	SupplierID      string                  `json:"supplier_id,omitempty"`
	TotalMinor      int64                   `json:"total_minor"`
	Currency        string                  `json:"currency,omitempty"`
	CreateRequest   *CreateRequest          `json:"create_request,omitempty"`
	UnifiedRequest  *UnifiedCheckoutRequest `json:"unified_request,omitempty"`
}

// ApprovalRequestDTO mirrors one RetailerApprovalRequests row.
type ApprovalRequestDTO struct {
	RequestID       string                   `json:"request_id"`
	RetailerID      string                   `json:"retailer_id"`
	SupplierID      string                   `json:"supplier_id,omitempty"`
	RequestedByUser string                   `json:"requested_by_user"`
	RequestedByName string                   `json:"requested_by_name,omitempty"`
	ClientOrderID   string                   `json:"client_order_id"`
	AmountMinor     int64                    `json:"amount_minor"`
	TrippedRule     string                   `json:"tripped_rule"`
	Status          string                   `json:"status"`
	PayloadKind     string                   `json:"payload_kind,omitempty"`
	PaymentMethod   string                   `json:"payment_method,omitempty"`
	OrderPayload    *ApprovalPayloadEnvelope `json:"order_payload,omitempty"`
	OrderPayloadRaw string                   `json:"-"`
	ApprovedByUser  string                   `json:"approved_by_user,omitempty"`
	DecisionReason  string                   `json:"decision_reason,omitempty"`
	ExpiresAt       string                   `json:"expires_at"`
	CreatedAt       string                   `json:"created_at"`
	UpdatedAt       string                   `json:"updated_at"`

	createdAtTime time.Time
	expiresAtTime time.Time
}

// ApprovalListResponse is returned by GET /v1/retailer/approvals.
type ApprovalListResponse struct {
	RetailerID string               `json:"retailer_id"`
	Items      []ApprovalRequestDTO `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

// ApprovalDecisionResponse is returned by POST .../approve and .../reject.
type ApprovalDecisionResponse struct {
	RequestID       string                   `json:"request_id"`
	Status          string                   `json:"status"`
	DecisionReason  string                   `json:"decision_reason,omitempty"`
	Approval        ApprovalRequestDTO       `json:"approval"`
	OrderID         string                   `json:"order_id,omitempty"`
	ParentOrderID   string                   `json:"parent_order_id,omitempty"`
	Order           *CreateResponse          `json:"order,omitempty"`
	UnifiedCheckout *UnifiedCheckoutResponse `json:"unified_checkout,omitempty"`
}

type approvalUpsertParams struct {
	RetailerID      string
	SupplierID      string
	RequestedByUser string
	ClientOrderID   string
	AmountMinor     int64
	TrippedRule     string
	Envelope        ApprovalPayloadEnvelope
}

func canManageApprovals(claims auth.Claims) bool {
	if claims.Role != auth.RoleRetailer {
		return false
	}
	if auth.HasRetailerPerm(claims, auth.PermStaffManage) {
		return true
	}
	role := auth.EffectiveRetailerRole(claims)
	return role == "OWNER" || role == "ADMIN" || role == "MANAGER"
}

func encodeApprovalCursor(createdAt time.Time, requestID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(createdAt.UTC().Format(time.RFC3339Nano) + "|" + requestID))
}

func decodeApprovalCursor(cursor string) (time.Time, string, bool, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return time.Time{}, "", false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", false, ErrApprovalInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return time.Time{}, "", false, ErrApprovalInvalidCursor
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false, ErrApprovalInvalidCursor
	}
	return ts.UTC(), parts[1], true, nil
}

func normalizeApprovalLimit(limit int) int {
	if limit <= 0 {
		return defaultApprovalsLimit
	}
	if limit > maxApprovalsPageLimit {
		return maxApprovalsPageLimit
	}
	return limit
}

func (s *Service) lockApprovalRequest(requestID string) func() {
	s.usageMu.Lock()
	if s.approvalApproveMu == nil {
		s.approvalApproveMu = make(map[string]*sync.Mutex)
	}
	mu, ok := s.approvalApproveMu[requestID]
	if !ok {
		mu = &sync.Mutex{}
		s.approvalApproveMu[requestID] = mu
	}
	s.usageMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

func (s *Service) upsertApprovalRequest(ctx context.Context, p approvalUpsertParams) (ApprovalRequestDTO, error) {
	payloadBytes, err := json.Marshal(p.Envelope)
	if err != nil {
		return ApprovalRequestDTO{}, fmt.Errorf("marshal approval payload: %w", err)
	}
	payloadStr := string(payloadBytes)
	now := s.now().UTC()
	expiresAt := now.Add(defaultApprovalTTL)

	if s.spannerClient == nil {
		s.usageMu.Lock()
		defer s.usageMu.Unlock()
		if s.approvalRequests == nil {
			s.approvalRequests = make(map[string]ApprovalRequestDTO)
		}
		if s.approvalsByClientOrder == nil {
			s.approvalsByClientOrder = make(map[string]string)
		}
		ck := p.RetailerID + "|" + p.ClientOrderID
		if existingID, ok := s.approvalsByClientOrder[ck]; ok {
			if existing, ok2 := s.approvalRequests[existingID]; ok2 {
				if existing.Status == ApprovalStatusPendingOwner {
					envCopy := p.Envelope
					existing.SupplierID = p.SupplierID
					existing.AmountMinor = p.AmountMinor
					existing.TrippedRule = p.TrippedRule
					existing.PayloadKind = p.Envelope.Kind
					existing.PaymentMethod = p.Envelope.PaymentMethod
					existing.OrderPayload = &envCopy
					existing.OrderPayloadRaw = payloadStr
					existing.ExpiresAt = expiresAt.Format(time.RFC3339Nano)
					existing.expiresAtTime = expiresAt
					existing.UpdatedAt = now.Format(time.RFC3339Nano)
					s.approvalRequests[existingID] = existing
				}
				return existing, nil
			}
		}
		reqID := uuid.NewString()
		envCopy := p.Envelope
		dto := ApprovalRequestDTO{
			RequestID:       reqID,
			RetailerID:      p.RetailerID,
			SupplierID:      p.SupplierID,
			RequestedByUser: p.RequestedByUser,
			ClientOrderID:   p.ClientOrderID,
			AmountMinor:     p.AmountMinor,
			TrippedRule:     p.TrippedRule,
			Status:          ApprovalStatusPendingOwner,
			PayloadKind:     p.Envelope.Kind,
			PaymentMethod:   p.Envelope.PaymentMethod,
			OrderPayload:    &envCopy,
			OrderPayloadRaw: payloadStr,
			ExpiresAt:       expiresAt.Format(time.RFC3339Nano),
			CreatedAt:       now.Format(time.RFC3339Nano),
			UpdatedAt:       now.Format(time.RFC3339Nano),
			createdAtTime:   now,
			expiresAtTime:   expiresAt,
		}
		s.approvalRequests[reqID] = dto
		s.approvalsByClientOrder[ck] = reqID
		return dto, nil
	}

	var out ApprovalRequestDTO
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT RequestId, RetailerId, SupplierId, RequestedByUser, ClientOrderId,
				AmountMinor, TrippedRule, Status, OrderPayloadJson, ApprovedByUser,
				DecisionReason, ExpiresAt, CreatedAt, UpdatedAt
				FROM RetailerApprovalRequests@{FORCE_INDEX=UQ_RetailerApprovals_ClientOrder}
				WHERE RetailerId = @rid AND ClientOrderId = @cid
				LIMIT 1`,
			Params: map[string]any{"rid": p.RetailerID, "cid": p.ClientOrderID},
		}
		iter := txn.Query(txCtx, stmt)
		row, iterErr := iter.Next()
		iter.Stop()
		if iterErr != nil && iterErr != iterator.Done {
			return iterErr
		}
		if iterErr == nil {
			existing, decErr := decodeApprovalRequestRow(row)
			if decErr != nil {
				return decErr
			}
			if existing.Status == ApprovalStatusPendingOwner {
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.UpdateMap("RetailerApprovalRequests", map[string]any{
						"RequestId":        existing.RequestID,
						"SupplierId":       nullableString(p.SupplierID),
						"AmountMinor":      p.AmountMinor,
						"TrippedRule":      p.TrippedRule,
						"OrderPayloadJson": payloadStr,
						"ExpiresAt":        expiresAt,
						"UpdatedAt":        spanner.CommitTimestamp,
					}),
				}); err != nil {
					return err
				}
				envCopy := p.Envelope
				existing.SupplierID = p.SupplierID
				existing.AmountMinor = p.AmountMinor
				existing.TrippedRule = p.TrippedRule
				existing.PayloadKind = p.Envelope.Kind
				existing.PaymentMethod = p.Envelope.PaymentMethod
				existing.OrderPayload = &envCopy
				existing.OrderPayloadRaw = payloadStr
				existing.ExpiresAt = expiresAt.Format(time.RFC3339Nano)
				existing.expiresAtTime = expiresAt
			}
			out = existing
			return nil
		}

		reqID := uuid.NewString()
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("RetailerApprovalRequests", map[string]any{
				"RequestId":        reqID,
				"RetailerId":       p.RetailerID,
				"SupplierId":       nullableString(p.SupplierID),
				"RequestedByUser":  p.RequestedByUser,
				"ClientOrderId":    p.ClientOrderID,
				"AmountMinor":      p.AmountMinor,
				"TrippedRule":      p.TrippedRule,
				"Status":           ApprovalStatusPendingOwner,
				"OrderPayloadJson": payloadStr,
				"ExpiresAt":        expiresAt,
				"CreatedAt":        spanner.CommitTimestamp,
				"UpdatedAt":        spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		envCopy := p.Envelope
		out = ApprovalRequestDTO{
			RequestID:       reqID,
			RetailerID:      p.RetailerID,
			SupplierID:      p.SupplierID,
			RequestedByUser: p.RequestedByUser,
			ClientOrderID:   p.ClientOrderID,
			AmountMinor:     p.AmountMinor,
			TrippedRule:     p.TrippedRule,
			Status:          ApprovalStatusPendingOwner,
			PayloadKind:     p.Envelope.Kind,
			PaymentMethod:   p.Envelope.PaymentMethod,
			OrderPayload:    &envCopy,
			OrderPayloadRaw: payloadStr,
			ExpiresAt:       expiresAt.Format(time.RFC3339Nano),
			CreatedAt:       now.Format(time.RFC3339Nano),
			UpdatedAt:       now.Format(time.RFC3339Nano),
			createdAtTime:   now,
			expiresAtTime:   expiresAt,
		}
		return nil
	})
	if err != nil {
		return ApprovalRequestDTO{}, err
	}
	return out, nil
}

func decodeApprovalRequestRow(row *spanner.Row) (ApprovalRequestDTO, error) {
	var dto ApprovalRequestDTO
	var supplierID, approvedBy, decisionReason spanner.NullString
	var payloadJSON string
	var expiresAt, createdAt, updatedAt time.Time
	if err := row.Columns(
		&dto.RequestID,
		&dto.RetailerID,
		&supplierID,
		&dto.RequestedByUser,
		&dto.ClientOrderID,
		&dto.AmountMinor,
		&dto.TrippedRule,
		&dto.Status,
		&payloadJSON,
		&approvedBy,
		&decisionReason,
		&expiresAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return ApprovalRequestDTO{}, err
	}
	if supplierID.Valid {
		dto.SupplierID = supplierID.StringVal
	}
	if approvedBy.Valid {
		dto.ApprovedByUser = approvedBy.StringVal
	}
	if decisionReason.Valid {
		dto.DecisionReason = decisionReason.StringVal
	}
	dto.OrderPayloadRaw = payloadJSON
	var env ApprovalPayloadEnvelope
	if err := json.Unmarshal([]byte(payloadJSON), &env); err == nil {
		dto.PayloadKind = env.Kind
		dto.PaymentMethod = env.PaymentMethod
		dto.OrderPayload = &env
	}
	dto.createdAtTime = createdAt.UTC()
	dto.expiresAtTime = expiresAt.UTC()
	dto.ExpiresAt = expiresAt.UTC().Format(time.RFC3339Nano)
	dto.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	dto.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	return dto, nil
}

func (s *Service) getApprovalRequest(ctx context.Context, requestID string) (ApprovalRequestDTO, bool, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ApprovalRequestDTO{}, false, nil
	}
	if s.spannerClient == nil {
		s.usageMu.Lock()
		defer s.usageMu.Unlock()
		if s.approvalRequests == nil {
			return ApprovalRequestDTO{}, false, nil
		}
		rec, ok := s.approvalRequests[requestID]
		return rec, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerApprovalRequests", spanner.Key{requestID}, []string{
		"RequestId", "RetailerId", "SupplierId", "RequestedByUser", "ClientOrderId",
		"AmountMinor", "TrippedRule", "Status", "OrderPayloadJson", "ApprovedByUser",
		"DecisionReason", "ExpiresAt", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ApprovalRequestDTO{}, false, nil
		}
		return ApprovalRequestDTO{}, false, err
	}
	dto, err := decodeApprovalRequestRow(row)
	if err != nil {
		return ApprovalRequestDTO{}, false, err
	}
	return dto, true, nil
}

func (s *Service) saveApprovalDecision(ctx context.Context, dto ApprovalRequestDTO) error {
	if s.spannerClient == nil {
		s.usageMu.Lock()
		defer s.usageMu.Unlock()
		if s.approvalRequests == nil {
			s.approvalRequests = make(map[string]ApprovalRequestDTO)
		}
		dto.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
		s.approvalRequests[dto.RequestID] = dto
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("RetailerApprovalRequests", map[string]any{
				"RequestId":      dto.RequestID,
				"Status":         dto.Status,
				"ApprovedByUser": nullableString(dto.ApprovedByUser),
				"DecisionReason": nullableString(dto.DecisionReason),
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		auditBuf := &spannerTxnBuffer{}
		action := "ORDER_APPROVAL_" + dto.Status
		if err := outbox.WriteAudit(txCtx, auditBuf, dto.RetailerID, dto.ApprovedByUser, "OWNER", action, "ApprovalRequest", dto.RequestID, map[string]any{
			"client_order_id": dto.ClientOrderID,
			"amount_minor":    dto.AmountMinor,
			"tripped_rule":    dto.TrippedRule,
			"reason":          dto.DecisionReason,
		}); err != nil {
			return err
		}
		return writeAuditEntriesInTxn(txn, auditBuf.audits)
	})
	return err
}

// ListApprovalsPage lists approval requests newest-first using opaque keyset cursors.
func (s *Service) ListApprovalsPage(ctx context.Context, retailerID, requestedByFilter, status string, limit int, cursor string) ([]ApprovalRequestDTO, string, error) {
	retailerID = strings.TrimSpace(retailerID)
	requestedByFilter = strings.TrimSpace(requestedByFilter)
	status = strings.ToUpper(strings.TrimSpace(status))
	limit = normalizeApprovalLimit(limit)

	cursorTs, cursorID, hasCursor, err := decodeApprovalCursor(cursor)
	if err != nil {
		return nil, "", err
	}

	if s.spannerClient == nil {
		s.usageMu.Lock()
		var all []ApprovalRequestDTO
		for _, rec := range s.approvalRequests {
			if rec.RetailerID != retailerID {
				continue
			}
			if requestedByFilter != "" && rec.RequestedByUser != requestedByFilter {
				continue
			}
			if status != "" && rec.Status != status {
				continue
			}
			all = append(all, rec)
		}
		s.usageMu.Unlock()

		sort.Slice(all, func(i, j int) bool {
			if !all[i].createdAtTime.Equal(all[j].createdAtTime) {
				return all[i].createdAtTime.After(all[j].createdAtTime)
			}
			return all[i].RequestID > all[j].RequestID
		})
		filtered := all[:0]
		for _, rec := range all {
			if hasCursor {
				if rec.createdAtTime.After(cursorTs) {
					continue
				}
				if rec.createdAtTime.Equal(cursorTs) && rec.RequestID >= cursorID {
					continue
				}
			}
			filtered = append(filtered, rec)
		}
		if len(filtered) > limit {
			page := filtered[:limit]
			last := page[len(page)-1]
			return page, encodeApprovalCursor(last.createdAtTime, last.RequestID), nil
		}
		if filtered == nil {
			return []ApprovalRequestDTO{}, "", nil
		}
		return filtered, "", nil
	}

	tableExpr := "RetailerApprovalRequests"
	if status != "" {
		tableExpr = "RetailerApprovalRequests@{FORCE_INDEX=Idx_RetailerApprovals_ByRetailerStatus}"
	}
	sql := fmt.Sprintf(`SELECT RequestId, RetailerId, SupplierId, RequestedByUser, ClientOrderId,
		AmountMinor, TrippedRule, Status, OrderPayloadJson, ApprovedByUser,
		DecisionReason, ExpiresAt, CreatedAt, UpdatedAt
		FROM %s WHERE RetailerId = @rid`, tableExpr)
	params := map[string]any{
		"rid": retailerID,
		"lim": int64(limit + 1),
	}
	if status != "" {
		sql += ` AND Status = @st`
		params["st"] = status
	}
	if requestedByFilter != "" {
		sql += ` AND RequestedByUser = @uid`
		params["uid"] = requestedByFilter
	}
	if hasCursor {
		sql += ` AND (CreatedAt < @cursorTs OR (CreatedAt = @cursorTs AND RequestId < @cursorId))`
		params["cursorTs"] = cursorTs
		params["cursorId"] = cursorID
	}
	sql += ` ORDER BY CreatedAt DESC, RequestId DESC LIMIT @lim`

	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var rows []ApprovalRequestDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("query approval requests: %w", err)
		}
		dto, err := decodeApprovalRequestRow(row)
		if err != nil {
			return nil, "", err
		}
		rows = append(rows, dto)
	}
	s.enrichApprovalRequesterNames(ctx, rows)

	if len(rows) > limit {
		page := rows[:limit]
		last := page[len(page)-1]
		return page, encodeApprovalCursor(last.createdAtTime, last.RequestID), nil
	}
	if rows == nil {
		return []ApprovalRequestDTO{}, "", nil
	}
	return rows, "", nil
}

func (s *Service) enrichApprovalRequesterNames(ctx context.Context, items []ApprovalRequestDTO) {
	if s.spannerClient == nil || len(items) == 0 {
		return
	}
	userIDs := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if it.RequestedByUser == "" {
			continue
		}
		if _, ok := seen[it.RequestedByUser]; !ok {
			seen[it.RequestedByUser] = struct{}{}
			userIDs = append(userIDs, it.RequestedByUser)
		}
	}
	if len(userIDs) == 0 {
		return
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT UserId, IFNULL(Name, '') FROM RetailerUsers WHERE UserId IN UNNEST(@ids)`,
		Params: map[string]any{"ids": userIDs},
	})
	defer iter.Stop()
	names := make(map[string]string, len(userIDs))
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var uid, name string
		if colErr := row.Columns(&uid, &name); colErr == nil && strings.TrimSpace(name) != "" {
			names[uid] = strings.TrimSpace(name)
		}
	}
	for i := range items {
		if n, ok := names[items[i].RequestedByUser]; ok {
			items[i].RequestedByName = n
		}
	}
}

// HandleListApprovals serves GET /v1/retailer/approvals.
func (s *Service) HandleListApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	retailerID := auth.ResolveRetailerOrgID(claims)
	if retailerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	limit := defaultApprovalsLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))

	requestedByFilter := ""
	if !canManageApprovals(claims) {
		requestedByFilter = auth.ResolveRetailerUserID(claims)
	}

	items, nextCursor, err := s.ListApprovalsPage(r.Context(), retailerID, requestedByFilter, status, limit, cursor)
	if err != nil {
		if errors.Is(err, ErrApprovalInvalidCursor) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_approvals_failed"})
		return
	}
	writeJSON(w, http.StatusOK, ApprovalListResponse{
		RetailerID: retailerID,
		Items:      items,
		NextCursor: nextCursor,
	})
}

type approvalDecisionBody struct {
	Reason string `json:"reason"`
}

// HandleApproveRequest serves POST /v1/retailer/approvals/{requestID}/approve.
func (s *Service) HandleApproveRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	retailerID := auth.ResolveRetailerOrgID(claims)
	if retailerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !canManageApprovals(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "owner_or_manager_required"})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "requestID"))
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}

	var req approvalDecisionBody
	if r.Body != nil {
		if err := json.NewDecoder(io.LimitReader(r.Body, 16*1024)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}
	reason := strings.TrimSpace(req.Reason)
	approverID := auth.ResolveRetailerUserID(claims)
	if approverID == "" {
		approverID = retailerID
	}

	resp, err := s.ApproveRequest(r.Context(), claims, retailerID, approverID, requestID, reason)
	if err != nil {
		writeApprovalDecisionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleRejectRequest serves POST /v1/retailer/approvals/{requestID}/reject.
func (s *Service) HandleRejectRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	retailerID := auth.ResolveRetailerOrgID(claims)
	if retailerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !canManageApprovals(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "owner_or_manager_required"})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "requestID"))
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}

	var req approvalDecisionBody
	if r.Body != nil {
		if err := json.NewDecoder(io.LimitReader(r.Body, 16*1024)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}
	reason := strings.TrimSpace(req.Reason)
	approverID := auth.ResolveRetailerUserID(claims)
	if approverID == "" {
		approverID = retailerID
	}

	resp, err := s.RejectRequest(r.Context(), retailerID, approverID, requestID, reason)
	if err != nil {
		writeApprovalDecisionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeApprovalDecisionError(w http.ResponseWriter, err error) {
	var ice *InventoryCheckoutError
	switch {
	case errors.Is(err, ErrApprovalNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "approval_request_not_found"})
	case errors.Is(err, ErrApprovalForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, ErrApprovalInvalidState):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrInventoryExhausted):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "inventory_exhausted", "code": "inventory_exhausted"})
	case errors.As(err, &ice):
		writeJSON(w, http.StatusConflict, ice)
	case errors.Is(err, ErrCreditLimitBreached), errors.Is(err, credit.ErrLimitBreached):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":           "credit_limit_breached",
			"code":            "CREDIT_LIMIT_BREACHED",
			"action_guidance": "Wire bank payment or settle outstanding overdue invoices to restore credit headroom.",
		})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
}

// ApproveRequest validates ownership, atomically claims APPROVING state, marks APPROVED, and replays OrderPayloadJson idempotently.
func (s *Service) ApproveRequest(ctx context.Context, claims auth.Claims, retailerID, approverID, requestID, reason string) (ApprovalDecisionResponse, error) {
	unlock := s.lockApprovalRequest(requestID)
	defer unlock()

	rec, claimedForExec, err := s.claimApprovalForExecution(ctx, retailerID, approverID, requestID)
	if err != nil {
		return ApprovalDecisionResponse{}, err
	}

	if rec.Status == ApprovalStatusApproved && !claimedForExec {
		s.usageMu.Lock()
		cached, okCached := s.approvedReplayResults[requestID]
		s.usageMu.Unlock()
		if okCached {
			return cached, nil
		}
		if existingResp, okExisting, exErr := s.reconstructApprovedReplay(ctx, rec); exErr == nil && okExisting {
			return existingResp, nil
		}
	}

	var env ApprovalPayloadEnvelope
	if err := json.Unmarshal([]byte(rec.OrderPayloadRaw), &env); err != nil {
		if claimedForExec {
			_ = s.revertApprovalClaim(ctx, requestID)
		}
		return ApprovalDecisionResponse{}, fmt.Errorf("invalid stored order payload: %w", err)
	}
	usageDate := sellerUsageDate(s.now())
	if strings.TrimSpace(env.UsageDate) != "" {
		if parsed, perr := civil.ParseDate(env.UsageDate); perr == nil {
			usageDate = parsed
		}
	}

	replayClaims := claims
	replayClaims.Role = auth.RoleRetailer
	replayClaims.RetailerOrgID = retailerID
	replayClaims.RetailerUserID = env.RequestedByUser
	if replayClaims.RetailerUserID == "" {
		replayClaims.RetailerUserID = rec.RequestedByUser
	}
	replayCtx := auth.WithClaims(ctx, replayClaims)
	replayCtx = WithClientOrderID(replayCtx, rec.ClientOrderID)
	replayCtx = WithApprovedApprovalContext(replayCtx, ApprovedReplayContext{
		RequestID:       rec.RequestID,
		ClientOrderID:   rec.ClientOrderID,
		RequestedByUser: rec.RequestedByUser,
		ApprovedByUser:  approverID,
		UsageDate:       usageDate,
	})

	decResp := ApprovalDecisionResponse{
		RequestID:      rec.RequestID,
		Status:         ApprovalStatusApproved,
		DecisionReason: reason,
	}

	switch env.Kind {
	case ApprovalPayloadCreate:
		if env.CreateRequest == nil {
			if claimedForExec {
				_ = s.revertApprovalClaim(ctx, requestID)
			}
			return ApprovalDecisionResponse{}, errors.New("approval payload missing create_request")
		}
		reqCopy := *env.CreateRequest
		reqCopy.ClientOrderID = rec.ClientOrderID
		if strings.TrimSpace(reqCopy.PaymentMethod) == "" {
			reqCopy.PaymentMethod = env.PaymentMethod
		}
		if strings.TrimSpace(reqCopy.SupplierID) == "" && strings.TrimSpace(env.SupplierID) != "" {
			reqCopy.SupplierID = env.SupplierID
		}
		if strings.TrimSpace(reqCopy.Currency) == "" && strings.TrimSpace(env.Currency) != "" {
			reqCopy.Currency = env.Currency
		}
		created, cerr := s.Create(replayCtx, retailerID, reqCopy)
		if cerr != nil {
			if claimedForExec {
				_ = s.revertApprovalClaim(ctx, requestID)
			}
			return ApprovalDecisionResponse{}, cerr
		}
		decResp.OrderID = created.OrderID
		decResp.Order = &created

	case ApprovalPayloadUnifiedCheckout:
		if env.UnifiedRequest == nil {
			if claimedForExec {
				_ = s.revertApprovalClaim(ctx, requestID)
			}
			return ApprovalDecisionResponse{}, errors.New("approval payload missing unified_request")
		}
		reqCopy := *env.UnifiedRequest
		reqCopy.RetailerID = retailerID
		reqCopy.ClientOrderID = rec.ClientOrderID
		if strings.TrimSpace(reqCopy.PaymentMethod) == "" {
			reqCopy.PaymentMethod = env.PaymentMethod
		}
		if strings.TrimSpace(reqCopy.Currency) == "" && strings.TrimSpace(env.Currency) != "" {
			reqCopy.Currency = env.Currency
		}
		ucResp, uerr := s.UnifiedCheckout(replayCtx, retailerID, reqCopy)
		if uerr != nil {
			if claimedForExec {
				_ = s.revertApprovalClaim(ctx, requestID)
			}
			return ApprovalDecisionResponse{}, uerr
		}
		decResp.ParentOrderID = ucResp.ParentOrderID
		if len(ucResp.SupplierOrders) == 1 {
			decResp.OrderID = ucResp.SupplierOrders[0].OrderID
		}
		decResp.UnifiedCheckout = &ucResp

	default:
		if claimedForExec {
			_ = s.revertApprovalClaim(ctx, requestID)
		}
		return ApprovalDecisionResponse{}, fmt.Errorf("unsupported approval payload kind %q", env.Kind)
	}

	if rec.Status != ApprovalStatusApproved {
		rec.Status = ApprovalStatusApproved
		rec.ApprovedByUser = approverID
		rec.DecisionReason = reason
		if err := s.saveApprovalDecision(ctx, rec); err != nil {
			return ApprovalDecisionResponse{}, fmt.Errorf("mark approval approved: %w", err)
		}
	}
	decResp.Approval = rec

	s.usageMu.Lock()
	if s.approvedReplayResults == nil {
		s.approvedReplayResults = make(map[string]ApprovalDecisionResponse)
	}
	if len(s.approvedReplayResults) >= maxApprovedReplayCache {
		s.approvedReplayResults = make(map[string]ApprovalDecisionResponse)
	}
	s.approvedReplayResults[requestID] = decResp
	s.usageMu.Unlock()

	return decResp, nil
}

func (s *Service) claimApprovalForExecution(ctx context.Context, retailerID, approverID, requestID string) (ApprovalRequestDTO, bool, error) {
	if s.spannerClient == nil {
		rec, found, err := s.getApprovalRequest(ctx, requestID)
		if err != nil {
			return ApprovalRequestDTO{}, false, fmt.Errorf("load approval request: %w", err)
		}
		if !found || rec.RetailerID != retailerID {
			return ApprovalRequestDTO{}, false, ErrApprovalNotFound
		}
		switch rec.Status {
		case ApprovalStatusRejected:
			return ApprovalRequestDTO{}, false, fmt.Errorf("%w: already rejected", ErrApprovalInvalidState)
		case ApprovalStatusExpired:
			return ApprovalRequestDTO{}, false, fmt.Errorf("%w: expired", ErrApprovalInvalidState)
		case ApprovalStatusApproved:
			return rec, false, nil
		case ApprovalStatusPendingOwner:
			if !rec.expiresAtTime.IsZero() && s.now().UTC().After(rec.expiresAtTime) {
				rec.Status = ApprovalStatusExpired
				_ = s.saveApprovalDecision(ctx, rec)
				return ApprovalRequestDTO{}, false, fmt.Errorf("%w: expired", ErrApprovalInvalidState)
			}
			rec.Status = ApprovalStatusApproving
			_ = s.saveApprovalDecision(ctx, rec)
			return rec, true, nil
		case ApprovalStatusApproving:
			return ApprovalRequestDTO{}, false, fmt.Errorf("%w: approval already in progress", ErrApprovalInvalidState)
		default:
			return ApprovalRequestDTO{}, false, fmt.Errorf("%w: status=%s", ErrApprovalInvalidState, rec.Status)
		}
	}

	var rec ApprovalRequestDTO
	var claimed bool
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rerr := txn.ReadRow(txCtx, "RetailerApprovalRequests", spanner.Key{requestID}, []string{
			"RequestId", "RetailerId", "SupplierId", "RequestedByUser", "ClientOrderId",
			"AmountMinor", "TrippedRule", "Status", "OrderPayloadJson", "ApprovedByUser",
			"DecisionReason", "ExpiresAt", "CreatedAt", "UpdatedAt",
		})
		if rerr != nil {
			if spanner.ErrCode(rerr) == codes.NotFound {
				return ErrApprovalNotFound
			}
			return fmt.Errorf("load approval request: %w", rerr)
		}
		dto, decErr := decodeApprovalRequestRow(row)
		if decErr != nil {
			return decErr
		}
		if dto.RetailerID != retailerID {
			return ErrApprovalNotFound
		}
		switch dto.Status {
		case ApprovalStatusRejected:
			return fmt.Errorf("%w: already rejected", ErrApprovalInvalidState)
		case ApprovalStatusExpired:
			return fmt.Errorf("%w: expired", ErrApprovalInvalidState)
		case ApprovalStatusApproved:
			rec = dto
			claimed = false
			return nil
		case ApprovalStatusApproving:
			updAt, _ := time.Parse(time.RFC3339Nano, dto.UpdatedAt)
			if !updAt.IsZero() && s.now().UTC().Sub(updAt) < 60*time.Second {
				return fmt.Errorf("%w: approval already in progress", ErrApprovalInvalidState)
			}
			// Stale APPROVING takeover after pod crash > 60s.
		case ApprovalStatusPendingOwner:
			if !dto.expiresAtTime.IsZero() && s.now().UTC().After(dto.expiresAtTime) {
				return txn.BufferWrite([]*spanner.Mutation{
					spanner.UpdateMap("RetailerApprovalRequests", map[string]any{
						"RequestId": requestID,
						"Status":    ApprovalStatusExpired,
						"UpdatedAt": spanner.CommitTimestamp,
					}),
				})
			}
		default:
			return fmt.Errorf("%w: status=%s", ErrApprovalInvalidState, dto.Status)
		}
		dto.Status = ApprovalStatusApproving
		rec = dto
		claimed = true
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("RetailerApprovalRequests", map[string]any{
				"RequestId":      requestID,
				"Status":         ApprovalStatusApproving,
				"ApprovedByUser": nullableString(approverID),
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		return ApprovalRequestDTO{}, false, err
	}
	if !claimed && rec.Status == "" {
		return ApprovalRequestDTO{}, false, fmt.Errorf("%w: expired", ErrApprovalInvalidState)
	}
	return rec, claimed, nil
}

func (s *Service) revertApprovalClaim(ctx context.Context, requestID string) error {
	if s.spannerClient == nil {
		s.usageMu.Lock()
		if rec, ok := s.approvalRequests[requestID]; ok && rec.Status == ApprovalStatusApproving {
			rec.Status = ApprovalStatusPendingOwner
			s.approvalRequests[requestID] = rec
		}
		s.usageMu.Unlock()
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rerr := txn.ReadRow(txCtx, "RetailerApprovalRequests", spanner.Key{requestID}, []string{"Status"})
		if rerr != nil {
			return rerr
		}
		var st string
		if err := row.Columns(&st); err != nil {
			return err
		}
		if st != ApprovalStatusApproving {
			return nil
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("RetailerApprovalRequests", map[string]any{
				"RequestId": requestID,
				"Status":    ApprovalStatusPendingOwner,
				"UpdatedAt": spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

func (s *Service) reconstructApprovedReplay(ctx context.Context, rec ApprovalRequestDTO) (ApprovalDecisionResponse, bool, error) {
	if rec.OrderPayload == nil {
		return ApprovalDecisionResponse{}, false, nil
	}
	env := rec.OrderPayload
	if env.Kind == ApprovalPayloadCreate {
		o, found, err := s.findExistingOrderByIdempotencyKey(ctx, rec.RetailerID, env.SupplierID, rec.ClientOrderID)
		if err != nil || !found {
			return ApprovalDecisionResponse{}, false, err
		}
		cr := CreateResponse{
			OrderID:            o.OrderID,
			WarehouseID:        o.WarehouseID,
			Status:             o.Status,
			Source:             o.Source,
			ConfirmationStatus: o.ConfirmationStatus,
			AutoConfirmOptIn:   o.AutoConfirmOptIn,
			HoldReason:         o.HoldReason,
			TotalMinor:         o.TotalMinor,
			Currency:           o.Currency,
			CreatedAt:          o.CreatedAt.UTC().Format(time.RFC3339Nano),
			LineItems:          o.LineItems,
		}
		return ApprovalDecisionResponse{
			RequestID:      rec.RequestID,
			Status:         rec.Status,
			DecisionReason: rec.DecisionReason,
			Approval:       rec,
			OrderID:        o.OrderID,
			Order:          &cr,
		}, true, nil
	}
	if env.Kind == ApprovalPayloadUnifiedCheckout && env.UnifiedRequest != nil {
		lines := make([]LineItem, 0, len(env.UnifiedRequest.Items))
		for _, it := range env.UnifiedRequest.Items {
			lines = append(lines, LineItem{
				SKU:       it.SkuID,
				Quantity:  it.Quantity,
				UnitPrice: it.UnitPrice,
			})
		}
		groups, err := s.groupCheckoutLinesBySupplier(ctx, env.UnifiedRequest.Items, lines)
		if err != nil || len(groups) == 0 {
			return ApprovalDecisionResponse{}, false, err
		}
		var parentID, currency string
		var total int64
		supOrders := make([]SupplierOrderResult, 0, len(groups))
		for _, g := range groups {
			key := legClientOrderIDWithRetry(rec.ClientOrderID, g.SupplierID, 0)
			o, found, err := s.findExistingOrderByIdempotencyKey(ctx, rec.RetailerID, g.SupplierID, key)
			if err != nil || !found {
				return ApprovalDecisionResponse{}, false, err
			}
			if parentID == "" {
				parentID = o.ParentOrderID
			}
			if currency == "" {
				currency = o.Currency
			}
			total += o.TotalMinor
			supOrders = append(supOrders, SupplierOrderResult{
				OrderID:    o.OrderID,
				SupplierID: g.SupplierID,
				Total:      o.TotalMinor,
				Currency:   o.Currency,
				ItemCount:  g.ItemCount,
			})
		}
		ucResp := UnifiedCheckoutResponse{
			Status:         "ok",
			Total:          total,
			Currency:       currency,
			ParentOrderID:  parentID,
			SupplierOrders: supOrders,
		}
		decResp := ApprovalDecisionResponse{
			RequestID:       rec.RequestID,
			Status:          rec.Status,
			DecisionReason:  rec.DecisionReason,
			Approval:        rec,
			ParentOrderID:   parentID,
			UnifiedCheckout: &ucResp,
		}
		if len(supOrders) == 1 {
			decResp.OrderID = supOrders[0].OrderID
		}
		return decResp, true, nil
	}
	return ApprovalDecisionResponse{}, false, nil
}

// RejectRequest transitions a PENDING_OWNER approval request to REJECTED cleanly.
func (s *Service) RejectRequest(ctx context.Context, retailerID, approverID, requestID, reason string) (ApprovalDecisionResponse, error) {
	unlock := s.lockApprovalRequest(requestID)
	defer unlock()

	rec, found, err := s.getApprovalRequest(ctx, requestID)
	if err != nil {
		return ApprovalDecisionResponse{}, fmt.Errorf("load approval request: %w", err)
	}
	if !found || rec.RetailerID != retailerID {
		return ApprovalDecisionResponse{}, ErrApprovalNotFound
	}
	if rec.Status == ApprovalStatusRejected {
		return ApprovalDecisionResponse{
			RequestID:      rec.RequestID,
			Status:         rec.Status,
			DecisionReason: rec.DecisionReason,
			Approval:       rec,
		}, nil
	}
	if rec.Status != ApprovalStatusPendingOwner {
		return ApprovalDecisionResponse{}, fmt.Errorf("%w: status=%s", ErrApprovalInvalidState, rec.Status)
	}

	rec.Status = ApprovalStatusRejected
	rec.ApprovedByUser = approverID
	rec.DecisionReason = reason
	if err := s.saveApprovalDecision(ctx, rec); err != nil {
		return ApprovalDecisionResponse{}, fmt.Errorf("save rejection: %w", err)
	}
	return ApprovalDecisionResponse{
		RequestID:      rec.RequestID,
		Status:         rec.Status,
		DecisionReason: rec.DecisionReason,
		Approval:       rec,
	}, nil
}
