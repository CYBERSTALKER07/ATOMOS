package loyalty

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

type Handlers struct {
	Spanner *spanner.Client
}

func (h *Handlers) HandleRetailerTier(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rid := auth.ResolveRetailerOrgID(claims)
	if rid == "" {
		web.JSONError(w, "retailer_scope_missing", http.StatusUnprocessableEntity)
		return
	}
	sid := strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID))
	if sid == "" || h == nil || h.Spanner == nil {
		web.JSONResponse(w, http.StatusOK, TierView{Enrolled: false})
		return
	}
	prog, err := ReadProgramStrong(r.Context(), h.Spanner, sid)
	if err != nil {
		web.JSONError(w, "loyalty_query_failed", http.StatusInternalServerError)
		return
	}
	life, avail, err := ReadAccount(r.Context(), h.Spanner, sid, rid)
	if err != nil {
		web.JSONError(w, "loyalty_query_failed", http.StatusInternalServerError)
		return
	}
	if prog == nil && life == 0 {
		web.JSONResponse(w, http.StatusOK, TierView{Enrolled: false, SupplierID: sid})
		return
	}
	tiers := DefaultTiers
	earnBps := defaultEarnBps
	if prog != nil {
		tiers = prog.Tiers
		earnBps = prog.EarnBps
	}
	cur, next := TierFor(life, tiers)
	view := TierView{
		Enrolled:        true,
		Tier:            cur.Name,
		LifetimePoints:  life,
		AvailablePoints: avail,
		EarnBps:         earnBps,
		SupplierID:      sid,
	}
	if next != nil {
		view.NextTier = next.Name
		if next.MinPoints > life {
			view.PointsToNext = next.MinPoints - life
		}
	}
	web.JSONResponse(w, http.StatusOK, view)
}

func (h *Handlers) HandleRetailerLedger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rid := auth.ResolveRetailerOrgID(claims)
	sid := strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID))
	if rid == "" || sid == "" || h == nil || h.Spanner == nil {
		web.JSONResponse(w, http.StatusOK, map[string]any{"entries": []LedgerEntry{}})
		return
	}
	iter := h.Spanner.Single().Query(r.Context(), spanner.Statement{
		SQL: `SELECT LedgerId, OrderId, Points, EarnBps, AmountMinor, CreatedAt
		      FROM LoyaltyLedger WHERE SupplierId = @sid AND RetailerId = @rid
		      ORDER BY CreatedAt DESC LIMIT 50`,
		Params: map[string]any{"sid": sid, "rid": rid},
	})
	defer iter.Stop()
	entries := []LedgerEntry{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			web.JSONError(w, "loyalty_query_failed", http.StatusInternalServerError)
			return
		}
		var e LedgerEntry
		var created time.Time
		if err := row.Columns(&e.LedgerID, &e.OrderID, &e.Points, &e.EarnBps, &e.AmountMinor, &created); err != nil {
			continue
		}
		e.CreatedAt = created.UTC().Format(time.RFC3339)
		entries = append(entries, e)
	}
	web.JSONResponse(w, http.StatusOK, map[string]any{"entries": entries})
}

func (h *Handlers) HandleSupplierProgram(w http.ResponseWriter, r *http.Request) {
	sid := strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), ""))
	if sid == "" {
		if id, ok := auth.ResolveSupplierID(r.Context()); ok {
			sid = strings.TrimSpace(id)
		}
	}
	if sid == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		prog, err := ReadProgramStrong(r.Context(), h.Spanner, sid)
		if err != nil {
			web.JSONError(w, "loyalty_query_failed", http.StatusInternalServerError)
			return
		}
		if prog == nil {
			web.JSONResponse(w, http.StatusOK, Program{
				SupplierID: sid,
				EarnBps:    defaultEarnBps,
				Tiers:      DefaultTiers,
				Source:     "unconfigured",
			})
			return
		}
		web.JSONResponse(w, http.StatusOK, prog)
	case http.MethodPatch:
		var req Program
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			web.JSONError(w, "invalid_json", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Reason) == "" {
			web.JSONError(w, "reason_required", http.StatusBadRequest)
			return
		}
		if h == nil || h.Spanner == nil {
			web.JSONError(w, "loyalty_unavailable", http.StatusServiceUnavailable)
			return
		}
		bps := req.EarnBps
		if bps <= 0 {
			bps = defaultEarnBps
		}
		tiers := req.Tiers
		if len(tiers) == 0 {
			tiers = DefaultTiers
		}
		tiersJSON, _ := json.Marshal(tiers)
		claims, _ := auth.FromContext(r.Context())
		actor := ""
		if claims.Subject != "" {
			actor = claims.Subject
		}
		err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			buf := outbox.NewSpannerTxnBuffer(txn)
			if err := outbox.EmitJSON(ctx, buf, events.AggregateSupplier, sid, events.TopicMain, map[string]any{
				"type":        "LOYALTY_PROGRAM_UPDATED",
				"supplier_id": sid,
				"earn_bps":    bps,
				"reason":      req.Reason,
			}); err != nil {
				return err
			}
			if err := buf.Flush(ctx); err != nil {
				return err
			}
			return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("LoyaltyPrograms", map[string]any{
				"SupplierId": sid,
				"EarnBps":    bps,
				"TiersJson":  string(tiersJSON),
				"Reason":     req.Reason,
				"UpdatedBy":  actor,
				"CreatedAt":  spanner.CommitTimestamp,
				"UpdatedAt":  spanner.CommitTimestamp,
			})})
		})
		if err != nil {
			web.JSONError(w, "update_failed", http.StatusInternalServerError)
			return
		}
		web.JSONResponse(w, http.StatusOK, Program{SupplierID: sid, EarnBps: bps, Tiers: tiers, Reason: req.Reason, Source: "program"})
	default:
		web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
	}
}
