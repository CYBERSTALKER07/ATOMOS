package ar

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// PromiseBreachFunc is called when an active payment promise expires with positive remaining undisputed balance.
type PromiseBreachFunc func(ctx context.Context, supplierID, retailerID, promiseID string) error

// PromiseNotifyFunc is called after a promise transitions to FULFILLED or BREACHED.
type PromiseNotifyFunc func(ctx context.Context, promise PaymentPromise, outcome string) error

// PromiseBreachWorker transitions expired ACTIVE ArPaymentPromises rows to FULFILLED or BREACHED.
type PromiseBreachWorker struct {
	svc       *Service
	log       *slog.Logger
	onBreach  PromiseBreachFunc
	notify    PromiseNotifyFunc
	now       func() time.Time
	customNow bool
}

func NewPromiseBreachWorker(svc *Service, log *slog.Logger) *PromiseBreachWorker {
	if log == nil {
		log = slog.Default()
	}
	return &PromiseBreachWorker{
		svc: svc,
		log: log,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (w *PromiseBreachWorker) SetOnBreach(fn func(ctx context.Context, supplierID, retailerID, promiseID string) error) {
	w.onBreach = fn
}

func (w *PromiseBreachWorker) SetNotify(fn func(ctx context.Context, promise PaymentPromise, outcome string) error) {
	w.notify = fn
}

func (w *PromiseBreachWorker) SetNow(fn func() time.Time) {
	w.now = fn
	w.customNow = fn != nil
}

// RunPromiseBreachOnce evaluates all ACTIVE promises with DueAt < now across keyset-paginated batches.
func (w *PromiseBreachWorker) RunPromiseBreachOnce(ctx context.Context) (fulfilled, breached int, err error) {
	if w == nil || w.svc == nil || w.svc.repo == nil {
		return 0, 0, nil
	}
	now := time.Now().UTC()
	switch {
	case w.customNow && w.now != nil:
		now = w.now().UTC()
	case w.svc.now != nil:
		now = w.svc.now().UTC()
	case w.now != nil:
		now = w.now().UTC()
	}

	const pageSize = 200
	cursor := ""
	var errs []error

	for {
		promises, nextCursor, listErr := w.listExpiredActivePromisesPage(ctx, now, pageSize, cursor)
		if listErr != nil {
			return fulfilled, breached, listErr
		}
		for _, p := range promises {
			undisputed, calcErr := w.remainingUndisputedForPromise(ctx, p)
			if calcErr != nil {
				w.log.Warn("promise balance check failed", "promise_id", p.PromiseID, "err", calcErr)
				errs = append(errs, calcErr)
				continue
			}
			if undisputed <= 0 {
				changed, updErr := w.transitionPromiseStatus(ctx, p.PromiseID, PromiseStatusFulfilled, now)
				if updErr != nil {
					w.log.Warn("promise fulfill failed", "promise_id", p.PromiseID, "err", updErr)
					errs = append(errs, updErr)
					continue
				}
				if changed {
					fulfilled++
					p.Status = PromiseStatusFulfilled
					p.UpdatedAt = now
					if w.notify != nil {
						if nErr := w.notify(ctx, p, PromiseStatusFulfilled); nErr != nil {
							w.log.Warn("promise fulfill notify failed", "promise_id", p.PromiseID, "err", nErr)
						}
					}
				}
				continue
			}

			if w.onBreach != nil {
				if bErr := w.onBreach(ctx, p.SupplierID, p.RetailerID, p.PromiseID); bErr != nil {
					w.log.Warn("promise breach callback failed", "promise_id", p.PromiseID, "err", bErr)
					errs = append(errs, bErr)
					continue
				}
			}
			changed, updErr := w.transitionPromiseStatus(ctx, p.PromiseID, PromiseStatusBreached, now)
			if updErr != nil {
				w.log.Warn("promise breach status update failed", "promise_id", p.PromiseID, "err", updErr)
				errs = append(errs, updErr)
				continue
			}
			if changed {
				breached++
				p.Status = PromiseStatusBreached
				p.UpdatedAt = now
				if w.notify != nil {
					if nErr := w.notify(ctx, p, PromiseStatusBreached); nErr != nil {
						w.log.Warn("promise breach notify failed", "promise_id", p.PromiseID, "err", nErr)
					}
				}
			}
		}
		if nextCursor == "" || nextCursor == cursor || len(promises) == 0 {
			break
		}
		cursor = nextCursor
	}
	return fulfilled, breached, errors.Join(errs...)
}

// RunOnce runs one promise breach evaluation sweep.
func (w *PromiseBreachWorker) RunOnce(ctx context.Context) error {
	fulfilled, breached, err := w.RunPromiseBreachOnce(ctx)
	w.log.Info("ar promise breach pass", "fulfilled", fulfilled, "breached", breached)
	return err
}

// Start polls expired promises periodically until ctx is cancelled.
func (w *PromiseBreachWorker) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil {
				w.log.Warn("promise breach worker pass failed", "err", err)
			}
		}
	}
}

// HandleRunPromisesOnce handles POST /v1/admin/ar/promises/run-once.
func (w *PromiseBreachWorker) HandleRunPromisesOnce(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(rw, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(rw, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin {
		writeJSON(rw, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	fulfilled, breached, err := w.RunPromiseBreachOnce(r.Context())
	if err != nil {
		writeJSON(rw, http.StatusInternalServerError, map[string]any{
			"error":     err.Error(),
			"fulfilled": fulfilled,
			"breached":  breached,
		})
		return
	}
	writeJSON(rw, http.StatusOK, map[string]any{
		"status":    "ok",
		"fulfilled": fulfilled,
		"breached":  breached,
	})
}

func (w *PromiseBreachWorker) remainingUndisputedForPromise(ctx context.Context, p PaymentPromise) (int64, error) {
	if p.InvoiceID != "" {
		inv, err := w.svc.repo.GetByID(ctx, p.InvoiceID)
		if err != nil {
			if strings.Contains(err.Error(), "invoice_not_found") || spanner.ErrCode(err) == codes.NotFound {
				return 0, nil
			}
			return 0, err
		}
		if inv.Status == StatusPaid || inv.Status == StatusVoid || inv.BalanceMinor <= 0 {
			return 0, nil
		}
		disputes, err := w.svc.lookupDisputes(ctx, []string{inv.OrderID})
		if err != nil {
			return 0, err
		}
		_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
		return und, nil
	}

	allOpen, err := w.svc.listAllInvoicesForRetailer(ctx, p.RetailerID, StatusOpen+","+StatusPartial)
	if err != nil {
		return 0, err
	}
	var orderIDs []string
	var scoped []Invoice
	for _, inv := range allOpen {
		if inv.SupplierID != p.SupplierID {
			continue
		}
		scoped = append(scoped, inv)
		if inv.OrderID != "" {
			orderIDs = append(orderIDs, inv.OrderID)
		}
	}
	if len(scoped) == 0 {
		return 0, nil
	}
	disputes, err := w.svc.lookupDisputes(ctx, orderIDs)
	if err != nil {
		return 0, err
	}
	var totalUndisputed int64
	for _, inv := range scoped {
		_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
		totalUndisputed += und
	}
	return totalUndisputed, nil
}

func (w *PromiseBreachWorker) listExpiredActivePromisesPage(ctx context.Context, now time.Time, limit int, cursor string) ([]PaymentPromise, string, error) {
	switch repo := w.svc.repo.(type) {
	case *MemoryRepository:
		cursorTs, cursorID, hasCursor, err := decodePromiseCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		repo.mu.RLock()
		var expired []PaymentPromise
		for _, p := range repo.promises {
			if p.Status != PromiseStatusActive || !p.DueAt.Before(now) {
				continue
			}
			expired = append(expired, p)
		}
		repo.mu.RUnlock()
		sort.Slice(expired, func(i, j int) bool {
			if !expired[i].DueAt.Equal(expired[j].DueAt) {
				return expired[i].DueAt.Before(expired[j].DueAt)
			}
			return expired[i].PromiseID < expired[j].PromiseID
		})
		filtered := expired
		if hasCursor {
			filtered = make([]PaymentPromise, 0, len(expired))
			for _, p := range expired {
				due := p.DueAt.UTC()
				if due.After(cursorTs) || (due.Equal(cursorTs) && p.PromiseID > cursorID) {
					filtered = append(filtered, p)
				}
			}
		}
		if len(filtered) > limit {
			page := filtered[:limit]
			last := page[len(page)-1]
			return page, encodePromiseCursor(last.DueAt, last.PromiseID), nil
		}
		return filtered, "", nil
	case *SpannerRepository:
		if repo == nil || repo.client == nil {
			return nil, "", nil
		}
		cursorTs, cursorID, hasCursor, err := decodePromiseCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		sql := `SELECT PromiseId, RetailerId, SupplierId, IFNULL(InvoiceId, ''), QuarterKey, PromisedAt, DueAt, PromisedMinor, Status, IFNULL(ApprovedBy, ''), CreatedAt, UpdatedAt
			FROM ArPaymentPromises
			WHERE Status = 'ACTIVE' AND DueAt < @now`
		params := map[string]any{
			"now":   now.UTC(),
			"limit": int64(limit + 1),
		}
		if hasCursor {
			sql += " AND (DueAt > @cursorTs OR (DueAt = @cursorTs AND PromiseId > @cursorId))"
			params["cursorTs"] = cursorTs
			params["cursorId"] = cursorID
		}
		sql += " ORDER BY DueAt ASC, PromiseId ASC LIMIT @limit"

		iter := repo.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()
		var out []PaymentPromise
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, "", err
			}
			p, err := scanPromise(row)
			if err != nil {
				return nil, "", err
			}
			out = append(out, p)
		}
		if len(out) > limit {
			page := out[:limit]
			last := page[len(page)-1]
			return page, encodePromiseCursor(last.DueAt, last.PromiseID), nil
		}
		return out, "", nil
	default:
		return nil, "", nil
	}
}

func (w *PromiseBreachWorker) transitionPromiseStatus(ctx context.Context, promiseID, nextStatus string, now time.Time) (bool, error) {
	switch repo := w.svc.repo.(type) {
	case *MemoryRepository:
		repo.mu.Lock()
		defer repo.mu.Unlock()
		p, ok := repo.promises[promiseID]
		if !ok || p.Status != PromiseStatusActive {
			return false, nil
		}
		p.Status = nextStatus
		p.UpdatedAt = now
		repo.promises[promiseID] = p
		return true, nil
	case *SpannerRepository:
		if repo == nil || repo.client == nil {
			return false, nil
		}
		changed := false
		err := spannerutils.RunReadWriteTransaction(ctx, repo.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "ArPaymentPromises", spanner.Key{promiseID}, []string{"Status", "RetailerId", "SupplierId", "InvoiceId"})
			if err != nil {
				if spanner.ErrCode(err) == codes.NotFound {
					return nil
				}
				return err
			}
			var st, rid, sid string
			var invID spanner.NullString
			if err := row.Columns(&st, &rid, &sid, &invID); err != nil {
				return err
			}
			if st != PromiseStatusActive {
				changed = false
				return nil
			}
			buf := outbox.NewSpannerTxnBuffer(txn)
			if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, promiseID, events.TopicMain, map[string]any{
				"type":        EventARPaymentPromiseStatus,
				"timestamp":   now.Format(time.RFC3339Nano),
				"promise_id":  promiseID,
				"retailer_id": rid,
				"supplier_id": sid,
				"invoice_id":  invID.StringVal,
				"status":      nextStatus,
			}); err != nil {
				return err
			}
			if err := txn.BufferWrite([]*spanner.Mutation{
				spanner.UpdateMap("ArPaymentPromises", map[string]any{
					"PromiseId": promiseID,
					"Status":    nextStatus,
					"UpdatedAt": spanner.CommitTimestamp,
				}),
			}); err != nil {
				return err
			}
			if err := buf.Flush(ctx); err != nil {
				return err
			}
			changed = true
			return nil
		})
		return changed, err
	default:
		return false, nil
	}
}
