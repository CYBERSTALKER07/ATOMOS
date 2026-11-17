package order

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	sellerMandatesPackID = "SELLER_MANDATES"

	PaymentMethodCredit = "CREDIT"
	PaymentMethodCash   = "CASH"
	PaymentMethodCard   = "CARD"

	TrippedRuleCreditMandateDisabled = "CREDIT_MANDATE_DISABLED"
	TrippedRuleCashMandateDisabled   = "CASH_MANDATE_DISABLED"
	TrippedRuleMaxSingleOrder        = "MAX_SINGLE_ORDER_EXCEEDED"
	TrippedRuleDailyCreditCap        = "DAILY_CREDIT_CAP_EXCEEDED"
	TrippedRuleDailyCashCap          = "DAILY_CASH_CAP_EXCEEDED"

	LockStageUsage         = "RetailerSellerDailyUsage"
	LockStageCreditProfile = "CreditProfile"
	LockStageOrders        = "Orders"
)

// SellerMandate governs ordering and receiving permissions/caps for non-owner staff.
// Zero cap fields indicate no numeric cap configured (still gated by boolean flags).
type SellerMandate struct {
	CanReceiveGoods     bool   `json:"can_receive_goods"`
	CanPayCash          bool   `json:"can_pay_cash"`
	CanUseCredit        bool   `json:"can_use_credit"`
	DailyCreditCapMinor int64  `json:"daily_credit_cap_minor"`
	DailyCashCapMinor   int64  `json:"daily_cash_cap_minor"`
	MaxSingleOrderMinor int64  `json:"max_single_order_minor"`
	UpdatedAt           string `json:"updated_at,omitempty"`
	UpdatedBy           string `json:"updated_by,omitempty"`
}

// MandateLookupResult is returned by MandateReader implementations.
type MandateLookupResult struct {
	Mandate SellerMandate
	Role    string
	IsOwner bool
	Found   bool
}

// MandateReader resolves an effective SellerMandate for a retailer staff user.
type MandateReader interface {
	ResolveSellerMandate(ctx context.Context, retailerID, userID, fallbackRole string) (MandateLookupResult, error)
}

// SellerDailyUsage mirrors a RetailerSellerDailyUsage row.
type SellerDailyUsage struct {
	RetailerID       string
	UserID           string
	UsageDate        civil.Date
	SpentCreditMinor int64
	SpentCashMinor   int64
	OrderCount       int64
	UpdatedAt        time.Time
}

type sellerUsageKey struct {
	RetailerID string
	UserID     string
	UsageDate  string
}

type orderUsageDelta struct {
	RetailerID       string
	UserID           string
	UsageDate        civil.Date
	PaymentMethod    string
	SpentCreditMinor int64
	SpentCashMinor   int64
	OrderCount       int64
	RolledBack       bool
}

// OwnerApprovalRequiredError indicates a checkout/order tripped seller mandate rules
// and was parked in RetailerApprovalRequests for owner review.
type OwnerApprovalRequiredError struct {
	RequestID              string   `json:"request_id"`
	ClientOrderID          string   `json:"client_order_id"`
	TrippedRule            string   `json:"tripped_rule"`
	AmountMinor            int64    `json:"amount_minor"`
	ThresholdMinor         int64    `json:"threshold_minor,omitempty"`
	RemainingCapMinor      int64    `json:"remaining_cap_minor,omitempty"`
	FallbackPaymentMethods []string `json:"fallback_payment_methods"`
}

func (e *OwnerApprovalRequiredError) Error() string {
	if e == nil {
		return "owner_approval_required"
	}
	return fmt.Sprintf("owner_approval_required: rule=%s request_id=%s", e.TrippedRule, e.RequestID)
}

func writeOwnerApprovalConflict(w http.ResponseWriter, e *OwnerApprovalRequiredError) {
	if e == nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":                     "owner_approval_required",
			"error":                    "owner_approval_required",
			"status":                   ApprovalStatusPendingOwner,
			"fallback_payment_methods": []string{},
		})
		return
	}
	fallbacks := e.FallbackPaymentMethods
	if fallbacks == nil {
		fallbacks = []string{}
	}
	writeJSON(w, http.StatusConflict, map[string]any{
		"code":                     "owner_approval_required",
		"error":                    "owner_approval_required",
		"message":                  "Order exceeds seller mandate and requires owner approval.",
		"request_id":               e.RequestID,
		"client_order_id":          e.ClientOrderID,
		"status":                   ApprovalStatusPendingOwner,
		"tripped_rule":             e.TrippedRule,
		"amount_minor":             e.AmountMinor,
		"threshold_minor":          e.ThresholdMinor,
		"remaining_cap_minor":      e.RemainingCapMinor,
		"fallback_payment_methods": fallbacks,
	})
}

type mandateConfigDoc struct {
	Mandates map[string]SellerMandate `json:"mandates"`
}

// DefaultSellerMandateForRole returns role-scoped defaults when no explicit mandate was saved.
func DefaultSellerMandateForRole(role string, isOwner bool) SellerMandate {
	normalized := strings.ToUpper(strings.TrimSpace(role))
	if isOwner || normalized == "OWNER" {
		return SellerMandate{
			CanReceiveGoods: true,
			CanPayCash:      true,
			CanUseCredit:    true,
		}
	}
	switch normalized {
	case "ADMIN", "MANAGER":
		return SellerMandate{
			CanReceiveGoods: true,
			CanPayCash:      true,
			CanUseCredit:    true,
		}
	case "BUYER":
		return SellerMandate{
			CanReceiveGoods: false,
			CanPayCash:      true,
			CanUseCredit:    true,
		}
	case "RECEIVER":
		return SellerMandate{
			CanReceiveGoods: true,
			CanPayCash:      false,
			CanUseCredit:    false,
		}
	case "CASHIER":
		return SellerMandate{
			CanReceiveGoods: false,
			CanPayCash:      true,
			CanUseCredit:    false,
		}
	case "STOCK_CLERK":
		return SellerMandate{
			CanReceiveGoods: true,
			CanPayCash:      false,
			CanUseCredit:    false,
		}
	default:
		return SellerMandate{}
	}
}

// NormalizeCheckoutPaymentMethod maps payment_method / payment_gateway into CREDIT | CASH | CARD.
func NormalizeCheckoutPaymentMethod(paymentMethod, paymentGateway string) string {
	raw := strings.ToUpper(strings.TrimSpace(paymentMethod))
	if raw == "" {
		raw = strings.ToUpper(strings.TrimSpace(paymentGateway))
	}
	switch raw {
	case "CASH", "COD":
		return PaymentMethodCash
	case "CARD", "PAYME", "CLICK", "UZCARD", "HUMO", "STRIPE", "VISA", "MASTERCARD":
		return PaymentMethodCard
	case "CREDIT", "PAY_LATER", "AR", "B2B_CREDIT", "":
		return PaymentMethodCredit
	default:
		return PaymentMethodCard
	}
}

func sellerUsageDate(now time.Time) civil.Date {
	return civil.DateOf(now.UTC())
}

type ctxKeyMandate int

const (
	ctxKeyClientOrderID ctxKeyMandate = iota
	ctxKeyApprovedOverride
	ctxKeyMandatePrechecked
	ctxKeyOrderLockPlan
	ctxKeyLockTracer
)

// ApprovedReplayContext marks internal approval replay execution.
type ApprovedReplayContext struct {
	RequestID       string
	ClientOrderID   string
	RequestedByUser string
	ApprovedByUser  string
	UsageDate       civil.Date
}

// WithClientOrderID attaches an explicit idempotency/client order identifier to ctx.
func WithClientOrderID(ctx context.Context, clientOrderID string) context.Context {
	clientOrderID = strings.TrimSpace(clientOrderID)
	if clientOrderID == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyClientOrderID, clientOrderID)
}

func clientOrderIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyClientOrderID).(string)
	return strings.TrimSpace(v)
}

// WithApprovedApprovalContext marks execution as an owner-approved replay.
func WithApprovedApprovalContext(ctx context.Context, info ApprovedReplayContext) context.Context {
	return context.WithValue(ctx, ctxKeyApprovedOverride, info)
}

func approvedApprovalFromContext(ctx context.Context) (ApprovedReplayContext, bool) {
	if ctx == nil {
		return ApprovedReplayContext{}, false
	}
	v, ok := ctx.Value(ctxKeyApprovedOverride).(ApprovedReplayContext)
	return v, ok
}

func withMandatePrechecked(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKeyMandatePrechecked, true)
}

func isMandatePrechecked(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(ctxKeyMandatePrechecked).(bool)
	return v
}

// WithLockOrderTracer attaches an optional callback invoked in exact RW txn lock order.
func WithLockOrderTracer(ctx context.Context, fn func(stage string)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyLockTracer, fn)
}

func recordLockStage(ctx context.Context, serviceTracer func(string), stage string) {
	if ctx != nil {
		if fn, ok := ctx.Value(ctxKeyLockTracer).(func(string)); ok && fn != nil {
			fn(stage)
		}
	}
	if serviceTracer != nil {
		serviceTracer(stage)
	}
}

type preOrderTxnPlanFunc func(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) error

func withPreOrderTxnPlan(ctx context.Context, fn preOrderTxnPlanFunc) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyOrderLockPlan, fn)
}

func runPreOrderLocksInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) error {
	if ctx == nil || txn == nil {
		return nil
	}
	if fn, ok := ctx.Value(ctxKeyOrderLockPlan).(preOrderTxnPlanFunc); ok && fn != nil {
		return fn(ctx, txn, o)
	}
	return nil
}

// SetMandateReader wires an optional SellerMandate lookup adapter (e.g. *retailer.Service).
func (s *Service) SetMandateReader(r MandateReader) {
	if s == nil {
		return
	}
	s.mandateReader = r
}

func resolveOrderActorAndRole(ctx context.Context, retailerID string) (userID string, role string, hasClaims bool) {
	if approved, ok := approvedApprovalFromContext(ctx); ok && strings.TrimSpace(approved.RequestedByUser) != "" {
		return strings.TrimSpace(approved.RequestedByUser), "OWNER", true
	}
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return strings.TrimSpace(retailerID), "OWNER", false
	}
	userID = strings.TrimSpace(auth.ResolveRetailerUserID(claims))
	if userID == "" {
		userID = strings.TrimSpace(retailerID)
	}
	role = auth.EffectiveRetailerRole(claims)
	return userID, role, true
}

func (s *Service) resolveEffectiveMandate(ctx context.Context, retailerID, userID, fallbackRole string) (MandateLookupResult, error) {
	retailerID = strings.TrimSpace(retailerID)
	userID = strings.TrimSpace(userID)
	fallbackRole = strings.ToUpper(strings.TrimSpace(fallbackRole))
	if fallbackRole == "" {
		fallbackRole = "OWNER"
	}

	if s.mandateReader != nil {
		return s.mandateReader.ResolveSellerMandate(ctx, retailerID, userID, fallbackRole)
	}

	role := fallbackRole
	isOwner := role == "OWNER"
	if s.spannerClient != nil {
		if userID != "" && userID != retailerID {
			row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerUsers", spanner.Key{userID}, []string{"RetailerId", "RetailerRole", "IsOwner"})
			if err == nil {
				var rowRetID, rowRole string
				var rowOwner bool
				if colErr := row.Columns(&rowRetID, &rowRole, &rowOwner); colErr == nil && rowRetID == retailerID {
					if r := strings.ToUpper(strings.TrimSpace(rowRole)); r != "" {
						role = r
					}
					isOwner = rowOwner || role == "OWNER"
				}
			} else if spanner.ErrCode(err) != codes.NotFound {
				return MandateLookupResult{}, fmt.Errorf("read retailer user for mandate: %w", err)
			}
		}
		packRow, err := s.spannerClient.Single().ReadRow(ctx, "RetailerCapabilityPacks", spanner.Key{retailerID, sellerMandatesPackID}, []string{"ConfigJson"})
		if err == nil {
			var raw spanner.NullString
			if colErr := packRow.Columns(&raw); colErr == nil && raw.Valid && strings.TrimSpace(raw.StringVal) != "" {
				var doc mandateConfigDoc
				if jerr := json.Unmarshal([]byte(raw.StringVal), &doc); jerr == nil && doc.Mandates != nil {
					if m, ok := doc.Mandates[userID]; ok {
						return MandateLookupResult{
							Mandate: m,
							Role:    role,
							IsOwner: isOwner,
							Found:   true,
						}, nil
					}
				}
			}
		} else if spanner.ErrCode(err) != codes.NotFound {
			return MandateLookupResult{}, fmt.Errorf("read seller mandate pack: %w", err)
		}
	}

	return MandateLookupResult{
		Mandate: DefaultSellerMandateForRole(role, isOwner),
		Role:    role,
		IsOwner: isOwner,
		Found:   false,
	}, nil
}

type mandateEvaluation struct {
	Allowed           bool
	TrippedRule       string
	ThresholdMinor    int64
	RemainingCapMinor int64
	Fallbacks         []string
}

func evaluateSellerMandate(m SellerMandate, isOwner bool, role, paymentMethod string, amountMinor int64, usage SellerDailyUsage) mandateEvaluation {
	if isOwner || strings.EqualFold(strings.TrimSpace(role), "OWNER") {
		return mandateEvaluation{Allowed: true}
	}
	paymentMethod = NormalizeCheckoutPaymentMethod(paymentMethod, "")
	fallbacks := make([]string, 0, 2)

	canCashFallback := func() bool {
		if !m.CanPayCash {
			return false
		}
		if m.MaxSingleOrderMinor > 0 && amountMinor > m.MaxSingleOrderMinor {
			return false
		}
		if m.DailyCashCapMinor > 0 && usage.SpentCashMinor+amountMinor > m.DailyCashCapMinor {
			return false
		}
		return true
	}
	canCardFallback := func() bool {
		return m.MaxSingleOrderMinor == 0 || amountMinor <= m.MaxSingleOrderMinor
	}
	buildFallbacks := func() []string {
		out := make([]string, 0, 2)
		if paymentMethod != PaymentMethodCash && canCashFallback() {
			out = append(out, PaymentMethodCash)
		}
		if paymentMethod != PaymentMethodCard && canCardFallback() {
			out = append(out, PaymentMethodCard)
		}
		return out
	}

	if m.MaxSingleOrderMinor > 0 && amountMinor > m.MaxSingleOrderMinor {
		return mandateEvaluation{
			Allowed:           false,
			TrippedRule:       TrippedRuleMaxSingleOrder,
			ThresholdMinor:    m.MaxSingleOrderMinor,
			RemainingCapMinor: m.MaxSingleOrderMinor,
			Fallbacks:         fallbacks,
		}
	}

	switch paymentMethod {
	case PaymentMethodCredit:
		if !m.CanUseCredit {
			return mandateEvaluation{
				Allowed:           false,
				TrippedRule:       TrippedRuleCreditMandateDisabled,
				ThresholdMinor:    0,
				RemainingCapMinor: 0,
				Fallbacks:         buildFallbacks(),
			}
		}
		if m.DailyCreditCapMinor > 0 && usage.SpentCreditMinor+amountMinor > m.DailyCreditCapMinor {
			rem := m.DailyCreditCapMinor - usage.SpentCreditMinor
			if rem < 0 {
				rem = 0
			}
			return mandateEvaluation{
				Allowed:           false,
				TrippedRule:       TrippedRuleDailyCreditCap,
				ThresholdMinor:    m.DailyCreditCapMinor,
				RemainingCapMinor: rem,
				Fallbacks:         buildFallbacks(),
			}
		}
	case PaymentMethodCash:
		if !m.CanPayCash {
			return mandateEvaluation{
				Allowed:           false,
				TrippedRule:       TrippedRuleCashMandateDisabled,
				ThresholdMinor:    0,
				RemainingCapMinor: 0,
				Fallbacks:         buildFallbacks(),
			}
		}
		if m.DailyCashCapMinor > 0 && usage.SpentCashMinor+amountMinor > m.DailyCashCapMinor {
			rem := m.DailyCashCapMinor - usage.SpentCashMinor
			if rem < 0 {
				rem = 0
			}
			return mandateEvaluation{
				Allowed:           false,
				TrippedRule:       TrippedRuleDailyCashCap,
				ThresholdMinor:    m.DailyCashCapMinor,
				RemainingCapMinor: rem,
				Fallbacks:         buildFallbacks(),
			}
		}
	}

	return mandateEvaluation{Allowed: true}
}

func (s *Service) readSellerDailyUsage(ctx context.Context, retailerID, userID string, usageDate civil.Date) (SellerDailyUsage, error) {
	if s.spannerClient != nil {
		row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerSellerDailyUsage", spanner.Key{retailerID, userID, usageDate},
			[]string{"SpentCreditMinor", "SpentCashMinor", "OrderCount", "UpdatedAt"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return SellerDailyUsage{RetailerID: retailerID, UserID: userID, UsageDate: usageDate}, nil
			}
			return SellerDailyUsage{}, fmt.Errorf("read seller daily usage: %w", err)
		}
		var out SellerDailyUsage
		out.RetailerID = retailerID
		out.UserID = userID
		out.UsageDate = usageDate
		if err := row.Columns(&out.SpentCreditMinor, &out.SpentCashMinor, &out.OrderCount, &out.UpdatedAt); err != nil {
			return SellerDailyUsage{}, err
		}
		return out, nil
	}

	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	if s.sellerDailyUsage == nil {
		return SellerDailyUsage{RetailerID: retailerID, UserID: userID, UsageDate: usageDate}, nil
	}
	u, ok := s.sellerDailyUsage[sellerUsageKey{RetailerID: retailerID, UserID: userID, UsageDate: usageDate.String()}]
	if !ok {
		return SellerDailyUsage{RetailerID: retailerID, UserID: userID, UsageDate: usageDate}, nil
	}
	return u, nil
}

// GetSellerDailyUsage returns today's (or specified date's) RetailerSellerDailyUsage snapshot.
func (s *Service) GetSellerDailyUsage(ctx context.Context, retailerID, userID string, usageDate civil.Date) (SellerDailyUsage, error) {
	if usageDate == (civil.Date{}) {
		usageDate = sellerUsageDate(s.now())
	}
	return s.readSellerDailyUsage(ctx, strings.TrimSpace(retailerID), strings.TrimSpace(userID), usageDate)
}

func usageDeltasForPayment(paymentMethod string, amountMinor int64) (creditDelta, cashDelta int64) {
	if amountMinor <= 0 {
		return 0, 0
	}
	switch NormalizeCheckoutPaymentMethod(paymentMethod, "") {
	case PaymentMethodCredit:
		return amountMinor, 0
	case PaymentMethodCash:
		return 0, amountMinor
	default:
		return 0, 0
	}
}

func (s *Service) recordOrderDailyUsageMemory(orderID, retailerID, userID string, usageDate civil.Date, paymentMethod string, amountMinor int64) {
	orderID = strings.TrimSpace(orderID)
	retailerID = strings.TrimSpace(retailerID)
	userID = strings.TrimSpace(userID)
	if orderID == "" || retailerID == "" || userID == "" {
		return
	}
	creditDelta, cashDelta := usageDeltasForPayment(paymentMethod, amountMinor)
	k := sellerUsageKey{RetailerID: retailerID, UserID: userID, UsageDate: usageDate.String()}

	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	if s.sellerDailyUsage == nil {
		s.sellerDailyUsage = make(map[sellerUsageKey]SellerDailyUsage)
	}
	if s.recordedOrderUsage == nil {
		s.recordedOrderUsage = make(map[string]orderUsageDelta)
	}
	if _, already := s.recordedOrderUsage[orderID]; already {
		return
	}
	cur := s.sellerDailyUsage[k]
	cur.RetailerID = retailerID
	cur.UserID = userID
	cur.UsageDate = usageDate
	cur.SpentCreditMinor += creditDelta
	cur.SpentCashMinor += cashDelta
	cur.OrderCount++
	cur.UpdatedAt = s.now()
	s.sellerDailyUsage[k] = cur

	s.recordedOrderUsage[orderID] = orderUsageDelta{
		RetailerID:       retailerID,
		UserID:           userID,
		UsageDate:        usageDate,
		PaymentMethod:    paymentMethod,
		SpentCreditMinor: creditDelta,
		SpentCashMinor:   cashDelta,
		OrderCount:       1,
	}
}

func (s *Service) rollbackOrderDailyUsageMemory(orderID string) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return
	}
	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	if s.recordedOrderUsage == nil {
		return
	}
	rec, ok := s.recordedOrderUsage[orderID]
	if !ok || rec.RolledBack {
		return
	}
	k := sellerUsageKey{RetailerID: rec.RetailerID, UserID: rec.UserID, UsageDate: rec.UsageDate.String()}
	if s.sellerDailyUsage != nil {
		cur := s.sellerDailyUsage[k]
		cur.SpentCreditMinor -= rec.SpentCreditMinor
		if cur.SpentCreditMinor < 0 {
			cur.SpentCreditMinor = 0
		}
		cur.SpentCashMinor -= rec.SpentCashMinor
		if cur.SpentCashMinor < 0 {
			cur.SpentCashMinor = 0
		}
		cur.OrderCount -= rec.OrderCount
		if cur.OrderCount < 0 {
			cur.OrderCount = 0
		}
		cur.UpdatedAt = s.now()
		s.sellerDailyUsage[k] = cur
	}
	rec.RolledBack = true
	s.recordedOrderUsage[orderID] = rec
}

func resolveEffectiveClientOrderID(ctx context.Context, explicitClientOrderID string) string {
	if id := strings.TrimSpace(explicitClientOrderID); id != "" {
		return truncateClientOrderID(id)
	}
	if id := clientOrderIDFromContext(ctx); id != "" {
		return truncateClientOrderID(id)
	}
	return uuid.NewString()
}

func truncateClientOrderID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 128 {
		return id
	}
	return id[:128]
}

func legClientOrderID(parentClientOrderID, supplierID string) string {
	parentClientOrderID = strings.TrimSpace(parentClientOrderID)
	supplierID = strings.TrimSpace(supplierID)
	if parentClientOrderID == "" {
		return ""
	}
	if supplierID == "" {
		return truncateClientOrderID(parentClientOrderID)
	}
	joined := parentClientOrderID + ":" + supplierID
	if len(joined) <= 128 {
		return joined
	}
	return truncateClientOrderID(uuid.NewSHA1(uuid.NameSpaceOID, []byte(joined)).String())
}

func (s *Service) checkSellerMandateGate(
	ctx context.Context,
	retailerID, userID, role, supplierID, clientOrderID, paymentMethod string,
	amountMinor int64,
	envelope ApprovalPayloadEnvelope,
) error {
	if _, approved := approvedApprovalFromContext(ctx); approved {
		return nil
	}
	lookup, err := s.resolveEffectiveMandate(ctx, retailerID, userID, role)
	if err != nil {
		return fmt.Errorf("resolve seller mandate: %w", err)
	}
	if lookup.IsOwner || strings.EqualFold(strings.TrimSpace(lookup.Role), "OWNER") {
		return nil
	}
	usageDate := sellerUsageDate(s.now())
	usage, err := s.readSellerDailyUsage(ctx, retailerID, userID, usageDate)
	if err != nil {
		return err
	}
	eval := evaluateSellerMandate(lookup.Mandate, lookup.IsOwner, lookup.Role, paymentMethod, amountMinor, usage)
	if eval.Allowed {
		return nil
	}

	if clientOrderID == "" {
		clientOrderID = uuid.NewString()
	}
	envelope.ClientOrderID = clientOrderID
	envelope.RetailerID = retailerID
	envelope.RequestedByUser = userID
	envelope.PaymentMethod = NormalizeCheckoutPaymentMethod(paymentMethod, "")
	envelope.UsageDate = usageDate.String()
	if strings.TrimSpace(envelope.SupplierID) == "" {
		envelope.SupplierID = supplierID
	}
	envelope.TotalMinor = amountMinor

	reqRec, err := s.upsertApprovalRequest(ctx, approvalUpsertParams{
		RetailerID:      retailerID,
		SupplierID:      supplierID,
		RequestedByUser: userID,
		ClientOrderID:   clientOrderID,
		AmountMinor:     amountMinor,
		TrippedRule:     eval.TrippedRule,
		Envelope:        envelope,
	})
	if err != nil {
		return fmt.Errorf("upsert approval request: %w", err)
	}
	return &OwnerApprovalRequiredError{
		RequestID:              reqRec.RequestID,
		ClientOrderID:          reqRec.ClientOrderID,
		TrippedRule:            eval.TrippedRule,
		AmountMinor:            amountMinor,
		ThresholdMinor:         eval.ThresholdMinor,
		RemainingCapMinor:      eval.RemainingCapMinor,
		FallbackPaymentMethods: eval.Fallbacks,
	}
}

type lockUsageTxnResult struct {
	UsageMutation   *spanner.Mutation
	BaseCreditMinor int64
	BaseCashMinor   int64
	BaseOrderCount  int64
	CreditDelta     int64
	CashDelta       int64
	UsageDate       civil.Date
}

func newMandateUUID() string {
	return uuid.NewString()
}

func buildSellerDailyUsageMutation(
	retailerID, userID string,
	usageDate civil.Date,
	paymentMethod string,
	amountMinor int64,
	base lockUsageTxnResult,
) (*spanner.Mutation, int64, int64) {
	creditDelta, cashDelta := usageDeltasForPayment(paymentMethod, amountMinor)
	mut := spanner.InsertOrUpdateMap("RetailerSellerDailyUsage", map[string]any{
		"RetailerId":       retailerID,
		"UserId":           userID,
		"UsageDate":        usageDate,
		"SpentCreditMinor": base.BaseCreditMinor + creditDelta,
		"SpentCashMinor":   base.BaseCashMinor + cashDelta,
		"OrderCount":       base.BaseOrderCount + 1,
		"UpdatedAt":        spanner.CommitTimestamp,
	})
	return mut, creditDelta, cashDelta
}

// acquireCreateLocksInTxn enforces mandatory RW transaction lock order:
// RetailerSellerDailyUsage -> CreditProfile -> Orders.
func (s *Service) acquireCreateLocksInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	retailerID, userID, supplierID string,
	usageDate civil.Date,
	paymentMethod string,
	amountMinor int64,
	mandate MandateLookupResult,
	enforceCap bool,
) (lockUsageTxnResult, error) {
	creditDelta, cashDelta := usageDeltasForPayment(paymentMethod, amountMinor)
	var curCredit, curCash, curCount int64

	// 1. Lock RetailerSellerDailyUsage first.
	recordLockStage(ctx, s.lockOrderTracer, LockStageUsage)
	usageRow, err := txn.ReadRow(ctx, "RetailerSellerDailyUsage", spanner.Key{retailerID, userID, usageDate},
		[]string{"SpentCreditMinor", "SpentCashMinor", "OrderCount"})
	if err == nil {
		if colErr := usageRow.Columns(&curCredit, &curCash, &curCount); colErr != nil {
			return lockUsageTxnResult{}, colErr
		}
	} else if spanner.ErrCode(err) != codes.NotFound {
		return lockUsageTxnResult{}, fmt.Errorf("lock RetailerSellerDailyUsage: %w", err)
	}

	if enforceCap {
		curUsage := SellerDailyUsage{
			RetailerID:       retailerID,
			UserID:           userID,
			UsageDate:        usageDate,
			SpentCreditMinor: curCredit,
			SpentCashMinor:   curCash,
			OrderCount:       curCount,
		}
		eval := evaluateSellerMandate(mandate.Mandate, mandate.IsOwner, mandate.Role, paymentMethod, amountMinor, curUsage)
		if !eval.Allowed {
			return lockUsageTxnResult{}, &OwnerApprovalRequiredError{
				TrippedRule:            eval.TrippedRule,
				AmountMinor:            amountMinor,
				ThresholdMinor:         eval.ThresholdMinor,
				RemainingCapMinor:      eval.RemainingCapMinor,
				FallbackPaymentMethods: eval.Fallbacks,
			}
		}
	}

	usageMut := spanner.InsertOrUpdateMap("RetailerSellerDailyUsage", map[string]any{
		"RetailerId":       retailerID,
		"UserId":           userID,
		"UsageDate":        usageDate,
		"SpentCreditMinor": curCredit + creditDelta,
		"SpentCashMinor":   curCash + cashDelta,
		"OrderCount":       curCount + 1,
		"UpdatedAt":        spanner.CommitTimestamp,
	})

	// 2. Lock CreditProfile second (before Orders or stock rows).
	if NormalizeCheckoutPaymentMethod(paymentMethod, "") == PaymentMethodCredit && s.credit != nil && amountMinor > 0 && creditReserveAtCreateEnabled() {
		recordLockStage(ctx, s.lockOrderTracer, LockStageCreditProfile)
		cpRow, cpErr := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
			[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Status"})
		if cpErr != nil {
			if spanner.ErrCode(cpErr) != codes.NotFound {
				return lockUsageTxnResult{}, fmt.Errorf("lock CreditProfile: %w", cpErr)
			}
		} else {
			var limit, balance, reserved int64
			var status string
			if colErr := cpRow.Columns(&limit, &balance, &reserved, &status); colErr != nil {
				return lockUsageTxnResult{}, colErr
			}
			if status == string(credit.StatusActive) && limit-(balance+reserved) < amountMinor {
				return lockUsageTxnResult{}, ErrCreditLimitBreached
			}
		}
	}

	return lockUsageTxnResult{
		UsageMutation:   usageMut,
		BaseCreditMinor: curCredit,
		BaseCashMinor:   curCash,
		BaseOrderCount:  curCount,
		CreditDelta:     creditDelta,
		CashDelta:       cashDelta,
		UsageDate:       usageDate,
	}, nil
}

func (s *Service) findExistingOrderByIdempotencyKey(ctx context.Context, retailerID, supplierID, idempotencyKey string) (Order, bool, error) {
	retailerID = strings.TrimSpace(retailerID)
	supplierID = strings.TrimSpace(supplierID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if retailerID == "" || idempotencyKey == "" {
		return Order{}, false, nil
	}
	if s.spannerClient != nil {
		sql := `SELECT OrderId FROM Orders@{FORCE_INDEX=UQ_Orders_ByRetailerIdempotency}
			WHERE RetailerId = @rid AND IdempotencyKey = @ikey`
		params := map[string]any{"rid": retailerID, "ikey": idempotencyKey}
		if supplierID != "" {
			sql += ` AND SupplierId = @sid`
			params["sid"] = supplierID
		}
		sql += ` LIMIT 1`
		iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return Order{}, false, nil
		}
		if err != nil {
			return Order{}, false, fmt.Errorf("lookup order by idempotency key: %w", err)
		}
		var orderID string
		if err := row.Columns(&orderID); err != nil {
			return Order{}, false, err
		}
		if s.repo != nil {
			return s.repo.GetOrder(ctx, orderID)
		}
		return Order{OrderID: orderID, RetailerID: retailerID, SupplierID: supplierID, IdempotencyKey: idempotencyKey}, true, nil
	}

	if s.repo == nil {
		return Order{}, false, nil
	}
	s.usageMu.Lock()
	if s.ordersByIdemKey == nil {
		s.usageMu.Unlock()
		return Order{}, false, nil
	}
	var o Order
	var ok bool
	if supplierID != "" {
		o, ok = s.ordersByIdemKey[retailerID+"|"+supplierID+"|"+idempotencyKey]
	} else {
		o, ok = s.ordersByIdemKey[retailerID+"||"+idempotencyKey]
	}
	s.usageMu.Unlock()
	if !ok {
		return Order{}, false, nil
	}
	if s.repo != nil {
		if full, found, err := s.repo.GetOrder(ctx, o.OrderID); err == nil && found {
			return full, true, nil
		}
	}
	return o, true, nil
}

func (s *Service) recordOrderIdempotencyMemory(o Order) {
	if s == nil || strings.TrimSpace(o.IdempotencyKey) == "" || strings.TrimSpace(o.RetailerID) == "" {
		return
	}
	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	if s.ordersByIdemKey == nil {
		s.ordersByIdemKey = make(map[string]Order)
	}
	if strings.TrimSpace(o.SupplierID) != "" {
		s.ordersByIdemKey[o.RetailerID+"|"+o.SupplierID+"|"+o.IdempotencyKey] = o
	}
	s.ordersByIdemKey[o.RetailerID+"||"+o.IdempotencyKey] = o
}

func writeAuditEntriesInTxn(txn *spanner.ReadWriteTransaction, audits []outbox.AuditEntry) error {
	if txn == nil || len(audits) == 0 {
		return nil
	}
	muts := make([]*spanner.Mutation, 0, len(audits))
	for _, a := range audits {
		muts = append(muts, spanner.InsertMap("AuditLog", a.AuditRowMap()))
	}
	return txn.BufferWrite(muts)
}

func findExistingOrderByIdempotencyInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, idempotencyKey string) (string, bool, error) {
	if txn == nil || strings.TrimSpace(retailerID) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return "", false, nil
	}
	sql := `SELECT OrderId FROM Orders@{FORCE_INDEX=UQ_Orders_ByRetailerIdempotency}
		WHERE RetailerId = @rid AND SupplierId = @sid AND IdempotencyKey = @ikey LIMIT 1`
	iter := txn.Query(ctx, spanner.Statement{
		SQL: sql,
		Params: map[string]any{
			"rid":  strings.TrimSpace(retailerID),
			"sid":  strings.TrimSpace(supplierID),
			"ikey": strings.TrimSpace(idempotencyKey),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	var orderID string
	if err := row.Columns(&orderID); err != nil {
		return "", false, err
	}
	return orderID, true, nil
}

type orderUsageAuditDetails struct {
	UsageUserID      string `json:"usage_user_id"`
	UsageDate        string `json:"usage_date"`
	PaymentMethod    string `json:"payment_method"`
	SpentCreditMinor int64  `json:"spent_credit_minor"`
	SpentCashMinor   int64  `json:"spent_cash_minor"`
	OrderCount       int64  `json:"order_count"`
}

// rollbackOrdersUsageInTxn atomically reverses RetailerSellerDailyUsage across cancelled child orders
// inside one Spanner ReadWriteTransaction, aggregating per (RetailerId, UserId, UsageDate) key to avoid
// duplicate mutations to the same Spanner row and guarding each order with ORDER_USAGE_ROLLED_BACK.
func rollbackOrdersUsageInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderIDs []string, fallbackRetailerID string) error {
	if txn == nil || len(orderIDs) == 0 {
		return nil
	}

	type aggregatedDelta struct {
		retailerID  string
		userID      string
		usageDate   civil.Date
		creditMinor int64
		cashMinor   int64
		orderCount  int64
	}
	type auditRollback struct {
		orderID     string
		retailerID  string
		userID      string
		usageDate   string
		creditMinor int64
		cashMinor   int64
		orderCount  int64
	}

	byKey := make(map[sellerUsageKey]*aggregatedDelta)
	var orderedKeys []sellerUsageKey
	var audits []auditRollback

	seenOrder := make(map[string]struct{}, len(orderIDs))
	for _, rawOID := range orderIDs {
		orderID := strings.TrimSpace(rawOID)
		if orderID == "" {
			continue
		}
		if _, dup := seenOrder[orderID]; dup {
			continue
		}
		seenOrder[orderID] = struct{}{}

		checkStmt := spanner.Statement{
			SQL: `SELECT AuditId FROM AuditLog@{FORCE_INDEX=Idx_AuditLog_ByAggregateCreated}
				WHERE AggregateType = 'Order' AND AggregateId = @oid AND Action = 'ORDER_USAGE_ROLLED_BACK'
				LIMIT 1`,
			Params: map[string]any{"oid": orderID},
		}
		checkIter := txn.Query(ctx, checkStmt)
		_, checkErr := checkIter.Next()
		checkIter.Stop()
		if checkErr == nil {
			continue // already rolled back
		}
		if checkErr != iterator.Done {
			return fmt.Errorf("check usage rollback audit: %w", checkErr)
		}

		readStmt := spanner.Statement{
			SQL: `SELECT DetailsJson FROM AuditLog@{FORCE_INDEX=Idx_AuditLog_ByAggregateCreated}
				WHERE AggregateType = 'Order' AND AggregateId = @oid AND Action = 'ORDER_CREATED'
				ORDER BY CreatedAt DESC LIMIT 1`,
			Params: map[string]any{"oid": orderID},
		}
		readIter := txn.Query(ctx, readStmt)
		row, readErr := readIter.Next()
		readIter.Stop()
		if readErr == iterator.Done {
			continue
		}
		if readErr != nil {
			return fmt.Errorf("read order usage audit: %w", readErr)
		}
		var raw []byte
		if err := row.Columns(&raw); err != nil {
			return err
		}
		var details orderUsageAuditDetails
		if err := json.Unmarshal(raw, &details); err != nil {
			continue
		}
		if details.UsageUserID == "" || details.UsageDate == "" || (details.SpentCreditMinor == 0 && details.SpentCashMinor == 0 && details.OrderCount == 0) {
			continue
		}
		usageDate, err := civil.ParseDate(details.UsageDate)
		if err != nil {
			continue
		}
		retailerID := strings.TrimSpace(fallbackRetailerID)
		if retailerID == "" {
			oRow, oErr := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"RetailerId"})
			if oErr != nil {
				if spanner.ErrCode(oErr) == codes.NotFound {
					continue
				}
				return oErr
			}
			if err := oRow.Columns(&retailerID); err != nil {
				return err
			}
		}
		deltaCount := details.OrderCount
		if deltaCount <= 0 {
			deltaCount = 1
		}

		k := sellerUsageKey{RetailerID: retailerID, UserID: details.UsageUserID, UsageDate: details.UsageDate}
		agg, exists := byKey[k]
		if !exists {
			agg = &aggregatedDelta{
				retailerID: retailerID,
				userID:     details.UsageUserID,
				usageDate:  usageDate,
			}
			byKey[k] = agg
			orderedKeys = append(orderedKeys, k)
		}
		agg.creditMinor += details.SpentCreditMinor
		agg.cashMinor += details.SpentCashMinor
		agg.orderCount += deltaCount
		audits = append(audits, auditRollback{
			orderID:     orderID,
			retailerID:  retailerID,
			userID:      details.UsageUserID,
			usageDate:   details.UsageDate,
			creditMinor: details.SpentCreditMinor,
			cashMinor:   details.SpentCashMinor,
			orderCount:  deltaCount,
		})
	}

	if len(orderedKeys) == 0 {
		return nil
	}

	var muts []*spanner.Mutation
	for _, k := range orderedKeys {
		agg := byKey[k]
		var curCredit, curCash, curCount int64
		uRow, uErr := txn.ReadRow(ctx, "RetailerSellerDailyUsage", spanner.Key{agg.retailerID, agg.userID, agg.usageDate},
			[]string{"SpentCreditMinor", "SpentCashMinor", "OrderCount"})
		if uErr != nil {
			if spanner.ErrCode(uErr) == codes.NotFound {
				continue
			}
			return fmt.Errorf("read RetailerSellerDailyUsage for rollback: %w", uErr)
		}
		if err := uRow.Columns(&curCredit, &curCash, &curCount); err != nil {
			return err
		}
		newCredit := curCredit - agg.creditMinor
		if newCredit < 0 {
			newCredit = 0
		}
		newCash := curCash - agg.cashMinor
		if newCash < 0 {
			newCash = 0
		}
		newCount := curCount - agg.orderCount
		if newCount < 0 {
			newCount = 0
		}
		muts = append(muts, spanner.UpdateMap("RetailerSellerDailyUsage", map[string]any{
			"RetailerId":       agg.retailerID,
			"UserId":           agg.userID,
			"UsageDate":        agg.usageDate,
			"SpentCreditMinor": newCredit,
			"SpentCashMinor":   newCash,
			"OrderCount":       newCount,
			"UpdatedAt":        spanner.CommitTimestamp,
		}))
	}

	if len(muts) > 0 {
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
	}

	auditBuf := &spannerTxnBuffer{}
	for _, a := range audits {
		if err := outbox.WriteAudit(ctx, auditBuf, a.retailerID, "system", "SYSTEM", "ORDER_USAGE_ROLLED_BACK", "Order", a.orderID, map[string]any{
			"usage_user_id":      a.userID,
			"usage_date":         a.usageDate,
			"spent_credit_minor": a.creditMinor,
			"spent_cash_minor":   a.cashMinor,
			"order_count":        a.orderCount,
		}); err != nil {
			return err
		}
	}
	return writeAuditEntriesInTxn(txn, auditBuf.audits)
}
