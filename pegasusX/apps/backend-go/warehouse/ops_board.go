package warehouse

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

// OpsBoardResponse is GET /v1/warehouse/ops/board.
type OpsBoardResponse struct {
	Date              string                   `json:"date"`
	WarehouseID       string                   `json:"warehouse_id"`
	Preorders         []OpsBoardOrder          `json:"preorders"`
	DeliverBefore     []OpsBoardOrder          `json:"deliver_before"`
	StockCommitments  []OpsBoardStockCommit    `json:"stock_commitments"`
	DraftManifests    []OpsBoardManifest       `json:"draft_manifests"`
	LoadingManifests  []OpsBoardManifest       `json:"loading_manifests"`
	FetchedAt         string                   `json:"fetched_at"`
}

type OpsBoardOrder struct {
	OrderID             string                         `json:"order_id"`
	Status              string                         `json:"status"`
	RetailerID          string                         `json:"retailer_id,omitempty"`
	TotalMinor          int64                          `json:"total_minor"`
	DeliveryExpectation *order.DeliveryExpectation     `json:"delivery_expectation,omitempty"`
}

type OpsBoardStockCommit struct {
	SKUId       string `json:"sku_id"`
	CommittedQty int64 `json:"committed_qty"`
}

type OpsBoardManifest struct {
	ManifestID  string `json:"manifest_id"`
	State       string `json:"state"`
	StopCount   int64  `json:"stop_count"`
	DriverName  string `json:"driver_name,omitempty"`
}

// HandleOpsBoard serves GET /v1/warehouse/ops/board?date=YYYY-MM-DD.
func (s *Service) HandleOpsBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	dateStr := strings.TrimSpace(r.URL.Query().Get("date"))
	if dateStr == "" {
		dateStr = s.now().UTC().Format("2006-01-02")
	}
	resp, err := s.loadOpsBoard(r.Context(), whID, dateStr)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ops_board_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) loadOpsBoard(ctx context.Context, warehouseID, dateStr string) (OpsBoardResponse, error) {
	now := s.now().UTC()
	out := OpsBoardResponse{
		Date:        dateStr,
		WarehouseID: warehouseID,
		FetchedAt:   now.Format(time.RFC3339Nano),
	}
	if s.spannerClient == nil || warehouseID == "" {
		return out, nil
	}
	dayStart, dayEnd, err := boardDayBounds(dateStr)
	if err != nil {
		return out, err
	}
	preorders, err := s.boardPreorders(ctx, warehouseID, dayStart, dayEnd, now)
	if err != nil {
		return out, err
	}
	deliverBefore, err := s.boardDeliverBefore(ctx, warehouseID, dayStart, dayEnd, now)
	if err != nil {
		return out, err
	}
	manifests, err := s.boardManifests(ctx, warehouseID)
	if err != nil {
		return out, err
	}
	out.Preorders = preorders
	out.DeliverBefore = deliverBefore
	out.DraftManifests = manifests.draft
	out.LoadingManifests = manifests.loading
	out.StockCommitments = []OpsBoardStockCommit{}
	return out, nil
}

func boardDayBounds(dateStr string) (time.Time, time.Time, error) {
	day, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date: %w", err)
	}
	start := day.UTC()
	end := start.Add(24 * time.Hour)
	return start, end, nil
}

func (s *Service) boardPreorders(ctx context.Context, warehouseID string, dayStart, dayEnd, now time.Time) ([]OpsBoardOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, Status, RetailerId, TotalMinor, Source, ConfirmationStatus, DeliveryPriority,
		             DeliverBefore, RequestedDeliveryDate, ProposedDeliveryDate, ReceivingWindowOpen, ReceivingWindowClose
		      FROM Orders
		      WHERE WarehouseId = @wh
		        AND Source IN UNNEST(@sources)
		        AND RequestedDeliveryDate >= @start
		        AND RequestedDeliveryDate < @end
		      ORDER BY RequestedDeliveryDate ASC
		      LIMIT 100`,
		Params: map[string]any{
			"wh":      warehouseID,
			"sources": []string{"MANUAL_PREORDER", "AI_PREORDER"},
			"start":   dayStart,
			"end":     dayEnd,
		},
	}
	return s.queryBoardOrders(ctx, stmt, now)
}

func (s *Service) boardDeliverBefore(ctx context.Context, warehouseID string, dayStart, dayEnd, now time.Time) ([]OpsBoardOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, Status, RetailerId, TotalMinor, Source, ConfirmationStatus, DeliveryPriority,
		             DeliverBefore, RequestedDeliveryDate, ProposedDeliveryDate, ReceivingWindowOpen, ReceivingWindowClose
		      FROM Orders
		      WHERE WarehouseId = @wh
		        AND DeliverBefore >= @start
		        AND DeliverBefore < @end
		        AND Status NOT IN UNNEST(@terminal)
		      ORDER BY DeliverBefore ASC
		      LIMIT 100`,
		Params: map[string]any{
			"wh":       warehouseID,
			"start":    dayStart,
			"end":      dayEnd,
			"terminal": []string{"COMPLETED", "CANCELLED"},
		},
	}
	return s.queryBoardOrders(ctx, stmt, now)
}

func (s *Service) queryBoardOrders(ctx context.Context, stmt spanner.Statement, now time.Time) ([]OpsBoardOrder, error) {
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]OpsBoardOrder, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var (
			orderID, status, retailerID, source, confirmation, priority string
			totalMinor                                                  int64
			deliverBefore, requested, proposed                          spanner.NullTime
			windowOpen, windowClose                                     spanner.NullString
		)
		if err := row.Columns(&orderID, &status, &retailerID, &totalMinor, &source, &confirmation, &priority,
			&deliverBefore, &requested, &proposed, &windowOpen, &windowClose); err != nil {
			continue
		}
		o := order.Order{
			OrderID:              orderID,
			RetailerID:           retailerID,
			TotalMinor:           totalMinor,
			Status:               order.Status(status),
			Source:               order.OrderSource(source),
			ConfirmationStatus:   order.ConfirmationStatus(confirmation),
			DeliveryPriority:     order.DeliveryPriority(priority),
			ReceivingWindowOpen:  windowOpen.StringVal,
			ReceivingWindowClose: windowClose.StringVal,
		}
		if deliverBefore.Valid {
			t := deliverBefore.Time
			o.DeliverBefore = &t
		}
		if requested.Valid {
			t := requested.Time
			o.RequestedDeliveryDate = &t
		}
		if proposed.Valid {
			t := proposed.Time
			o.ProposedDeliveryDate = &t
		}
		exp := order.ComputeDeliveryExpectation(now, o)
		out = append(out, OpsBoardOrder{
			OrderID:             orderID,
			Status:              status,
			RetailerID:          retailerID,
			TotalMinor:          totalMinor,
			DeliveryExpectation: &exp,
		})
	}
}

type boardManifestSplit struct {
	draft   []OpsBoardManifest
	loading []OpsBoardManifest
}

func (s *Service) boardManifests(ctx context.Context, warehouseID string) (boardManifestSplit, error) {
	stmt := spanner.Statement{
		SQL: `SELECT m.ManifestId, m.State, m.StopCount, COALESCE(d.Name, m.DriverId)
		      FROM SupplierTruckManifests m
		      LEFT JOIN Drivers d ON d.DriverId = m.DriverId
		      WHERE m.WarehouseId = @wh
		        AND m.State IN ('DRAFT', 'LOADING')
		      ORDER BY m.UpdatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{"wh": warehouseID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out boardManifestSplit
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return boardManifestSplit{}, err
		}
		var manifestID, state, driverName string
		var stopCount int64
		if err := row.Columns(&manifestID, &state, &stopCount, &driverName); err != nil {
			continue
		}
		entry := OpsBoardManifest{
			ManifestID: manifestID,
			State:      state,
			StopCount:  stopCount,
			DriverName: strings.TrimSpace(driverName),
		}
		switch strings.ToUpper(state) {
		case "LOADING":
			out.loading = append(out.loading, entry)
		default:
			out.draft = append(out.draft, entry)
		}
	}
}
