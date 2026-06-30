package supplier

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	h3 "github.com/uber/h3-go/v4"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"google.golang.org/api/iterator"
)

// ExceptionMapCell is one H3 bucket on the supplier exception weather map.
type ExceptionMapCell struct {
	H3Cell         string            `json:"h3_cell"`
	Lat            float64           `json:"lat"`
	Lng            float64           `json:"lng"`
	Severity       string            `json:"severity"`
	Counts         map[string]int    `json:"counts"`
	SampleOrderIDs []string          `json:"sample_order_ids,omitempty"`
	DeepLink       string            `json:"deep_link"`
}

type exceptionMapBucket struct {
	h3Cell     string
	shopClosed int
	delayed    int
	manifest   int
	orderIDs   []string
}

// HandleExceptionMap serves GET /v1/supplier/ops/exception-map.
func (s *Service) HandleExceptionMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"cells": []ExceptionMapCell{}, "window_hours": 24})
		return
	}

	windowHours := 24
	if raw := strings.TrimSpace(r.URL.Query().Get("window_hours")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 168 {
			windowHours = v
		}
	}

	sid := s.scopedSupplierID(r)
	cells, err := s.buildExceptionMap(r.Context(), sid, windowHours)
	if err != nil {
		s.log.ErrorContext(r.Context(), "exception map failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exception_map_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cells": cells, "window_hours": windowHours})
}

func (s *Service) buildExceptionMap(ctx context.Context, supplierID string, windowHours int) ([]ExceptionMapCell, error) {
	since := s.now().UTC().Add(-time.Duration(windowHours) * time.Hour)
	buckets := map[string]*exceptionMapBucket{}

	add := func(cell string, kind string, orderID string) {
		cell = strings.TrimSpace(cell)
		if len(cell) != 15 {
			return
		}
		b, ok := buckets[cell]
		if !ok {
			b = &exceptionMapBucket{h3Cell: cell}
			buckets[cell] = b
		}
		switch kind {
		case "shop_closed":
			b.shopClosed++
		case "delayed":
			b.delayed++
		case "manifest_gate":
			b.manifest++
		}
		if orderID != "" && len(b.orderIDs) < 5 {
			dup := false
			for _, existing := range b.orderIDs {
				if existing == orderID {
					dup = true
					break
				}
			}
			if !dup {
				b.orderIDs = append(b.orderIDs, orderID)
			}
		}
	}

	if err := s.collectShopClosedBuckets(ctx, supplierID, since, add); err != nil {
		return nil, err
	}
	if err := s.collectDelayedOrderBuckets(ctx, supplierID, since, add); err != nil {
		return nil, err
	}
	if err := s.collectManifestExceptionBuckets(ctx, supplierID, add); err != nil {
		return nil, err
	}

	out := make([]ExceptionMapCell, 0, len(buckets))
	for _, b := range buckets {
		total := b.shopClosed + b.delayed + b.manifest
		if total == 0 {
			continue
		}
		lat, lng := h3CellCenter(b.h3Cell)
		out = append(out, ExceptionMapCell{
			H3Cell:   b.h3Cell,
			Lat:      lat,
			Lng:      lng,
			Severity: exceptionSeverity(b.shopClosed, b.delayed, b.manifest),
			Counts: map[string]int{
				"shop_closed":    b.shopClosed,
				"delayed":        b.delayed,
				"manifest_gate":  b.manifest,
				"total":          total,
			},
			SampleOrderIDs: append([]string(nil), b.orderIDs...),
			DeepLink:       fmt.Sprintf("/exceptions?cell=%s", b.h3Cell),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Counts["total"] > out[j].Counts["total"]
	})
	if len(out) > 120 {
		out = out[:120]
	}
	return out, nil
}

func (s *Service) collectShopClosedBuckets(ctx context.Context, supplierID string, since time.Time, add func(cell, kind, orderID string)) error {
	stmt := spanner.Statement{
		SQL: `SELECT s.OrderId, s.GPSLat, s.GPSLng
		      FROM ShopClosedAttempts s
		      JOIN Orders o ON s.OrderId = o.OrderId
		      WHERE o.SupplierId = @supplierId
		        AND s.ReportedAt >= @since
		        AND s.ResolvedAt IS NULL`,
		Params: map[string]any{"supplierId": supplierID, "since": since},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var orderID string
		var lat, lng float64
		if err := row.Columns(&orderID, &lat, &lng); err != nil {
			continue
		}
		add(proximity.H3CellFromLatLng(lat, lng), "shop_closed", orderID)
	}
}

func (s *Service) collectDelayedOrderBuckets(ctx context.Context, supplierID string, since time.Time, add func(cell, kind, orderID string)) error {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, H3Cell, Status, ConfirmationStatus, RequestedDeliveryDate, ProposedDeliveryDate, UpdatedAt
		      FROM Orders@{FORCE_INDEX=Idx_Orders_BySupplierUpdated}
		      WHERE SupplierId = @supplierId
		        AND UpdatedAt >= @since
		        AND Status NOT IN ('COMPLETED', 'CANCELLED', 'REJECTED')
		      LIMIT 500`,
		Params: map[string]any{"supplierId": supplierID, "since": since},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	now := s.now().UTC()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var (
			orderID, h3Cell, status, confirmation string
			requested, proposed                   spanner.NullTime
			updatedAt                             time.Time
		)
		if err := row.Columns(&orderID, &h3Cell, &status, &confirmation, &requested, &proposed, &updatedAt); err != nil {
			continue
		}
		if !exceptionOrderIsDelayed(now, status, confirmation, requested, proposed) {
			continue
		}
		cell := strings.TrimSpace(h3Cell)
		if cell == "" {
			continue
		}
		add(cell, "delayed", orderID)
	}
}

func (s *Service) collectManifestExceptionBuckets(ctx context.Context, supplierID string, add func(cell, kind, orderID string)) error {
	rows, err := s.listSupplierManifestExceptions(ctx, supplierID, false)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	orderIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if id := strings.TrimSpace(row.OrderID); id != "" {
			orderIDs = append(orderIDs, id)
		}
	}
	if len(orderIDs) == 0 {
		return nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT OrderId, H3Cell FROM Orders WHERE OrderId IN UNNEST(@ids)`,
		Params: map[string]any{"ids": orderIDs},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var orderID, h3Cell string
		if err := row.Columns(&orderID, &h3Cell); err != nil {
			continue
		}
		add(strings.TrimSpace(h3Cell), "manifest_gate", orderID)
	}
}

func exceptionOrderIsDelayed(now time.Time, status, confirmation string, requested, proposed spanner.NullTime) bool {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "DELAYED" || status == "AWAITING_REVIEW" {
		return true
	}
	o := order.Order{
		Status:             order.Status(status),
		ConfirmationStatus: order.ConfirmationStatus(strings.TrimSpace(confirmation)),
	}
	if proposed.Valid {
		t := proposed.Time
		o.ProposedDeliveryDate = &t
		o.ConfirmationStatus = order.ConfirmationStatusPendingWarehouse
	}
	if requested.Valid {
		t := requested.Time
		o.RequestedDeliveryDate = &t
	}
	exp := order.ComputeDeliveryExpectation(now, time.UTC, o)
	return exp.Urgency == order.ExpectationUrgencyOverdue || exp.Delayed
}

func exceptionSeverity(shopClosed, delayed, manifest int) string {
	total := shopClosed + delayed + manifest
	weighted := shopClosed*3 + delayed*2 + manifest
	switch {
	case weighted >= 6 || total >= 5:
		return "high"
	case weighted >= 3 || total >= 2:
		return "medium"
	default:
		return "low"
	}
}

func h3CellCenter(cell string) (float64, float64) {
	parsed := h3.Cell(h3.IndexFromString(cell))
	if !parsed.IsValid() {
		return 0, 0
	}
	ll, err := h3.CellToLatLng(parsed)
	if err != nil {
		return 0, 0
	}
	return ll.Lat, ll.Lng
}
