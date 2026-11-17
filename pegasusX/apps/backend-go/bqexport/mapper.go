package bqexport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	kgo "github.com/segmentio/kafka-go"
)

// MapKafkaMessage converts a Kafka domain event message into one or more canonical BigQuery rows.
// Every valid JSON domain event emits a `raw.domain_events` row, and domain-specific events also
// project into `raw.ontology_objects`, `curated.gl_entries`, or `curated.recon_runs`.
func MapKafkaMessage(msg kgo.Message, defaultCellID string) (string, []Row, error) {
	trimmed := bytes.TrimSpace(msg.Value)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return "", nil, fmt.Errorf("%w: topic=%s partition=%d offset=%d", ErrMalformedEventPayload, msg.Topic, msg.Partition, msg.Offset)
	}

	var envelope map[string]any
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	if err := dec.Decode(&envelope); err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrMalformedEventPayload, err)
	}

	headers := headersToMap(msg.Headers)
	now := time.Now().UTC()

	eventID := firstNonEmpty(
		stringField(envelope, "event_id"),
		stringField(envelope, "id"),
		headers["event_id"],
		fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset),
	)
	eventType := firstNonEmpty(
		stringField(envelope, "type"),
		stringField(envelope, "event_type"),
		headers["type"],
		"UNKNOWN",
	)
	aggregateType := firstNonEmpty(
		stringField(envelope, "aggregate_type"),
		headers["aggregate_type"],
	)
	aggregateID := firstNonEmpty(
		stringField(envelope, "aggregate_id"),
		headers["aggregate_id"],
		string(msg.Key),
	)
	cellID := firstNonEmpty(
		stringField(envelope, "cell_id"),
		stringField(envelope, "home_cell"),
		headers["cell_id"],
		defaultCellID,
		"uz",
	)
	supplierID := firstNonEmpty(
		stringField(envelope, "supplier_id"),
		stringField(envelope, "entity_id"),
		stringField(envelope, "tenant_id"),
		headers["supplier_id"],
		"platform",
	)
	traceID := firstNonEmpty(
		stringField(envelope, "trace_id"),
		headers["trace_id"],
	)
	schemaVersion := firstNonEmpty(
		stringField(envelope, "schema_version"),
		numStringField(envelope, "version"),
		"1",
	)
	occurredAt := parseTimestampFallback(
		firstNonEmpty(
			stringField(envelope, "occurred_at"),
			stringField(envelope, "timestamp"),
			stringField(envelope, "updated_at"),
			stringField(envelope, "posted_at"),
			headers["timestamp"],
		),
		msg.Time,
		now,
	)

	domainRow := MapDomainEventRow(DomainEventInput{
		EventID:       eventID,
		EventType:     eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		CellID:        cellID,
		SupplierID:    supplierID,
		Topic:         msg.Topic,
		PartitionNum:  int64(msg.Partition),
		KafkaOffset:   msg.Offset,
		TraceID:       traceID,
		SchemaVersion: schemaVersion,
		PayloadJSON:   string(trimmed),
		OccurredAt:    occurredAt,
		IngestedAt:    now,
	})

	rows := make([]Row, 0, 2)
	rows = append(rows, domainRow)

	switch strings.TrimSpace(eventType) {
	case events.EventOntologyObjectChanged, events.EventOntologyObjectUpdated:
		ontoRow, ok, err := mapOntologyFromEnvelope(trimmed, envelope, eventID, eventType, cellID, supplierID, occurredAt, now)
		if err != nil {
			return "", nil, err
		}
		if ok {
			rows = append(rows, ontoRow)
		}
	case events.EventGLEntryPosted, events.EventGLEntryReversed:
		glRow, ok, err := mapGLEntryFromEnvelope(trimmed, envelope, eventID, supplierID, occurredAt, now)
		if err != nil {
			return "", nil, err
		}
		if ok {
			rows = append(rows, glRow)
		}
	case events.EventReconRunCompleted:
		if reconRow, ok := mapReconRunFromEnvelope(trimmed, envelope, supplierID, occurredAt, now); ok {
			rows = append(rows, reconRow)
		}
	}

	return eventID, rows, nil
}

// MapDomainEventRow converts a typed DomainEventInput into a BigQuery Row for raw.domain_events.
func MapDomainEventRow(in DomainEventInput) Row {
	ingestedAt := in.IngestedAt.UTC()
	if ingestedAt.IsZero() {
		ingestedAt = time.Now().UTC()
	}
	occurredAt := in.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = ingestedAt
	}
	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = fmt.Sprintf("%s:%d:%d", in.Topic, in.PartitionNum, in.KafkaOffset)
	}
	data := map[string]any{
		"event_id":      eventID,
		"event_type":    firstNonEmpty(in.EventType, "UNKNOWN"),
		"cell_id":       firstNonEmpty(in.CellID, "uz"),
		"supplier_id":   firstNonEmpty(in.SupplierID, "platform"),
		"topic":         in.Topic,
		"partition_num": in.PartitionNum,
		"kafka_offset":  in.KafkaOffset,
		"payload_json":  in.PayloadJSON,
		"occurred_at":   occurredAt.Format(time.RFC3339Nano),
		"ingested_at":   ingestedAt.Format(time.RFC3339Nano),
	}
	setOptionalString(data, "aggregate_type", in.AggregateType)
	setOptionalString(data, "aggregate_id", in.AggregateID)
	setOptionalString(data, "trace_id", in.TraceID)
	setOptionalString(data, "schema_version", in.SchemaVersion)

	return Row{
		Table:    TableRawDomainEvents,
		InsertID: "evt:" + eventID,
		Data:     data,
	}
}

// MapOntologyObjectRow converts a typed OntologyObjectInput into a BigQuery Row for raw.ontology_objects.
func MapOntologyObjectRow(in OntologyObjectInput) (Row, error) {
	cellID := strings.TrimSpace(in.CellID)
	objType := strings.TrimSpace(in.ObjectType)
	objID := strings.TrimSpace(in.ObjectID)
	if cellID == "" || objType == "" || objID == "" {
		return Row{}, fmt.Errorf("%w: ontology object requires cell_id, object_type, and object_id", ErrMalformedEventPayload)
	}
	version := in.Version
	if version <= 0 {
		version = 1
	}
	ingestedAt := in.IngestedAt.UTC()
	if ingestedAt.IsZero() {
		ingestedAt = time.Now().UTC()
	}
	updatedAt := in.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = ingestedAt
	}

	markings := make([]string, 0, len(in.SecurityMarkings))
	for _, m := range in.SecurityMarkings {
		if t := strings.TrimSpace(m); t != "" {
			markings = append(markings, t)
		}
	}

	data := map[string]any{
		"cell_id":           cellID,
		"object_type":       objType,
		"object_id":         objID,
		"supplier_id":       firstNonEmpty(in.SupplierID, "platform"),
		"version":           version,
		"security_markings": markings,
		"updated_at":        updatedAt.Format(time.RFC3339Nano),
		"ingested_at":       ingestedAt.Format(time.RFC3339Nano),
	}
	setOptionalString(data, "change_kind", in.ChangeKind)
	setOptionalString(data, "source_event_type", in.SourceEventType)
	setOptionalString(data, "source_event_id", in.SourceEventID)
	if strings.TrimSpace(in.PropertiesJSON) != "" && json.Valid([]byte(in.PropertiesJSON)) {
		data["properties_json"] = strings.TrimSpace(in.PropertiesJSON)
	}
	if in.ValidFrom != nil && !in.ValidFrom.IsZero() {
		data["valid_from"] = in.ValidFrom.UTC().Format(time.RFC3339Nano)
	}

	return Row{
		Table:    TableRawOntologyObjects,
		InsertID: fmt.Sprintf("onto:%s:%s:%s:%d", cellID, objType, objID, version),
		Data:     data,
	}, nil
}

// MapGLEntryRow converts a typed GLEntryInput into a BigQuery Row for curated.gl_entries.
func MapGLEntryRow(in GLEntryInput) (Row, error) {
	entityID := strings.TrimSpace(in.EntityID)
	entryID := strings.TrimSpace(in.EntryID)
	if entityID == "" || entryID == "" {
		return Row{}, fmt.Errorf("%w: gl entry requires entity_id and entry_id", ErrMalformedEventPayload)
	}
	if in.TotalDebitMinor < 0 || in.TotalCreditMinor < 0 || in.TotalDebitMinor != in.TotalCreditMinor {
		return Row{}, fmt.Errorf("%w: unbalanced gl entry %s (debit=%d credit=%d)", ErrMalformedEventPayload, entryID, in.TotalDebitMinor, in.TotalCreditMinor)
	}
	ingestedAt := in.IngestedAt.UTC()
	if ingestedAt.IsZero() {
		ingestedAt = time.Now().UTC()
	}
	postedAt := in.PostedAt.UTC()
	if postedAt.IsZero() {
		postedAt = ingestedAt
	}

	data := map[string]any{
		"entity_id":          entityID,
		"entry_id":           entryID,
		"period_id":          firstNonEmpty(in.PeriodID, postedAt.Format("2006-01")),
		"source_type":        firstNonEmpty(in.SourceType, "UNKNOWN"),
		"source_id":          firstNonEmpty(in.SourceID, entryID),
		"currency":           firstNonEmpty(in.Currency, "UZS"),
		"country_pack":       firstNonEmpty(in.CountryPack, "UZ"),
		"total_debit_minor":  in.TotalDebitMinor,
		"total_credit_minor": in.TotalCreditMinor,
		"entry_hash":         firstNonEmpty(in.EntryHash, entryID),
		"posted_at":          postedAt.Format(time.RFC3339Nano),
		"ingested_at":        ingestedAt.Format(time.RFC3339Nano),
	}
	setOptionalString(data, "source_event_id", in.SourceEventID)
	setOptionalString(data, "order_id", in.OrderID)
	setOptionalString(data, "reverses_entry_id", in.ReversesEntryID)
	setOptionalString(data, "prev_hash", in.PrevHash)
	setOptionalString(data, "posted_by", in.PostedBy)
	if in.OccurredAt != nil && !in.OccurredAt.IsZero() {
		data["occurred_at"] = in.OccurredAt.UTC().Format(time.RFC3339Nano)
	}

	return Row{
		Table:    TableCuratedGLEntries,
		InsertID: fmt.Sprintf("gl:%s:%s", entityID, entryID),
		Data:     data,
	}, nil
}

// MapSKUDemandDailyRow converts a typed SKUDemandDailyInput into a BigQuery Row for features.sku_demand_daily.
func MapSKUDemandDailyRow(in SKUDemandDailyInput) (Row, error) {
	if strings.TrimSpace(in.FeatureDate) == "" || strings.TrimSpace(in.SupplierID) == "" || strings.TrimSpace(in.ProductID) == "" {
		return Row{}, fmt.Errorf("%w: sku_demand_daily requires feature_date, supplier_id, and product_id", ErrMalformedEventPayload)
	}
	updatedAt := in.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	cellID := firstNonEmpty(in.CellID, "uz")
	whID := firstNonEmpty(in.WarehouseID, "default")
	data := map[string]any{
		"feature_date":    strings.TrimSpace(in.FeatureDate),
		"cell_id":         cellID,
		"supplier_id":     strings.TrimSpace(in.SupplierID),
		"warehouse_id":    whID,
		"product_id":      strings.TrimSpace(in.ProductID),
		"units_ordered":   in.UnitsOrdered,
		"units_delivered": in.UnitsDelivered,
		"revenue_minor":   in.RevenueMinor,
		"updated_at":      updatedAt.Format(time.RFC3339Nano),
	}
	setOptionalString(data, "sku", in.SKU)
	setOptionalString(data, "currency", in.Currency)
	setOptionalString(data, "demand_category", in.DemandCategory)
	if in.BaselineQty != nil {
		data["baseline_qty"] = *in.BaselineQty
	}
	if in.ConfidencePct != nil {
		data["confidence_pct"] = *in.ConfidencePct
	}
	if in.ADI != nil {
		data["adi"] = *in.ADI
	}
	if in.CV2 != nil {
		data["cv2"] = *in.CV2
	}
	if in.StockoutEvents != nil {
		data["stockout_events"] = *in.StockoutEvents
	}

	return Row{
		Table:    TableFeaturesSKUDemandDaily,
		InsertID: fmt.Sprintf("feat_sku:%s:%s:%s:%s:%s", strings.TrimSpace(in.FeatureDate), cellID, strings.TrimSpace(in.SupplierID), whID, strings.TrimSpace(in.ProductID)),
		Data:     data,
	}, nil
}

// MapRetailerCreditSignalRow converts a typed RetailerCreditSignalInput into a BigQuery Row for features.retailer_credit_signals.
func MapRetailerCreditSignalRow(in RetailerCreditSignalInput) (Row, error) {
	if strings.TrimSpace(in.SignalDate) == "" || strings.TrimSpace(in.SupplierID) == "" || strings.TrimSpace(in.RetailerID) == "" {
		return Row{}, fmt.Errorf("%w: retailer_credit_signals requires signal_date, supplier_id, and retailer_id", ErrMalformedEventPayload)
	}
	updatedAt := in.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	cellID := firstNonEmpty(in.CellID, "uz")
	data := map[string]any{
		"signal_date":           strings.TrimSpace(in.SignalDate),
		"cell_id":               cellID,
		"supplier_id":           strings.TrimSpace(in.SupplierID),
		"retailer_id":           strings.TrimSpace(in.RetailerID),
		"currency":              firstNonEmpty(in.Currency, "UZS"),
		"credit_limit_minor":    in.CreditLimitMinor,
		"current_balance_minor": in.CurrentBalanceMinor,
		"overdue_balance_minor": in.OverdueBalanceMinor,
		"open_invoice_count":    in.OpenInvoiceCount,
		"late_payments_60d":     in.LatePayments60d,
		"promise_breaches_90d":  in.PromiseBreaches90d,
		"risk_tier":             firstNonEmpty(in.RiskTier, "UNKNOWN"),
		"delinquent":            in.Delinquent,
		"on_hold":               in.OnHold,
		"updated_at":            updatedAt.Format(time.RFC3339Nano),
	}

	return Row{
		Table:    TableFeaturesRetailerCreditSignals,
		InsertID: fmt.Sprintf("feat_cred:%s:%s:%s:%s", strings.TrimSpace(in.SignalDate), cellID, strings.TrimSpace(in.SupplierID), strings.TrimSpace(in.RetailerID)),
		Data:     data,
	}, nil
}

func mapOntologyFromEnvelope(
	raw []byte,
	envelope map[string]any,
	eventID string,
	eventType string,
	cellID string,
	supplierID string,
	occurredAt time.Time,
	now time.Time,
) (Row, bool, error) {
	var ev events.OntologyObjectChangedEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return Row{}, false, fmt.Errorf("%w: parse ontology event: %v", ErrMalformedEventPayload, err)
	}
	objType := strings.TrimSpace(ev.ObjectType)
	objID := strings.TrimSpace(ev.ObjectID)
	if objType == "" || objID == "" {
		return Row{}, false, nil
	}
	effectiveCell := firstNonEmpty(ev.CellID, cellID)
	effectiveSupplier := firstNonEmpty(ev.SupplierID, supplierID)
	updatedAt := parseTimestampFallback(ev.UpdatedAt, occurredAt, now)

	var propsJSON string
	if p, ok := envelope["properties_json"].(string); ok && strings.TrimSpace(p) != "" {
		propsJSON = p
	} else if propsMap, ok := envelope["properties"]; ok && propsMap != nil {
		if b, err := json.Marshal(propsMap); err == nil {
			propsJSON = string(b)
		}
	}
	var validFromPtr *time.Time
	if vf := stringField(envelope, "valid_from"); vf != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, vf); err == nil {
			u := parsed.UTC()
			validFromPtr = &u
		}
	}

	row, err := MapOntologyObjectRow(OntologyObjectInput{
		CellID:           effectiveCell,
		ObjectType:       objType,
		ObjectID:         objID,
		SupplierID:       effectiveSupplier,
		Version:          ev.Version,
		ChangeKind:       firstNonEmpty(ev.ChangeKind, "UPSERT"),
		SourceEventType:  firstNonEmpty(ev.SourceEventType, eventType),
		SourceEventID:    firstNonEmpty(ev.SourceEventID, eventID),
		SecurityMarkings: ev.SecurityMarkings,
		PropertiesJSON:   propsJSON,
		ValidFrom:        validFromPtr,
		UpdatedAt:        updatedAt,
		IngestedAt:       now,
	})
	if err != nil {
		return Row{}, false, err
	}
	return row, true, nil
}

func mapGLEntryFromEnvelope(
	raw []byte,
	envelope map[string]any,
	eventID string,
	supplierID string,
	occurredAt time.Time,
	now time.Time,
) (Row, bool, error) {
	var ev events.GLEntryPostedEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return Row{}, false, fmt.Errorf("%w: parse gl event: %v", ErrMalformedEventPayload, err)
	}
	if strings.TrimSpace(ev.EntryID) == "" {
		return Row{}, false, nil
	}
	entityID := firstNonEmpty(ev.EntityID, supplierID)
	postedAt := parseTimestampFallback(stringField(envelope, "posted_at"), occurredAt, now)
	var occPtr *time.Time
	if !occurredAt.IsZero() {
		u := occurredAt.UTC()
		occPtr = &u
	}

	row, err := MapGLEntryRow(GLEntryInput{
		EntityID:         entityID,
		EntryID:          ev.EntryID,
		PeriodID:         ev.PeriodID,
		SourceEventID:    firstNonEmpty(stringField(envelope, "source_event_id"), eventID),
		SourceType:       ev.SourceType,
		SourceID:         ev.SourceID,
		OrderID:          stringField(envelope, "order_id"),
		Currency:         ev.Currency,
		CountryPack:      firstNonEmpty(stringField(envelope, "country_pack"), "UZ"),
		TotalDebitMinor:  ev.TotalDebitMinor,
		TotalCreditMinor: ev.TotalCreditMinor,
		ReversesEntryID:  ev.ReversesEntryID,
		PrevHash:         stringField(envelope, "prev_hash"),
		EntryHash:        ev.Hash,
		PostedBy:         stringField(envelope, "posted_by"),
		OccurredAt:       occPtr,
		PostedAt:         postedAt,
		IngestedAt:       now,
	})
	if err != nil {
		return Row{}, false, err
	}
	return row, true, nil
}

func mapReconRunFromEnvelope(
	raw []byte,
	envelope map[string]any,
	supplierID string,
	occurredAt time.Time,
	now time.Time,
) (Row, bool) {
	var ev events.ReconRunCompletedEvent
	if err := json.Unmarshal(raw, &ev); err != nil || strings.TrimSpace(ev.RunID) == "" {
		return Row{}, false
	}
	entityID := firstNonEmpty(ev.EntityID, supplierID)
	startedAt := parseTimestampFallback(ev.StartedAt, occurredAt, now)
	data := map[string]any{
		"entity_id":      entityID,
		"run_id":         strings.TrimSpace(ev.RunID),
		"period_id":      firstNonEmpty(ev.PeriodID, startedAt.Format("2006-01")),
		"mode":           firstNonEmpty(ev.Mode, "MANUAL"),
		"status":         firstNonEmpty(ev.Status, "COMPLETED"),
		"break_count":    ev.BreakCount,
		"exposure_minor": ev.ExposureMinor,
		"triggered_by":   firstNonEmpty(ev.TriggeredBy, "system"),
		"started_at":     startedAt.Format(time.RFC3339Nano),
		"ingested_at":    now.Format(time.RFC3339Nano),
	}
	setOptionalString(data, "currency", ev.Currency)
	if strings.TrimSpace(ev.CompletedAt) != "" {
		data["completed_at"] = parseTimestampFallback(ev.CompletedAt, startedAt, now).Format(time.RFC3339Nano)
	}
	if checks := stringField(envelope, "checks_json"); checks != "" && json.Valid([]byte(checks)) {
		data["checks_json"] = checks
	}
	if summary := stringField(envelope, "summary_json"); summary != "" && json.Valid([]byte(summary)) {
		data["summary_json"] = summary
	}

	return Row{
		Table:    TableCuratedReconRuns,
		InsertID: fmt.Sprintf("recon:%s:%s", entityID, strings.TrimSpace(ev.RunID)),
		Data:     data,
	}, true
}

func headersToMap(headers []kgo.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h.Key))
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(string(h.Value))
	}
	return out
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func numStringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case json.Number:
		return t.String()
	case string:
		return strings.TrimSpace(t)
	default:
		return ""
	}
}

func setOptionalString(m map[string]any, key, val string) {
	if trimmed := strings.TrimSpace(val); trimmed != "" {
		m[key] = trimmed
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func parseTimestampFallback(raw string, secondary, fallback time.Time) time.Time {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t.UTC()
		}
	}
	if !secondary.IsZero() {
		return secondary.UTC()
	}
	return fallback.UTC()
}
