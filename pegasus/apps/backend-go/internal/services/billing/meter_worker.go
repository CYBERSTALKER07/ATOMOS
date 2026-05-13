package billing

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	billingShardCount  int64  = 32
	billingAllTime     string = "ALL_TIME"
	billingAllCurrency string = "ALL"

	systemKeyPlatformFeePercent        string = "platform_fee_percent"
	systemKeyPlatformFeeBasisPoints    string = "platform_fee_basis_points"
	systemKeyBillingMilestoneOrderCnt  string = "billing_milestone_order_count"
	systemKeyBillingMilestoneStepBP    string = "billing_milestone_step_basis_points"
	systemKeyBillingMinFeeBP           string = "billing_min_fee_basis_points"
	systemKeyBillingLastMilestoneIndex string = "billing_last_milestone_index"

	defaultPlatformFeeBP          int64 = 0
	defaultBillingMilestoneOrders int64 = 100000
	defaultBillingMilestoneStepBP int64 = 25
	defaultBillingMinFeeBP        int64 = 25
	defaultBillingLastIndex       int64 = 0

	kafkaTopicMain       string = "pegasus-logistics-events"
	eventFeeRateAdjusted string = "FEE_RATE_ADJUSTED"
)

// FinalizedOrderInput is the normalized payload the Kafka worker passes to metering.
type FinalizedOrderInput struct {
	OrderID    string
	InvoiceID  string
	SupplierID string
	RetailerID string
	Timestamp  time.Time
}

// MeterWorker performs idempotent ORDER_FINALIZED billing metering.
type MeterWorker struct {
	Spanner *spanner.Client
}

// NewMeterWorker builds a billing meter worker.
func NewMeterWorker(sc *spanner.Client) *MeterWorker {
	return &MeterWorker{Spanner: sc}
}

type orderSnapshot struct {
	OrderID    string
	InvoiceID  string
	SupplierID string
	RetailerID string
	Amount     int64
	Currency   string
	Month      string
}

type feeRateAdjustedEvent struct {
	PreviousFeeBasisPoints int64     `json:"previous_fee_basis_points"`
	NewFeeBasisPoints      int64     `json:"new_fee_basis_points"`
	MilestoneOrderCount    int64     `json:"milestone_order_count"`
	GlobalOrderCount       int64     `json:"global_order_count"`
	MilestoneIndex         int64     `json:"milestone_index"`
	TriggerOrderID         string    `json:"trigger_order_id"`
	Timestamp              time.Time `json:"timestamp"`
}

// ProcessFinalizedOrder consumes one ORDER_FINALIZED event, updates sharded meters,
// and emits FEE_RATE_ADJUSTED when milestone thresholds are crossed.
func (w *MeterWorker) ProcessFinalizedOrder(ctx context.Context, input FinalizedOrderInput) error {
	if w == nil || w.Spanner == nil {
		return fmt.Errorf("billing meter worker: nil spanner client")
	}

	input.OrderID = strings.TrimSpace(input.OrderID)
	if input.OrderID == "" {
		return fmt.Errorf("billing meter worker: empty order_id")
	}

	eventTime := input.Timestamp.UTC()
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}
	month := eventTime.Format("2006-01")
	shardID := shardForOrder(input.OrderID)

	_, err := w.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		alreadyProcessed, err := meterEventExists(ctx, txn, input.OrderID)
		if err != nil {
			return fmt.Errorf("billing meter worker: meter event read: %w", err)
		}
		if alreadyProcessed {
			return nil
		}

		snapshot, err := resolveOrderSnapshot(ctx, txn, input, month)
		if err != nil {
			return fmt.Errorf("billing meter worker: resolve snapshot for %s: %w", input.OrderID, err)
		}

		feeBasisPoints, err := readPlatformFeeBasisPoints(ctx, txn)
		if err != nil {
			return fmt.Errorf("billing meter worker: read fee basis points: %w", err)
		}
		feeAmount := (snapshot.Amount * feeBasisPoints) / 10000

		globalOrdersBefore, err := readGlobalOrderCount(ctx, txn)
		if err != nil {
			return fmt.Errorf("billing meter worker: read global order count: %w", err)
		}

		if err := insertMeterEvent(txn, snapshot, feeAmount, feeBasisPoints, eventTime); err != nil {
			return fmt.Errorf("billing meter worker: insert meter event: %w", err)
		}
		if err := incrementSupplierMeter(ctx, txn, snapshot, shardID, feeAmount); err != nil {
			return fmt.Errorf("billing meter worker: increment supplier meter: %w", err)
		}
		if err := incrementGlobalMeter(ctx, txn, snapshot.Month, snapshot.Currency, shardID, snapshot.Amount, feeAmount); err != nil {
			return fmt.Errorf("billing meter worker: increment global meter month: %w", err)
		}
		if err := incrementGlobalMeter(ctx, txn, billingAllTime, billingAllCurrency, shardID, 0, 0); err != nil {
			return fmt.Errorf("billing meter worker: increment global meter all-time: %w", err)
		}

		nextGlobalOrders := globalOrdersBefore + 1
		if err := maybeAdjustFee(ctx, txn, input.OrderID, feeBasisPoints, nextGlobalOrders, eventTime); err != nil {
			return fmt.Errorf("billing meter worker: milestone adjustment: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func meterEventExists(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (bool, error) {
	_, err := txn.ReadRow(ctx, "BillingMeterEvents", spanner.Key{orderID}, []string{"OrderId"})
	if err == nil {
		return true, nil
	}
	if errors.Is(err, spanner.ErrRowNotFound) {
		return false, nil
	}
	return false, err
}

func resolveOrderSnapshot(ctx context.Context, txn *spanner.ReadWriteTransaction, input FinalizedOrderInput, month string) (orderSnapshot, error) {
	snapshot := orderSnapshot{
		OrderID:    input.OrderID,
		InvoiceID:  strings.TrimSpace(input.InvoiceID),
		SupplierID: strings.TrimSpace(input.SupplierID),
		RetailerID: strings.TrimSpace(input.RetailerID),
		Currency:   "UZS",
		Month:      month,
	}

	orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{input.OrderID}, []string{"SupplierId", "RetailerId", "InvoiceId", "Amount", "Currency"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return snapshot, fmt.Errorf("order not found")
		}
		return snapshot, err
	}

	var orderSupplierID, orderRetailerID, orderInvoiceID, orderCurrency spanner.NullString
	var orderAmount spanner.NullInt64
	if err := orderRow.Columns(&orderSupplierID, &orderRetailerID, &orderInvoiceID, &orderAmount, &orderCurrency); err != nil {
		return snapshot, err
	}

	if snapshot.SupplierID == "" && orderSupplierID.Valid {
		snapshot.SupplierID = strings.TrimSpace(orderSupplierID.StringVal)
	}
	if snapshot.RetailerID == "" && orderRetailerID.Valid {
		snapshot.RetailerID = strings.TrimSpace(orderRetailerID.StringVal)
	}
	if snapshot.InvoiceID == "" && orderInvoiceID.Valid {
		snapshot.InvoiceID = strings.TrimSpace(orderInvoiceID.StringVal)
	}
	if orderAmount.Valid {
		snapshot.Amount = orderAmount.Int64
	}
	if orderCurrency.Valid && strings.TrimSpace(orderCurrency.StringVal) != "" {
		snapshot.Currency = strings.ToUpper(strings.TrimSpace(orderCurrency.StringVal))
	}

	if snapshot.InvoiceID != "" {
		if err := hydrateFromInvoice(ctx, txn, &snapshot, snapshot.InvoiceID); err != nil {
			return snapshot, err
		}
	} else {
		invoiceID, err := findInvoiceIDByOrder(ctx, txn, input.OrderID)
		if err != nil {
			return snapshot, err
		}
		if invoiceID != "" {
			snapshot.InvoiceID = invoiceID
			if err := hydrateFromInvoice(ctx, txn, &snapshot, invoiceID); err != nil {
				return snapshot, err
			}
		}
	}

	if snapshot.SupplierID == "" {
		return snapshot, fmt.Errorf("missing supplier_id")
	}
	if snapshot.Currency == "" {
		snapshot.Currency = "UZS"
	}

	return snapshot, nil
}

func hydrateFromInvoice(ctx context.Context, txn *spanner.ReadWriteTransaction, snapshot *orderSnapshot, invoiceID string) error {
	row, err := txn.ReadRow(ctx, "MasterInvoices", spanner.Key{invoiceID}, []string{"Total", "Currency"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return nil
		}
		return err
	}

	var total spanner.NullInt64
	var currency spanner.NullString
	if err := row.Columns(&total, &currency); err != nil {
		return err
	}
	if total.Valid {
		snapshot.Amount = total.Int64
	}
	if currency.Valid && strings.TrimSpace(currency.StringVal) != "" {
		snapshot.Currency = strings.ToUpper(strings.TrimSpace(currency.StringVal))
	}
	return nil
}

func findInvoiceIDByOrder(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (string, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT InvoiceId FROM MasterInvoices WHERE OrderId = @orderId LIMIT 1`,
		Params: map[string]interface{}{"orderId": orderID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return "", nil
		}
		return "", err
	}

	var invoiceID spanner.NullString
	if err := row.Columns(&invoiceID); err != nil {
		return "", err
	}
	if !invoiceID.Valid {
		return "", nil
	}
	return strings.TrimSpace(invoiceID.StringVal), nil
}

func readPlatformFeeBasisPoints(ctx context.Context, txn *spanner.ReadWriteTransaction) (int64, error) {
	if bp, ok, err := readSystemConfigInt64(ctx, txn, systemKeyPlatformFeeBasisPoints); err != nil {
		return 0, err
	} else if ok {
		if bp < 0 {
			return 0, nil
		}
		if bp > 10000 {
			return 10000, nil
		}
		return bp, nil
	}

	if pct, ok, err := readSystemConfigInt64(ctx, txn, systemKeyPlatformFeePercent); err != nil {
		return 0, err
	} else if ok {
		if pct < 0 {
			return 0, nil
		}
		if pct > 100 {
			return 10000, nil
		}
		return pct * 100, nil
	}

	return defaultPlatformFeeBP, nil
}

func readSystemConfigInt64(ctx context.Context, txn *spanner.ReadWriteTransaction, key string) (int64, bool, error) {
	row, err := txn.ReadRow(ctx, "SystemConfig", spanner.Key{key}, []string{"ConfigValue"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}

	var value string
	if err := row.Columns(&value); err != nil {
		return 0, false, err
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}

	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, nil
	}
	return n, true, nil
}

func upsertSystemConfigInt64(txn *spanner.ReadWriteTransaction, key string, value int64) error {
	return txn.BufferWrite([]*spanner.Mutation{
		spanner.InsertOrUpdate("SystemConfig",
			[]string{"ConfigKey", "ConfigValue", "UpdatedAt"},
			[]interface{}{key, strconv.FormatInt(value, 10), spanner.CommitTimestamp}),
	})
}

func insertMeterEvent(txn *spanner.ReadWriteTransaction, snapshot orderSnapshot, feeAmount, feeBasisPoints int64, eventTime time.Time) error {
	return txn.BufferWrite([]*spanner.Mutation{
		spanner.Insert("BillingMeterEvents",
			[]string{"OrderId", "SupplierId", "InvoiceId", "RetailerId", "PeriodMonth", "Currency", "Amount", "FeeAmount", "FeeBasisPts", "EventTimestamp", "ProcessedAt"},
			[]interface{}{snapshot.OrderID, snapshot.SupplierID, snapshot.InvoiceID, snapshot.RetailerID, snapshot.Month, snapshot.Currency, snapshot.Amount, feeAmount, feeBasisPoints, eventTime, spanner.CommitTimestamp}),
	})
}

func incrementSupplierMeter(ctx context.Context, txn *spanner.ReadWriteTransaction, snapshot orderSnapshot, shardID int64, feeAmount int64) error {
	key := spanner.Key{snapshot.SupplierID, snapshot.Month, snapshot.Currency, shardID}
	orderCount, grossAmount, feeTotal, err := readMeterRow(ctx, txn, "BillingSupplierMeters", key)
	if err != nil {
		return err
	}

	return txn.BufferWrite([]*spanner.Mutation{
		spanner.InsertOrUpdate("BillingSupplierMeters",
			[]string{"SupplierId", "PeriodMonth", "Currency", "ShardId", "OrderCount", "GrossAmount", "FeeAmount", "UpdatedAt"},
			[]interface{}{snapshot.SupplierID, snapshot.Month, snapshot.Currency, shardID, orderCount + 1, grossAmount + snapshot.Amount, feeTotal + feeAmount, spanner.CommitTimestamp}),
	})
}

func incrementGlobalMeter(ctx context.Context, txn *spanner.ReadWriteTransaction, periodMonth, currency string, shardID int64, grossDelta, feeDelta int64) error {
	key := spanner.Key{periodMonth, currency, shardID}
	orderCount, grossAmount, feeAmount, err := readMeterRow(ctx, txn, "BillingGlobalMeters", key)
	if err != nil {
		return err
	}

	return txn.BufferWrite([]*spanner.Mutation{
		spanner.InsertOrUpdate("BillingGlobalMeters",
			[]string{"PeriodMonth", "Currency", "ShardId", "OrderCount", "GrossAmount", "FeeAmount", "UpdatedAt"},
			[]interface{}{periodMonth, currency, shardID, orderCount + 1, grossAmount + grossDelta, feeAmount + feeDelta, spanner.CommitTimestamp}),
	})
}

func readMeterRow(ctx context.Context, txn *spanner.ReadWriteTransaction, table string, key spanner.Key) (int64, int64, int64, error) {
	row, err := txn.ReadRow(ctx, table, key, []string{"OrderCount", "GrossAmount", "FeeAmount"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return 0, 0, 0, nil
		}
		return 0, 0, 0, err
	}

	var orderCount, grossAmount, feeAmount spanner.NullInt64
	if err := row.Columns(&orderCount, &grossAmount, &feeAmount); err != nil {
		return 0, 0, 0, err
	}

	return orderCount.Int64, grossAmount.Int64, feeAmount.Int64, nil
}

func readGlobalOrderCount(ctx context.Context, txn *spanner.ReadWriteTransaction) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT IFNULL(SUM(OrderCount), 0)
		      FROM BillingGlobalMeters
		      WHERE PeriodMonth = @periodMonth AND Currency = @currency`,
		Params: map[string]interface{}{
			"periodMonth": billingAllTime,
			"currency":    billingAllCurrency,
		},
	}

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return 0, nil
		}
		return 0, err
	}

	var total spanner.NullInt64
	if err := row.Columns(&total); err != nil {
		return 0, err
	}
	return total.Int64, nil
}

func maybeAdjustFee(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string, currentFeeBP int64, nextGlobalOrders int64, eventTime time.Time) error {
	milestoneOrders, err := readConfigInt64OrDefault(ctx, txn, systemKeyBillingMilestoneOrderCnt, defaultBillingMilestoneOrders)
	if err != nil {
		return err
	}
	if milestoneOrders <= 0 {
		return nil
	}

	stepBP, err := readConfigInt64OrDefault(ctx, txn, systemKeyBillingMilestoneStepBP, defaultBillingMilestoneStepBP)
	if err != nil {
		return err
	}
	if stepBP <= 0 {
		return nil
	}

	minFeeBP, err := readConfigInt64OrDefault(ctx, txn, systemKeyBillingMinFeeBP, defaultBillingMinFeeBP)
	if err != nil {
		return err
	}
	if minFeeBP < 0 {
		minFeeBP = 0
	}

	lastIndex, err := readConfigInt64OrDefault(ctx, txn, systemKeyBillingLastMilestoneIndex, defaultBillingLastIndex)
	if err != nil {
		return err
	}

	nextIndex := nextGlobalOrders / milestoneOrders
	if nextIndex <= lastIndex {
		return nil
	}

	reduction := (nextIndex - lastIndex) * stepBP
	newFeeBP := currentFeeBP - reduction
	if newFeeBP < minFeeBP {
		newFeeBP = minFeeBP
	}
	if newFeeBP < 0 {
		newFeeBP = 0
	}

	if err := upsertSystemConfigInt64(txn, systemKeyBillingLastMilestoneIndex, nextIndex); err != nil {
		return err
	}

	if newFeeBP == currentFeeBP {
		return nil
	}

	if err := upsertSystemConfigInt64(txn, systemKeyPlatformFeeBasisPoints, newFeeBP); err != nil {
		return err
	}
	if err := upsertSystemConfigInt64(txn, systemKeyPlatformFeePercent, newFeeBP/100); err != nil {
		return err
	}

	return outbox.EmitJSON(txn,
		"BillingConfig",
		"platform_fee",
		eventFeeRateAdjusted,
		kafkaTopicMain,
		feeRateAdjustedEvent{
			PreviousFeeBasisPoints: currentFeeBP,
			NewFeeBasisPoints:      newFeeBP,
			MilestoneOrderCount:    milestoneOrders,
			GlobalOrderCount:       nextGlobalOrders,
			MilestoneIndex:         nextIndex,
			TriggerOrderID:         orderID,
			Timestamp:              eventTime,
		},
		telemetry.TraceIDFromContext(ctx),
	)
}

func readConfigInt64OrDefault(ctx context.Context, txn *spanner.ReadWriteTransaction, key string, fallback int64) (int64, error) {
	value, ok, err := readSystemConfigInt64(ctx, txn, key)
	if err != nil {
		return 0, err
	}
	if !ok {
		return fallback, nil
	}
	return value, nil
}

func shardForOrder(orderID string) int64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(orderID))
	return int64(h.Sum32() % uint32(billingShardCount))
}
