// chargeback/reversal, and gateway webhooks.
package payment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func (s *Service) HandleClaimChargebacks(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/supplier/claim-chargebacks"
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	if s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "ledger_repository_unavailable", "Ledger repository is unavailable", endpoint, false, "")
		return
	}
	supplierID, ok := resolvePaymentSupplierScope(w, r, s.seedSupplierID, endpoint)
	if !ok {
		return
	}
	limit, err := parseBoundedIntQuery(strings.TrimSpace(r.URL.Query().Get("limit")), 50, 1, 200)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 200", endpoint, false, "")
		return
	}
	// Pull a wider page then filter to claim-originated chargebacks.
	items, err := s.repo.ListLedgerEntries(r.Context(), LedgerQuery{
		SupplierID: supplierID,
		OrderID:    strings.TrimSpace(r.URL.Query().Get("order_id")),
		EntryType:  "CHARGEBACK_RECORDED",
		Limit:      limit * 3,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "ledger_query_failed", err.Error(), endpoint, false, "")
		return
	}
	out := make([]LedgerEntryRecord, 0, len(items))
	for _, it := range items {
		src := strings.ToLower(it.Source)
		ref := strings.ToLower(it.ReferenceID)
		if strings.Contains(src, "claims.settle") ||
			strings.HasPrefix(ref, "chargeback_clm_") ||
			strings.Contains(ref, "clm_") {
			out = append(out, it)
			if len(out) >= limit {
				break
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":       out,
		"count":       len(out),
		"limit":       limit,
		"supplier_id": supplierID,
	})
}

// HandleLedger serves GET /v1/payment/ledger.
func (s *Service) HandleLedger(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/payment/ledger"
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	if s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "ledger_repository_unavailable", "Ledger repository is unavailable", endpoint, false, "")
		return
	}

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.seedSupplierID, endpoint)
	if !ok {
		return
	}

	limit, err := parseBoundedIntQuery(strings.TrimSpace(r.URL.Query().Get("limit")), 100, 1, 500)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500", endpoint, false, "")
		return
	}

	occurredFrom, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_from")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_from", "occurred_from must be RFC3339", endpoint, false, "")
		return
	}
	occurredTo, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_to")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_to", "occurred_to must be RFC3339", endpoint, false, "")
		return
	}
	if occurredFrom != nil && occurredTo != nil && occurredFrom.After(*occurredTo) {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_range", "occurred_from must be before or equal to occurred_to", endpoint, false, "")
		return
	}

	items, err := s.repo.ListLedgerEntries(r.Context(), LedgerQuery{
		SupplierID:   supplierID,
		OrderID:      strings.TrimSpace(r.URL.Query().Get("order_id")),
		SessionID:    strings.TrimSpace(r.URL.Query().Get("session_id")),
		Gateway:      strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("gateway"))),
		EntryType:    strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("entry_type"))),
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
		Limit:        limit,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "ledger_query_failed", err.Error(), endpoint, false, "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":       items,
		"count":       len(items),
		"limit":       limit,
		"supplier_id": supplierID,
		"filters": map[string]any{
			"order_id":      strings.TrimSpace(r.URL.Query().Get("order_id")),
			"session_id":    strings.TrimSpace(r.URL.Query().Get("session_id")),
			"gateway":       strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("gateway"))),
			"entry_type":    strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("entry_type"))),
			"occurred_from": formatOptionalTime(occurredFrom),
			"occurred_to":   formatOptionalTime(occurredTo),
		},
	})
}

// HandleSettlementAuthority serves GET /v1/payment/settlement/authority.
func (s *Service) HandleSettlementAuthority(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/payment/settlement/authority"
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	if s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "settlement_repository_unavailable", "Settlement repository is unavailable", endpoint, false, "")
		return
	}

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.seedSupplierID, endpoint)
	if !ok {
		return
	}

	groupLimit, err := parseBoundedIntQuery(strings.TrimSpace(r.URL.Query().Get("group_limit")), 200, 1, 1000)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_group_limit", "group_limit must be between 1 and 1000", endpoint, false, "")
		return
	}

	occurredFrom, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_from")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_from", "occurred_from must be RFC3339", endpoint, false, "")
		return
	}
	occurredTo, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_to")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_to", "occurred_to must be RFC3339", endpoint, false, "")
		return
	}
	if occurredFrom != nil && occurredTo != nil && occurredFrom.After(*occurredTo) {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_range", "occurred_from must be before or equal to occurred_to", endpoint, false, "")
		return
	}

	gateway := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("gateway")))
	entryType := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("entry_type")))

	rows, err := s.repo.SummarizeLedgerEntries(r.Context(), SettlementAuthorityQuery{
		SupplierID:   supplierID,
		Gateway:      gateway,
		EntryType:    entryType,
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
		GroupLimit:   groupLimit,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "settlement_summary_failed", err.Error(), endpoint, false, "")
		return
	}

	totalsByCurrency := make(map[string]SettlementCurrencyTotal)
	var entryCountTotal int64
	for _, row := range rows {
		entryCountTotal += row.EntryCount
		totals := totalsByCurrency[row.Currency]
		totals.Currency = row.Currency
		totals.EntryCount += row.EntryCount
		totals.AmountMinorTotal += row.AmountMinorTotal
		totalsByCurrency[row.Currency] = totals
	}

	currencyTotals := make([]SettlementCurrencyTotal, 0, len(totalsByCurrency))
	for _, totals := range totalsByCurrency {
		currencyTotals = append(currencyTotals, totals)
	}
	sort.Slice(currencyTotals, func(i, j int) bool {
		return currencyTotals[i].Currency < currencyTotals[j].Currency
	})

	operating := fxrates.NormalizeCurrency(s.currency)
	if operating == "" {
		if cur, err := auth.CurrencyFromContext(r.Context(), supplierID); err == nil {
			operating = cur
		}
	}
	opTotal, opPartial := rollupOperatingCurrencyMinor(r.Context(), s.fx, operating, rows, s.now)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":                          rows,
		"count":                          len(rows),
		"group_limit":                    groupLimit,
		"supplier_id":                    supplierID,
		"entry_count_total":              entryCountTotal,
		"totals_by_currency":             currencyTotals,
		"operating_currency":             operating,
		"operating_currency_total_minor": opTotal,
		"operating_conversion_partial":   opPartial,
		"filters": map[string]any{
			"gateway":       gateway,
			"entry_type":    entryType,
			"occurred_from": formatOptionalTime(occurredFrom),
			"occurred_to":   formatOptionalTime(occurredTo),
		},
	})
}

// HandleReconciliationMismatches serves GET /v1/payment/reconciliation/mismatches.
func (s *Service) HandleReconciliationMismatches(w http.ResponseWriter, r *http.Request) {
	const endpoint = "/v1/payment/reconciliation/mismatches"
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", endpoint, false, "")
		return
	}
	if s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "reconciliation_repository_unavailable", "Reconciliation repository is unavailable", endpoint, false, "")
		return
	}

	supplierID, ok := resolvePaymentSupplierScope(w, r, s.seedSupplierID, endpoint)
	if !ok {
		return
	}

	groupLimit, err := parseBoundedIntQuery(strings.TrimSpace(r.URL.Query().Get("group_limit")), 200, 1, 1000)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_group_limit", "group_limit must be between 1 and 1000", endpoint, false, "")
		return
	}

	mismatchThresholdMinor, err := parseNonNegativeInt64Query(strings.TrimSpace(r.URL.Query().Get("mismatch_threshold_minor")), 0)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_mismatch_threshold_minor", "mismatch_threshold_minor must be a non-negative integer", endpoint, false, "")
		return
	}

	occurredFrom, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_from")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_from", "occurred_from must be RFC3339", endpoint, false, "")
		return
	}
	occurredTo, err := parseRFC3339QueryTime(strings.TrimSpace(r.URL.Query().Get("occurred_to")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_to", "occurred_to must be RFC3339", endpoint, false, "")
		return
	}
	if occurredFrom != nil && occurredTo != nil && occurredFrom.After(*occurredTo) {
		writeJSONError(w, http.StatusBadRequest, "invalid_occurred_range", "occurred_from must be before or equal to occurred_to", endpoint, false, "")
		return
	}

	gateway := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("gateway")))
	rows, err := s.repo.SummarizeLedgerEntries(r.Context(), SettlementAuthorityQuery{
		SupplierID:   supplierID,
		Gateway:      gateway,
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
		GroupLimit:   groupLimit,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "reconciliation_summary_failed", err.Error(), endpoint, false, "")
		return
	}

	type gatewayCurrencyAggregate struct {
		Gateway         string
		Currency        string
		EntryCountTotal int64
		GroupCount      int64
		CreditAmount    int64
		DebitAmount     int64
		NetAmount       int64
		FirstOccurredAt time.Time
		LastOccurredAt  time.Time
		EntryTypeTotals []ReconciliationEntryTypeTotal
	}

	aggByKey := make(map[string]*gatewayCurrencyAggregate)
	for _, row := range rows {
		key := strings.Join([]string{row.Gateway, row.Currency}, "|")
		agg, exists := aggByKey[key]
		if !exists {
			agg = &gatewayCurrencyAggregate{
				Gateway:         row.Gateway,
				Currency:        row.Currency,
				FirstOccurredAt: row.FirstOccurredAt,
				LastOccurredAt:  row.LastOccurredAt,
			}
			aggByKey[key] = agg
		}

		sign := reconciliationEntrySign(row.EntryType)
		signedAmount := row.AmountMinorTotal * sign
		if signedAmount > 0 {
			agg.CreditAmount += signedAmount
		} else if signedAmount < 0 {
			agg.DebitAmount += -signedAmount
		}

		agg.EntryCountTotal += row.EntryCount
		agg.GroupCount++
		agg.NetAmount += signedAmount
		if row.FirstOccurredAt.Before(agg.FirstOccurredAt) {
			agg.FirstOccurredAt = row.FirstOccurredAt
		}
		if row.LastOccurredAt.After(agg.LastOccurredAt) {
			agg.LastOccurredAt = row.LastOccurredAt
		}
		agg.EntryTypeTotals = append(agg.EntryTypeTotals, ReconciliationEntryTypeTotal{
			EntryType:              row.EntryType,
			EntryCount:             row.EntryCount,
			AmountMinorTotal:       row.AmountMinorTotal,
			SignedAmountMinorTotal: signedAmount,
		})
	}

	mismatches := make([]ReconciliationMismatchRow, 0, len(aggByKey))
	for _, agg := range aggByKey {
		if absInt64(agg.NetAmount) <= mismatchThresholdMinor {
			continue
		}

		sort.Slice(agg.EntryTypeTotals, func(i, j int) bool {
			left := absInt64(agg.EntryTypeTotals[i].SignedAmountMinorTotal)
			right := absInt64(agg.EntryTypeTotals[j].SignedAmountMinorTotal)
			if left == right {
				return agg.EntryTypeTotals[i].EntryType < agg.EntryTypeTotals[j].EntryType
			}
			return left > right
		})

		mismatches = append(mismatches, ReconciliationMismatchRow{
			Gateway:         agg.Gateway,
			Currency:        agg.Currency,
			EntryCountTotal: agg.EntryCountTotal,
			GroupCount:      agg.GroupCount,
			CreditAmount:    agg.CreditAmount,
			DebitAmount:     agg.DebitAmount,
			NetAmount:       agg.NetAmount,
			FirstOccurredAt: agg.FirstOccurredAt,
			LastOccurredAt:  agg.LastOccurredAt,
			EntryTypeTotals: agg.EntryTypeTotals,
		})
	}

	sort.Slice(mismatches, func(i, j int) bool {
		left := absInt64(mismatches[i].NetAmount)
		right := absInt64(mismatches[j].NetAmount)
		if left == right {
			if mismatches[i].Gateway == mismatches[j].Gateway {
				return mismatches[i].Currency < mismatches[j].Currency
			}
			return mismatches[i].Gateway < mismatches[j].Gateway
		}
		return left > right
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":                    mismatches,
		"count":                    len(mismatches),
		"analyzed_group_count":     len(rows),
		"group_limit":              groupLimit,
		"mismatch_threshold_minor": mismatchThresholdMinor,
		"supplier_id":              supplierID,
		"filters": map[string]any{
			"gateway":       gateway,
			"occurred_from": formatOptionalTime(occurredFrom),
			"occurred_to":   formatOptionalTime(occurredTo),
		},
	})
}

// HandleDeprecatedGlobalPayInitiate serves POST /v1/payment/global_pay/initiate.
func (s *Service) HandleDeprecatedGlobalPayInitiate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/payment/global_pay/initiate", true, "/v1/order/card-checkout")
		return
	}
	writeJSONError(w, http.StatusGone, "endpoint_deprecated", "Deprecated endpoint", "/v1/payment/global_pay/initiate", true, "/v1/order/card-checkout")
}

func (s *Service) isWebhookReplay(ctx context.Context, webhookKey, bodyHash string) (bool, error) {
	if s.idem == nil {
		return false, nil
	}
	_, hit, err := idempotency.Guard(ctx, s.idem, webhookKey, bodyHash)
	if err != nil {
		return false, err
	}
	return hit, nil
}

func (s *Service) handleCheckout(mode string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "/v1/checkout", false, "")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "Unable to read request body", "/v1/checkout", false, "")
		return
	}
	defer r.Body.Close()
	s.handleCheckoutWithBody(mode, w, r, body)
}

func (s *Service) handleCheckoutWithBody(mode string, w http.ResponseWriter, r *http.Request, body []byte) {
	path := "/v1/checkout/unified"
	if mode == "B2B" {
		path = "/v1/checkout/b2b"
	}
	writeJSONError(w, http.StatusGone, "payment_before_delivery_removed",
		"Pre-delivery payment checkout is disabled; pay at delivery after offload via /v1/order/card-checkout or /v1/order/cash-checkout",
		path, true, "/v1/order/card-checkout")
}

func (s *Service) persistWebhookWithOutbox(ctx context.Context, row WebhookRecord, source string, now time.Time) error {
	eventType := events.EventPaymentRequired
	if isPaymentSuccessStatus(row.Status) {
		eventType = events.EventPaymentCleared
	} else if isPaymentFailureStatus(row.Status) {
		eventType = events.EventPaymentFailed
	}
	if err := s.repo.SaveWebhook(ctx, row, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateSession, row.WebhookID, events.TopicMain, events.FinanceEvent{
			BaseEvent:     events.BaseEvent{Type: eventType, Timestamp: now.Format(time.RFC3339Nano)},
			SessionID:     row.SessionID,
			OrderID:       row.OrderID,
			SupplierID:    row.SupplierID,
			RetailerID:    row.RetailerID,
			Gateway:       row.Gateway,
			Status:        row.Status,
			AmountMinor:   row.AmountMinor,
			Currency:      row.Currency,
			TransactionID: row.TransactionID,
			Source:        source,
		})
	}); err != nil {
		if s.webhookInbox != nil {
			if qErr := s.webhookInbox.Enqueue(ctx, row, source); qErr == nil {
				return nil
			}
		}
		return err
	}

	keys := make([]string, 0, 2)
	if row.OrderID != "" {
		keys = append(keys, paymentOrderKey(row.OrderID))
	}
	if row.SessionID != "" {
		keys = append(keys, paymentSessionKey(row.SessionID))
	}
	if len(keys) > 0 {
		s.cache.Invalidate(ctx, keys...)
	}
	return nil
}

func (s *Service) writeExecutionError(w http.ResponseWriter, endpoint string, err error) {
	var policyErr *GatewayPolicyError
	if errors.As(err, &policyErr) {
		code := strings.TrimSpace(policyErr.Code)
		if code == "" {
			code = "payment_gateway_policy_violation"
		}
		status := http.StatusUnprocessableEntity
		if code == "no_live_keys" {
			status = http.StatusNotImplemented
		}
		writeJSONError(w, status, code, policyErr.Error(), endpoint, false, "")
		return
	}
	writeJSONError(w, http.StatusBadGateway, "payment_gateway_execution_failed", err.Error(), endpoint, false, "")
}

func (s *Service) writeWebhookReplayIfExists(w http.ResponseWriter, r *http.Request, endpoint, webhookKey, bodyHash string) bool {
	replay, hit, err := idempotency.Guard(r.Context(), s.idem, webhookKey, bodyHash)
	if errors.Is(err, idempotency.ErrConflict) {
		writeJSONError(w, http.StatusConflict, "idempotency_key_payload_mismatch", "idempotency key payload mismatch", endpoint, false, "")
		return true
	}
	if err != nil {
		s.log.Warn("webhook idempotency guard failed", "key", webhookKey, "err", err)
		return false
	}
	if !hit {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(replay.StatusCode)
	_, _ = w.Write(replay.Response)
	return true
}

func (s *Service) requireMoneyIdempotencyKey(w http.ResponseWriter, r *http.Request, endpoint string) bool {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		key = strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	}
	if key == "" {
		writeJSONError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key is required", endpoint, false, "")
		return false
	}
	return true
}

func (s *Service) handleIdempotentHit(w http.ResponseWriter, r *http.Request, body []byte) (idempotency.Record, bool) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || s.idem == nil {
		return idempotency.Record{}, false
	}
	rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, sha256Hex(body))
	if errors.Is(err, idempotency.ErrConflict) {
		writeJSONError(w, http.StatusConflict, "idempotency_key_payload_mismatch", "idempotency key payload mismatch", r.URL.Path, false, "")
		return idempotency.Record{}, true
	}
	if err != nil {
		s.log.Warn("idempotency guard failed", "path", r.URL.Path, "err", err)
		return idempotency.Record{}, false
	}
	if !hit {
		return idempotency.Record{}, false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rec.StatusCode)
	_, _ = w.Write(rec.Response)
	return rec, true
}

func (s *Service) persistIdempotencyRecord(ctx context.Context, key, bodyHash string, statusCode int, payload []byte, ttl time.Duration) {
	if strings.TrimSpace(key) == "" || s.idem == nil {
		return
	}
	if err := s.idem.Save(ctx, key, idempotency.Record{
		BodyHash:   bodyHash,
		StatusCode: statusCode,
		Response:   payload,
		StoredAt:   s.now(),
	}, ttl); err != nil {
		s.log.Warn("idempotency save failed", "key", key, "err", err)
	}
}

func resolvePaymentSupplierScope(w http.ResponseWriter, r *http.Request, fallbackSupplierID string, endpoint string) (string, bool) {
	supplierID := auth.PreferTenantSupplierID(r.Context(), fallbackSupplierID)

	requestedSupplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if requestedSupplierID != "" {
		if auth.IsPlatformAdmin(r.Context()) {
			return requestedSupplierID, true
		}
		if supplierID == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "tenant_required", endpoint, false, "")
			return "", false
		}
		if requestedSupplierID != supplierID {
			writeJSONError(w, http.StatusForbidden, "forbidden", "supplier scope mismatch", endpoint, false, "")
			return "", false
		}
	}

	return supplierID, true
}

// resolveSupplierID prefers request TenantContext over the bootstrap seed.
func (s *Service) resolveSupplierID(ctx context.Context) string {
	return auth.PreferTenantSupplierID(ctx, s.seedSupplierID)
}

// resolveWebhookSupplierID prefers payment-session tenant, then seed (webhooks lack JWT).
func (s *Service) resolveWebhookSupplierID(ctx context.Context, orderID string) string {
	if oid := strings.TrimSpace(orderID); oid != "" && s.repo != nil {
		if session, ok, err := s.repo.GetSessionByOrderID(ctx, oid); err == nil && ok {
			if sid := strings.TrimSpace(session.SupplierID); sid != "" {
				return sid
			}
		}
	}
	return strings.TrimSpace(s.seedSupplierID)
}

func parseBoundedIntQuery(raw string, defaultValue int, minValue int, maxValue int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < minValue || parsed > maxValue {
		return 0, fmt.Errorf("out of range")
	}
	return parsed, nil
}

func parseNonNegativeInt64Query(raw string, defaultValue int64) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("out of range")
	}
	return parsed, nil
}

func parseRFC3339QueryTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func formatOptionalTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.UTC().Format(time.RFC3339Nano)
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// SignedSettlementEntryAmount applies the canonical reconciliation sign rules
// to a grouped settlement authority amount so adjacent read models can stay in
// lockstep with finance mismatch math.
func SignedSettlementEntryAmount(entryType string, amountMinor int64) int64 {
	return amountMinor * reconciliationEntrySign(entryType)
}

func reconciliationEntrySign(entryType string) int64 {
	t := strings.ToUpper(strings.TrimSpace(entryType))
	if t == "" {
		return 0
	}
	if strings.Contains(t, "FAILED") || strings.Contains(t, "CANCEL") || strings.Contains(t, "VOID") || strings.Contains(t, "PENDING") || strings.Contains(t, "REQUIRED") || strings.Contains(t, "UNKNOWN") {
		return 0
	}
	if strings.Contains(t, "REVERSAL") {
		return 1
	}
	if strings.Contains(t, "CHARGEBACK") || strings.Contains(t, "REFUND") {
		return -1
	}
	if strings.Contains(t, "PAID") || strings.Contains(t, "CAPTURED") || strings.Contains(t, "CLEARED") || strings.Contains(t, "SETTLED") || strings.Contains(t, "SUCCESS") || strings.Contains(t, "SUCCEEDED") || strings.Contains(t, "COMPLETED") || strings.Contains(t, "AUTHORIZED") {
		return 1
	}
	return 0
}

func coalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isPaymentSuccessStatus(status string) bool {
	s := strings.ToUpper(strings.TrimSpace(status))
	return s == "PAID" || s == "CAPTURED" || s == "SETTLED" || s == "SUCCESS"
}

func isPaymentFailureStatus(status string) bool {
	s := strings.ToUpper(strings.TrimSpace(status))
	return s == "FAILED" || s == "DECLINED" || s == "CANCELLED" || s == "VOIDED"
}

func writeJSONError(w http.ResponseWriter, status int, code, message, endpoint string, deprecated bool, migrateTo string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	payload := map[string]any{
		"error":      message,
		"code":       code,
		"message":    message,
		"endpoint":   endpoint,
		"deprecated": deprecated,
	}
	if migrateTo != "" {
		payload["migrate_to"] = migrateTo
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func paymentOrderKey(orderID string) string {
	return "payment:order:" + orderID
}

func paymentSessionKey(sessionID string) string {
	return "payment:session:" + sessionID
}

func paymentRetailerKey(retailerID string) string {
	return "payment:retailer:" + retailerID
}

// assertSessionCurrency rejects requestCurrency when a known session currency differs (theatre #13).
func (s *Service) assertSessionCurrency(ctx context.Context, sessionID, orderID, requestCurrency string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	requestCurrency = strings.TrimSpace(requestCurrency)
	if requestCurrency == "" {
		return nil
	}
	var session SessionRecord
	var ok bool
	var err error
	if sid := strings.TrimSpace(sessionID); sid != "" {
		session, ok, err = s.repo.GetSession(ctx, sid)
	} else if oid := strings.TrimSpace(orderID); oid != "" {
		session, ok, err = s.repo.GetSessionByOrderID(ctx, oid)
	}
	if err != nil || !ok {
		return nil
	}
	return fxrates.AssertSameCurrency(session.Currency, requestCurrency)
}
