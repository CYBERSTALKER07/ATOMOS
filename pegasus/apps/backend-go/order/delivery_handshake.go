package order

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const (
	DeliverySessionStateProximityLock   = "PROXIMITY_LOCK"
	DeliverySessionStateHandshakeStart  = "HANDSHAKE_START"
	DeliverySessionStateReconciliation  = "RECONCILIATION"
	DeliverySessionStateSettlementAwait = "SETTLEMENT_AWAIT"
	DeliverySessionStateFinalSettlement = "FINAL_SETTLEMENT"
	DeliverySessionStateDisputed        = "DISPUTED"

	deliveryHandshakeMaxDistanceMeters = 100.0
)

// VerifyHandshakeRequest validates the scanned retailer token and geofence.
type VerifyHandshakeRequest struct {
	OrderID         string  `json:"order_id"`
	HandshakeToken  string  `json:"handshake_token"`
	DriverID        string  `json:"driver_id,omitempty"`
	DriverLatitude  float64 `json:"driver_latitude"`
	DriverLongitude float64 `json:"driver_longitude"`
}

// VerifyHandshakeResponse returns the active session snapshot after verification.
type VerifyHandshakeResponse struct {
	SessionID        string  `json:"session_id"`
	OrderID          string  `json:"order_id"`
	RetailerID       string  `json:"retailer_id"`
	DriverID         string  `json:"driver_id"`
	SupplierID       string  `json:"supplier_id,omitempty"`
	State            string  `json:"state"`
	Amount           int64   `json:"amount"`
	Currency         string  `json:"currency"`
	FeeBasisPoints   int64   `json:"fee_basis_points"`
	FeeAmount        int64   `json:"fee_amount"`
	DistanceM        float64 `json:"distance_m"`
	HandshakeJWTUsed bool    `json:"handshake_jwt_used"`
}

// UpdateOrderDuringDeliveryRequest applies driver reconciliation edits.
type UpdateOrderDuringDeliveryRequest struct {
	OrderID     string         `json:"order_id"`
	DriverID    string         `json:"driver_id,omitempty"`
	Items       []AmendItemReq `json:"items"`
	DriverNotes string         `json:"driver_notes,omitempty"`
}

// UpdateOrderDuringDeliveryResponse returns amended settlement totals.
type UpdateOrderDuringDeliveryResponse struct {
	SessionID      string `json:"session_id"`
	OrderID        string `json:"order_id"`
	State          string `json:"state"`
	AmendmentID    string `json:"amendment_id"`
	OriginalAmount int64  `json:"original_amount"`
	AdjustedAmount int64  `json:"adjusted_amount"`
	FeeBasisPoints int64  `json:"fee_basis_points"`
	FeeAmount      int64  `json:"fee_amount"`
	Currency       string `json:"currency"`
	RetailerID     string `json:"retailer_id,omitempty"`
	DriverID       string `json:"driver_id,omitempty"`
	SupplierID     string `json:"supplier_id,omitempty"`
}

type deliveryHandshakeClaims struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id"`
	jwt.RegisteredClaims
}

type deliverySessionUpsertInput struct {
	OrderID              string
	RetailerID           string
	DriverID             string
	SupplierID           string
	State                string
	OriginalAmount       int64
	AdjustedAmount       int64
	FeeBasisPoints       int64
	FeeAmount            int64
	Currency             string
	HandshakeTokenHash   string
	RetailerLat          *float64
	RetailerLng          *float64
	DriverLat            *float64
	DriverLng            *float64
	DistanceM            *float64
	LastErrorCode        string
	LastErrorMessage     string
	PaymentSessionID     string
	InvoiceID            string
	HandshakeVerifiedAt  *time.Time
	SettlementRequiredAt *time.Time
	PaymentClearedAt     *time.Time
	DisputedAt           *time.Time
}

// VerifyHandshake validates QR/JWT token plus geofence and opens handshake state.
func (s *OrderService) VerifyHandshake(ctx context.Context, req VerifyHandshakeRequest) (*VerifyHandshakeResponse, error) {
	if strings.TrimSpace(req.OrderID) == "" || strings.TrimSpace(req.HandshakeToken) == "" {
		return nil, fmt.Errorf("order_id and handshake_token are required")
	}

	resp := &VerifyHandshakeResponse{OrderID: req.OrderID}
	handshakeToken := strings.TrimSpace(req.HandshakeToken)
	driverID := strings.TrimSpace(req.DriverID)

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"State", "RetailerId", "DriverId", "SupplierId", "Amount", "Currency", "ShopLocation", "DeliveryToken", "QRValidatedAt"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", req.OrderID, err)
		}

		var state string
		var retailerID string
		var assignedDriverID spanner.NullString
		var supplierID spanner.NullString
		var amount int64
		var currency string
		var shopLocation spanner.NullString
		var deliveryToken spanner.NullString
		var qrValidatedAt spanner.NullTime
		if err := row.Columns(&state, &retailerID, &assignedDriverID, &supplierID, &amount, &currency, &shopLocation, &deliveryToken, &qrValidatedAt); err != nil {
			return fmt.Errorf("read order snapshot: %w", err)
		}

		if state != "ARRIVED" && state != "AWAITING_PAYMENT" {
			return fmt.Errorf("order %s must be ARRIVED or AWAITING_PAYMENT to verify handshake (current: %s)", req.OrderID, state)
		}

		if assignedDriverID.Valid && assignedDriverID.StringVal != "" {
			if driverID == "" {
				driverID = assignedDriverID.StringVal
			}
			if driverID != assignedDriverID.StringVal {
				return fmt.Errorf("driver %s is not assigned to order %s", driverID, req.OrderID)
			}
		}
		if driverID == "" {
			return fmt.Errorf("driver assignment required before handshake")
		}

		if !shopLocation.Valid || strings.TrimSpace(shopLocation.StringVal) == "" {
			return fmt.Errorf("order %s has no retailer location for handshake validation", req.OrderID)
		}
		retailerLoc, parseErr := parseWKTPoint(shopLocation.StringVal)
		if parseErr != nil {
			return fmt.Errorf("parse retailer location: %w", parseErr)
		}

		distanceM := getDistance(req.DriverLatitude, req.DriverLongitude, retailerLoc.Latitude, retailerLoc.Longitude)
		if distanceM > deliveryHandshakeMaxDistanceMeters {
			return fmt.Errorf("driver is %.0fm from retailer (max %.0fm)", distanceM, deliveryHandshakeMaxDistanceMeters)
		}

		jwtUsed, tokenErr := validateDeliveryHandshakeToken(handshakeToken, req.OrderID, retailerID)
		if tokenErr != nil {
			if !deliveryToken.Valid || subtle.ConstantTimeCompare([]byte(deliveryToken.StringVal), []byte(handshakeToken)) != 1 {
				return fmt.Errorf("invalid handshake token")
			}
			jwtUsed = false
		}
		resp.HandshakeJWTUsed = jwtUsed

		if !qrValidatedAt.Valid {
			if _, err := txn.Update(ctx, spanner.Statement{
				SQL:    `UPDATE Orders SET QRValidatedAt = CURRENT_TIMESTAMP() WHERE OrderId = @id`,
				Params: map[string]interface{}{"id": req.OrderID},
			}); err != nil {
				return fmt.Errorf("stamp QRValidatedAt: %w", err)
			}
		}

		feeBasisPoints := s.feeBasisPoints()
		feeAmount := (amount * feeBasisPoints) / 10000
		now := time.Now().UTC()
		retailerLat := retailerLoc.Latitude
		retailerLng := retailerLoc.Longitude
		driverLat := req.DriverLatitude
		driverLng := req.DriverLongitude
		distanceCopy := distanceM

		sessionID, err := s.upsertDeliverySessionTxn(ctx, txn, deliverySessionUpsertInput{
			OrderID:             req.OrderID,
			RetailerID:          retailerID,
			DriverID:            driverID,
			SupplierID:          strings.TrimSpace(supplierID.StringVal),
			State:               DeliverySessionStateHandshakeStart,
			OriginalAmount:      amount,
			AdjustedAmount:      amount,
			FeeBasisPoints:      feeBasisPoints,
			FeeAmount:           feeAmount,
			Currency:            normalizeHandshakeCurrency(currency),
			HandshakeTokenHash:  hashHandshakeToken(handshakeToken),
			RetailerLat:         &retailerLat,
			RetailerLng:         &retailerLng,
			DriverLat:           &driverLat,
			DriverLng:           &driverLng,
			DistanceM:           &distanceCopy,
			HandshakeVerifiedAt: &now,
		})
		if err != nil {
			return err
		}

		traceID := telemetry.TraceIDFromContext(ctx)
		if err := outbox.EmitJSON(txn, "DeliverySession", sessionID, kafkaEvents.EventDeliverySessionUpdated, topicLogisticsEvents, kafkaEvents.DeliverySessionUpdatedEvent{
			SessionID:      sessionID,
			OrderID:        req.OrderID,
			RetailerID:     retailerID,
			DriverID:       driverID,
			State:          DeliverySessionStateHandshakeStart,
			OriginalAmount: amount,
			AdjustedAmount: amount,
			FeeBasisPoints: feeBasisPoints,
			FeeAmount:      feeAmount,
			Currency:       normalizeHandshakeCurrency(currency),
			Timestamp:      now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit DELIVERY_SESSION_UPDATED: %w", err)
		}

		resp.SessionID = sessionID
		resp.RetailerID = retailerID
		resp.DriverID = driverID
		resp.SupplierID = strings.TrimSpace(supplierID.StringVal)
		resp.State = DeliverySessionStateHandshakeStart
		resp.Amount = amount
		resp.Currency = normalizeHandshakeCurrency(currency)
		resp.FeeBasisPoints = feeBasisPoints
		resp.FeeAmount = feeAmount
		resp.DistanceM = distanceM
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// UpdateOrderDuringDelivery applies reconciliation edits and updates session totals.
func (s *OrderService) UpdateOrderDuringDelivery(ctx context.Context, req UpdateOrderDuringDeliveryRequest) (*UpdateOrderDuringDeliveryResponse, error) {
	if strings.TrimSpace(req.OrderID) == "" || len(req.Items) == 0 {
		return nil, fmt.Errorf("order_id and at least one item are required")
	}

	amendmentID := fmt.Sprintf("DLS-AMD-%s", GenerateSecureToken())
	amendResp, err := s.AmendOrder(ctx, AmendOrderRequest{
		OrderID:     req.OrderID,
		AmendmentID: amendmentID,
		Items:       req.Items,
		DriverNotes: req.DriverNotes,
	})
	if err != nil {
		return nil, err
	}

	resp := &UpdateOrderDuringDeliveryResponse{
		OrderID:        req.OrderID,
		State:          DeliverySessionStateReconciliation,
		AmendmentID:    amendmentID,
		AdjustedAmount: amendResp.AdjustedTotal,
		RetailerID:     amendResp.RetailerID,
		DriverID:       req.DriverID,
		SupplierID:     amendResp.SupplierID,
	}

	if strings.TrimSpace(resp.DriverID) == "" {
		resp.DriverID = amendResp.DriverID
	}

	_, err = s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		orderStmt := spanner.Statement{
			SQL: `SELECT RetailerId, COALESCE(DriverId, ''), COALESCE(SupplierId, ''), Amount, Currency
			      FROM Orders WHERE OrderId = @orderId LIMIT 1`,
			Params: map[string]interface{}{"orderId": req.OrderID},
		}
		orderIter := txn.Query(ctx, orderStmt)
		orderRow, orderErr := orderIter.Next()
		orderIter.Stop()
		if orderErr != nil {
			return fmt.Errorf("order snapshot for reconciliation failed: %w", orderErr)
		}

		var retailerID string
		var driverID string
		var supplierID string
		var adjustedAmount int64
		var currency string
		if err := orderRow.Columns(&retailerID, &driverID, &supplierID, &adjustedAmount, &currency); err != nil {
			return fmt.Errorf("scan order reconciliation snapshot: %w", err)
		}

		if strings.TrimSpace(resp.DriverID) == "" {
			resp.DriverID = strings.TrimSpace(driverID)
		}
		if strings.TrimSpace(resp.DriverID) == "" {
			return fmt.Errorf("driver_id is required for reconciliation")
		}
		if strings.TrimSpace(driverID) != "" && strings.TrimSpace(driverID) != strings.TrimSpace(resp.DriverID) {
			return fmt.Errorf("driver %s is not assigned to order %s", resp.DriverID, req.OrderID)
		}

		resp.RetailerID = retailerID
		resp.SupplierID = supplierID
		resp.AdjustedAmount = adjustedAmount
		resp.Currency = normalizeHandshakeCurrency(currency)
		resp.FeeBasisPoints = s.feeBasisPoints()
		resp.FeeAmount = (adjustedAmount * resp.FeeBasisPoints) / 10000

		prevStmt := spanner.Statement{
			SQL: `SELECT OriginalAmount FROM DeliverySessions
			      WHERE OrderId = @orderId
			      ORDER BY UpdatedAt DESC
			      LIMIT 1`,
			Params: map[string]interface{}{"orderId": req.OrderID},
		}
		prevIter := txn.Query(ctx, prevStmt)
		prevRow, prevErr := prevIter.Next()
		prevIter.Stop()
		if prevErr == nil {
			if err := prevRow.Columns(&resp.OriginalAmount); err != nil {
				return fmt.Errorf("scan existing session amount: %w", err)
			}
		} else if !errors.Is(prevErr, iterator.Done) {
			return fmt.Errorf("read existing session amount: %w", prevErr)
		}
		if resp.OriginalAmount == 0 {
			resp.OriginalAmount = resp.AdjustedAmount
		}

		sessionID, err := s.upsertDeliverySessionTxn(ctx, txn, deliverySessionUpsertInput{
			OrderID:        req.OrderID,
			RetailerID:     retailerID,
			DriverID:       strings.TrimSpace(resp.DriverID),
			SupplierID:     strings.TrimSpace(supplierID),
			State:          DeliverySessionStateReconciliation,
			OriginalAmount: resp.OriginalAmount,
			AdjustedAmount: resp.AdjustedAmount,
			FeeBasisPoints: resp.FeeBasisPoints,
			FeeAmount:      resp.FeeAmount,
			Currency:       resp.Currency,
		})
		if err != nil {
			return err
		}
		resp.SessionID = sessionID

		skuIDs := make([]string, 0, len(req.Items))
		for _, item := range req.Items {
			sku := strings.TrimSpace(item.ProductId)
			if sku == "" {
				return fmt.Errorf("item product_id is required")
			}
			skuIDs = append(skuIDs, sku)
		}

		lineItemStmt := spanner.Statement{
			SQL: `SELECT LineItemId, SkuId, UnitPrice FROM OrderLineItems
			      WHERE OrderId = @orderId AND SkuId IN UNNEST(@skuIds)`,
			Params: map[string]interface{}{
				"orderId": req.OrderID,
				"skuIds":  skuIDs,
			},
		}
		lineIter := txn.Query(ctx, lineItemStmt)
		type lineMeta struct {
			lineItemID string
			unitPrice  int64
		}
		lineItemsBySKU := map[string]lineMeta{}
		for {
			lineRow, lineErr := lineIter.Next()
			if errors.Is(lineErr, iterator.Done) {
				break
			}
			if lineErr != nil {
				lineIter.Stop()
				return fmt.Errorf("query order line items for reconciliation: %w", lineErr)
			}
			var lineItemID, skuID string
			var unitPrice int64
			if err := lineRow.Columns(&lineItemID, &skuID, &unitPrice); err != nil {
				lineIter.Stop()
				return fmt.Errorf("scan order line item metadata: %w", err)
			}
			lineItemsBySKU[skuID] = lineMeta{lineItemID: lineItemID, unitPrice: unitPrice}
		}
		lineIter.Stop()

		adjustmentMutations := make([]*spanner.Mutation, 0, len(req.Items))
		for _, item := range req.Items {
			meta, ok := lineItemsBySKU[strings.TrimSpace(item.ProductId)]
			if !ok {
				return fmt.Errorf("line item metadata missing for sku %s", item.ProductId)
			}
			adjustmentID := uuid.NewString()
			adjustmentMutations = append(adjustmentMutations,
				spanner.Insert("DeliverySessionAdjustments",
					[]string{"AdjustmentId", "SessionId", "OrderId", "LineItemId", "SkuId", "OriginalQty", "AcceptedQty", "RejectedQty", "UnitPrice", "Reason", "CreatedAt"},
					[]interface{}{
						adjustmentID,
						sessionID,
						req.OrderID,
						nullableStringValue(meta.lineItemID),
						strings.TrimSpace(item.ProductId),
						item.AcceptedQty + item.RejectedQty,
						item.AcceptedQty,
						item.RejectedQty,
						meta.unitPrice,
						nullableStringValue(item.Reason),
						spanner.CommitTimestamp,
					},
				),
			)
		}
		if len(adjustmentMutations) > 0 {
			if err := txn.BufferWrite(adjustmentMutations); err != nil {
				return fmt.Errorf("persist delivery session adjustments: %w", err)
			}
		}

		now := time.Now().UTC()
		if err := outbox.EmitJSON(txn, "DeliverySession", sessionID, kafkaEvents.EventDeliverySessionUpdated, topicLogisticsEvents, kafkaEvents.DeliverySessionUpdatedEvent{
			SessionID:      sessionID,
			OrderID:        req.OrderID,
			RetailerID:     retailerID,
			DriverID:       strings.TrimSpace(resp.DriverID),
			State:          DeliverySessionStateReconciliation,
			OriginalAmount: resp.OriginalAmount,
			AdjustedAmount: resp.AdjustedAmount,
			FeeBasisPoints: resp.FeeBasisPoints,
			FeeAmount:      resp.FeeAmount,
			Currency:       resp.Currency,
			Timestamp:      now,
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit DELIVERY_SESSION_UPDATED (reconciliation): %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *OrderService) upsertDeliverySessionTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, input deliverySessionUpsertInput) (string, error) {
	if strings.TrimSpace(input.OrderID) == "" {
		return "", fmt.Errorf("delivery session upsert requires order_id")
	}
	if strings.TrimSpace(input.RetailerID) == "" {
		return "", fmt.Errorf("delivery session upsert requires retailer_id")
	}
	if strings.TrimSpace(input.DriverID) == "" {
		return "", fmt.Errorf("delivery session upsert requires driver_id")
	}
	if strings.TrimSpace(input.State) == "" {
		return "", fmt.Errorf("delivery session upsert requires state")
	}

	sessionID := ""
	currentVersion := int64(0)

	lookupStmt := spanner.Statement{
		SQL: `SELECT SessionId, Version
		      FROM DeliverySessions
		      WHERE OrderId = @orderId
		      ORDER BY UpdatedAt DESC
		      LIMIT 1`,
		Params: map[string]interface{}{"orderId": input.OrderID},
	}
	lookupIter := txn.Query(ctx, lookupStmt)
	lookupRow, lookupErr := lookupIter.Next()
	lookupIter.Stop()
	if lookupErr == nil {
		if err := lookupRow.Columns(&sessionID, &currentVersion); err != nil {
			return "", fmt.Errorf("scan delivery session snapshot: %w", err)
		}
	} else if !errors.Is(lookupErr, iterator.Done) {
		return "", fmt.Errorf("lookup delivery session snapshot: %w", lookupErr)
	}

	currency := normalizeHandshakeCurrency(input.Currency)
	if currency == "" {
		currency = "UZS"
	}

	if sessionID == "" {
		sessionID = uuid.NewString()
		mutation := spanner.Insert("DeliverySessions",
			[]string{
				"SessionId", "OrderId", "RetailerId", "DriverId", "SupplierId", "State",
				"OriginalAmount", "AdjustedAmount", "FeeBasisPts", "FeeAmount", "Currency",
				"HandshakeTokenHash", "RetailerLat", "RetailerLng", "DriverLat", "DriverLng", "DistanceM",
				"LastErrorCode", "LastErrorMessage", "PaymentSessionId", "InvoiceId",
				"HandshakeVerifiedAt", "SettlementRequiredAt", "PaymentClearedAt", "DisputedAt",
				"Version", "CreatedAt", "UpdatedAt",
			},
			[]interface{}{
				sessionID,
				input.OrderID,
				input.RetailerID,
				input.DriverID,
				nullableStringValue(input.SupplierID),
				input.State,
				input.OriginalAmount,
				input.AdjustedAmount,
				input.FeeBasisPoints,
				input.FeeAmount,
				currency,
				nullableStringValue(input.HandshakeTokenHash),
				nullableFloat64Value(input.RetailerLat),
				nullableFloat64Value(input.RetailerLng),
				nullableFloat64Value(input.DriverLat),
				nullableFloat64Value(input.DriverLng),
				nullableFloat64Value(input.DistanceM),
				nullableStringValue(input.LastErrorCode),
				nullableStringValue(input.LastErrorMessage),
				nullableStringValue(input.PaymentSessionID),
				nullableStringValue(input.InvoiceID),
				nullableTimeValue(input.HandshakeVerifiedAt),
				nullableTimeValue(input.SettlementRequiredAt),
				nullableTimeValue(input.PaymentClearedAt),
				nullableTimeValue(input.DisputedAt),
				int64(1),
				spanner.CommitTimestamp,
				spanner.CommitTimestamp,
			},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{mutation}); err != nil {
			return "", fmt.Errorf("insert delivery session: %w", err)
		}
		return sessionID, nil
	}

	nextVersion := currentVersion + 1
	updateMutation := spanner.Update("DeliverySessions",
		[]string{
			"SessionId", "OrderId", "RetailerId", "DriverId", "SupplierId", "State",
			"OriginalAmount", "AdjustedAmount", "FeeBasisPts", "FeeAmount", "Currency",
			"HandshakeTokenHash", "RetailerLat", "RetailerLng", "DriverLat", "DriverLng", "DistanceM",
			"LastErrorCode", "LastErrorMessage", "PaymentSessionId", "InvoiceId",
			"HandshakeVerifiedAt", "SettlementRequiredAt", "PaymentClearedAt", "DisputedAt",
			"Version", "UpdatedAt",
		},
		[]interface{}{
			sessionID,
			input.OrderID,
			input.RetailerID,
			input.DriverID,
			nullableStringValue(input.SupplierID),
			input.State,
			input.OriginalAmount,
			input.AdjustedAmount,
			input.FeeBasisPoints,
			input.FeeAmount,
			currency,
			nullableStringValue(input.HandshakeTokenHash),
			nullableFloat64Value(input.RetailerLat),
			nullableFloat64Value(input.RetailerLng),
			nullableFloat64Value(input.DriverLat),
			nullableFloat64Value(input.DriverLng),
			nullableFloat64Value(input.DistanceM),
			nullableStringValue(input.LastErrorCode),
			nullableStringValue(input.LastErrorMessage),
			nullableStringValue(input.PaymentSessionID),
			nullableStringValue(input.InvoiceID),
			nullableTimeValue(input.HandshakeVerifiedAt),
			nullableTimeValue(input.SettlementRequiredAt),
			nullableTimeValue(input.PaymentClearedAt),
			nullableTimeValue(input.DisputedAt),
			nextVersion,
			spanner.CommitTimestamp,
		},
	)
	if err := txn.BufferWrite([]*spanner.Mutation{updateMutation}); err != nil {
		return "", fmt.Errorf("update delivery session %s: %w", sessionID, err)
	}

	return sessionID, nil
}

func validateDeliveryHandshakeToken(tokenString, expectedOrderID, expectedRetailerID string) (bool, error) {
	if strings.Count(tokenString, ".") != 2 {
		return false, fmt.Errorf("not a JWT handshake token")
	}

	secret := strings.TrimSpace(os.Getenv("DELIVERY_HANDSHAKE_JWT_SECRET"))
	if secret == "" && len(auth.JWTSecret) > 0 {
		secret = string(auth.JWTSecret)
	}
	if secret == "" {
		return false, fmt.Errorf("handshake jwt secret is not configured")
	}

	claims := &deliveryHandshakeClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return false, fmt.Errorf("parse handshake jwt: %w", err)
	}
	if !token.Valid {
		return false, fmt.Errorf("invalid handshake jwt")
	}

	if claims.OrderID != expectedOrderID {
		return false, fmt.Errorf("handshake jwt order mismatch")
	}
	if strings.TrimSpace(claims.RetailerID) != "" && strings.TrimSpace(expectedRetailerID) != "" && claims.RetailerID != expectedRetailerID {
		return false, fmt.Errorf("handshake jwt retailer mismatch")
	}

	return true, nil
}

func hashHandshakeToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// BindDeliverySessionPaymentLink attaches payment session artifacts to the
// most recent delivery session for an order.
func (s *OrderService) BindDeliverySessionPaymentLink(ctx context.Context, orderID, paymentSessionID, invoiceID string) error {
	if strings.TrimSpace(orderID) == "" {
		return fmt.Errorf("order_id is required")
	}
	if strings.TrimSpace(paymentSessionID) == "" && strings.TrimSpace(invoiceID) == "" {
		return nil
	}

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT SessionId FROM DeliverySessions
			      WHERE OrderId = @orderId
			      ORDER BY UpdatedAt DESC
			      LIMIT 1`,
			Params: map[string]interface{}{"orderId": orderID},
		}
		iter := txn.Query(ctx, stmt)
		row, readErr := iter.Next()
		iter.Stop()
		if errors.Is(readErr, iterator.Done) {
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("lookup delivery session for payment link: %w", readErr)
		}

		var sessionID string
		if err := row.Columns(&sessionID); err != nil {
			return fmt.Errorf("scan delivery session id for payment link: %w", err)
		}

		mutation := spanner.Update("DeliverySessions",
			[]string{"SessionId", "PaymentSessionId", "InvoiceId", "UpdatedAt"},
			[]interface{}{sessionID, nullableStringValue(paymentSessionID), nullableStringValue(invoiceID), spanner.CommitTimestamp},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{mutation}); err != nil {
			return fmt.Errorf("update delivery session payment linkage: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func normalizeHandshakeCurrency(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "UZS"
	}
	return normalized
}

func nullableStringValue(value string) spanner.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: trimmed, Valid: true}
}

func nullableFloat64Value(value *float64) spanner.NullFloat64 {
	if value == nil {
		return spanner.NullFloat64{}
	}
	return spanner.NullFloat64{Float64: *value, Valid: true}
}

func nullableTimeValue(value *time.Time) spanner.NullTime {
	if value == nil {
		return spanner.NullTime{}
	}
	return spanner.NullTime{Time: *value, Valid: true}
}
