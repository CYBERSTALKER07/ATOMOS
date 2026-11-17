package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

const (
	defaultSupplierTimezone = "Asia/Tashkent"
	defaultCutoffHHMM       = "18:00"
	defaultLeadTimeDays     = int64(1)
)

// FastHomeDeliverySummary is the compact summary of the active/upcoming delivery.
type FastHomeDeliverySummary struct {
	OrderID      string `json:"order_id"`
	SupplierID   string `json:"supplier_id,omitempty"`
	SupplierName string `json:"supplier_name,omitempty"`
	Status       string `json:"status"`
	ETAAt        string `json:"eta_at,omitempty"`
	DeliveryDate string `json:"delivery_date,omitempty"`
	TotalMinor   int64  `json:"total_minor"`
	Currency     string `json:"currency"`
	ItemCount    int    `json:"item_count,omitempty"`
	DriverID     string `json:"driver_id,omitempty"`
}

// FastHomeProposalItem represents one SKU line inside the pending AI/preorder proposal.
type FastHomeProposalItem struct {
	SKU            string `json:"sku"`
	ProductID      string `json:"product_id,omitempty"`
	Name           string `json:"name,omitempty"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor,omitempty"`
	LineTotalMinor int64  `json:"line_total_minor,omitempty"`
}

// FastHomePendingProposal is the latest unconfirmed AI or scheduled preorder draft.
type FastHomePendingProposal struct {
	OrderID               string                 `json:"order_id"`
	SupplierID            string                 `json:"supplier_id,omitempty"`
	SupplierName          string                 `json:"supplier_name,omitempty"`
	OrderSource           string                 `json:"order_source,omitempty"`
	ConfirmationStatus    string                 `json:"confirmation_status"`
	AutoConfirmOptIn      bool                   `json:"auto_confirm_opt_in"`
	AutoConfirmAt         string                 `json:"auto_confirm_at,omitempty"`
	CanConfirm            bool                   `json:"can_confirm"`
	TotalMinor            int64                  `json:"total_minor"`
	Currency              string                 `json:"currency"`
	RequestedDeliveryDate string                 `json:"requested_delivery_date,omitempty"`
	Items                 []FastHomeProposalItem `json:"items"`
}

// FastHomeActivePromise summarizes the nearest active payment promise for the retailer.
type FastHomeActivePromise struct {
	PromiseID     string `json:"promise_id"`
	SupplierID    string `json:"supplier_id,omitempty"`
	DueAt         string `json:"due_at"`
	PromisedMinor int64  `json:"promised_minor"`
	Status        string `json:"status"`
}

// FastHomeResponse is the consolidated Fast Mode home payload served by GET /v1/retailer/fast-home.
type FastHomeResponse struct {
	ServerNow             string                   `json:"server_now"`
	CutoffAt              string                   `json:"cutoff_at"`
	NextDeliveryDate      string                   `json:"next_delivery_date"`
	TomorrowDelivery      *FastHomeDeliverySummary `json:"tomorrow_delivery"`
	BalanceOwedMinor      int64                    `json:"balance_owed_minor"`
	OverdueMinor          int64                    `json:"overdue_minor"`
	DaysOverdue           int                      `json:"days_overdue"`
	ActivePromise         *FastHomeActivePromise   `json:"active_promise"`
	CreditHeadroomMinor   int64                    `json:"credit_headroom_minor"`
	PendingProposal       *FastHomePendingProposal `json:"pending_proposal"`
	PendingApprovalsCount int64                    `json:"pending_approvals_count"`
	Locale                string                   `json:"locale"`
}

// SetFastHomeQuery overrides fast-home loading in unit tests without Spanner.
func (s *Service) SetFastHomeQuery(fn func(context.Context, string, string) (*FastHomeResponse, error)) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.fastHomeQuery = fn
	s.mu.Unlock()
}

// HandleFastHome serves GET /v1/retailer/fast-home in <= 2 Spanner roundtrips.
func (s *Service) HandleFastHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	claims, hasClaims := auth.FromContext(r.Context())
	userID := ""
	isOwner := true
	if hasClaims {
		userID = strings.TrimSpace(claims.RetailerUserID)
		if userID == "" {
			userID = strings.TrimSpace(claims.Subject)
		}
		isOwner = auth.EffectiveRetailerRole(claims) == "OWNER"
	}

	s.mu.RLock()
	queryFn := s.fastHomeQuery
	s.mu.RUnlock()
	if queryFn != nil {
		resp, qErr := queryFn(r.Context(), retailerID, userID)
		if qErr != nil {
			s.log.Warn("fast-home query hook failed", "retailer_id", retailerID, "err", qErr)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_fast_home_failed"})
			return
		}
		if resp == nil {
			resp = &FastHomeResponse{}
		}
		if !isOwner {
			resp.PendingApprovalsCount = 0
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	resp, err := s.buildFastHome(r.Context(), retailerID, userID, isOwner)
	if err != nil {
		s.log.Warn("fast-home load failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_fast_home_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) buildFastHome(ctx context.Context, retailerID, userID string, isOwner bool) (*FastHomeResponse, error) {
	now := s.now().UTC()
	out := &FastHomeResponse{
		ServerNow: now.Format(time.RFC3339Nano),
		Locale:    normalizeRetailerLocale(""),
	}

	if s.spannerClient != nil {
		return s.buildFastHomeSpanner(ctx, retailerID, userID, isOwner, now)
	}
	return s.buildFastHomeMemory(ctx, retailerID, userID, isOwner, now, out), nil
}

type fastHomeOrderCandidate struct {
	orderID               string
	supplierID            string
	supplierName          string
	status                string
	orderSource           string
	confirmationStatus    string
	lineItemsRaw          []byte
	totalMinor            int64
	currency              string
	requestedDeliveryDate spanner.NullTime
	deliverBefore         spanner.NullTime
	autoConfirmAt         spanner.NullTime
	autoConfirmOptIn      bool
	timezone              string
	driverID              string
}

func (s *Service) buildFastHomeSpanner(ctx context.Context, retailerID, userID string, isOwner bool, now time.Time) (*FastHomeResponse, error) {
	out := &FastHomeResponse{
		ServerNow: now.Format(time.RFC3339Nano),
		Locale:    normalizeRetailerLocale(""),
	}

	// Roundtrip 1: active delivery + pending proposal orders using Idx_Orders_ByRetailerStatusCreated.
	ordStmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.SupplierId, COALESCE(s.Name, o.SupplierId), o.Status, o.OrderSource,
			       o.ConfirmationStatus, o.LineItemsJson, o.TotalMinor, o.Currency,
			       o.RequestedDeliveryDate, o.DeliverBefore, o.AutoConfirmAt,
			       COALESCE(o.AutoConfirmOptIn, FALSE), COALESCE(o.Timezone, ''), COALESCE(o.DriverId, '')
			FROM Orders o
			LEFT JOIN Suppliers s ON s.SupplierId = o.SupplierId
			WHERE o.RetailerId = @rid
			  AND o.Status IN (
				'DRAFT', 'PENDING', 'CONFIRMED', 'ACCEPTED', 'ALLOCATED',
				'PICKING', 'PACKING', 'READY_FOR_DISPATCH', 'DISPATCHED',
				'EN_ROUTE', 'OUT_FOR_DELIVERY', 'ARRIVING'
			  )
			ORDER BY o.UpdatedAt DESC
			LIMIT 40`,
		Params: map[string]any{"rid": retailerID},
	}

	var rows []fastHomeOrderCandidate
	ordIter := s.spannerClient.Single().Query(ctx, ordStmt)
	for {
		row, err := ordIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			ordIter.Stop()
			return nil, err
		}
		var item fastHomeOrderCandidate
		if err := row.Columns(
			&item.orderID, &item.supplierID, &item.supplierName, &item.status, &item.orderSource,
			&item.confirmationStatus, &item.lineItemsRaw, &item.totalMinor, &item.currency,
			&item.requestedDeliveryDate, &item.deliverBefore, &item.autoConfirmAt,
			&item.autoConfirmOptIn, &item.timezone, &item.driverID,
		); err != nil {
			ordIter.Stop()
			return nil, err
		}
		rows = append(rows, item)
	}
	ordIter.Stop()

	supplierScopeID := strings.TrimSpace(s.resolveSupplierScope(ctx))
	orderTZ := ""
	for _, item := range rows {
		if supplierScopeID == "" && strings.TrimSpace(item.supplierID) != "" {
			supplierScopeID = strings.TrimSpace(item.supplierID)
		}
		if orderTZ == "" && strings.TrimSpace(item.timezone) != "" {
			orderTZ = strings.TrimSpace(item.timezone)
		}

		confUpper := strings.ToUpper(strings.TrimSpace(item.confirmationStatus))
		statusUpper := strings.ToUpper(strings.TrimSpace(item.status))
		srcUpper := strings.ToUpper(strings.TrimSpace(item.orderSource))

		isAwaitingConfirm := confUpper == "PENDING" || confUpper == "AWAITING_CONFIRMATION" || confUpper == "PROPOSED" ||
			statusUpper == "DRAFT" ||
			((srcUpper == string(order.OrderSourceAIPreorder) || srcUpper == string(order.OrderSourceAutoOrder) || srcUpper == "PREORDER" || srcUpper == "AI_GENERATED") && confUpper != "CONFIRMED" && confUpper != "REJECTED")

		if out.PendingProposal == nil && isAwaitingConfirm {
			parsedItems := decodeFastHomeProposalItems(item.lineItemsRaw)
			prop := &FastHomePendingProposal{
				OrderID:            item.orderID,
				SupplierID:         item.supplierID,
				SupplierName:       item.supplierName,
				OrderSource:        item.orderSource,
				ConfirmationStatus: item.confirmationStatus,
				AutoConfirmOptIn:   item.autoConfirmOptIn,
				CanConfirm:         true,
				TotalMinor:         item.totalMinor,
				Currency:           firstNonEmptyFastHome(item.currency, "UZS"),
				Items:              parsedItems,
			}
			if item.autoConfirmAt.Valid {
				prop.AutoConfirmAt = item.autoConfirmAt.Time.UTC().Format(time.RFC3339Nano)
			}
			if item.requestedDeliveryDate.Valid {
				prop.RequestedDeliveryDate = item.requestedDeliveryDate.Time.UTC().Format("2006-01-02")
			}
			out.PendingProposal = prop
			continue
		}

		if out.TomorrowDelivery == nil && !isAwaitingConfirm {
			parsedItems := decodeFastHomeProposalItems(item.lineItemsRaw)
			summary := &FastHomeDeliverySummary{
				OrderID:      item.orderID,
				SupplierID:   item.supplierID,
				SupplierName: item.supplierName,
				Status:       item.status,
				TotalMinor:   item.totalMinor,
				Currency:     firstNonEmptyFastHome(item.currency, "UZS"),
				ItemCount:    len(parsedItems),
				DriverID:     item.driverID,
			}
			if item.deliverBefore.Valid {
				summary.ETAAt = item.deliverBefore.Time.UTC().Format(time.RFC3339Nano)
			} else if item.requestedDeliveryDate.Valid {
				summary.ETAAt = item.requestedDeliveryDate.Time.UTC().Format(time.RFC3339Nano)
			}
			if item.requestedDeliveryDate.Valid {
				summary.DeliveryDate = item.requestedDeliveryDate.Time.UTC().Format("2006-01-02")
			}
			out.TomorrowDelivery = summary
		}
	}

	// Roundtrip 2: single scalar summary across Retailer, Telegram link locale, SupplierServicePolicies, Credit, AR, Approvals, Promises.
	summaryStmt := spanner.Statement{
		SQL: `SELECT
			COALESCE((SELECT COALESCE(Timezone, '') FROM Retailers WHERE RetailerId = @rid LIMIT 1), '') AS RetailerTimezone,
			COALESCE((SELECT Locale FROM RetailerTelegramLinks WHERE RetailerId = @rid AND VerifiedAt IS NOT NULL ORDER BY VerifiedAt DESC, UpdatedAt DESC LIMIT 1), '') AS LinkLocale,
			COALESCE((SELECT COALESCE(NULLIF(NextDayCutoffTime, ''), NULLIF(SameDayCutoffTime, ''), '18:00') FROM SupplierServicePolicies WHERE SupplierId = @sid LIMIT 1), '18:00') AS CutoffHHMM,
			COALESCE((SELECT LeadTimeDays FROM SupplierServicePolicies WHERE SupplierId = @sid LIMIT 1), 1) AS LeadTimeDays,
			COALESCE((SELECT SUM(CASE WHEN CurrentBalanceMinor > 0 THEN CurrentBalanceMinor ELSE 0 END) FROM RetailerCreditProfiles WHERE RetailerId = @rid), 0) AS ProfileBalanceMinor,
			COALESCE((SELECT SUM(CASE WHEN AvailableCreditMinor > 0 THEN AvailableCreditMinor ELSE 0 END) FROM RetailerCreditProfiles WHERE RetailerId = @rid), 0) AS CreditHeadroomMinor,
			COALESCE((SELECT SUM(BalanceMinor) FROM ArInvoices WHERE RetailerId = @rid AND Status IN ('OPEN', 'PARTIAL', 'OVERDUE') AND BalanceMinor > 0), 0) AS InvoiceBalanceMinor,
			COALESCE((SELECT SUM(BalanceMinor) FROM ArInvoices WHERE RetailerId = @rid AND Status IN ('OPEN', 'PARTIAL', 'OVERDUE') AND BalanceMinor > 0 AND DueAt < @now), 0) AS OverdueMinor,
			(SELECT MIN(DueAt) FROM ArInvoices WHERE RetailerId = @rid AND Status IN ('OPEN', 'PARTIAL', 'OVERDUE') AND BalanceMinor > 0 AND DueAt < @now) AS EarliestOverdueDueAt,
			COALESCE((SELECT COUNT(*) FROM RetailerApprovalRequests WHERE RetailerId = @rid AND Status = 'PENDING_OWNER' AND ExpiresAt > @now), 0) AS PendingApprovalsCount,
			COALESCE((SELECT CONCAT(PromiseId, '|', SupplierId, '|', CAST(PromisedMinor AS STRING), '|', Status, '|', FORMAT_TIMESTAMP('%Y-%m-%dT%H:%M:%E9SZ', DueAt)) FROM ArPaymentPromises WHERE RetailerId = @rid AND Status IN ('ACTIVE', 'PENDING') ORDER BY DueAt ASC LIMIT 1), '') AS ActivePromisePacked`,
		Params: map[string]any{
			"rid": retailerID,
			"sid": supplierScopeID,
			"now": now,
		},
	}

	var retTZ, linkLocale, cutoffHHMM, packedPromise string
	var leadTimeDays, profileBalanceMinor, headroomMinor, invoiceBalanceMinor, overdueMinor, approvalsCount int64
	var earliestDue spanner.NullTime

	sumIter := s.spannerClient.Single().Query(ctx, summaryStmt)
	defer sumIter.Stop()
	if row, err := sumIter.Next(); err == nil {
		if colErr := row.Columns(
			&retTZ, &linkLocale, &cutoffHHMM, &leadTimeDays,
			&profileBalanceMinor, &headroomMinor, &invoiceBalanceMinor, &overdueMinor,
			&earliestDue, &approvalsCount, &packedPromise,
		); colErr != nil {
			return nil, colErr
		}
	} else if err != iterator.Done {
		return nil, err
	}

	tz := firstNonEmptyFastHome(retTZ, orderTZ, defaultSupplierTimezone)
	cutoffUTC, nextDate := computeSupplierCutoff(now, tz, cutoffHHMM, leadTimeDays)
	out.CutoffAt = cutoffUTC.Format(time.RFC3339Nano)
	out.NextDeliveryDate = nextDate

	balanceOwed := invoiceBalanceMinor
	if profileBalanceMinor > balanceOwed {
		balanceOwed = profileBalanceMinor
	}
	out.BalanceOwedMinor = balanceOwed
	out.OverdueMinor = overdueMinor
	out.CreditHeadroomMinor = headroomMinor
	if overdueMinor > 0 && earliestDue.Valid && now.After(earliestDue.Time) {
		days := int(now.Sub(earliestDue.Time).Hours() / 24)
		if days < 1 {
			days = 1
		}
		out.DaysOverdue = days
	}
	if isOwner {
		out.PendingApprovalsCount = approvalsCount
	}
	out.Locale = normalizeRetailerLocale(linkLocale)
	out.ActivePromise = unpackFastHomePromise(packedPromise)

	return out, nil
}

func (s *Service) buildFastHomeMemory(ctx context.Context, retailerID, userID string, isOwner bool, now time.Time, out *FastHomeResponse) *FastHomeResponse {
	tz := defaultSupplierTimezone
	if s.repo != nil {
		if ret, found, err := s.repo.GetRetailer(ctx, retailerID); err == nil && found {
			if strings.EqualFold(strings.TrimSpace(ret.CountryCode), "CA") {
				tz = "America/Toronto"
			}
		}
		if tracking, err := s.repo.ListTrackingOrders(ctx, retailerID, 10, 0); err == nil {
			for _, ord := range tracking {
				statusUpper := strings.ToUpper(strings.TrimSpace(ord.Status))
				if statusUpper == "CANCELLED" || statusUpper == "REJECTED" || statusUpper == "COMPLETED" || statusUpper == "DELIVERED" {
					continue
				}
				deliveryDate := ""
				if ord.DeliveryExpectation != nil && ord.DeliveryExpectation.TargetDate != nil {
					deliveryDate = *ord.DeliveryExpectation.TargetDate
				}
				out.TomorrowDelivery = &FastHomeDeliverySummary{
					OrderID:      ord.OrderID,
					SupplierID:   ord.SupplierID,
					Status:       ord.Status,
					DeliveryDate: deliveryDate,
					TotalMinor:   ord.TotalMinor,
					Currency:     firstNonEmptyFastHome(ord.Currency, "UZS"),
					ItemCount:    len(ord.Items),
					DriverID:     ord.DriverID,
				}
				break
			}
		}
	}

	if s.orders != nil {
		if preds, err := s.orders.ListRetailerAIPredictions(ctx, retailerID, 10); err == nil {
			for _, p := range preds {
				confUpper := strings.ToUpper(strings.TrimSpace(string(p.ConfirmationStatus)))
				if confUpper == "CONFIRMED" || confUpper == "REJECTED" {
					continue
				}
				items := make([]FastHomeProposalItem, 0, len(p.Items))
				for _, li := range p.Items {
					sku := strings.TrimSpace(li.SKU)
					items = append(items, FastHomeProposalItem{
						SKU:            sku,
						ProductID:      sku,
						Name:           li.Name,
						Quantity:       li.Quantity,
						UnitPriceMinor: li.UnitPrice,
						LineTotalMinor: li.UnitPrice * li.Quantity,
					})
				}
				out.PendingProposal = &FastHomePendingProposal{
					OrderID:               p.OrderID,
					SupplierID:            p.SupplierID,
					OrderSource:           string(p.Source),
					ConfirmationStatus:    string(p.ConfirmationStatus),
					AutoConfirmOptIn:      p.AutoConfirmOptIn,
					AutoConfirmAt:         p.AutoConfirmAt,
					CanConfirm:            true,
					TotalMinor:            p.TotalMinor,
					Currency:              firstNonEmptyFastHome(p.Currency, "UZS"),
					RequestedDeliveryDate: p.RequestedDeliveryDate,
					Items:                 items,
				}
				break
			}
		}
	}

	// Best-effort locale from memory Telegram link store.
	s.mu.RLock()
	for _, rec := range s.telegramLinksMem {
		if rec.RetailerID != retailerID {
			continue
		}
		if rec.UserID == userID && rec.Locale != "" {
			out.Locale = normalizeRetailerLocale(rec.Locale)
			break
		}
		if rec.VerifiedAt != nil && rec.Locale != "" {
			out.Locale = normalizeRetailerLocale(rec.Locale)
		}
	}
	s.mu.RUnlock()

	if !isOwner {
		out.PendingApprovalsCount = 0
	}
	cutoffUTC, nextDate := computeSupplierCutoff(now, tz, defaultCutoffHHMM, defaultLeadTimeDays)
	out.CutoffAt = cutoffUTC.Format(time.RFC3339Nano)
	out.NextDeliveryDate = nextDate
	return out
}

func computeSupplierCutoff(nowUTC time.Time, tzName, cutoffHHMM string, leadTimeDays int64) (time.Time, string) {
	if leadTimeDays < 1 {
		leadTimeDays = defaultLeadTimeDays
	}
	loc, err := time.LoadLocation(strings.TrimSpace(tzName))
	if err != nil {
		loc, err = time.LoadLocation(defaultSupplierTimezone)
		if err != nil {
			loc = time.FixedZone(defaultSupplierTimezone, 5*3600)
		}
	}
	hour, minute := 18, 0
	parts := strings.Split(strings.TrimSpace(cutoffHHMM), ":")
	if len(parts) >= 2 {
		if h, hErr := strconv.Atoi(parts[0]); hErr == nil && h >= 0 && h < 24 {
			if m, mErr := strconv.Atoi(parts[1]); mErr == nil && m >= 0 && m < 60 {
				hour, minute = h, m
			}
		}
	}
	nowLocal := nowUTC.In(loc)
	cutoffLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), hour, minute, 0, 0, loc)
	deliveryOffset := int(leadTimeDays)
	if !nowLocal.Before(cutoffLocal) {
		cutoffLocal = cutoffLocal.AddDate(0, 0, 1)
		deliveryOffset++
	}
	nextDeliveryDate := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc).
		AddDate(0, 0, deliveryOffset).
		Format("2006-01-02")
	return cutoffLocal.UTC(), nextDeliveryDate
}

func decodeFastHomeProposalItems(raw []byte) []FastHomeProposalItem {
	if len(raw) == 0 {
		return nil
	}
	var lineItems []struct {
		SKU            string `json:"sku"`
		ProductID      string `json:"product_id"`
		Name           string `json:"name"`
		ProductName    string `json:"product_name"`
		Quantity       int64  `json:"quantity"`
		Qty            int64  `json:"qty"`
		UnitPriceMinor int64  `json:"unit_price_minor"`
		UnitPrice      int64  `json:"unit_price"`
		LineTotalMinor int64  `json:"line_total_minor"`
		LineTotal      int64  `json:"line_total"`
	}
	if err := json.Unmarshal(raw, &lineItems); err != nil {
		return nil
	}
	out := make([]FastHomeProposalItem, 0, len(lineItems))
	for _, li := range lineItems {
		sku := firstNonEmptyFastHome(li.SKU, li.ProductID)
		qty := li.Quantity
		if qty <= 0 {
			qty = li.Qty
		}
		unitMinor := li.UnitPriceMinor
		if unitMinor == 0 {
			unitMinor = li.UnitPrice
		}
		lineMinor := li.LineTotalMinor
		if lineMinor == 0 {
			lineMinor = li.LineTotal
		}
		out = append(out, FastHomeProposalItem{
			SKU:            sku,
			ProductID:      firstNonEmptyFastHome(li.ProductID, sku),
			Name:           firstNonEmptyFastHome(li.Name, li.ProductName),
			Quantity:       qty,
			UnitPriceMinor: unitMinor,
			LineTotalMinor: lineMinor,
		})
	}
	return out
}

func unpackFastHomePromise(packed string) *FastHomeActivePromise {
	packed = strings.TrimSpace(packed)
	if packed == "" {
		return nil
	}
	parts := strings.SplitN(packed, "|", 5)
	if len(parts) != 5 {
		return nil
	}
	promisedMinor, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil
	}
	return &FastHomeActivePromise{
		PromiseID:     parts[0],
		SupplierID:    parts[1],
		PromisedMinor: promisedMinor,
		Status:        parts[3],
		DueAt:         parts[4],
	}
}

func normalizeRetailerLocale(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "uz", "ru", "en":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "uz"
	}
}

func firstNonEmptyFastHome(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

// Keep unused order import satisfied if OrderSource constants are checked cleanly.
var _ = order.OrderSourceAIPreorder
