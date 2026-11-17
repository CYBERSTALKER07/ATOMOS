package ontology

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	pegasuskafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	kafka "github.com/segmentio/kafka-go"
)

// nonMoneyFloatKeys lists upstream geo/volume telemetry keys allowed on legacy Kafka
// domain events; stripped before constructing integer-only OntologyObject properties.
var nonMoneyFloatKeys = map[string]bool{
	"lat":                true,
	"lng":                true,
	"latitude":           true,
	"longitude":          true,
	"gps_lat":            true,
	"gps_lng":            true,
	"unit_volume_vu":     true,
	"total_volume_vu":    true,
	"max_volume_vu":      true,
	"coverage_radius_km": true,
	"utilization_pct":    true,
	"confidence":         true,
}

// Service manages semantic ontology projection from Kafka domain events,
// nightly Spanner drift rebuilds, redaction-aware graph queries, and EKG unification.
type Service struct {
	repo          Repository
	rebuildSource RebuildSource
	log           *slog.Logger
	now           func() time.Time
	defaultCell   string
}

// NewService constructs the ontology projector service.
// When repo also implements RebuildSource (e.g. *SpannerRepository), RebuildFromSpanner
// automatically uses it to snapshot live domain tables.
func NewService(repo Repository, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	svc := &Service{
		repo:        repo,
		log:         log,
		now:         func() time.Time { return time.Now().UTC() },
		defaultCell: DefaultCellID,
	}
	if rs, ok := repo.(RebuildSource); ok {
		svc.rebuildSource = rs
	}
	return svc
}

// WithRebuildSource overrides the snapshot source used by RebuildFromSpanner.
func (s *Service) WithRebuildSource(src RebuildSource) *Service {
	if s != nil {
		s.rebuildSource = src
	}
	return s
}

// SetDefaultCell overrides the fallback CellID used when incoming events omit cell_id.
func (s *Service) SetDefaultCell(cellID string) {
	if s != nil && strings.TrimSpace(cellID) != "" {
		s.defaultCell = NormalizeCellID(cellID)
	}
}

// Repository returns the underlying ontology repository.
func (s *Service) Repository() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// ListTypes returns the canonical v1 registry of 17 object types, 7 link types, and security markings.
func (s *Service) ListTypes() TypesCatalog {
	return ListTypes()
}

// UpsertDirect validates and persists objects/edges through the monotonic repository.
func (s *Service) UpsertDirect(ctx context.Context, objs []*OntologyObject, edges []*OntologyGraphEdge, opts UpsertOptions) (UpsertStats, error) {
	if s == nil || s.repo == nil {
		return UpsertStats{}, ErrRepoUnavailable
	}
	return s.repo.UpsertBatch(ctx, objs, edges, opts)
}

// GetObject fetches a single ontology object and redacts sensitive fields unless viewerRole is finance/admin.
func (s *Service) GetObject(ctx context.Context, cellID, objectType, objectID, viewerRole string) (*OntologyObject, bool, error) {
	if s == nil || s.repo == nil {
		return nil, false, ErrRepoUnavailable
	}
	obj, err := s.repo.GetObject(ctx, NormalizeCellID(cellID), strings.TrimSpace(objectType), strings.TrimSpace(objectID))
	if err != nil {
		return nil, false, err
	}
	redacted, didRedact := RedactObjectForViewer(obj, viewerRole)
	return redacted, didRedact, nil
}

// ListObjects queries ontology objects and applies marking redaction for the caller's viewer role.
func (s *Service) ListObjects(ctx context.Context, filter ListObjectsFilter) (ListObjectsResult, error) {
	cell := NormalizeCellID(filter.CellID)
	res := ListObjectsResult{
		CellID:     cell,
		SupplierID: strings.TrimSpace(filter.SupplierID),
		Objects:    []*OntologyObject{},
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}
	if res.Limit <= 0 {
		res.Limit = 50
	} else if res.Limit > 200 {
		res.Limit = 200
	}
	if res.Offset < 0 {
		res.Offset = 0
	}
	if s == nil || s.repo == nil {
		return res, ErrRepoUnavailable
	}
	filter.CellID = cell
	filter.Limit = res.Limit
	filter.Offset = res.Offset
	objs, total, err := s.repo.ListObjects(ctx, filter)
	if err != nil {
		return res, err
	}
	redacted, piiMasked, finMasked := RedactObjectsForViewer(objs, filter.ViewerRole)
	res.Objects = redacted
	res.Total = total
	res.PIIMasked = piiMasked
	res.FinanceMasked = finMasked
	return res, nil
}

// GetNeighbors traverses outgoing & incoming ontology graph edges up to depth hops and applies viewer redaction.
func (s *Service) GetNeighbors(ctx context.Context, cellID, supplierID, objectType, objectID string, depth int, linkTypes []string, viewerRole string) (NeighborResult, error) {
	cell := NormalizeCellID(cellID)
	if depth <= 0 {
		depth = 2
	} else if depth > 4 {
		depth = 4
	}
	res := NeighborResult{
		CellID:     cell,
		SupplierID: strings.TrimSpace(supplierID),
		Depth:      depth,
		Nodes:      []NeighborNode{},
		Edges:      []*OntologyGraphEdge{},
	}
	if s == nil || s.repo == nil {
		return res, ErrRepoUnavailable
	}
	root, nodes, edges, err := s.repo.GetNeighbors(ctx, cell, res.SupplierID, objectType, objectID, depth, linkTypes)
	if err != nil {
		return res, err
	}
	canView := CanViewSensitiveMarkings(viewerRole)
	redactedRoot, rootMasked := RedactObjectForViewer(root, viewerRole)
	anyMasked := rootMasked

	redactedNodes := make([]NeighborNode, 0, len(nodes))
	for _, n := range nodes {
		rn, masked := RedactObjectForViewer(n.Object, viewerRole)
		if masked {
			anyMasked = true
		}
		redactedNodes = append(redactedNodes, NeighborNode{
			Object:      rn,
			Hop:         n.Hop,
			ViaLinkType: n.ViaLinkType,
			Direction:   n.Direction,
			ParentType:  n.ParentType,
			ParentID:    n.ParentID,
		})
	}
	sortNeighborsDeterministic(redactedNodes)

	if edges == nil {
		edges = []*OntologyGraphEdge{}
	}
	res.Root = redactedRoot
	res.Nodes = redactedNodes
	res.Edges = edges
	res.TotalNodes = len(redactedNodes)
	res.TotalEdges = len(edges)
	res.PIIMasked = !canView || anyMasked
	res.FinanceMasked = !canView || anyMasked
	return res, nil
}

// GetKnowledgeGraphUnified materializes the supplier's EKG from unified OntologyObjects & OntologyGraphEdges,
// honoring viewer role markings while returning backward-compatible KGNode/KGEdge shapes.
func (s *Service) GetKnowledgeGraphUnified(ctx context.Context, cellID, supplierID, viewerRole string) (UnifiedKnowledgeGraph, error) {
	sid := strings.TrimSpace(supplierID)
	kg := UnifiedKnowledgeGraph{
		SupplierID: sid,
		Nodes:      []KGNode{},
		Edges:      []KGEdge{},
		Source:     "unified_ontology",
	}
	if s == nil || s.repo == nil {
		return kg, ErrRepoUnavailable
	}
	cell := s.resolveCellID(cellID)
	objs, edges, err := s.repo.GetGraphForSupplier(ctx, cell, sid)
	if err != nil {
		return kg, err
	}
	canView := CanViewSensitiveMarkings(viewerRole)
	redactedObjs, _, _ := RedactObjectsForViewer(objs, viewerRole)

	seenNode := make(map[string]struct{}, len(redactedObjs))
	for _, o := range redactedObjs {
		if o == nil {
			continue
		}
		nodeType := mapOntologyTypeToKGType(o.ObjectType)
		key := nodeType + ":" + o.ObjectID
		if _, exists := seenNode[key]; exists {
			continue
		}
		seenNode[key] = struct{}{}
		kg.Nodes = append(kg.Nodes, KGNode{
			ID:   o.ObjectID,
			Type: nodeType,
			Name: extractNodeLabel(o, canView),
		})
	}
	sort.Slice(kg.Nodes, func(i, j int) bool {
		if kg.Nodes[i].Type != kg.Nodes[j].Type {
			return kg.Nodes[i].Type < kg.Nodes[j].Type
		}
		return kg.Nodes[i].ID < kg.Nodes[j].ID
	})

	seenEdge := make(map[string]struct{}, len(edges))
	for _, e := range edges {
		if e == nil {
			continue
		}
		key := e.SourceObjectID + "|" + e.LinkType + "|" + e.TargetObjectID
		if _, exists := seenEdge[key]; exists {
			continue
		}
		seenEdge[key] = struct{}{}
		kg.Edges = append(kg.Edges, KGEdge{
			From:     e.SourceObjectID,
			To:       e.TargetObjectID,
			Relation: e.LinkType,
		})
	}
	sort.Slice(kg.Edges, func(i, j int) bool {
		if kg.Edges[i].From != kg.Edges[j].From {
			return kg.Edges[i].From < kg.Edges[j].From
		}
		if kg.Edges[i].Relation != kg.Edges[j].Relation {
			return kg.Edges[i].Relation < kg.Edges[j].Relation
		}
		return kg.Edges[i].To < kg.Edges[j].To
	})

	kg.OntologyObjectsCount = len(redactedObjs)
	kg.OntologyEdgesCount = len(edges)
	return kg, nil
}

func mapOntologyTypeToKGType(objType string) string {
	switch objType {
	case TypeSupplier:
		return "supplier"
	case TypeLegalEntity:
		return "legal_entity"
	case TypeWarehouse:
		return "warehouse"
	case TypeFactory:
		return "factory"
	case TypeSKU:
		return "sku"
	case TypeLot:
		return "lot"
	case TypeShop:
		return "retailer"
	case TypeOrder:
		return "order"
	case TypeOrderLine:
		return "order_line"
	case TypeManifest:
		return "manifest"
	case TypeVehicle:
		return "vehicle"
	case TypeDriver:
		return "driver"
	case TypeNasiyaInvoice:
		return "nasiya_invoice"
	case TypeCreditLine:
		return "credit_line"
	case TypePayment:
		return "payment"
	case TypeFiscalDoc:
		return "fiscal_doc"
	case TypeProposal:
		return "proposal"
	default:
		return strings.ToLower(objType)
	}
}

func extractNodeLabel(obj *OntologyObject, canViewSensitive bool) string {
	if obj == nil {
		return ""
	}
	switch obj.ObjectType {
	case TypeLegalEntity:
		if !canViewSensitive {
			return RedactedPlaceholder
		}
		return firstNonEmpty(propString(obj.Properties, "legal_name"), obj.ObjectID)
	case TypeDriver:
		if !canViewSensitive {
			return RedactedPlaceholder
		}
		return firstNonEmpty(propString(obj.Properties, "name"), obj.ObjectID)
	case TypeOrder:
		return firstNonEmpty(propString(obj.Properties, "status"), obj.ObjectID)
	case TypeOrderLine:
		return firstNonEmpty(propString(obj.Properties, "name"), propString(obj.Properties, "sku"), obj.ObjectID)
	case TypeVehicle:
		return firstNonEmpty(propString(obj.Properties, "label"), propString(obj.Properties, "license_plate"), obj.ObjectID)
	case TypeLot:
		return firstNonEmpty(propString(obj.Properties, "lot_code"), propString(obj.Properties, "status"), obj.ObjectID)
	case TypeManifest:
		return firstNonEmpty(propString(obj.Properties, "state"), obj.ObjectID)
	case TypeNasiyaInvoice, TypeCreditLine, TypePayment, TypeFiscalDoc, TypeProposal:
		return firstNonEmpty(propString(obj.Properties, "status"), obj.ObjectID)
	default:
		return firstNonEmpty(propString(obj.Properties, "name"), propString(obj.Properties, "sku"), obj.ObjectID)
	}
}

// RebuildFromSpanner loads live Spanner domain tables for a supplier (or all active suppliers
// when supplierID is empty) and batch-upserts unified OntologyObjects and OntologyGraphEdges
// in <=900-mutation transactions so nightly drift reconciliation never hits Spanner limits.
func (s *Service) RebuildFromSpanner(ctx context.Context, cellID, supplierID string) (RebuildResult, error) {
	started := s.now()
	cell := s.resolveCellID(cellID)
	sid := strings.TrimSpace(supplierID)
	res := RebuildResult{
		CellID:        cell,
		SupplierID:    sid,
		Status:        "COMPLETED",
		TablesScanned: []string{},
		RebuiltAt:     started,
	}
	if s == nil || s.repo == nil {
		res.Status = "UNAVAILABLE"
		return res, ErrRepoUnavailable
	}
	if s.rebuildSource == nil {
		res.Status = "NO_REBUILD_SOURCE"
		return res, ErrRepoUnavailable
	}

	objs, edges, tables, err := s.rebuildSource.LoadDomainSnapshot(ctx, cell, sid)
	if err != nil {
		res.Status = "FAILED"
		return res, err
	}
	res.TablesScanned = tables

	stats, err := s.repo.UpsertBatch(ctx, objs, edges, UpsertOptions{
		StrictMonotonic:     false,
		EmitOutbox:          true,
		SkipNoOpIfUnchanged: true,
		ChangeKind:          "REBUILD",
		SourceEventType:     "ontology.rebuild",
		SourceEventID:       fmt.Sprintf("rebuild-%d", started.UnixNano()),
	})
	if err != nil {
		res.Status = "FAILED"
		return res, err
	}
	res.ObjectsUpserted = stats.ObjectsUpserted
	res.EdgesUpserted = stats.EdgesUpserted
	res.BatchesCommitted = stats.BatchesCommitted
	res.DurationMs = s.now().Sub(started).Milliseconds()
	return res, nil
}

// RebuildAllFromSpanner reconciles every active supplier sequentially for nightly drift sweeper execution.
func (s *Service) RebuildAllFromSpanner(ctx context.Context) (RebuildResult, error) {
	return s.RebuildFromSpanner(ctx, s.defaultCell, "")
}

// ── Kafka domain-event projection ────────────────────────────────────────────

// EventConsumer implements Kafka Consumer handler parity (like twin.EventConsumer)
// for the void-ontology-projector consumer group.
type EventConsumer struct {
	service *Service
	log     *slog.Logger
}

// NewEventConsumer wraps Service into a Kafka consumer handler.
func NewEventConsumer(service *Service, log *slog.Logger) *EventConsumer {
	if log == nil {
		log = slog.Default()
	}
	return &EventConsumer{service: service, log: log}
}

// HandleEvent consumes one Kafka message and projects domain state into OntologyObjects/Edges.
func (c *EventConsumer) HandleEvent(ctx context.Context, msg kafka.Message) error {
	if c == nil || c.service == nil {
		return nil
	}
	ctx = pegasuskafka.WithTraceFromMessage(ctx, msg)
	env, err := pegasuskafka.EnvelopeFromMessage(msg)
	if err != nil {
		c.log.ErrorContext(ctx, "ontology projector: invalid kafka envelope", "err", err, "topic", msg.Topic, "offset", msg.Offset)
		return err
	}
	eventType := strings.TrimSpace(env.Type)
	if eventType == "" {
		// Fallback to raw payload "event_type" if headers/type absent.
		if rawMap, decErr := decodeKafkaPayloadMap(msg.Value); decErr == nil {
			eventType = firstNonEmpty(propString(rawMap, "type"), propString(rawMap, "event_type"))
		}
	}
	if eventType == "" {
		return nil
	}
	return c.service.HandleDomainEvent(ctx, eventType, env.Version, msg.Value)
}

// HandleDomainEvent projects a typed domain event payload (JSON bytes) into typed v1 ontology objects & edges.
func (s *Service) HandleDomainEvent(ctx context.Context, eventType string, envelopeVersion int64, rawPayload []byte) error {
	if s == nil || s.repo == nil {
		return ErrRepoUnavailable
	}
	evType := strings.TrimSpace(eventType)
	if evType == "" || evType == events.EventOntologyObjectChanged {
		return nil
	}

	payload, err := decodeKafkaPayloadMap(rawPayload)
	if err != nil {
		return err
	}
	if evType == "" {
		evType = firstNonEmpty(propString(payload, "type"), propString(payload, "event_type"))
	}

	cellID := s.resolveCellID(propString(payload, "cell_id"))
	supID := strings.TrimSpace(firstNonEmpty(propString(payload, "supplier_id"), propString(payload, "entity_id")))
	version := extractIncomingVersion(payload, envelopeVersion)
	eventID := firstNonEmpty(propString(payload, "event_id"), propString(payload, "trace_id"), propString(payload, "id"))

	objs, edges := s.buildProjectionForEvent(ctx, cellID, supID, evType, version, payload)
	if len(objs) == 0 && len(edges) == 0 {
		return nil
	}

	_, err = s.repo.UpsertBatch(ctx, objs, edges, UpsertOptions{
		StrictMonotonic: true,
		EmitOutbox:      true,
		ChangeKind:      "PROJECTED",
		SourceEventType: evType,
		SourceEventID:   eventID,
	})
	return err
}

func (s *Service) resolveCellID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		return trimmed
	}
	if strings.TrimSpace(s.defaultCell) != "" {
		return s.defaultCell
	}
	return DefaultCellID
}

func (s *Service) buildProjectionForEvent(ctx context.Context, cellID, supID, evType string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	lower := strings.ToLower(evType)
	upper := strings.ToUpper(evType)

	switch {
	case strings.HasPrefix(lower, "order.") || strings.HasPrefix(upper, "ORDER_") ||
		strings.HasPrefix(upper, "PRE_ORDER_") || strings.HasPrefix(upper, "PARENT_ORDER_") ||
		upper == events.EventMissingItemsReported || upper == events.EventPartialOffload || upper == events.EventCreditLeave:
		return s.projectOrderDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(lower, "stocklot.") || strings.HasPrefix(lower, "lot.") ||
		strings.HasPrefix(upper, "LOT_") || lower == events.EventInventoryLotReleased:
		return s.projectLotDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(lower, "inventory.") || strings.HasPrefix(upper, "INVENTORY_") ||
		strings.HasPrefix(upper, "WMS_") || strings.HasPrefix(upper, "PRODUCT_") ||
		strings.HasPrefix(upper, "RETAILER_PRICE_OVERRIDE"):
		return s.projectInventoryAndSKUDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(lower, "manifest.") || strings.HasPrefix(upper, "MANIFEST_") ||
		strings.HasPrefix(upper, "ROUTE_") || upper == events.EventSplitShipmentCreated:
		return s.projectManifestDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(lower, "credit.") || strings.HasPrefix(upper, "RETAILER_CREDIT_") ||
		strings.HasPrefix(upper, "SUPPLIER_CREDIT_") || strings.HasPrefix(upper, "CREDIT_DELIVERY_"):
		return s.projectCreditDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(lower, "ar.") || strings.HasPrefix(upper, "AR_INVOICE_"):
		return s.projectARDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(lower, "payment.") || strings.HasPrefix(upper, "PAYMENT_") ||
		strings.HasPrefix(upper, "SPLIT_PAYMENT_") || strings.HasPrefix(upper, "REFUND_") ||
		upper == events.EventSettlementRequired || upper == events.EventCashShortfall ||
		upper == events.EventCashOverage || upper == events.EventAMLExposureUpdated:
		return s.projectPaymentDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(lower, "fiscal.") || strings.HasPrefix(upper, "FISCAL_") ||
		strings.HasPrefix(upper, "BUYER_ACCEPTANCE_"):
		return s.projectFiscalDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(lower, "gl.") || strings.HasPrefix(lower, "recon."):
		return s.projectGLAndReconDomain(ctx, cellID, supID, lower, version, p)

	case strings.HasPrefix(upper, "NEGOTIATION_") || strings.HasPrefix(upper, "AI_RECOMMENDATION_") ||
		strings.HasPrefix(upper, "REPLENISHMENT_"):
		return s.projectProposalDomain(cellID, supID, upper, version, p)

	case strings.HasPrefix(upper, "SUPPLIER_") || strings.HasPrefix(upper, "WAREHOUSE_") ||
		strings.HasPrefix(upper, "FACTORY_") || strings.HasPrefix(upper, "RETAILER_") ||
		strings.HasPrefix(upper, "DRIVER_") || strings.HasPrefix(upper, "VEHICLE_") ||
		strings.HasPrefix(upper, "FLEET_DVIR_") || upper == events.EventOntologyObjectUpdated:
		return s.projectMasterEntityDomain(cellID, supID, upper, version, p)
	}
	return nil, nil
}

func (s *Service) projectOrderDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	oid := propString(p, "order_id")
	if oid == "" {
		return nil, nil
	}
	rid := propString(p, "retailer_id")
	wid := propString(p, "warehouse_id")
	did := propString(p, "driver_id")
	vid := propString(p, "vehicle_id")
	mid := propString(p, "manifest_id")
	currency := firstNonEmpty(propString(p, "currency"), "UZS")
	status := firstNonEmpty(propString(p, "status"), upper)
	totalMinor := propInt64(p, "total_minor")

	props := map[string]any{
		"order_id":                oid,
		"parent_order_id":         propString(p, "parent_order_id"),
		"supplier_id":             supID,
		"retailer_id":             rid,
		"warehouse_id":            wid,
		"driver_id":               did,
		"vehicle_id":              vid,
		"route_id":                propString(p, "route_id"),
		"manifest_id":             mid,
		"status":                  status,
		"order_source":            propString(p, "order_source"),
		"confirmation_status":     propString(p, "confirmation_status"),
		"total_minor":             totalMinor,
		"currency":                currency,
		"h3_cell":                 propString(p, "h3_cell"),
		"requested_delivery_date": propString(p, "requested_delivery_date"),
		"version":                 version,
	}
	if orig, ok := p["original_total_minor"]; ok {
		props["original_total_minor"] = asInt64(orig)
	}
	if dfee, ok := p["delivery_fee_minor"]; ok {
		props["delivery_fee_minor"] = asInt64(dfee)
	}

	markings := []string{MarkingFinanceOnly}
	if strings.Contains(upper, "QUARANTINE") || strings.Contains(strings.ToUpper(status), "QUARANTINE") {
		markings = append(markings, MarkingDiscrepantQuarantine)
	}
	objs := []*OntologyObject{{
		CellID:           cellID,
		ObjectType:       TypeOrder,
		ObjectID:         oid,
		SupplierID:       supID,
		Version:          version,
		Properties:       props,
		SecurityMarkings: markings,
	}}
	var edges []*OntologyGraphEdge
	if rid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeShop, rid, LinkOrdered, TypeOrder, oid, supID))
	}
	if wid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeWarehouse, wid, LinkSupplies, TypeOrder, oid, supID))
	}
	if mid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeOrder, oid, LinkShippedOn, TypeManifest, mid, supID))
	}
	if did != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeOrder, oid, LinkShippedOn, TypeDriver, did, supID))
	}
	if vid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeOrder, oid, LinkShippedOn, TypeVehicle, vid, supID))
	}

	// Project OrderLine objects if line_items present on the event.
	if rawLines, ok := p["line_items"].([]any); ok {
		for idx, item := range rawLines {
			lm, ok := item.(map[string]any)
			if !ok {
				continue
			}
			pid := firstNonEmpty(propString(lm, "product_id"), propString(lm, "sku"))
			if pid == "" {
				continue
			}
			lineID := firstNonEmpty(propString(lm, "order_line_id"), fmt.Sprintf("%s#%s", oid, pid))
			if lineID == oid+"#" {
				lineID = fmt.Sprintf("%s#%d", oid, idx)
			}
			qty := firstPositiveInt64(propInt64(lm, "quantity"), 1)
			unitPrice := propInt64(lm, "unit_price_minor")
			lineTotal := propInt64(lm, "line_total_minor")
			if lineTotal == 0 && unitPrice > 0 {
				lineTotal = unitPrice * qty
			}
			objs = append(objs, &OntologyObject{
				CellID:           cellID,
				ObjectType:       TypeOrderLine,
				ObjectID:         lineID,
				SupplierID:       supID,
				Version:          version,
				SecurityMarkings: []string{MarkingFinanceOnly},
				Properties: map[string]any{
					"order_line_id":    lineID,
					"order_id":         oid,
					"supplier_id":      supID,
					"warehouse_id":     wid,
					"product_id":       pid,
					"sku":              firstNonEmpty(propString(lm, "sku"), pid),
					"name":             firstNonEmpty(propString(lm, "name"), pid),
					"quantity":         qty,
					"unit_price_minor": unitPrice,
					"line_total_minor": lineTotal,
					"currency":         currency,
				},
			})
			edges = append(edges,
				makeProjectedEdge(cellID, TypeOrder, oid, LinkOrdered, TypeOrderLine, lineID, supID),
				makeProjectedEdge(cellID, TypeOrderLine, lineID, LinkOrdered, TypeSKU, pid, supID),
			)
		}
	}
	return objs, edges
}

func (s *Service) projectLotDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	var objs []*OntologyObject
	var edges []*OntologyGraphEdge

	// InventoryLotReleasedEvent encodes released_lots: [{lot_id, quantity}].
	if rawReleased, ok := p["released_lots"].([]any); ok && len(rawReleased) > 0 {
		oid := propString(p, "order_id")
		wid := propString(p, "warehouse_id")
		for _, item := range rawReleased {
			lm, ok := item.(map[string]any)
			if !ok {
				continue
			}
			lotID := propString(lm, "lot_id")
			if lotID == "" {
				continue
			}
			qty := propInt64(lm, "quantity")
			pid := firstNonEmpty(propString(lm, "product_id"), propString(p, "product_id"), "unknown-sku")
			objs = append(objs, &OntologyObject{
				CellID:     cellID,
				ObjectType: TypeLot,
				ObjectID:   lotID,
				SupplierID: supID,
				Version:    version,
				Properties: map[string]any{
					"lot_id":             lotID,
					"supplier_id":        supID,
					"warehouse_id":       wid,
					"product_id":         pid,
					"status":             "ACTIVE",
					"quantity_available": qty,
				},
			})
			if wid != "" {
				edges = append(edges, makeProjectedEdge(cellID, TypeWarehouse, wid, LinkStocks, TypeLot, lotID, supID))
			}
			if oid != "" {
				edges = append(edges, makeProjectedEdge(cellID, TypeLot, lotID, LinkOrdered, TypeOrder, oid, supID))
			}
		}
		return objs, edges
	}

	lotID := firstNonEmpty(propString(p, "lot_id"), propString(p, "lot_code"))
	if lotID == "" {
		return nil, nil
	}
	wid := firstNonEmpty(propString(p, "warehouse_id"), "default")
	pid := firstNonEmpty(propString(p, "product_id"), propString(p, "sku"), "unknown-sku")
	status := firstNonEmpty(propString(p, "status"), "ACTIVE")
	var markings []string
	if upper == events.EventLotRecallInitiated || strings.Contains(upper, "RECALL") {
		status = "RECALLED"
		markings = append(markings, MarkingLotRecallQuarantine)
	} else if upper == events.EventLotQuarantined || strings.Contains(upper, "QUARANTIN") {
		status = "QUARANTINED"
		markings = append(markings, MarkingDiscrepantQuarantine)
	} else if upper == events.EventLotReleased {
		status = "ACTIVE"
	}

	objs = append(objs, &OntologyObject{
		CellID:           cellID,
		ObjectType:       TypeLot,
		ObjectID:         lotID,
		SupplierID:       supID,
		Version:          version,
		SecurityMarkings: markings,
		Properties: map[string]any{
			"lot_id":             lotID,
			"supplier_id":        supID,
			"warehouse_id":       wid,
			"product_id":         pid,
			"sku":                propString(p, "sku"),
			"lot_code":           propString(p, "lot_code"),
			"status":             status,
			"recall_reason":      propString(p, "recall_reason"),
			"quantity_on_hand":   propInt64(p, "quantity_on_hand"),
			"quantity_reserved":  propInt64(p, "quantity_reserved"),
			"quantity_available": propInt64(p, "quantity_available"),
		},
	})
	if wid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeWarehouse, wid, LinkStocks, TypeLot, lotID, supID))
	}
	if pid != "" && pid != "unknown-sku" {
		edges = append(edges, makeProjectedEdge(cellID, TypeSKU, pid, LinkStocks, TypeLot, lotID, supID))
	}
	if oid := propString(p, "order_id"); oid != "" {
		lt := LinkOrdered
		if len(markings) > 0 {
			lt = LinkRecalledWith
		}
		edges = append(edges, makeProjectedEdge(cellID, TypeLot, lotID, lt, TypeOrder, oid, supID))
	}
	return objs, edges
}

func (s *Service) projectInventoryAndSKUDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	var objs []*OntologyObject
	var edges []*OntologyGraphEdge

	pid := firstNonEmpty(propString(p, "product_id"), propString(p, "sku"))
	wid := propString(p, "warehouse_id")
	lotID := propString(p, "lot_id")

	if upper == events.EventWMSTemperatureBreach {
		if lotID != "" {
			objs = append(objs, &OntologyObject{
				CellID:           cellID,
				ObjectType:       TypeLot,
				ObjectID:         lotID,
				SupplierID:       supID,
				Version:          version,
				SecurityMarkings: []string{MarkingThermalQuarantineSpoiled},
				Properties: map[string]any{
					"lot_id":       lotID,
					"supplier_id":  supID,
					"warehouse_id": firstNonEmpty(wid, "default"),
					"product_id":   firstNonEmpty(pid, "unknown-sku"),
					"status":       "THERMAL_QUARANTINE",
				},
			})
		}
		return objs, edges
	}

	if pid != "" {
		currency := firstNonEmpty(propString(p, "currency"), "UZS")
		name := firstNonEmpty(propString(p, "name"), propString(p, "sku"), pid)
		priceMinor := propInt64(p, "price_minor")
		stockQty := firstNonZeroInt64(propInt64(p, "stock_quantity"), propInt64(p, "quantity_on_hand"), propInt64(p, "projected_units"))

		objs = append(objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeSKU,
			ObjectID:   pid,
			SupplierID: supID,
			Version:    version,
			Properties: map[string]any{
				"product_id":     pid,
				"sku":            firstNonEmpty(propString(p, "sku"), pid),
				"supplier_id":    supID,
				"category_id":    propString(p, "category_id"),
				"name":           name,
				"barcode":        propString(p, "barcode"),
				"price_minor":    priceMinor,
				"currency":       currency,
				"stock_quantity": stockQty,
				"handling_class": propString(p, "handling_class"),
				"is_hazardous":   propBool(p, "is_hazardous"),
				"is_perishable":  propBool(p, "is_perishable"),
				"is_active":      upper != events.EventProductDeactivated,
				"version":        version,
			},
		})
		if supID != "" {
			edges = append(edges, makeProjectedEdge(cellID, TypeSupplier, supID, LinkSupplies, TypeSKU, pid, supID))
		}
		if wid != "" {
			edges = append(edges, makeProjectedEdge(cellID, TypeWarehouse, wid, LinkStocks, TypeSKU, pid, supID))
		}
	}
	if lotID != "" && pid != "" {
		objs = append(objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeLot,
			ObjectID:   lotID,
			SupplierID: supID,
			Version:    version,
			Properties: map[string]any{
				"lot_id":           lotID,
				"supplier_id":      supID,
				"warehouse_id":     firstNonEmpty(wid, "default"),
				"product_id":       pid,
				"status":           firstNonEmpty(propString(p, "status"), "ACTIVE"),
				"quantity_on_hand": propInt64(p, "quantity_on_hand"),
			},
		})
		if wid != "" {
			edges = append(edges, makeProjectedEdge(cellID, TypeWarehouse, wid, LinkStocks, TypeLot, lotID, supID))
		}
		edges = append(edges, makeProjectedEdge(cellID, TypeSKU, pid, LinkStocks, TypeLot, lotID, supID))
	}
	return objs, edges
}

func (s *Service) projectManifestDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	mid := propString(p, "manifest_id")
	if mid == "" {
		return nil, nil
	}
	wid := propString(p, "warehouse_id")
	vid := firstNonEmpty(propString(p, "vehicle_id"), propString(p, "to_vehicle_id"))
	did := firstNonEmpty(propString(p, "driver_id"), propString(p, "to_driver_id"))
	oid := propString(p, "order_id")
	state := firstNonEmpty(propString(p, "state"), propString(p, "status"), stateFromManifestEvent(upper))

	var markings []string
	if upper == events.EventManifestOrderException || upper == events.EventManifestDLQEscalation {
		markings = append(markings, MarkingDiscrepantQuarantine)
	}

	objs := []*OntologyObject{{
		CellID:           cellID,
		ObjectType:       TypeManifest,
		ObjectID:         mid,
		SupplierID:       supID,
		Version:          version,
		SecurityMarkings: markings,
		Properties: map[string]any{
			"manifest_id":  mid,
			"supplier_id":  supID,
			"warehouse_id": wid,
			"route_id":     propString(p, "route_id"),
			"vehicle_id":   vid,
			"driver_id":    did,
			"state":        state,
			"stop_count":   propInt64(p, "stop_count"),
		},
	}}
	var edges []*OntologyGraphEdge
	if vid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeManifest, mid, LinkShippedOn, TypeVehicle, vid, supID))
	}
	if did != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeManifest, mid, LinkShippedOn, TypeDriver, did, supID))
	}
	if wid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeManifest, mid, LinkShippedOn, TypeWarehouse, wid, supID))
	}
	if oid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeOrder, oid, LinkShippedOn, TypeManifest, mid, supID))
	}
	return objs, edges
}

func (s *Service) projectCreditDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	rid := propString(p, "retailer_id")
	id := firstNonEmpty(propString(p, "credit_line_id"), propString(p, "profile_id"))
	if id == "" && supID != "" && rid != "" {
		id = supID + ":" + rid
	}
	if id == "" {
		return nil, nil
	}
	if rid == "" {
		rid = id
	}
	limitMinor := propInt64(p, "credit_limit_minor")
	balanceMinor := firstNonZeroInt64(propInt64(p, "current_balance_minor"), propInt64(p, "current_balance"))
	reservedMinor := propInt64(p, "reserved_minor")
	availMinor := propInt64(p, "available_credit_minor")
	if availMinor == 0 && limitMinor > 0 {
		availMinor = limitMinor - balanceMinor - reservedMinor
		if availMinor < 0 {
			availMinor = 0
		}
	}
	status := firstNonEmpty(propString(p, "status"), propString(p, "risk_tier"), "ACTIVE")
	if upper == events.EventRetailerCreditLimitBreached {
		status = "BREACHED"
	}

	objs := []*OntologyObject{{
		CellID:           cellID,
		ObjectType:       TypeCreditLine,
		ObjectID:         id,
		SupplierID:       supID,
		Version:          version,
		SecurityMarkings: []string{MarkingFinanceOnly},
		Properties: map[string]any{
			"credit_line_id":         id,
			"supplier_id":            supID,
			"retailer_id":            rid,
			"status":                 status,
			"credit_limit_minor":     limitMinor,
			"current_balance_minor":  balanceMinor,
			"reserved_minor":         reservedMinor,
			"available_credit_minor": availMinor,
			"currency":               firstNonEmpty(propString(p, "currency"), "UZS"),
		},
	}}
	var edges []*OntologyGraphEdge
	if rid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeShop, rid, LinkOwes, TypeCreditLine, id, supID))
	}
	return objs, edges
}

func (s *Service) projectARDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	invID := propString(p, "invoice_id")
	if invID == "" {
		return nil, nil
	}
	rid := firstNonEmpty(propString(p, "retailer_id"), "unknown-retailer")
	oid := propString(p, "order_id")
	principal := propInt64(p, "principal_minor")
	balance := propInt64(p, "balance_minor")
	amount := propInt64(p, "amount_minor")
	if principal == 0 && amount > 0 {
		principal = amount
	}
	status := firstNonEmpty(propString(p, "status"), arStatusFromEvent(upper))

	objs := []*OntologyObject{{
		CellID:           cellID,
		ObjectType:       TypeNasiyaInvoice,
		ObjectID:         invID,
		SupplierID:       supID,
		Version:          version,
		SecurityMarkings: []string{MarkingFinanceOnly},
		Properties: map[string]any{
			"invoice_id":      invID,
			"supplier_id":     supID,
			"retailer_id":     rid,
			"order_id":        oid,
			"status":          status,
			"principal_minor": principal,
			"balance_minor":   balance,
			"currency":        firstNonEmpty(propString(p, "currency"), "UZS"),
			"aging_bucket":    propString(p, "aging_bucket"),
			"dunning_step":    propInt64(p, "dunning_step"),
		},
	}}
	var edges []*OntologyGraphEdge
	if rid != "" && rid != "unknown-retailer" {
		edges = append(edges, makeProjectedEdge(cellID, TypeShop, rid, LinkOwes, TypeNasiyaInvoice, invID, supID))
	}
	if oid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeOrder, oid, LinkOwes, TypeNasiyaInvoice, invID, supID))
	}
	return objs, edges
}

func (s *Service) projectPaymentDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	if upper == events.EventAMLExposureUpdated {
		payerID := firstNonEmpty(propString(p, "legal_entity_id"), propString(p, "payer_id"), propString(p, "normalized_tax_id"))
		if payerID == "" {
			return nil, nil
		}
		markings := []string{MarkingFinanceOnly, MarkingPII}
		if propBool(p, "aml_flagged") || propInt64(p, "daily_cash_accumulated_minor") > propInt64(p, "daily_cash_ceiling_minor") && propInt64(p, "daily_cash_ceiling_minor") > 0 {
			markings = append(markings, MarkingAMLInvestigation)
		}
		return []*OntologyObject{{
			CellID:           cellID,
			ObjectType:       TypeLegalEntity,
			ObjectID:         payerID,
			SupplierID:       supID,
			Version:          version,
			SecurityMarkings: markings,
			Properties: map[string]any{
				"legal_entity_id":              payerID,
				"supplier_id":                  firstNonEmpty(supID, payerID),
				"legal_name":                   firstNonEmpty(propString(p, "legal_name"), payerID),
				"tin":                          propString(p, "tin"),
				"daily_cash_accumulated_minor": propInt64(p, "daily_cash_accumulated_minor"),
				"daily_cash_ceiling_minor":     propInt64(p, "daily_cash_ceiling_minor"),
				"currency":                     firstNonEmpty(propString(p, "currency"), "UZS"),
			},
		}}, nil
	}

	payID := firstNonEmpty(propString(p, "payment_id"), propString(p, "session_id"), propString(p, "transaction_id"), propString(p, "order_id"))
	if payID == "" {
		return nil, nil
	}
	oid := propString(p, "order_id")
	invID := propString(p, "invoice_id")
	rid := propString(p, "retailer_id")
	amount := firstNonZeroInt64(propInt64(p, "amount_minor"), propInt64(p, "cash_minor")+propInt64(p, "card_minor"))
	status := firstNonEmpty(propString(p, "status"), upper)
	markings := []string{MarkingFinanceOnly}
	if upper == events.EventCashShortfall || upper == events.EventCashOverage {
		markings = append(markings, MarkingDiscrepantQuarantine)
		amount = firstNonZeroInt64(amount, propInt64(p, "shortfall_minor"), propInt64(p, "overage_minor"))
	}

	objs := []*OntologyObject{{
		CellID:           cellID,
		ObjectType:       TypePayment,
		ObjectID:         payID,
		SupplierID:       supID,
		Version:          version,
		SecurityMarkings: markings,
		Properties: map[string]any{
			"payment_id":         payID,
			"session_id":         propString(p, "session_id"),
			"order_id":           oid,
			"invoice_id":         invID,
			"supplier_id":        supID,
			"retailer_id":        rid,
			"gateway":            propString(p, "gateway"),
			"mode":               propString(p, "execution_mode"),
			"status":             status,
			"amount_minor":       amount,
			"cash_minor":         propInt64(p, "cash_minor"),
			"card_minor":         propInt64(p, "card_minor"),
			"currency":           firstNonEmpty(propString(p, "currency"), "UZS"),
			"provider_reference": propString(p, "provider_reference"),
		},
	}}
	var edges []*OntologyGraphEdge
	if oid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypePayment, payID, LinkOrdered, TypeOrder, oid, supID))
	}
	if invID != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypePayment, payID, LinkOwes, TypeNasiyaInvoice, invID, supID))
	}
	return objs, edges
}

func (s *Service) projectFiscalDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	oid := propString(p, "order_id")
	docID := firstNonEmpty(propString(p, "fiscal_doc_id"), propString(p, "attempt_id"), propString(p, "ehf_id"), propString(p, "fiscal_receipt_id"), oid)
	if docID == "" {
		return nil, nil
	}
	if oid == "" {
		oid = docID
	}
	provider := firstNonEmpty(propString(p, "provider"), "SOLIQ_OFD")
	status := firstNonEmpty(propString(p, "status"), upper)
	amount := propInt64(p, "amount_minor")
	currency := firstNonEmpty(propString(p, "currency"), "UZS")

	objs := []*OntologyObject{{
		CellID:           cellID,
		ObjectType:       TypeFiscalDoc,
		ObjectID:         docID,
		SupplierID:       supID,
		Version:          version,
		SecurityMarkings: []string{MarkingFinanceOnly},
		Properties: map[string]any{
			"fiscal_doc_id":     docID,
			"attempt_id":        propString(p, "attempt_id"),
			"order_id":          oid,
			"supplier_id":       supID,
			"retailer_id":       propString(p, "retailer_id"),
			"provider":          provider,
			"status":            status,
			"fiscal_receipt_id": propString(p, "fiscal_receipt_id"),
			"ehf_id":            propString(p, "ehf_id"),
			"amount_minor":      amount,
			"currency":          currency,
			"payment_method":    propString(p, "payment_method"),
			"fiscal_qr":         propString(p, "fiscal_qr"),
		},
	}}
	edges := []*OntologyGraphEdge{
		makeProjectedEdge(cellID, TypeFiscalDoc, docID, LinkOrdered, TypeOrder, oid, supID),
	}
	return objs, edges
}

func (s *Service) projectGLAndReconDomain(ctx context.Context, cellID, supID, lower string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	if strings.HasPrefix(lower, "recon.break.") {
		objType := propString(p, "object_type")
		objID := propString(p, "object_id")
		if _, ok := LookupType(objType); !ok || objID == "" {
			return nil, nil
		}
		if strings.EqualFold(lower, events.EventReconBreakOpened) && s.repo != nil {
			existing, err := s.repo.GetObject(ctx, cellID, objType, objID)
			if err != nil || existing == nil {
				return nil, nil
			}
			cp := existing.Clone()
			cp.Version = version
			cp.SecurityMarkings = MergeMarkings(cp.SecurityMarkings, []string{MarkingFinanceOnly, MarkingDiscrepantQuarantine})
			return []*OntologyObject{cp}, nil
		}
		return nil, nil
	}

	// gl.entry.posted / gl.entry.reversed
	srcType := strings.ToUpper(propString(p, "source_type"))
	srcID := propString(p, "source_id")
	if srcID == "" {
		return nil, nil
	}
	amount := firstNonZeroInt64(propInt64(p, "total_debit_minor"), propInt64(p, "total_credit_minor"))
	currency := firstNonEmpty(propString(p, "currency"), "UZS")
	glStatus := "GL_POSTED"
	if lower == events.EventGLEntryReversed {
		glStatus = "GL_REVERSED"
	}

	switch srcType {
	case "INVOICE", "AR_INVOICE":
		rid := firstNonEmpty(propString(p, "retailer_id"), "unknown-retailer")
		return []*OntologyObject{{
			CellID:           cellID,
			ObjectType:       TypeNasiyaInvoice,
			ObjectID:         srcID,
			SupplierID:       supID,
			Version:          version,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"invoice_id":      srcID,
				"supplier_id":     supID,
				"retailer_id":     rid,
				"status":          glStatus,
				"principal_minor": amount,
				"balance_minor":   amount,
				"currency":        currency,
				"gl_entry_id":     propString(p, "entry_id"),
				"period_id":       propString(p, "period_id"),
			},
		}}, nil
	case "PAYMENT", "SETTLEMENT", "CASH_COLLECT":
		return []*OntologyObject{{
			CellID:           cellID,
			ObjectType:       TypePayment,
			ObjectID:         srcID,
			SupplierID:       supID,
			Version:          version,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"payment_id":   srcID,
				"supplier_id":  supID,
				"status":       glStatus,
				"amount_minor": amount,
				"currency":     currency,
				"gl_entry_id":  propString(p, "entry_id"),
				"period_id":    propString(p, "period_id"),
			},
		}}, nil
	case "ORDER":
		rid := firstNonEmpty(propString(p, "retailer_id"), "unknown-retailer")
		return []*OntologyObject{{
			CellID:           cellID,
			ObjectType:       TypeOrder,
			ObjectID:         srcID,
			SupplierID:       supID,
			Version:          version,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"order_id":    srcID,
				"supplier_id": supID,
				"retailer_id": rid,
				"status":      glStatus,
				"total_minor": amount,
				"currency":    currency,
				"gl_entry_id": propString(p, "entry_id"),
			},
		}}, nil
	}
	return nil, nil
}

func (s *Service) projectProposalDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	pid := firstNonEmpty(propString(p, "proposal_id"), propString(p, "negotiation_id"), propString(p, "recommendation_id"), propString(p, "insight_id"))
	if pid == "" {
		return nil, nil
	}
	kind := "REPLENISHMENT"
	if strings.HasPrefix(upper, "NEGOTIATION_") {
		kind = "NEGOTIATION"
	} else if strings.HasPrefix(upper, "AI_RECOMMENDATION_") {
		kind = "AI_RECOMMENDATION"
	}
	status := firstNonEmpty(propString(p, "status"), propString(p, "decision"), "PROPOSED")
	oid := propString(p, "order_id")
	sku := propString(p, "product_id")
	if sku == "" {
		sku = propString(p, "sku")
	}

	objs := []*OntologyObject{{
		CellID:     cellID,
		ObjectType: TypeProposal,
		ObjectID:   pid,
		SupplierID: supID,
		Version:    version,
		Properties: map[string]any{
			"proposal_id":           pid,
			"proposal_kind":         kind,
			"supplier_id":           supID,
			"retailer_id":           propString(p, "retailer_id"),
			"order_id":              oid,
			"driver_id":             propString(p, "driver_id"),
			"warehouse_id":          propString(p, "warehouse_id"),
			"sku":                   sku,
			"status":                status,
			"proposed_amount_minor": propInt64(p, "proposed_price_minor"),
			"reason_code":           propString(p, "reason_code"),
			"resolution":            propString(p, "resolution"),
		},
	}}
	var edges []*OntologyGraphEdge
	if oid != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeProposal, pid, LinkOrdered, TypeOrder, oid, supID))
	}
	if sku != "" {
		edges = append(edges, makeProjectedEdge(cellID, TypeProposal, pid, LinkStocks, TypeSKU, sku, supID))
	}
	return objs, edges
}

func (s *Service) projectMasterEntityDomain(cellID, supID, upper string, version int64, p map[string]any) ([]*OntologyObject, []*OntologyGraphEdge) {
	switch {
	case strings.HasPrefix(upper, "SUPPLIER_"):
		if supID == "" {
			return nil, nil
		}
		var objs []*OntologyObject
		objs = append(objs, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeSupplier,
			ObjectID:   supID,
			SupplierID: supID,
			Version:    version,
			Properties: map[string]any{
				"supplier_id":   supID,
				"name":          firstNonEmpty(propString(p, "name"), propString(p, "legal_name"), supID),
				"country_code":  firstNonEmpty(propString(p, "country"), propString(p, "country_code"), "UZ"),
				"currency":      firstNonEmpty(propString(p, "currency"), "UZS"),
				"is_configured": propBool(p, "is_configured"),
				"is_registered": propBool(p, "is_registered"),
			},
		})
		if legalName := propString(p, "legal_name"); legalName != "" {
			objs = append(objs, &OntologyObject{
				CellID:           cellID,
				ObjectType:       TypeLegalEntity,
				ObjectID:         supID,
				SupplierID:       supID,
				Version:          version,
				SecurityMarkings: []string{MarkingFinanceOnly, MarkingPII},
				Properties: map[string]any{
					"legal_entity_id": supID,
					"supplier_id":     supID,
					"legal_name":      legalName,
					"contact_name":    propString(p, "contact_name"),
					"email":           propString(p, "email"),
					"phone":           propString(p, "phone"),
					"bank_name":       propString(p, "bank_name"),
					"account_holder":  propString(p, "account_holder"),
				},
			})
		}
		return objs, nil

	case strings.HasPrefix(upper, "WAREHOUSE_"):
		wid := propString(p, "warehouse_id")
		if wid == "" {
			return nil, nil
		}
		objs := []*OntologyObject{{
			CellID:     cellID,
			ObjectType: TypeWarehouse,
			ObjectID:   wid,
			SupplierID: supID,
			Version:    version,
			Properties: map[string]any{
				"warehouse_id": wid,
				"supplier_id":  supID,
				"name":         firstNonEmpty(propString(p, "name"), wid),
				"is_active":    true,
			},
		}}
		var edges []*OntologyGraphEdge
		if supID != "" {
			edges = append(edges, makeProjectedEdge(cellID, TypeSupplier, supID, LinkSupplies, TypeWarehouse, wid, supID))
		}
		if fid := propString(p, "factory_id"); fid != "" {
			edges = append(edges, makeProjectedEdge(cellID, TypeFactory, fid, LinkSupplies, TypeWarehouse, wid, supID))
		}
		return objs, edges

	case strings.HasPrefix(upper, "FACTORY_"):
		fid := propString(p, "factory_id")
		if fid == "" {
			return nil, nil
		}
		objs := []*OntologyObject{{
			CellID:     cellID,
			ObjectType: TypeFactory,
			ObjectID:   fid,
			SupplierID: supID,
			Version:    version,
			Properties: map[string]any{
				"factory_id":  fid,
				"supplier_id": supID,
				"name":        firstNonEmpty(propString(p, "name"), propString(p, "legal_name"), fid),
				"h3_cell":     propString(p, "h3_cell"),
				"is_active":   true,
			},
		}}
		var edges []*OntologyGraphEdge
		if wid := propString(p, "assigned_warehouse_id"); wid != "" {
			edges = append(edges, makeProjectedEdge(cellID, TypeFactory, fid, LinkSupplies, TypeWarehouse, wid, supID))
		}
		return objs, edges

	case strings.HasPrefix(upper, "RETAILER_"):
		rid := propString(p, "retailer_id")
		if rid == "" {
			return nil, nil
		}
		objs := []*OntologyObject{{
			CellID:           cellID,
			ObjectType:       TypeShop,
			ObjectID:         rid,
			SupplierID:       supID,
			Version:          version,
			SecurityMarkings: []string{MarkingPII},
			Properties: map[string]any{
				"retailer_id":  rid,
				"supplier_id":  supID,
				"name":         firstNonEmpty(propString(p, "name"), propString(p, "legal_name"), rid),
				"phone":        propString(p, "phone"),
				"email":        propString(p, "email"),
				"h3_cell":      propString(p, "h3_cell"),
				"country_code": propString(p, "country_code"),
			},
		}}
		var edges []*OntologyGraphEdge
		if supID != "" {
			edges = append(edges, makeProjectedEdge(cellID, TypeSupplier, supID, LinkSupplies, TypeShop, rid, supID))
		}
		return objs, edges

	case strings.HasPrefix(upper, "DRIVER_"):
		did := propString(p, "driver_id")
		if did == "" {
			return nil, nil
		}
		objs := []*OntologyObject{{
			CellID:           cellID,
			ObjectType:       TypeDriver,
			ObjectID:         did,
			SupplierID:       supID,
			Version:          version,
			SecurityMarkings: []string{MarkingPII},
			Properties: map[string]any{
				"driver_id":      did,
				"supplier_id":    supID,
				"name":           firstNonEmpty(propString(p, "name"), did),
				"phone":          propString(p, "phone"),
				"home_node_type": propString(p, "home_node_type"),
				"home_node_id":   propString(p, "home_node_id"),
				"on_shift":       propBool(p, "on_shift"),
				"is_active":      true,
			},
		}}
		return objs, nil

	case strings.HasPrefix(upper, "VEHICLE_") || strings.HasPrefix(upper, "FLEET_DVIR_"):
		vid := propString(p, "vehicle_id")
		if vid == "" {
			return nil, nil
		}
		markings := []string(nil)
		if upper == events.EventFleetDVIRRedTag || propBool(p, "red_tag_applied") {
			markings = []string{MarkingDiscrepantQuarantine}
		}
		return []*OntologyObject{{
			CellID:           cellID,
			ObjectType:       TypeVehicle,
			ObjectID:         vid,
			SupplierID:       supID,
			Version:          version,
			SecurityMarkings: markings,
			Properties: map[string]any{
				"vehicle_id":         vid,
				"supplier_id":        supID,
				"license_plate":      firstNonEmpty(propString(p, "license_plate"), vid),
				"home_node_type":     propString(p, "home_node_type"),
				"home_node_id":       propString(p, "home_node_id"),
				"is_active":          propBool(p, "is_active"),
				"unavailable_reason": propString(p, "unavailable_reason"),
				"red_tag_applied":    upper == events.EventFleetDVIRRedTag || propBool(p, "red_tag_applied"),
			},
		}}, nil
	}
	return nil, nil
}

func makeProjectedEdge(cellID, srcType, srcID, linkType, dstType, dstID, supID string) *OntologyGraphEdge {
	return &OntologyGraphEdge{
		CellID:            cellID,
		SourceObjectType:  srcType,
		SourceObjectID:    srcID,
		LinkType:          linkType,
		TargetObjectType:  dstType,
		TargetObjectID:    dstID,
		SupplierID:        supID,
		CreatedByActionID: "kafka-ontology-projector",
	}
}

// decodeKafkaPayloadMap parses raw JSON with json.UseNumber(), converting integers to int64,
// rejecting floats on monetary/integer fields, and stripping non-monetary telemetry floats (lat/lng/volume).
func decodeKafkaPayloadMap(raw []byte) (map[string]any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("ontology: decode kafka payload: %w", err)
	}
	sanitized, err := sanitizeKafkaMap(m)
	if err != nil {
		return nil, err
	}
	return sanitized, nil
}

func sanitizeKafkaMap(m map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if nonMoneyFloatKeys[strings.ToLower(k)] {
			// Strip geo/volume telemetry floats so zero-float property invariant holds strictly.
			continue
		}
		nv, err := sanitizeKafkaValue(k, v)
		if err != nil {
			return nil, err
		}
		out[k] = nv
	}
	return out, nil
}

func sanitizeKafkaValue(field string, v any) (any, error) {
	switch tv := v.(type) {
	case json.Number:
		s := tv.String()
		if strings.ContainsAny(s, ".eE") {
			return nil, fmt.Errorf("%w: field %s (%s)", ErrFloatForbidden, field, s)
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("ontology: invalid integer on %s (%q): %w", field, s, err)
		}
		return n, nil
	case map[string]any:
		return sanitizeKafkaMap(tv)
	case []any:
		out := make([]any, len(tv))
		for i, item := range tv {
			nv, err := sanitizeKafkaValue(fmt.Sprintf("%s[%d]", field, i), item)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}

func extractIncomingVersion(p map[string]any, envelopeVer int64) int64 {
	if v := propInt64(p, "version"); v > 0 {
		return v
	}
	if v := propInt64(p, "v"); v > 0 {
		return v
	}
	if envelopeVer > 0 {
		return envelopeVer
	}
	return 0
}

func propString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch tv := v.(type) {
	case string:
		return strings.TrimSpace(tv)
	default:
		return strings.TrimSpace(fmt.Sprint(tv))
	}
}

func propInt64(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	return asInt64(v)
}

func asInt64(v any) int64 {
	switch tv := v.(type) {
	case int64:
		return tv
	case int:
		return int64(tv)
	case int32:
		return int64(tv)
	case uint64:
		return int64(tv)
	case json.Number:
		n, _ := strconv.ParseInt(tv.String(), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(tv), 10, 64)
		return n
	default:
		return 0
	}
}

func propBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	b, ok := m[key].(bool)
	return ok && b
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstNonZeroInt64(vals ...int64) int64 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}

func firstPositiveInt64(vals ...int64) int64 {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
}

func stateFromManifestEvent(upper string) string {
	switch upper {
	case events.EventManifestDraftCreated:
		return "DRAFT"
	case events.EventManifestLoadingStarted:
		return "LOADING"
	case events.EventManifestSealed:
		return "SEALED"
	case events.EventManifestDispatched:
		return "DISPATCHED"
	case events.EventManifestCompleted:
		return "COMPLETED"
	case events.EventManifestCancelled:
		return "CANCELLED"
	default:
		return upper
	}
}

func arStatusFromEvent(upper string) string {
	switch upper {
	case events.EventARInvoiceOpened:
		return "OPEN"
	case events.EventARInvoiceSettled:
		return "SETTLED"
	case events.EventARInvoiceDunned:
		return "DUNNED"
	case events.EventARInvoicePayment:
		return "PARTIALLY_PAID"
	default:
		return "OPEN"
	}
}
