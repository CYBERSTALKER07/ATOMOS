package supplier

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	importDiscoveryConfidenceThreshold = 0.80
	importDiscoveryMaxRows             = 10000
)

type importFieldDefinition struct {
	Name        string
	Description string
	Aliases     []string
	Required    bool
}

type importMappingCandidate struct {
	SourceColumn  string  `json:"source_column"`
	TargetField   string  `json:"target_field"`
	Confidence    float64 `json:"confidence"`
	Reason        string  `json:"reason,omitempty"`
	Deterministic bool    `json:"deterministic,omitempty"`
}

type importAnomaly struct {
	Kind     string `json:"kind"`
	Column   string `json:"column,omitempty"`
	Detail   string `json:"detail"`
	Severity string `json:"severity,omitempty"`
}

type importDiscoveryOutcome struct {
	Mappings  []importMappingCandidate
	Anomalies []importAnomaly
	Model     string
}

var pegasusXImportTargetFields = []importFieldDefinition{
	{Name: "warehouse_id", Description: "Warehouse identifier", Aliases: []string{"warehouse", "warehouse_id", "warehouse code"}, Required: true},
	{Name: "product_id", Description: "Product identifier", Aliases: []string{"product", "product_id", "product code", "sku", "sku_id", "item code"}, Required: true},
	{Name: "quantity_on_hand", Description: "On-hand stock quantity", Aliases: []string{"quantity", "qty", "quantity_on_hand", "quantity_available", "available", "stock", "on hand"}, Required: true},
	{Name: "reorder_threshold", Description: "Reorder threshold", Aliases: []string{"reorder_threshold", "threshold", "reorder", "min_stock_level", "safety_stock"}},
	{Name: "product_name", Description: "Product display name", Aliases: []string{"product name", "name", "item name"}},
	{Name: "category_id", Description: "Category identifier", Aliases: []string{"category", "category_id", "category code"}},
	{Name: "price_minor", Description: "Unit price in minor currency units", Aliases: []string{"price", "unit price", "unit_price", "price_minor", "cost"}},
}

func discoverImportDelimited(body []byte, delimiter rune) (importDiscoveryOutcome, []string, []map[string]any, error) {
	headers, rows, err := readImportDelimitedSample(bytes.NewReader(body), delimiter, importDiscoveryMaxRows)
	if err != nil {
		return importDiscoveryOutcome{}, nil, nil, err
	}
	if len(headers) == 0 {
		return importDiscoveryOutcome{}, nil, nil, fmt.Errorf("no headers found in uploaded file")
	}

	outcome := discoverImportMapping(headers)
	outcome.Anomalies = append(outcome.Anomalies, detectImportAnomalies(rows, outcome.Mappings)...)
	if outcome.Model == "" {
		outcome.Model = "deterministic"
	}
	return outcome, headers, rows, nil
}

func discoverImportMapping(headers []string) importDiscoveryOutcome {
	resolved, unresolved := importDeterministicMappings(headers, pegasusXImportTargetFields)
	outcome := importDiscoveryOutcome{Model: "deterministic"}

	for _, fallback := range importHeuristicFallbackMappings(unresolved, pegasusXImportTargetFields) {
		if existing, ok := resolved[fallback.SourceColumn]; ok && existing.Confidence >= fallback.Confidence {
			continue
		}
		resolved[fallback.SourceColumn] = fallback
	}

	outcome.Mappings = importMappingsInHeaderOrder(headers, resolved)
	return outcome
}

func resolveImportDiscoveryStatus(mappings []importMappingCandidate) (string, bool) {
	if len(mappings) == 0 {
		return "MAPPING_REQUIRED", true
	}
	for _, mapping := range mappings {
		if mapping.Confidence < importDiscoveryConfidenceThreshold {
			return "MAPPING_REQUIRED", true
		}
	}
	required := map[string]struct{}{
		"warehouse_id":     {},
		"product_id":       {},
		"quantity_on_hand": {},
	}
	mapped := make(map[string]struct{}, len(mappings))
	for _, mapping := range mappings {
		mapped[mapping.TargetField] = struct{}{}
	}
	for field := range required {
		if _, ok := mapped[field]; !ok {
			return "MAPPING_REQUIRED", true
		}
	}
	return "DISCOVERED", false
}

func buildImportStagedRows(
	supplierID string,
	rows []map[string]any,
	mappings []importMappingCandidate,
	warehouseIDs map[string]struct{},
	productExists func(productID string) bool,
) ([]ImportStagedRowRecord, map[string]any) {
	sourceByTarget := make(map[string]string, len(mappings))
	for _, mapping := range mappings {
		sourceByTarget[mapping.TargetField] = mapping.SourceColumn
	}

	staged := make([]ImportStagedRowRecord, 0, len(rows))
	summary := map[string]any{
		"staged_rows":  len(rows),
		"valid_rows":   0,
		"invalid_rows": 0,
	}

	for idx, raw := range rows {
		rawJSON, _ := json.Marshal(raw)
		cleaned := applyImportMappings(raw, sourceByTarget)
		cleaned["supplier_id"] = supplierID

		var validationErrors []string
		warehouseID := importCleanedString(cleaned, "warehouse_id")
		if warehouseID == "" {
			validationErrors = append(validationErrors, "warehouse_id_required")
		} else if _, ok := warehouseIDs[warehouseID]; !ok {
			validationErrors = append(validationErrors, "warehouse_not_in_topology")
		}

		productID := importCleanedString(cleaned, "product_id")
		isNewProduct := false
		if productID == "" {
			validationErrors = append(validationErrors, "product_id_required")
		} else if !productExists(productID) {
			isNewProduct = true
			if importCleanedString(cleaned, "product_name") == "" {
				cleaned["product_name"] = fmt.Sprintf("Imported product %d", idx+1)
			}
		}

		qtyRaw := importCleanedString(cleaned, "quantity_on_hand")
		if qtyRaw == "" {
			validationErrors = append(validationErrors, "quantity_on_hand_required")
		} else if qty, err := strconv.ParseInt(strings.ReplaceAll(qtyRaw, ",", ""), 10, 64); err != nil || qty < 0 {
			validationErrors = append(validationErrors, "invalid_quantity_on_hand")
		} else {
			cleaned["quantity_on_hand"] = qty
		}

		if rawThreshold := importCleanedString(cleaned, "reorder_threshold"); rawThreshold != "" {
			if threshold, err := strconv.ParseInt(strings.ReplaceAll(rawThreshold, ",", ""), 10, 64); err != nil || threshold < 0 {
				validationErrors = append(validationErrors, "invalid_reorder_threshold")
			} else {
				cleaned["reorder_threshold"] = threshold
			}
		}

		cleanedJSON, _ := json.Marshal(cleaned)
		if len(validationErrors) == 0 {
			summary["valid_rows"] = summary["valid_rows"].(int) + 1
		} else {
			summary["invalid_rows"] = summary["invalid_rows"].(int) + 1
			cleanedJSON = nil
		}

		staged = append(staged, ImportStagedRowRecord{
			RowIndex:         int64(idx + 1),
			RawData:          rawJSON,
			CleanedData:      cleanedJSON,
			ValidationErrors: validationErrors,
			IsNewProduct:     isNewProduct,
		})
	}
	return staged, summary
}

func importCleanedString(cleaned map[string]any, key string) string {
	value, ok := cleaned[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func applyImportMappings(raw map[string]any, sourceByTarget map[string]string) map[string]any {
	out := make(map[string]any, len(sourceByTarget))
	for target, source := range sourceByTarget {
		if value, ok := raw[source]; ok {
			out[target] = strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return out
}

func readImportDelimitedSample(reader io.Reader, delimiter rune, limit int) ([]string, []map[string]any, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true

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
			headers = importNormalizeHeaders(record)
			continue
		}
		if importIsEmptyRecord(record) {
			continue
		}
		rows = append(rows, importRowToMap(headers, record))
		if len(rows) >= limit {
			break
		}
	}
	return headers, rows, nil
}

func importNormalizeHeaders(headers []string) []string {
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

func importRowToMap(headers []string, record []string) map[string]any {
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

func importIsEmptyRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func importDeterministicMappings(headers []string, fields []importFieldDefinition) (map[string]importMappingCandidate, []string) {
	aliasToField := make(map[string]string)
	for _, field := range fields {
		aliasToField[importNormalizeToken(field.Name)] = field.Name
		for _, alias := range field.Aliases {
			aliasToField[importNormalizeToken(alias)] = field.Name
		}
	}

	resolved := make(map[string]importMappingCandidate)
	unresolved := make([]string, 0, len(headers))
	for _, header := range headers {
		normalized := importNormalizeToken(header)
		if target, ok := aliasToField[normalized]; ok {
			resolved[header] = importMappingCandidate{
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

func importHeuristicFallbackMappings(headers []string, fields []importFieldDefinition) []importMappingCandidate {
	out := make([]importMappingCandidate, 0, len(headers))
	for _, header := range headers {
		normHeader := importNormalizeToken(header)
		bestField := ""
		bestScore := 0.0
		for _, field := range fields {
			score := importSimilarity(normHeader, importNormalizeToken(field.Name))
			for _, alias := range field.Aliases {
				score = math.Max(score, importSimilarity(normHeader, importNormalizeToken(alias)))
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
		out = append(out, importMappingCandidate{
			SourceColumn: header,
			TargetField:  bestField,
			Confidence:   confidence,
			Reason:       "heuristic_fallback",
		})
	}
	return out
}

func importMappingsInHeaderOrder(headers []string, mapped map[string]importMappingCandidate) []importMappingCandidate {
	out := make([]importMappingCandidate, 0, len(mapped))
	for _, header := range headers {
		if candidate, ok := mapped[header]; ok {
			out = append(out, candidate)
		}
	}
	return out
}

func detectImportAnomalies(rows []map[string]any, mappings []importMappingCandidate) []importAnomaly {
	if len(rows) == 0 || len(mappings) == 0 {
		return nil
	}
	targetColumn := make(map[string]string)
	for _, mapping := range mappings {
		targetColumn[mapping.TargetField] = mapping.SourceColumn
	}

	anomalies := make([]importAnomaly, 0)
	if priceCol, ok := targetColumn["price_minor"]; ok {
		prices := make([]float64, 0, len(rows))
		for _, row := range rows {
			if value, ok := importParseNumber(row[priceCol]); ok {
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
						anomalies = append(anomalies, importAnomaly{
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
	return anomalies
}

func importParseNumber(raw any) (float64, bool) {
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

func importNormalizeToken(input string) string {
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

func importSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 0.82
	}
	distance := importLevenshteinDistance(a, b)
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 0
	}
	return 1 - float64(distance)/float64(maxLen)
}

func importLevenshteinDistance(a, b string) int {
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
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[len(b)]
}

func importParseDate(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{time.RFC3339, "2006-01-02", "02.01.2006", "01/02/2006", "02/01/2006"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}
