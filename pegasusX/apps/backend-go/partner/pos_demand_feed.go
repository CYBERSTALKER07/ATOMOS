package partner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// POSDemandFeedItem is one chain-POS sale line for demand ingestion.
type POSDemandFeedItem struct {
	ExternalID string `json:"external_id"`
	ProductID  string `json:"product_id"`
	GTIN       string `json:"gtin"`
	Quantity   int64  `json:"quantity"`
	Day        string `json:"day"` // YYYY-MM-DD
	LocationID string `json:"location_id"`
	Void       bool   `json:"void"`
}

// IngestPOSDemandFeed accepts batch POS sell-through from a retailer's partner key
// and emits DEMAND_SIGNAL events for auto-order/flywheel consumers.
func (s *Service) IngestPOSDemandFeed(ctx context.Context, p Principal, items []POSDemandFeedItem) (int, error) {
	if strings.ToUpper(strings.TrimSpace(p.TenantType)) != TenantRetailer {
		return 0, fmt.Errorf("retailer_tenant_required")
	}
	if len(items) == 0 {
		return 0, fmt.Errorf("items_required")
	}
	if len(items) > maxMasterDataBatch {
		return 0, fmt.Errorf("batch_too_large")
	}
	accepted := 0
	for _, it := range items {
		ext := strings.TrimSpace(it.ExternalID)
		if ext == "" {
			return accepted, fmt.Errorf("external_id_required")
		}
		sku := strings.TrimSpace(it.ProductID)
		if sku == "" {
			sku = strings.TrimSpace(it.GTIN)
		}
		if sku == "" {
			return accepted, fmt.Errorf("product_id_or_gtin_required")
		}
		qty := it.Quantity
		kind := "sale"
		if it.Void {
			qty = -qty
			kind = "void"
		}
		if qty == 0 {
			continue
		}
		day := strings.TrimSpace(it.Day)
		if day == "" {
			day = s.now().UTC().Format("2006-01-02")
		}
		payload := events.DemandSignalEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventDemandSignal,
				Timestamp: s.now().UTC().Format(time.RFC3339Nano),
			},
			RetailerID: p.TenantID,
			LocationID: strings.TrimSpace(it.LocationID),
			SKU:        sku,
			Day:        day,
			QtyDelta:   qty,
			Source:     "CHAIN_POS",
			Kind:       kind,
		}
		_ = s.EnqueueEvent(ctx, "pos:"+ext, events.EventDemandSignal, map[string]any{
			"type":        events.EventDemandSignal,
			"retailer_id": p.TenantID,
			"sku":         sku,
			"qty_delta":   qty,
			"source":      "CHAIN_POS",
			"kind":        kind,
			"external_id": ext,
			"day":         day,
			"location_id": strings.TrimSpace(it.LocationID),
			"trace_id":    uuid.NewString(),
		})
		if s.posFeedSink != nil {
			if err := s.posFeedSink(ctx, ext, payload); err != nil {
				return accepted, err
			}
		}
		accepted++
	}
	return accepted, nil
}

// SetPOSFeedSink wires durable DEMAND_SIGNAL persistence (Spanner outbox).
func (s *Service) SetPOSFeedSink(fn func(context.Context, string, events.DemandSignalEvent) error) {
	if s == nil {
		return
	}
	s.posFeedSink = fn
}

// HandlePOSDemandFeed POST /partner/v1/demand/pos-feed
func (h *Handlers) HandlePOSDemandFeed(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := readMasterDataBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		Items []POSDemandFeedItem `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	h.withPartnerIdempotency(w, r, p, "POST /partner/v1/demand/pos-feed", body, func() (int, any, error) {
		n, err := h.Svc.IngestPOSDemandFeed(r.Context(), p, req.Items)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusAccepted, map[string]any{"accepted": n}, nil
	})
}

// NewSpannerPOSFeedSink writes DEMAND_SIGNAL to the transactional outbox.
func NewSpannerPOSFeedSink(client *spanner.Client) func(context.Context, string, events.DemandSignalEvent) error {
	return func(ctx context.Context, externalID string, ev events.DemandSignalEvent) error {
		if client == nil {
			return nil
		}
		aggID := strings.TrimSpace(externalID)
		if aggID == "" {
			aggID = uuid.NewString()
		}
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			buf := &spannerOutboxBuf{txn: txn}
			return outbox.EmitJSON(ctx, buf, "DemandSignal", aggID, events.TopicDemand, ev)
		})
		return err
	}
}

type spannerOutboxBuf struct {
	txn *spanner.ReadWriteTransaction
}

func (b *spannerOutboxBuf) BufferOutbox(_ context.Context, e outbox.Event) error {
	return b.txn.BufferWrite([]*spanner.Mutation{
		spanner.InsertMap("OutboxEvents", map[string]any{
			"EventId":       e.EventID,
			"AggregateType": e.AggregateType,
			"AggregateId":   e.AggregateID,
			"TopicName":     e.TopicName,
			"Payload":       e.Payload,
			"CreatedAt":     spanner.CommitTimestamp,
		}),
	})
}
