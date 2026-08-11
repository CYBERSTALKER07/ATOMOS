package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// ConditionType classifies an order condition report.
type ConditionType string

const (
	ConditionTypeDamaged           ConditionType = "DAMAGED"
	ConditionTypeExpired           ConditionType = "EXPIRED"
	ConditionTypeTemperatureBreach ConditionType = "TEMPERATURE_BREACH"
	ConditionTypeMissing           ConditionType = "MISSING"
	ConditionTypeQualityReject     ConditionType = "QUALITY_REJECT"
	ConditionTypeOther             ConditionType = "OTHER"
)

// Valid returns true for known condition types.
func (c ConditionType) Valid() bool {
	switch c {
	case ConditionTypeDamaged, ConditionTypeExpired, ConditionTypeTemperatureBreach,
		ConditionTypeMissing, ConditionTypeQualityReject, ConditionTypeOther:
		return true
	}
	return false
}

// Severity classifies the operational urgency of a condition report.
type Severity string

const (
	SeverityLow    Severity = "LOW"
	SeverityMedium Severity = "MEDIUM"
	SeverityHigh   Severity = "HIGH"
)

// ResolutionStatus tracks whether a condition report has been addressed.
type ResolutionStatus string

const (
	ResolutionStatusOpen      ResolutionStatus = "OPEN"
	ResolutionStatusResolved  ResolutionStatus = "RESOLVED"
	ResolutionStatusDisputed  ResolutionStatus = "DISPUTED"
	ResolutionStatusEscalated ResolutionStatus = "ESCALATED"
)

// ConditionReport is an immutable quality/condition record linked to an order.
type ConditionReport struct {
	ReportID         string           `json:"report_id"`
	OrderID          string           `json:"order_id"`
	SupplierID       string           `json:"supplier_id"`
	RetailerID       string           `json:"retailer_id"`
	LineItemIndex    *int64           `json:"line_item_index,omitempty"`
	SKU              string           `json:"sku,omitempty"`
	ConditionType    ConditionType    `json:"condition_type"`
	Severity         Severity         `json:"severity"`
	Description      string           `json:"description,omitempty"`
	PhotoURLs        []string         `json:"photo_urls,omitempty"`
	ProofIDs         []string         `json:"proof_ids,omitempty"`
	ReportedBy       string           `json:"reported_by"`
	ReportedByRole   string           `json:"reported_by_role"`
	ResolutionStatus ResolutionStatus `json:"resolution_status"`
	ResolvedBy       string           `json:"resolved_by,omitempty"`
	ResolvedAt       *time.Time       `json:"resolved_at,omitempty"`
	ResolutionNotes  string           `json:"resolution_notes,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

// ConditionReportRequest is the payload for POST /v1/delivery/report-condition.
type ConditionReportRequest struct {
	OrderID       string        `json:"order_id"`
	LineItemIndex *int64        `json:"line_item_index,omitempty"`
	SKU           string        `json:"sku,omitempty"`
	ConditionType ConditionType `json:"condition_type"`
	Severity      Severity      `json:"severity,omitempty"`
	Description   string        `json:"description,omitempty"`
	PhotoURLs     []string      `json:"photo_urls,omitempty"`
	ProofIDs      []string      `json:"proof_ids,omitempty"`
}

// normalizeSeverity returns a valid severity or defaults to MEDIUM.
func normalizeSeverity(s Severity) Severity {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh:
		return s
	}
	return SeverityMedium
}

// reporterAuthorizedForOrder returns true when the caller may file a condition
// report against the order. Drivers may report their own assigned order;
// retailers may report their own order; warehouse/factory staff may report
// orders assigned to a node they operate.
func reporterAuthorizedForOrder(claims auth.Claims, o Order) bool {
	switch claims.Role {
	case auth.RoleDriver:
		return o.DriverID != "" && o.DriverID == claims.Subject
	case auth.RoleRetailer:
		return o.RetailerID != "" && o.RetailerID == claims.Subject
	case auth.RoleWarehouseAdmin, auth.RoleWarehouse, auth.RoleFactoryAdmin:
		return o.WarehouseID != "" && o.WarehouseID == claims.HomeNodeID
	}
	return false
}

// conditionReportable returns true when the order status allows a condition report.
func conditionReportable(status Status) bool {
	switch status {
	case StatusInTransit, StatusArrived, StatusAwaitingPayment,
		StatusPendingCashCollection, StatusDeliveredOnCredit, StatusCompleted:
		return true
	}
	return false
}

// ReportCondition persists a structured condition report and emits an event.
func (s *Service) ReportCondition(ctx context.Context, claims auth.Claims, req ConditionReportRequest) (ConditionReport, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return ConditionReport{}, errors.New("order_id required")
	}
	if !req.ConditionType.Valid() {
		return ConditionReport{}, errors.New("invalid condition_type")
	}

	current, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return ConditionReport{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !found {
		return ConditionReport{}, ErrOrderNotFound
	}
	if !reporterAuthorizedForOrder(claims, current) {
		return ConditionReport{}, ErrOrderForbidden
	}
	if !conditionReportable(current.Status) {
		return ConditionReport{}, fmt.Errorf("order %s status %s cannot receive condition report", orderID, current.Status)
	}

	report := ConditionReport{
		ReportID:         s.newID(),
		OrderID:          current.OrderID,
		SupplierID:       current.SupplierID,
		RetailerID:       current.RetailerID,
		LineItemIndex:    req.LineItemIndex,
		SKU:              strings.TrimSpace(req.SKU),
		ConditionType:    req.ConditionType,
		Severity:         normalizeSeverity(req.Severity),
		Description:      strings.TrimSpace(req.Description),
		PhotoURLs:        req.PhotoURLs,
		ProofIDs:         req.ProofIDs,
		ReportedBy:       claims.Subject,
		ReportedByRole:   string(claims.Role),
		ResolutionStatus: ResolutionStatusOpen,
		CreatedAt:        s.now(),
	}

	if err := s.repo.CreateConditionReport(ctx, report, func(txn outbox.TxnBuffer) error {
		payload := events.ConditionEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventOrderConditionReported,
				Timestamp: report.CreatedAt.UTC().Format(time.RFC3339Nano),
			},
			ReportID:      report.ReportID,
			OrderID:       report.OrderID,
			SupplierID:    report.SupplierID,
			RetailerID:    report.RetailerID,
			ReporterID:    report.ReportedBy,
			ReporterRole:  report.ReportedByRole,
			ConditionType: string(report.ConditionType),
			SKU:           report.SKU,
			Quantity:      0,
			GCSPaths:      report.PhotoURLs,
			Notes:         report.Description,
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateConditionReport, report.ReportID, events.TopicMain, payload)
	}); err != nil {
		return ConditionReport{}, fmt.Errorf("create condition report for order %s: %w", orderID, err)
	}

	s.afterOrderMutation(ctx, current)
	return report, nil
}

// HandleReportCondition serves POST /v1/delivery/report-condition.
func (s *Service) HandleReportCondition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req ConditionReportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	report, err := s.ReportCondition(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "report condition failed", req.OrderID, err)
		return
	}

	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"success":     true,
		"report_id":   report.ReportID,
		"condition":   report.ConditionType,
		"severity":    report.Severity,
		"reported_at": report.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

// ListConditionReports serves GET /v1/order/{orderID}/condition-reports.
func (s *Service) ListConditionReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	current, found, err := s.loadOrderForRequest(r.Context(), orderID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list condition reports failed", "err", err, "order_id", orderID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if !reporterAuthorizedForOrder(claims, current) && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	reports, err := s.repo.ListConditionReports(r.Context(), orderID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list condition reports failed", "err", err, "order_id", orderID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order_id": orderID, "reports": reports})
}
