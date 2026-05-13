package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	aibridge "aibridge"

	"cloud.google.com/go/spanner"
	gcs "cloud.google.com/go/storage"
	"github.com/segmentio/kafka-go"
	"github.com/xuri/excelize/v2"
	"google.golang.org/grpc/codes"
)

const (
	topicInventoryImportEvents       = "inventory.import.events"
	eventInventoryImportUploaded     = "INVENTORY_IMPORT_UPLOADED"
	eventInventoryImportStatusUpdate = "INVENTORY_IMPORT_STATUS_UPDATE"
	importStatusFrameType            = "IMPORT_STATUS"
	defaultImportSampleRows          = 50
	defaultImportWorkerConcurrency   = 5
	confidenceThreshold              = 0.80
)

var supplierInventoryTargetFields = []aibridge.FieldDefinition{
	{Name: "supplier_id", Description: "Supplier identifier", Aliases: []string{"supplier", "supplier_id"}, Required: true},
	{Name: "warehouse_id", Description: "Warehouse identifier", Aliases: []string{"warehouse", "warehouse_id", "warehouse code"}, Required: true},
	{Name: "product_id", Description: "Product identifier", Aliases: []string{"product", "product_id", "product code"}},
	{Name: "sku_id", Description: "SKU identifier", Aliases: []string{"sku", "sku_id", "sku code", "item code"}, Required: true},
	{Name: "product_name", Description: "Product display name", Aliases: []string{"product name", "name", "item name"}},
	{Name: "category_id", Description: "Category identifier", Aliases: []string{"category", "category_id", "category code"}},
	{Name: "quantity_available", Description: "Available stock quantity", Aliases: []string{"quantity", "qty", "available", "stock", "on hand"}, Required: true},
	{Name: "unit_price", Description: "Unit price in minor currency units", Aliases: []string{"price", "unit price", "unit_price", "cost"}},
	{Name: "currency", Description: "ISO-4217 currency", Aliases: []string{"currency", "currency_code"}},
	{Name: "safety_stock_level", Description: "Safety stock floor", Aliases: []string{"safety stock", "safety_stock", "buffer stock"}},
	{Name: "min_stock_level", Description: "Minimum stock floor", Aliases: []string{"min stock", "minimum stock", "reorder point"}},
	{Name: "max_stock_level", Description: "Maximum stock ceiling", Aliases: []string{"max stock", "maximum stock"}},
	{Name: "h3_cell", Description: "H3 geospatial index cell", Aliases: []string{"h3", "h3_cell", "geo_cell"}},
	{Name: "updated_at", Description: "Last inventory update timestamp", Aliases: []string{"updated at", "updated_at", "last updated", "date"}},
}

type inventoryImportUploadedEvent struct {
	SessionID  string `json:"session_id"`
	SupplierID string `json:"supplier_id"`
	GCSPath    string `json:"gcs_path"`
}

type inventoryImportStatusUpdateEvent struct {
	Type              string    `json:"type"`
	SessionID         string    `json:"session_id"`
	SupplierID        string    `json:"supplier_id"`
	Status            string    `json:"status"`
	SuggestedMappings int       `json:"suggested_mappings"`
	Timestamp         time.Time `json:"timestamp"`
}

type inventoryImportRuntime struct {
	reader       *kafka.Reader
	statusWriter *kafka.Writer
	storage      *gcs.Client
	worker       *inventoryImportWorker
	logger       *slog.Logger
	concurrency  int
}

type inventoryImportWorker struct {
	spanner    *spanner.Client
	bucket     string
	storage    *gcs.Client
	mapper     aibridge.InventoryMapper
	statusFeed *kafka.Writer
	logger     *slog.Logger
}

type importDiscoveryOutcome struct {
	Mappings      []aibridge.MappingCandidate
	Anomalies     []aibridge.Anomaly
	Usage         aibridge.TokenUsage
	Model         string
	ProviderError string
}

func newInventoryImportRuntime(ctx context.Context, logger *slog.Logger, brokerAddress string, spannerClient *spanner.Client) (*inventoryImportRuntime, error) {
	if spannerClient == nil {
		return nil, errors.New("spanner client is required for import worker")
	}
	bucket := strings.TrimSpace(os.Getenv("GCS_BUCKET_NAME"))
	if bucket == "" {
		return nil, errors.New("GCS_BUCKET_NAME is required for import worker")
	}

	storageClient, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("init gcs client: %w", err)
	}

	mapper, mapperErr := aibridge.NewGeminiProviderFromEnv(ctx, nil)
	if mapperErr != nil {
		logger.Warn("gemini provider unavailable; falling back to deterministic mapping", "err", mapperErr)
	}

	concurrency := envInt("IMPORT_WORKER_CONCURRENCY", defaultImportWorkerConcurrency)
	if concurrency <= 0 {
		concurrency = defaultImportWorkerConcurrency
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    topicInventoryImportEvents,
		GroupID:  "ai-worker-import-group",
		MinBytes: 1e3,
		MaxBytes: 10e6,
	})

	statusWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddress),
		Topic:        topicMain,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  5,
		BatchTimeout: 10 * time.Millisecond,
	}

	return &inventoryImportRuntime{
		reader:       reader,
		statusWriter: statusWriter,
		storage:      storageClient,
		logger:       logger,
		concurrency:  concurrency,
		worker: &inventoryImportWorker{
			spanner:    spannerClient,
			bucket:     bucket,
			storage:    storageClient,
			mapper:     mapper,
			statusFeed: statusWriter,
			logger:     logger,
		},
	}, nil
}

func (r *inventoryImportRuntime) Close() {
	if r == nil {
		return
	}
	if r.reader != nil {
		_ = r.reader.Close()
	}
	if r.statusWriter != nil {
		_ = r.statusWriter.Close()
	}
	if r.storage != nil {
		_ = r.storage.Close()
	}
}

func (r *inventoryImportRuntime) Run(ctx context.Context, metrics *consumerLagMetrics) {
	if r == nil || r.reader == nil || r.worker == nil {
		return
	}
	importSem := make(chan struct{}, r.concurrency)
	runConsumerWithAck(ctx, r.reader, "inventory-import-consumer", metrics, func(msg kafka.Message) bool {
		eventType := kafkaEventType(msg.Headers, msg.Key)
		if eventType != eventInventoryImportUploaded {
			return true
		}

		var evt inventoryImportUploadedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			r.logger.Error("inventory import event parse failed", "err", err)
			return true
		}

		importSem <- struct{}{}
		defer func() { <-importSem }()

		if err := r.worker.ProcessUploaded(ctx, evt); err != nil {
			r.logger.Error("inventory import processing failed", "session_id", evt.SessionID, "supplier_id", evt.SupplierID, "err", err)
			return false
		}
		return true
	})
}

func (w *inventoryImportWorker) ProcessUploaded(ctx context.Context, evt inventoryImportUploadedEvent) error {
	if strings.TrimSpace(evt.SessionID) == "" || strings.TrimSpace(evt.SupplierID) == "" || strings.TrimSpace(evt.GCSPath) == "" {
		return fmt.Errorf("invalid inventory import uploaded payload")
	}

	acquired, err := w.markSessionDiscovering(ctx, evt.SupplierID, evt.SessionID)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}

	sample, readErr := w.readSample(ctx, evt.GCSPath, defaultImportSampleRows)
	if readErr != nil {
		if err := w.markSessionFailed(ctx, evt.SupplierID, evt.SessionID, fmt.Sprintf("sample read failed: %v", readErr)); err != nil {
			return err
		}
		_ = w.emitImportStatus(ctx, evt, "FAILED", 0)
		return nil
	}

	outcome := w.discoverMapping(ctx, sample)
	status, lowConfidence := resolveDiscoveryStatus(outcome.Mappings)

	errorSummary := map[string]any{
		"anomalies":          outcome.Anomalies,
		"low_confidence":     lowConfidence,
		"provider":           outcome.Model,
		"provider_error":     outcome.ProviderError,
		"suggested_mappings": len(outcome.Mappings),
		"usage":              outcome.Usage,
	}

	if err := w.persistDiscovery(ctx, evt.SupplierID, evt.SessionID, status, outcome, errorSummary); err != nil {
		return err
	}

	if err := w.emitImportStatus(ctx, evt, status, len(outcome.Mappings)); err != nil {
		w.logger.Warn("import status websocket event emit failed", "session_id", evt.SessionID, "err", err)
	}

	return nil
}

func (w *inventoryImportWorker) markSessionDiscovering(ctx context.Context, supplierID, sessionID string) (bool, error) {
	_, err := w.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			return readErr
		}
		var current string
		if err := row.Columns(&current); err != nil {
			return err
		}
		current = strings.ToUpper(strings.TrimSpace(current))
		if current != "UPLOADED" {
			return errAlreadyClaimed
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "updated_at"},
			[]any{supplierID, sessionID, "DISCOVERING", spanner.CommitTimestamp},
		)})
	})
	if err == nil {
		return true, nil
	}
	if errors.Is(err, errAlreadyClaimed) {
		return false, nil
	}
	if spanner.ErrCode(err) == codes.NotFound {
		return false, nil
	}
	return false, err
}

var errAlreadyClaimed = errors.New("session already claimed")

func (w *inventoryImportWorker) markSessionFailed(ctx context.Context, supplierID, sessionID, reason string) error {
	summaryBytes, _ := json.Marshal(map[string]any{"error": reason})
	summaryJSON, err := toNullJSON(summaryBytes)
	if err != nil {
		return err
	}

	_, err = w.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			return readErr
		}
		var current string
		if err := row.Columns(&current); err != nil {
			return err
		}
		if strings.ToUpper(strings.TrimSpace(current)) == "FAILED" {
			return nil
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "error_summary", "updated_at"},
			[]any{supplierID, sessionID, "FAILED", summaryJSON, spanner.CommitTimestamp},
		)})
	})
	return err
}

func (w *inventoryImportWorker) persistDiscovery(ctx context.Context, supplierID, sessionID, status string, outcome importDiscoveryOutcome, errorSummary map[string]any) error {
	mappingDoc := map[string]any{
		"mappings":     outcome.Mappings,
		"anomalies":    outcome.Anomalies,
		"model":        outcome.Model,
		"usage":        outcome.Usage,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
	mappingBytes, err := json.Marshal(mappingDoc)
	if err != nil {
		return err
	}
	errorBytes, err := json.Marshal(errorSummary)
	if err != nil {
		return err
	}
	mappingJSON, err := toNullJSON(mappingBytes)
	if err != nil {
		return err
	}
	errorJSON, err := toNullJSON(errorBytes)
	if err != nil {
		return err
	}

	_, err = w.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			return readErr
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update(
				"SupplierImportSessions",
				[]string{"supplier_id", "session_id", "status", "error_summary", "updated_at"},
				[]any{supplierID, sessionID, status, errorJSON, spanner.CommitTimestamp},
			),
			spanner.InsertOrUpdate(
				"SupplierImportMapping",
				[]string{"supplier_id", "session_id", "mapping_json", "created_at", "updated_at"},
				[]any{supplierID, sessionID, mappingJSON, spanner.CommitTimestamp, spanner.CommitTimestamp},
			),
		})
	})
	return err
}

func (w *inventoryImportWorker) emitImportStatus(ctx context.Context, evt inventoryImportUploadedEvent, status string, suggestedMappings int) error {
	payload := inventoryImportStatusUpdateEvent{
		Type:              importStatusFrameType,
		SessionID:         evt.SessionID,
		SupplierID:        evt.SupplierID,
		Status:            status,
		SuggestedMappings: suggestedMappings,
		Timestamp:         time.Now().UTC(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return w.statusFeed.WriteMessages(wctx, kafka.Message{
		Key:   []byte(evt.SessionID),
		Value: body,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(eventInventoryImportStatusUpdate)},
		},
	})
}

func (w *inventoryImportWorker) discoverMapping(ctx context.Context, sample aibridge.SampleData) importDiscoveryOutcome {
	resolved, unresolved := deterministicMappings(sample.Headers, supplierInventoryTargetFields)
	outcome := importDiscoveryOutcome{}

	if len(unresolved) > 0 {
		unresolvedSample := aibridge.SampleData{
			Headers:      unresolved,
			Rows:         filterRowsForHeaders(sample.Rows, unresolved),
			TargetFields: supplierInventoryTargetFields,
		}
		if w.mapper != nil {
			aiResult, err := w.mapper.DiscoverSchema(ctx, unresolvedSample)
			if err != nil {
				outcome.ProviderError = err.Error()
			} else {
				outcome.Model = aiResult.Model
				outcome.Usage = aiResult.Usage
				for _, candidate := range sanitizeMappingCandidates(aiResult.Mappings, supplierInventoryTargetFields) {
					prev, exists := resolved[candidate.SourceColumn]
					if !exists || candidate.Confidence > prev.Confidence {
						resolved[candidate.SourceColumn] = candidate
					}
				}
				outcome.Anomalies = append(outcome.Anomalies, aiResult.Anomalies...)
			}
		}

		for _, fallback := range heuristicFallbackMappings(unresolved, supplierInventoryTargetFields) {
			if existing, ok := resolved[fallback.SourceColumn]; ok && existing.Confidence >= fallback.Confidence {
				continue
			}
			resolved[fallback.SourceColumn] = fallback
		}
	}

	outcome.Mappings = mappingsInHeaderOrder(sample.Headers, resolved)
	deterministicAnomalies := detectDeterministicAnomalies(sample.Rows, outcome.Mappings)
	outcome.Anomalies = dedupeAnomalies(append(outcome.Anomalies, deterministicAnomalies...))
	if outcome.Model == "" {
		outcome.Model = "deterministic-fallback"
	}
	return outcome
}

func (w *inventoryImportWorker) readSample(ctx context.Context, objectPath string, limit int) (aibridge.SampleData, error) {
	if w.storage == nil || w.bucket == "" {
		return aibridge.SampleData{}, errors.New("gcs client not configured")
	}
	path := strings.TrimPrefix(strings.TrimSpace(objectPath), "/")
	if path == "" {
		return aibridge.SampleData{}, errors.New("object path is empty")
	}

	rc, err := w.storage.Bucket(w.bucket).Object(path).NewReader(ctx)
	if err != nil {
		return aibridge.SampleData{}, err
	}
	defer rc.Close()

	ext := strings.ToLower(strings.TrimSpace(fileExtension(path)))
	var headers []string
	var rows []map[string]any

	switch ext {
	case "csv", "txt":
		headers, rows, err = readDelimitedSample(rc, ',', limit)
	case "tsv":
		headers, rows, err = readDelimitedSample(rc, '\t', limit)
	case "xlsx", "xls":
		headers, rows, err = readExcelSample(rc, limit)
	default:
		headers, rows, err = readDelimitedSample(rc, ',', limit)
	}
	if err != nil {
		return aibridge.SampleData{}, err
	}

	return aibridge.SampleData{
		Headers:      headers,
		Rows:         rows,
		TargetFields: supplierInventoryTargetFields,
	}, nil
}

func readDelimitedSample(reader io.Reader, delimiter rune, limit int) ([]string, []map[string]any, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1

	headers := []string{}
	rows := make([]map[string]any, 0, limit)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if len(headers) == 0 {
			headers = normalizeHeaders(record)
			continue
		}
		if isEmptyRecord(record) {
			continue
		}
		rows = append(rows, rowToMap(headers, record))
		if len(rows) >= limit {
			break
		}
	}

	if len(headers) == 0 {
		return nil, nil, errors.New("no headers found in uploaded file")
	}
	return headers, rows, nil
}

func readExcelSample(reader io.Reader, limit int) ([]string, []map[string]any, error) {
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, errors.New("excel workbook has no sheets")
	}

	rowsIter, err := file.Rows(sheets[0])
	if err != nil {
		return nil, nil, err
	}
	defer rowsIter.Close()

	headers := []string{}
	rows := make([]map[string]any, 0, limit)
	for rowsIter.Next() {
		record, err := rowsIter.Columns()
		if err != nil {
			return nil, nil, err
		}
		if len(headers) == 0 {
			headers = normalizeHeaders(record)
			continue
		}
		if isEmptyRecord(record) {
			continue
		}
		rows = append(rows, rowToMap(headers, record))
		if len(rows) >= limit {
			break
		}
	}

	if len(headers) == 0 {
		return nil, nil, errors.New("excel sheet does not contain a header row")
	}
	return headers, rows, nil
}

func normalizeHeaders(headers []string) []string {
	normalized := make([]string, 0, len(headers))
	used := make(map[string]int)
	for idx, header := range headers {
		value := strings.TrimSpace(header)
		if value == "" {
			value = fmt.Sprintf("column_%d", idx+1)
		}
		if count, exists := used[value]; exists {
			used[value] = count + 1
			value = fmt.Sprintf("%s_%d", value, count+1)
		} else {
			used[value] = 1
		}
		normalized = append(normalized, value)
	}
	return normalized
}

func rowToMap(headers []string, record []string) map[string]any {
	out := make(map[string]any, len(headers))
	for i, header := range headers {
		if i < len(record) {
			out[header] = strings.TrimSpace(record[i])
			continue
		}
		out[header] = ""
	}
	return out
}

func isEmptyRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func deterministicMappings(headers []string, fields []aibridge.FieldDefinition) (map[string]aibridge.MappingCandidate, []string) {
	aliasToField := make(map[string]string)
	for _, field := range fields {
		aliasToField[normalizeToken(field.Name)] = field.Name
		for _, alias := range field.Aliases {
			aliasToField[normalizeToken(alias)] = field.Name
		}
	}

	resolved := make(map[string]aibridge.MappingCandidate)
	unresolved := make([]string, 0, len(headers))
	for _, header := range headers {
		normalized := normalizeToken(header)
		if target, ok := aliasToField[normalized]; ok {
			resolved[header] = aibridge.MappingCandidate{
				SourceColumn:  header,
				TargetField:   target,
				Confidence:    1,
				Reason:        "exact_header_match",
				Deterministic: true,
			}
			continue
		}
		unresolved = append(unresolved, header)
	}
	return resolved, unresolved
}

func sanitizeMappingCandidates(candidates []aibridge.MappingCandidate, fields []aibridge.FieldDefinition) []aibridge.MappingCandidate {
	validFields := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		validFields[field.Name] = struct{}{}
	}
	out := make([]aibridge.MappingCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.SourceColumn = strings.TrimSpace(candidate.SourceColumn)
		candidate.TargetField = strings.TrimSpace(candidate.TargetField)
		if candidate.SourceColumn == "" || candidate.TargetField == "" {
			continue
		}
		if _, ok := validFields[candidate.TargetField]; !ok {
			continue
		}
		if candidate.Confidence < 0 {
			candidate.Confidence = 0
		}
		if candidate.Confidence > 1 {
			candidate.Confidence = 1
		}
		out = append(out, candidate)
	}
	return out
}

func heuristicFallbackMappings(headers []string, fields []aibridge.FieldDefinition) []aibridge.MappingCandidate {
	out := make([]aibridge.MappingCandidate, 0, len(headers))
	for _, header := range headers {
		normHeader := normalizeToken(header)
		bestField := ""
		bestScore := 0.0
		for _, field := range fields {
			score := similarity(normHeader, normalizeToken(field.Name))
			for _, alias := range field.Aliases {
				score = math.Max(score, similarity(normHeader, normalizeToken(alias)))
			}
			if score > bestScore {
				bestScore = score
				bestField = field.Name
			}
		}
		if bestField == "" || bestScore < 0.55 {
			continue
		}
		confidence := bestScore
		if confidence < 0.65 {
			confidence = 0.65
		}
		out = append(out, aibridge.MappingCandidate{
			SourceColumn: header,
			TargetField:  bestField,
			Confidence:   confidence,
			Reason:       "heuristic_fallback",
		})
	}
	return out
}

func mappingsInHeaderOrder(headers []string, mapped map[string]aibridge.MappingCandidate) []aibridge.MappingCandidate {
	out := make([]aibridge.MappingCandidate, 0, len(mapped))
	for _, header := range headers {
		if candidate, ok := mapped[header]; ok {
			out = append(out, candidate)
		}
	}
	return out
}

func resolveDiscoveryStatus(mappings []aibridge.MappingCandidate) (string, bool) {
	if len(mappings) == 0 {
		return "MAPPING_REQUIRED", true
	}
	lowConfidence := false
	for _, mapping := range mappings {
		if mapping.Confidence < confidenceThreshold {
			lowConfidence = true
			break
		}
	}
	if lowConfidence {
		return "MAPPING_REQUIRED", true
	}
	return "DISCOVERED", false
}

func detectDeterministicAnomalies(rows []map[string]any, mappings []aibridge.MappingCandidate) []aibridge.Anomaly {
	if len(rows) == 0 || len(mappings) == 0 {
		return nil
	}

	targetColumn := make(map[string]string)
	for _, mapping := range mappings {
		targetColumn[mapping.TargetField] = mapping.SourceColumn
	}

	anomalies := make([]aibridge.Anomaly, 0)
	if priceCol, ok := targetColumn["unit_price"]; ok {
		prices := make([]float64, 0, len(rows))
		for _, row := range rows {
			if value, ok := parseNumber(row[priceCol]); ok {
				prices = append(prices, value)
			}
		}
		if len(prices) >= 3 {
			sorted := append([]float64(nil), prices...)
			sort.Float64s(sorted)
			median := sorted[len(sorted)/2]
			if median > 0 {
				for _, value := range prices {
					if value >= median*10 || value <= median*0.1 {
						anomalies = append(anomalies, aibridge.Anomaly{
							Kind:     "PRICE_VARIANCE_EXTREME",
							Column:   priceCol,
							Detail:   fmt.Sprintf("price %.2f deviates more than 1000%% from median %.2f", value, median),
							Severity: "warning",
						})
						break
					}
				}
			}
		}
	}

	dateColumns := make([]string, 0)
	for _, mapping := range mappings {
		if mapping.TargetField == "updated_at" || strings.Contains(strings.ToLower(mapping.SourceColumn), "date") {
			dateColumns = append(dateColumns, mapping.SourceColumn)
		}
	}
	now := time.Now().UTC().Add(24 * time.Hour)
	for _, column := range dateColumns {
		for _, row := range rows {
			raw := fmt.Sprint(row[column])
			if t, ok := parseDate(raw); ok && t.After(now) {
				anomalies = append(anomalies, aibridge.Anomaly{
					Kind:     "FUTURE_DATE",
					Column:   column,
					Detail:   fmt.Sprintf("detected future date %s", t.Format(time.RFC3339)),
					Severity: "warning",
				})
				break
			}
		}
	}

	return anomalies
}

func dedupeAnomalies(anomalies []aibridge.Anomaly) []aibridge.Anomaly {
	if len(anomalies) <= 1 {
		return anomalies
	}
	seen := make(map[string]struct{}, len(anomalies))
	out := make([]aibridge.Anomaly, 0, len(anomalies))
	for _, anomaly := range anomalies {
		key := anomaly.Kind + "|" + anomaly.Column + "|" + anomaly.Detail
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, anomaly)
	}
	return out
}

func parseNumber(raw any) (float64, bool) {
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, ",", "")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseDate(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"02.01.2006",
		"01/02/2006",
		"02/01/2006",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func normalizeToken(input string) string {
	trimmed := strings.ToLower(strings.TrimSpace(input))
	if trimmed == "" {
		return ""
	}
	b := strings.Builder{}
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func similarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 0.82
	}
	distance := levenshteinDistance(a, b)
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 0
	}
	return 1 - float64(distance)/float64(maxLen)
}

func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(
				curr[j-1]+1,
				min(prev[j]+1, prev[j-1]+cost),
			)
		}
		prev = curr
	}
	return prev[len(b)]
}

func fileExtension(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx < 0 || idx == len(path)-1 {
		return ""
	}
	return path[idx+1:]
}

func filterRowsForHeaders(rows []map[string]any, headers []string) []map[string]any {
	if len(rows) == 0 || len(headers) == 0 {
		return rows
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		filtered := make(map[string]any, len(headers))
		for _, header := range headers {
			filtered[header] = row[header]
		}
		out = append(out, filtered)
	}
	return out
}

func kafkaHeaderValue(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}

func kafkaEventType(headers []kafka.Header, key []byte) string {
	if eventType := kafkaHeaderValue(headers, "event_type"); eventType != "" {
		return eventType
	}
	return string(key)
}

func runConsumerWithAck(ctx context.Context, r *kafka.Reader, name string, metrics *consumerLagMetrics, handler func(m kafka.Message) bool) {
	n := max(runtime.GOMAXPROCS(0), 1)
	chans := make([]chan kafka.Message, n)
	var wg sync.WaitGroup
	for i := range chans {
		chans[i] = make(chan kafka.Message, 32)
		wg.Add(1)
		go func(in <-chan kafka.Message) {
			defer wg.Done()
			for m := range in {
				ack := handler(m)
				if !ack {
					continue
				}
				commitCtx := ctx
				if commitCtx.Err() != nil {
					var cancel context.CancelFunc
					commitCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
				}
				if err := r.CommitMessages(commitCtx, m); err != nil {
					logger.Error(name+": commit failed", "partition", m.Partition, "offset", m.Offset, "err", err)
				}
			}
		}(chans[i])
	}

	defer func() {
		for _, ch := range chans {
			close(ch)
		}
		wg.Wait()
	}()

	streak := 0
	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			d := time.Duration(100*1<<min(streak, 10)) * time.Millisecond
			if d > 30*time.Second {
				d = 30 * time.Second
			}
			streak++
			logger.Error(name+": fetch failed", "err", err, "streak", streak, "backoff", d)
			select {
			case <-ctx.Done():
				return
			case <-time.After(d):
			}
			continue
		}
		streak = 0
		metrics.observe(name, msg)
		idx := int(uint(msg.Partition)) % n
		select {
		case <-ctx.Done():
			return
		case chans[idx] <- msg:
		}
	}
}

func envInt(key string, def int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return parsed
}

func toNullJSON(raw []byte) (spanner.NullJSON, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return spanner.NullJSON{}, nil
	}
	var value any
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return spanner.NullJSON{}, err
	}
	return spanner.NullJSON{Value: value, Valid: true}, nil
}
