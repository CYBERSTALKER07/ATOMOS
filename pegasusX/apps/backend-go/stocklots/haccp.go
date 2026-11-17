package stocklots

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

const (
	defaultChilledMinMilliC int64 = 0
	defaultChilledMaxMilliC int64 = 8000
	defaultFrozenMinMilliC  int64 = -22000
	defaultFrozenMaxMilliC  int64 = -15000
)

// HACCPAuditInput carries audit metadata emitted when evaluating or exporting HACCP reports.
type HACCPAuditInput struct {
	TenantID   string
	ActorID    string
	ActorType  string
	Role       string
	Action     string
	ObjectType string
	ObjectID   string
	AfterJSON  string
	TraceID    string
	IP         string
	DeviceID   string
}

// HACCPAffectedLot summarizes a stock lot impacted by cold-chain excursions, breaches, or zone mismatches.
type HACCPAffectedLot struct {
	LotID          string `json:"lot_id"`
	WarehouseID    string `json:"warehouse_id"`
	ProductID      string `json:"product_id"`
	ProductName    string `json:"product_name,omitempty"`
	LocationID     string `json:"location_id"`
	LotCode        string `json:"lot_code"`
	Status         string `json:"status"`
	QuantityOnHand int64  `json:"quantity_on_hand"`
	RequiredZone   string `json:"required_zone,omitempty"`
	LocationZone   string `json:"location_zone,omitempty"`
	Reason         string `json:"reason"`
	ManifestID     string `json:"manifest_id,omitempty"`
}

// HACCPExcursion captures one out-of-band telemetry reading measured in integer milli-Celsius.
type HACCPExcursion struct {
	ReadingID         string `json:"reading_id"`
	ManifestID        string `json:"manifest_id"`
	WarehouseID       string `json:"warehouse_id,omitempty"`
	SensorID          string `json:"sensor_id,omitempty"`
	RecordedAt        string `json:"recorded_at"`
	TempMilliC        int64  `json:"temp_milli_c"`
	MinMilliC         int64  `json:"min_milli_c"`
	MaxMilliC         int64  `json:"max_milli_c"`
	BreachDeltaMilliC int64  `json:"breach_delta_milli_c"`
	Severity          string `json:"severity"`
	Status            string `json:"status"`
}

// HACCPLotZoneMismatch represents an inventory lot stored in staging or an incompatible temperature zone.
type HACCPLotZoneMismatch struct {
	LotID        string `json:"lot_id"`
	WarehouseID  string `json:"warehouse_id"`
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name,omitempty"`
	LocationID   string `json:"location_id"`
	RequiredZone string `json:"required_zone"`
	LocationZone string `json:"location_zone"`
	LocationType string `json:"location_type,omitempty"`
	Warning      string `json:"warning"`
}

// HACCPConditionReport summarizes a TEMPERATURE_BREACH order condition report within the reporting window.
type HACCPConditionReport struct {
	ReportID         string `json:"report_id"`
	OrderID          string `json:"order_id"`
	RetailerID       string `json:"retailer_id"`
	SKU              string `json:"sku"`
	ConditionType    string `json:"condition_type"`
	Severity         string `json:"severity"`
	ResolutionStatus string `json:"resolution_status"`
	Description      string `json:"description,omitempty"`
	ReportedBy       string `json:"reported_by,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// HACCPReport holds aggregated Cold-Chain HACCP telemetry, zone compliance, and breach metrics.
type HACCPReport struct {
	TenantID           string                 `json:"tenant_id"`
	From               string                 `json:"from"`
	To                 string                 `json:"to"`
	GeneratedAt        string                 `json:"generated_at"`
	TotalReadings      int64                  `json:"total_readings"`
	ExcursionCount     int64                  `json:"excursion_count"`
	ComplianceRateBps  int64                  `json:"compliance_rate_bps"`
	OpenBreaches       int64                  `json:"open_breaches"`
	AffectedLotsCount  int64                  `json:"affected_lots_count"`
	AffectedLots       []HACCPAffectedLot     `json:"affected_lots"`
	Excursions         []HACCPExcursion       `json:"excursions"`
	ColdZoneMismatches []HACCPLotZoneMismatch `json:"cold_zone_mismatches"`
	ConditionReports   []HACCPConditionReport `json:"condition_reports"`
}

type haccpOutboxBuffer struct {
	inner    *outbox.SpannerTxnBuffer
	tenantID string
}

func (b *haccpOutboxBuffer) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.tenantID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.tenantID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

type haccpTempReadingRow struct {
	ReadingID   string
	ManifestID  string
	WarehouseID string
	SensorID    string
	RecordedAt  time.Time
	TempMilliC  int64
}

type haccpStockLotRow struct {
	LotID          string
	WarehouseID    string
	ProductID      string
	LocationID     string
	LotCode        string
	Status         string
	QuantityOnHand int64
}

type haccpTempBand struct {
	MinMilliC int64
	MaxMilliC int64
}

// RegisterHACCPComplianceRoute mounts GET /v1/compliance/haccp with CookieAuth,
// RequireAuditorActive, tenant binding, role guard, and read permissions.
func RegisterHACCPComplianceRoute(r chi.Router, h *Handler, jwtSecret string, allowAuthBypass bool) {
	if r == nil || h == nil {
		return
	}
	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermAuditRead,
		auth.PermStockAdjustApprove,
		auth.PermReconManage,
		auth.PermConfigManage,
	)
	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(jwtSecret) != "" {
			gr.Use(auth.CookieAuth(jwtSecret))
		}
		gr.Use(auth.RequireAuditorActive)
		gr.Use(auth.AttachTenantFromClaims)
		if !allowAuthBypass {
			gr.Use(auth.RequireAnyAuthenticated())
		}
		gr.With(roleGuard, readPerm).Get("/v1/compliance/haccp", h.HandleHACCPCompliance)
	})
}

// HandleHACCPCompliance serves GET /v1/compliance/haccp (?from=&to=&format=json|csv).
func (h *Handler) HandleHACCPCompliance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}

	reqEntity := firstNonEmptyStr(
		r.URL.Query().Get("entity"),
		r.URL.Query().Get("supplier_id"),
		r.URL.Query().Get("tenant_id"),
	)
	sid, scopeErr := resolveHACCPTenantScope(r, h.SupplierID, reqEntity)
	if scopeErr != "" {
		status := http.StatusForbidden
		if scopeErr == "supplier_id_required" {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": scopeErr, "code": scopeErr})
		return
	}

	now := time.Now().UTC()
	from, to, explicitTo, rangeErr := parseHACCPWindow(r.URL.Query().Get("from"), r.URL.Query().Get("to"), now)
	if rangeErr != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": rangeErr, "code": rangeErr})
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_format", "code": "unsupported_format"})
		return
	}

	actorID, actorType, role, traceID := haccpActorMeta(r)
	report, err := evaluateHACCPComplianceWithOpt(r.Context(), h.Spanner, sid, from, to, explicitTo, now, actorID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "haccp_evaluation_failed",
			"message": err.Error(),
		})
		return
	}

	if h.AuditRecord != nil {
		action := "compliance.haccp.evaluated"
		if format == "csv" {
			action = "compliance.haccp.exported"
		}
		summaryBytes, _ := json.Marshal(map[string]any{
			"tenant_id":           report.TenantID,
			"from":                report.From,
			"to":                  report.To,
			"format":              format,
			"total_readings":      report.TotalReadings,
			"excursion_count":     report.ExcursionCount,
			"compliance_rate_bps": report.ComplianceRateBps,
			"open_breaches":       report.OpenBreaches,
			"affected_lots_count": report.AffectedLotsCount,
			"mismatch_count":      len(report.ColdZoneMismatches),
		})
		_ = h.AuditRecord(r.Context(), HACCPAuditInput{
			TenantID:   sid,
			ActorID:    actorID,
			ActorType:  actorType,
			Role:       role,
			Action:     action,
			ObjectType: "ComplianceHACCPReport",
			ObjectID:   sid + ":" + from.Format("2006-01-02") + ":" + to.Format("2006-01-02"),
			AfterJSON:  string(summaryBytes),
			TraceID:    traceID,
			IP:         r.RemoteAddr,
			DeviceID:   strings.TrimSpace(r.Header.Get("X-Device-Id")),
		})
	}

	if format == "csv" {
		csvBytes, cerr := marshalHACCPCSV(report)
		if cerr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "haccp_csv_failed"})
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="haccp-compliance-%s.csv"`, sid))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(csvBytes)
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func resolveHACCPTenantScope(r *http.Request, fallback, requested string) (string, string) {
	claims, ok := auth.FromContext(r.Context())
	if ok && strings.EqualFold(string(claims.Role), string(auth.RolePlatformAdmin)) && strings.TrimSpace(claims.SupplierID) == "" {
		sid := firstNonEmptyStr(requested, fallback)
		if sid == "" {
			return "", "supplier_id_required"
		}
		return sid, ""
	}
	return bindCallerSupplier(r, fallback, requested)
}

func haccpActorMeta(r *http.Request) (actorID, actorType, role, traceID string) {
	actorID = actorSubject(r)
	if actorID == "" {
		actorID = "system"
	}
	claims, ok := auth.FromContext(r.Context())
	if ok {
		actorType = auth.EffectiveActorType(claims)
		role = auth.EffectiveSupplierRole(claims)
		traceID = claims.TraceID
	}
	if actorType == "" {
		actorType = auth.ActorTypeHuman
	}
	if traceID == "" {
		traceID = firstNonEmptyStr(outbox.TraceIDFromContext(r.Context()), r.Header.Get("X-Trace-Id"))
	}
	return
}

func parseHACCPWindow(rawFrom, rawTo string, now time.Time) (time.Time, time.Time, bool, string) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	to := now.UTC()
	explicitTo := false
	if s := strings.TrimSpace(rawTo); s != "" {
		parsed, endOfDay, ok := parseHACCPTimestamp(s)
		if !ok {
			return time.Time{}, time.Time{}, false, "invalid_to_timestamp"
		}
		if endOfDay {
			parsed = parsed.Add(24*time.Hour - time.Nanosecond)
		}
		to = parsed
		explicitTo = true
	}
	from := to.Add(-30 * 24 * time.Hour)
	if s := strings.TrimSpace(rawFrom); s != "" {
		parsed, _, ok := parseHACCPTimestamp(s)
		if !ok {
			return time.Time{}, time.Time{}, explicitTo, "invalid_from_timestamp"
		}
		from = parsed
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, explicitTo, "invalid_date_range"
	}
	return from, to, explicitTo, ""
}

func parseHACCPTimestamp(raw string) (t time.Time, dateOnly bool, ok bool) {
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed.UTC(), false, true
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), false, true
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed.UTC(), true, true
	}
	return time.Time{}, false, false
}

// ComputeHACCPReport aggregates cold-chain telemetry, zone mismatches, and condition breaches in a ReadOnlyTransaction.
func ComputeHACCPReport(ctx context.Context, client *spanner.Client, tenantID string, from, to, now time.Time) (HACCPReport, int64, error) {
	return computeHACCPReportWithOpt(ctx, client, tenantID, from, to, true, now)
}

func computeHACCPReportWithOpt(ctx context.Context, client *spanner.Client, tenantID string, from, to time.Time, explicitTo bool, now time.Time) (HACCPReport, int64, error) {
	if client == nil {
		return HACCPReport{}, 0, fmt.Errorf("spanner client required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tid := strings.TrimSpace(tenantID)
	report := HACCPReport{
		TenantID:           tid,
		From:               from.UTC().Format(time.RFC3339),
		To:                 to.UTC().Format(time.RFC3339),
		GeneratedAt:        now.UTC().Format(time.RFC3339),
		ComplianceRateBps:  10000,
		AffectedLots:       []HACCPAffectedLot{},
		Excursions:         []HACCPExcursion{},
		ColdZoneMismatches: []HACCPLotZoneMismatch{},
		ConditionReports:   []HACCPConditionReport{},
	}

	ro := client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	defer ro.Close()

	readings, manifestIDs, err := loadHACCPReadingsRO(ctx, ro, tid, from, to, explicitTo)
	if err != nil {
		return HACCPReport{}, 0, err
	}
	report.TotalReadings = int64(len(readings))

	condReports, openBreaches, resolvedBreaches, orderBreachStatus, breachOrderIDs, err := loadHACCPConditionReportsRO(ctx, ro, tid, from, to, explicitTo)
	if err != nil {
		return HACCPReport{}, 0, err
	}
	report.ConditionReports = condReports
	report.OpenBreaches = openBreaches

	manifestBands, manifestOrders, pickLotsByManifest, err := loadManifestColdContextRO(ctx, ro, manifestIDs)
	if err != nil {
		return HACCPReport{}, 0, err
	}

	excursionManifestSet := make(map[string]struct{})
	for _, rd := range readings {
		band := manifestBands[rd.ManifestID]
		if band.MinMilliC == 0 && band.MaxMilliC == 0 {
			band = haccpTempBand{MinMilliC: defaultChilledMinMilliC, MaxMilliC: defaultChilledMaxMilliC}
		}
		var delta int64
		switch {
		case rd.TempMilliC < band.MinMilliC:
			delta = band.MinMilliC - rd.TempMilliC
		case rd.TempMilliC > band.MaxMilliC:
			delta = rd.TempMilliC - band.MaxMilliC
		}
		if delta == 0 {
			continue
		}
		excursionManifestSet[rd.ManifestID] = struct{}{}
		report.ExcursionCount++
		report.Excursions = append(report.Excursions, HACCPExcursion{
			ReadingID:         rd.ReadingID,
			ManifestID:        rd.ManifestID,
			WarehouseID:       rd.WarehouseID,
			SensorID:          rd.SensorID,
			RecordedAt:        rd.RecordedAt.UTC().Format(time.RFC3339),
			TempMilliC:        rd.TempMilliC,
			MinMilliC:         band.MinMilliC,
			MaxMilliC:         band.MaxMilliC,
			BreachDeltaMilliC: delta,
			Severity:          classifyHACCPExcursionSeverity(delta),
			Status:            resolveManifestExcursionStatus(manifestOrders[rd.ManifestID], orderBreachStatus),
		})
	}
	if report.TotalReadings > 0 {
		compliant := report.TotalReadings - report.ExcursionCount
		if compliant < 0 {
			compliant = 0
		}
		report.ComplianceRateBps = (compliant * 10000) / report.TotalReadings
	}

	excursionLotManifest := make(map[string]string)
	targetOrderSet := make(map[string]string)
	for mid := range excursionManifestSet {
		for _, lotID := range pickLotsByManifest[mid] {
			if _, exists := excursionLotManifest[lotID]; !exists {
				excursionLotManifest[lotID] = mid
			}
		}
		for _, oid := range manifestOrders[mid] {
			if _, exists := targetOrderSet[oid]; !exists {
				targetOrderSet[oid] = mid
			}
		}
	}
	for _, oid := range breachOrderIDs {
		if _, exists := targetOrderSet[oid]; !exists {
			targetOrderSet[oid] = ""
		}
	}
	if len(targetOrderSet) > 0 {
		orderIDs := make([]string, 0, len(targetOrderSet))
		for oid := range targetOrderSet {
			orderIDs = append(orderIDs, oid)
		}
		reservedLots, err := loadReservedLotsForOrdersRO(ctx, ro, orderIDs)
		if err != nil {
			return HACCPReport{}, 0, err
		}
		for _, rl := range reservedLots {
			if _, exists := excursionLotManifest[rl.LotID]; !exists {
				excursionLotManifest[rl.LotID] = targetOrderSet[rl.OrderID]
			}
		}
	}

	activeLots, err := loadSupplierStockLotsRO(ctx, ro, tid)
	if err != nil {
		return HACCPReport{}, 0, err
	}
	lotByID := make(map[string]haccpStockLotRow, len(activeLots))
	productIDs := make([]string, 0, len(activeLots))
	warehouseIDs := make([]string, 0, 8)
	for _, lot := range activeLots {
		lotByID[lot.LotID] = lot
		productIDs = appendUniqueTrimmed(productIDs, lot.ProductID)
		warehouseIDs = appendUniqueTrimmed(warehouseIDs, lot.WarehouseID)
	}

	var missingLotIDs []string
	for lotID := range excursionLotManifest {
		if _, ok := lotByID[lotID]; !ok {
			missingLotIDs = append(missingLotIDs, lotID)
		}
	}
	if len(missingLotIDs) > 0 {
		extraLots, err := loadStockLotsByIDsRO(ctx, ro, tid, missingLotIDs)
		if err != nil {
			return HACCPReport{}, 0, err
		}
		for _, lot := range extraLots {
			lotByID[lot.LotID] = lot
			productIDs = appendUniqueTrimmed(productIDs, lot.ProductID)
			warehouseIDs = appendUniqueTrimmed(warehouseIDs, lot.WarehouseID)
		}
	}

	prodMeta, err := loadProductsColdMetaRO(ctx, ro, productIDs)
	if err != nil {
		return HACCPReport{}, 0, err
	}
	binMetaByWh, err := loadWarehouseBinsMetaRO(ctx, ro, warehouseIDs)
	if err != nil {
		return HACCPReport{}, 0, err
	}

	affectedMap := make(map[string]HACCPAffectedLot)
	for _, lot := range lotByID {
		p := prodMeta[strings.TrimSpace(lot.ProductID)]
		b := binMetaByWh[lotKey(lot.WarehouseID, lot.LocationID)]
		reqZone, _, warn := ClassifyColdLotState(
			p.RequiresColdChain, p.IsPerishable, p.HandlingClass,
			lot.LocationID, b.Zone, b.LocationType,
		)
		hasZoneMismatch := warn != "" && lot.QuantityOnHand > 0
		if hasZoneMismatch {
			report.ColdZoneMismatches = append(report.ColdZoneMismatches, HACCPLotZoneMismatch{
				LotID:        lot.LotID,
				WarehouseID:  lot.WarehouseID,
				ProductID:    lot.ProductID,
				ProductName:  p.Name,
				LocationID:   lot.LocationID,
				RequiredZone: reqZone,
				LocationZone: b.Zone,
				LocationType: b.LocationType,
				Warning:      warn,
			})
		}

		excManifestID, hasExcursion := excursionLotManifest[lot.LotID]
		if !hasZoneMismatch && !hasExcursion {
			continue
		}
		reason := "COLD_ZONE_MISMATCH"
		if hasExcursion && hasZoneMismatch {
			reason = "EXCURSION_AND_ZONE_MISMATCH"
		} else if hasExcursion {
			if excManifestID != "" {
				reason = "EXCURSION_MANIFEST"
			} else {
				reason = "TEMPERATURE_BREACH"
			}
		}
		affectedMap[lot.LotID] = HACCPAffectedLot{
			LotID:          lot.LotID,
			WarehouseID:    lot.WarehouseID,
			ProductID:      lot.ProductID,
			ProductName:    p.Name,
			LocationID:     lot.LocationID,
			LotCode:        lot.LotCode,
			Status:         lot.Status,
			QuantityOnHand: lot.QuantityOnHand,
			RequiredZone:   reqZone,
			LocationZone:   b.Zone,
			Reason:         reason,
			ManifestID:     excManifestID,
		}
	}

	report.AffectedLots = make([]HACCPAffectedLot, 0, len(affectedMap))
	for _, al := range affectedMap {
		report.AffectedLots = append(report.AffectedLots, al)
	}
	sort.Slice(report.AffectedLots, func(i, j int) bool {
		return report.AffectedLots[i].LotID < report.AffectedLots[j].LotID
	})
	sort.Slice(report.ColdZoneMismatches, func(i, j int) bool {
		return report.ColdZoneMismatches[i].LotID < report.ColdZoneMismatches[j].LotID
	})
	report.AffectedLotsCount = int64(len(report.AffectedLots))
	return report, resolvedBreaches, nil
}

// EvaluateHACCPCompliance computes the HACCP report and emits EventHaccpExcursionReported when open excursion breaches exist.
func EvaluateHACCPCompliance(ctx context.Context, client *spanner.Client, tenantID string, from, to, now time.Time, actorID string) (HACCPReport, error) {
	return evaluateHACCPComplianceWithOpt(ctx, client, tenantID, from, to, true, now, actorID)
}

func evaluateHACCPComplianceWithOpt(ctx context.Context, client *spanner.Client, tenantID string, from, to time.Time, explicitTo bool, now time.Time, actorID string) (HACCPReport, error) {
	report, resolvedBreaches, err := computeHACCPReportWithOpt(ctx, client, tenantID, from, to, explicitTo, now)
	if err != nil {
		return HACCPReport{}, err
	}
	if report.OpenBreaches > 0 || (report.ExcursionCount > 0 && resolvedBreaches == 0) {
		if err := emitHACCPExcursionOutbox(ctx, client, report, actorID); err != nil {
			return HACCPReport{}, err
		}
	}
	return report, nil
}

func emitHACCPExcursionOutbox(ctx context.Context, client *spanner.Client, report HACCPReport, actorID string) error {
	if client == nil {
		return nil
	}
	lotIDs := make([]string, 0, len(report.AffectedLots))
	for _, al := range report.AffectedLots {
		lotIDs = append(lotIDs, al.LotID)
	}
	var manifestID, readingID, sensorID string
	var tempMilliC, minMilliC, maxMilliC int64
	if len(report.Excursions) > 0 {
		first := report.Excursions[0]
		manifestID = first.ManifestID
		readingID = first.ReadingID
		sensorID = first.SensorID
		tempMilliC = first.TempMilliC
		minMilliC = first.MinMilliC
		maxMilliC = first.MaxMilliC
	}
	aggID := strings.TrimSpace(manifestID)
	if aggID == "" {
		aggID = report.TenantID
	}

	_, err := client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		ev := events.HaccpExcursionReportedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventHaccpExcursionReported,
				Version:   1,
				Timestamp: report.GeneratedAt,
			},
			TenantID:          report.TenantID,
			ManifestID:        manifestID,
			ReadingID:         readingID,
			SensorID:          sensorID,
			TempMilliC:        tempMilliC,
			MinMilliC:         minMilliC,
			MaxMilliC:         maxMilliC,
			TotalReadings:     report.TotalReadings,
			ExcursionCount:    report.ExcursionCount,
			ComplianceRateBps: report.ComplianceRateBps,
			OpenBreaches:      report.OpenBreaches,
			AffectedLotIDs:    lotIDs,
			Status:            "OPEN_EXCURSIONS",
			ActorID:           strings.TrimSpace(actorID),
			OccurredAt:        report.GeneratedAt,
		}
		buf := &haccpOutboxBuffer{
			inner:    outbox.NewSpannerTxnBuffer(txn),
			tenantID: report.TenantID,
		}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateWarehouse, aggID, events.TopicTelemetryLogistics, ev); err != nil {
			return err
		}
		return buf.inner.Flush(txCtx)
	})
	return err
}

func loadHACCPReadingsRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, tid string, from, to time.Time, explicitTo bool) ([]haccpTempReadingRow, []string, error) {
	sqlStr := `SELECT tr.ReadingId,
		             tr.ManifestId,
		             COALESCE(m.WarehouseId, ''),
		             COALESCE(tr.SensorId, ''),
		             tr.RecordedAt,
		             CAST(ROUND(tr.TempC * 1000) AS INT64) AS TempMilliC
		      FROM TemperatureReadings tr
		      JOIN SupplierTruckManifests m ON m.ManifestId = tr.ManifestId
		      WHERE m.SupplierId = @sid
		        AND tr.RecordedAt >= @from`
	params := map[string]any{
		"sid":  tid,
		"from": from.UTC(),
	}
	if explicitTo {
		sqlStr += ` AND tr.RecordedAt <= @to`
		params["to"] = to.UTC()
	}
	sqlStr += ` ORDER BY tr.RecordedAt DESC LIMIT 5000`
	iter := ro.Query(ctx, spanner.Statement{SQL: sqlStr, Params: params})
	defer iter.Stop()

	var rows []haccpTempReadingRow
	manifestSet := make(map[string]struct{})
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("query TemperatureReadings: %w", err)
		}
		var r haccpTempReadingRow
		if err := row.Columns(&r.ReadingID, &r.ManifestID, &r.WarehouseID, &r.SensorID, &r.RecordedAt, &r.TempMilliC); err != nil {
			return nil, nil, fmt.Errorf("scan TemperatureReadings: %w", err)
		}
		rows = append(rows, r)
		manifestSet[r.ManifestID] = struct{}{}
	}
	manifests := make([]string, 0, len(manifestSet))
	for mid := range manifestSet {
		manifests = append(manifests, mid)
	}
	return rows, manifests, nil
}

func loadHACCPConditionReportsRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, tid string, from, to time.Time, explicitTo bool) (
	reports []HACCPConditionReport,
	openCount int64,
	resolvedCount int64,
	orderOpen map[string]bool,
	breachOrderIDs []string,
	err error,
) {
	orderOpen = make(map[string]bool)
	sqlStr := `SELECT ReportId,
		             OrderId,
		             COALESCE(RetailerId, ''),
		             COALESCE(SKU, ''),
		             COALESCE(ConditionType, ''),
		             COALESCE(Severity, ''),
		             COALESCE(ResolutionStatus, 'OPEN'),
		             COALESCE(Description, ''),
		             COALESCE(ReportedBy, ''),
		             CreatedAt
		      FROM OrderConditionReports
		      WHERE SupplierId = @sid
		        AND ConditionType = 'TEMPERATURE_BREACH'
		        AND CreatedAt >= @from`
	params := map[string]any{
		"sid":  tid,
		"from": from.UTC(),
	}
	if explicitTo {
		sqlStr += ` AND CreatedAt <= @to`
		params["to"] = to.UTC()
	}
	sqlStr += ` ORDER BY CreatedAt DESC LIMIT 1000`
	iter := ro.Query(ctx, spanner.Statement{SQL: sqlStr, Params: params})
	defer iter.Stop()

	seenOrders := make(map[string]struct{})
	for {
		row, rerr := iter.Next()
		if rerr == iterator.Done {
			break
		}
		if rerr != nil {
			return nil, 0, 0, nil, nil, fmt.Errorf("query OrderConditionReports: %w", rerr)
		}
		var cr HACCPConditionReport
		var createdAt time.Time
		if serr := row.Columns(&cr.ReportID, &cr.OrderID, &cr.RetailerID, &cr.SKU, &cr.ConditionType, &cr.Severity, &cr.ResolutionStatus, &cr.Description, &cr.ReportedBy, &createdAt); serr != nil {
			return nil, 0, 0, nil, nil, fmt.Errorf("scan OrderConditionReports: %w", serr)
		}
		cr.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		reports = append(reports, cr)

		normStatus := strings.ToUpper(strings.TrimSpace(cr.ResolutionStatus))
		if normStatus == "RESOLVED" || normStatus == "CLOSED" || normStatus == "DISMISSED" {
			resolvedCount++
			if _, exists := orderOpen[cr.OrderID]; !exists {
				orderOpen[cr.OrderID] = false
			}
		} else {
			openCount++
			orderOpen[cr.OrderID] = true
		}
		if _, exists := seenOrders[cr.OrderID]; !exists && strings.TrimSpace(cr.OrderID) != "" {
			seenOrders[cr.OrderID] = struct{}{}
			breachOrderIDs = append(breachOrderIDs, cr.OrderID)
		}
	}
	return reports, openCount, resolvedCount, orderOpen, breachOrderIDs, nil
}

func loadManifestColdContextRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, manifestIDs []string) (
	bands map[string]haccpTempBand,
	manifestOrders map[string][]string,
	pickLotsByManifest map[string][]string,
	err error,
) {
	bands = make(map[string]haccpTempBand, len(manifestIDs))
	manifestOrders = make(map[string][]string, len(manifestIDs))
	pickLotsByManifest = make(map[string][]string, len(manifestIDs))
	if len(manifestIDs) == 0 {
		return bands, manifestOrders, pickLotsByManifest, nil
	}

	manifestProducts := make(map[string][]string)
	allProductSet := make(map[string]struct{})
	addOrder := func(mid, oid string) {
		if strings.TrimSpace(oid) != "" {
			manifestOrders[mid] = appendUniqueTrimmed(manifestOrders[mid], oid)
		}
	}
	addProd := func(mid, pid string) {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			return
		}
		manifestProducts[mid] = appendUniqueTrimmed(manifestProducts[mid], pid)
		allProductSet[pid] = struct{}{}
	}

	for _, stmt := range []spanner.Statement{
		{
			SQL: `SELECT mo.ManifestId, mo.OrderId, o.LineItemsJson
			      FROM ManifestOrders mo
			      JOIN Orders o ON o.OrderId = mo.OrderId
			      WHERE mo.ManifestId IN UNNEST(@mids)`,
			Params: map[string]any{"mids": manifestIDs},
		},
		{
			SQL: `SELECT ManifestId, OrderId, LineItemsJson
			      FROM Orders
			      WHERE ManifestId IN UNNEST(@mids)`,
			Params: map[string]any{"mids": manifestIDs},
		},
	} {
		iter := ro.Query(ctx, stmt)
		for {
			row, rerr := iter.Next()
			if rerr == iterator.Done {
				break
			}
			if rerr != nil {
				iter.Stop()
				return nil, nil, nil, fmt.Errorf("query manifest orders: %w", rerr)
			}
			var mid, oid string
			var raw []byte
			if serr := row.Columns(&mid, &oid, &raw); serr != nil {
				continue
			}
			addOrder(mid, oid)
			for _, pid := range extractProductIDsFromLineItems(raw) {
				addProd(mid, pid)
			}
		}
		iter.Stop()
	}

	ptIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT pw.ManifestId, pt.OrderId, pt.ProductId, pt.LotId
		      FROM PickWaves pw
		      JOIN PickTasks pt ON pt.WaveId = pw.WaveId
		      WHERE pw.ManifestId IN UNNEST(@mids)`,
		Params: map[string]any{"mids": manifestIDs},
	})
	for {
		row, rerr := ptIter.Next()
		if rerr == iterator.Done {
			break
		}
		if rerr != nil {
			ptIter.Stop()
			return nil, nil, nil, fmt.Errorf("query pick tasks for manifests: %w", rerr)
		}
		var mid, oid, pid, lid string
		if serr := row.Columns(&mid, &oid, &pid, &lid); serr == nil {
			addOrder(mid, oid)
			addProd(mid, pid)
			pickLotsByManifest[mid] = appendUniqueTrimmed(pickLotsByManifest[mid], lid)
		}
	}
	ptIter.Stop()

	allProductIDs := make([]string, 0, len(allProductSet))
	for pid := range allProductSet {
		allProductIDs = append(allProductIDs, pid)
	}
	prodMeta, err := loadProductsColdMetaRO(ctx, ro, allProductIDs)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, mid := range manifestIDs {
		band := haccpTempBand{MinMilliC: defaultChilledMinMilliC, MaxMilliC: defaultChilledMaxMilliC}
		for _, pid := range manifestProducts[mid] {
			if strings.EqualFold(strings.TrimSpace(prodMeta[pid].HandlingClass), "FROZEN") {
				band = haccpTempBand{MinMilliC: defaultFrozenMinMilliC, MaxMilliC: defaultFrozenMaxMilliC}
				break
			}
		}
		bands[mid] = band
	}
	return bands, manifestOrders, pickLotsByManifest, nil
}

type haccpReservedLotRef struct {
	OrderID string
	LotID   string
}

func loadReservedLotsForOrdersRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, orderIDs []string) ([]haccpReservedLotRef, error) {
	if len(orderIDs) == 0 {
		return nil, nil
	}
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT OrderId, LotId FROM OrderLotReservations WHERE OrderId IN UNNEST(@oids)`,
		Params: map[string]any{"oids": orderIDs},
	})
	defer iter.Stop()
	var out []haccpReservedLotRef
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query OrderLotReservations: %w", err)
		}
		var ref haccpReservedLotRef
		if err := row.Columns(&ref.OrderID, &ref.LotID); err == nil {
			out = append(out, ref)
		}
	}
	return out, nil
}

func loadSupplierStockLotsRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, tid string) ([]haccpStockLotRow, error) {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT LotId, WarehouseId, ProductId, LocationId, COALESCE(LotCode, ''), COALESCE(Status, ''), QuantityOnHand
		      FROM StockLots
		      WHERE SupplierId = @sid AND QuantityOnHand > 0
		      LIMIT 2000`,
		Params: map[string]any{"sid": tid},
	})
	defer iter.Stop()
	var out []haccpStockLotRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query StockLots: %w", err)
		}
		var r haccpStockLotRow
		if err := row.Columns(&r.LotID, &r.WarehouseID, &r.ProductID, &r.LocationID, &r.LotCode, &r.Status, &r.QuantityOnHand); err == nil {
			out = append(out, r)
		}
	}
	return out, nil
}

func loadStockLotsByIDsRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, tid string, lotIDs []string) ([]haccpStockLotRow, error) {
	if len(lotIDs) == 0 {
		return nil, nil
	}
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT LotId, WarehouseId, ProductId, LocationId, COALESCE(LotCode, ''), COALESCE(Status, ''), QuantityOnHand
		      FROM StockLots
		      WHERE SupplierId = @sid AND LotId IN UNNEST(@lids)`,
		Params: map[string]any{"sid": tid, "lids": lotIDs},
	})
	defer iter.Stop()
	var out []haccpStockLotRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query StockLots by id: %w", err)
		}
		var r haccpStockLotRow
		if err := row.Columns(&r.LotID, &r.WarehouseID, &r.ProductID, &r.LocationID, &r.LotCode, &r.Status, &r.QuantityOnHand); err == nil {
			out = append(out, r)
		}
	}
	return out, nil
}

func loadProductsColdMetaRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, pids []string) (map[string]productColdMeta, error) {
	out := make(map[string]productColdMeta, len(pids))
	if len(pids) == 0 {
		return out, nil
	}
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, COALESCE(Name, ''), COALESCE(Barcode, ''), COALESCE(HandlingClass, ''), IsPerishable, RequiresColdChain
		      FROM Products WHERE ProductId IN UNNEST(@pids)`,
		Params: map[string]any{"pids": pids},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query Products: %w", err)
		}
		var pid, name, barcode, hc string
		var perishable, cold bool
		if err := row.Columns(&pid, &name, &barcode, &hc, &perishable, &cold); err == nil {
			out[strings.TrimSpace(pid)] = productColdMeta{
				Name:              name,
				Barcode:           barcode,
				HandlingClass:     hc,
				IsPerishable:      perishable,
				RequiresColdChain: cold,
			}
		}
	}
	return out, nil
}

func lotKey(warehouseID, locationID string) string {
	return strings.TrimSpace(warehouseID) + "::" + strings.TrimSpace(locationID)
}

func loadWarehouseBinsMetaRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, warehouseIDs []string) (map[string]binZoneMeta, error) {
	out := make(map[string]binZoneMeta)
	if len(warehouseIDs) == 0 {
		return out, nil
	}
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, LocationId, COALESCE(Zone, ''), COALESCE(LocationType, '')
		      FROM WarehouseLocations
		      WHERE WarehouseId IN UNNEST(@wids)`,
		Params: map[string]any{"wids": warehouseIDs},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query WarehouseLocations: %w", err)
		}
		var wid, lid, zone, locType string
		if err := row.Columns(&wid, &lid, &zone, &locType); err == nil {
			out[lotKey(wid, lid)] = binZoneMeta{Zone: zone, LocationType: locType}
		}
	}
	return out, nil
}

func classifyHACCPExcursionSeverity(deltaMilliC int64) string {
	switch {
	case deltaMilliC >= 5000:
		return "CRITICAL"
	case deltaMilliC >= 2000:
		return "HIGH"
	default:
		return "MEDIUM"
	}
}

func resolveManifestExcursionStatus(orderIDs []string, orderOpen map[string]bool) string {
	if len(orderIDs) == 0 {
		return "OPEN"
	}
	sawReport := false
	for _, oid := range orderIDs {
		isOpen, exists := orderOpen[oid]
		if !exists {
			continue
		}
		sawReport = true
		if isOpen {
			return "OPEN"
		}
	}
	if sawReport {
		return "RESOLVED"
	}
	return "OPEN"
}

func safeCSVCell(s string) string {
	if s == "" {
		return ""
	}
	switch s[0] {
	case '=', '+', '@', '\t', '\r':
		return "'" + s
	case '-':
		if _, err := strconv.ParseInt(s, 10, 64); err == nil {
			return s
		}
		return "'" + s
	default:
		return s
	}
}

func marshalHACCPCSV(r HACCPReport) ([]byte, error) {
	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	header := []string{
		"record_type", "tenant_id", "from", "to", "id", "manifest_or_order_id",
		"warehouse_or_sku", "product_or_sensor", "location_id", "temp_milli_c",
		"min_milli_c", "max_milli_c", "quantity_on_hand", "compliance_rate_bps",
		"severity", "status", "reason_or_warning", "recorded_or_created_at",
	}
	if err := cw.Write(header); err != nil {
		return nil, err
	}
	if err := cw.Write([]string{
		"SUMMARY", safeCSVCell(r.TenantID), r.From, r.To, "summary", "",
		"", "", "", "", "", "",
		strconv.FormatInt(r.AffectedLotsCount, 10),
		strconv.FormatInt(r.ComplianceRateBps, 10),
		"", fmt.Sprintf("readings=%d;excursions=%d;open_breaches=%d", r.TotalReadings, r.ExcursionCount, r.OpenBreaches),
		"", r.GeneratedAt,
	}); err != nil {
		return nil, err
	}
	for _, ex := range r.Excursions {
		if err := cw.Write([]string{
			"EXCURSION", safeCSVCell(r.TenantID), r.From, r.To, safeCSVCell(ex.ReadingID),
			safeCSVCell(ex.ManifestID), safeCSVCell(ex.WarehouseID), safeCSVCell(ex.SensorID), "",
			strconv.FormatInt(ex.TempMilliC, 10), strconv.FormatInt(ex.MinMilliC, 10), strconv.FormatInt(ex.MaxMilliC, 10),
			"", strconv.FormatInt(r.ComplianceRateBps, 10),
			safeCSVCell(ex.Severity), safeCSVCell(ex.Status),
			strconv.FormatInt(ex.BreachDeltaMilliC, 10), ex.RecordedAt,
		}); err != nil {
			return nil, err
		}
	}
	for _, lot := range r.AffectedLots {
		if err := cw.Write([]string{
			"AFFECTED_LOT", safeCSVCell(r.TenantID), r.From, r.To, safeCSVCell(lot.LotID),
			safeCSVCell(lot.ManifestID), safeCSVCell(lot.WarehouseID), safeCSVCell(lot.ProductName), safeCSVCell(lot.LocationID),
			"", "", "", strconv.FormatInt(lot.QuantityOnHand, 10), "",
			"", safeCSVCell(lot.Status), safeCSVCell(lot.Reason), r.GeneratedAt,
		}); err != nil {
			return nil, err
		}
	}
	for _, mm := range r.ColdZoneMismatches {
		if err := cw.Write([]string{
			"COLD_ZONE_MISMATCH", safeCSVCell(r.TenantID), r.From, r.To, safeCSVCell(mm.LotID),
			"", safeCSVCell(mm.WarehouseID), safeCSVCell(mm.ProductName), safeCSVCell(mm.LocationID),
			"", "", "", "", "",
			"HIGH", safeCSVCell(mm.RequiredZone + " vs " + mm.LocationZone), safeCSVCell(mm.Warning), r.GeneratedAt,
		}); err != nil {
			return nil, err
		}
	}
	for _, cr := range r.ConditionReports {
		if err := cw.Write([]string{
			"CONDITION_REPORT", safeCSVCell(r.TenantID), r.From, r.To, safeCSVCell(cr.ReportID),
			safeCSVCell(cr.OrderID), safeCSVCell(cr.SKU), safeCSVCell(cr.ReportedBy), "",
			"", "", "", "", "",
			safeCSVCell(cr.Severity), safeCSVCell(cr.ResolutionStatus), safeCSVCell(cr.Description), cr.CreatedAt,
		}); err != nil {
			return nil, err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
