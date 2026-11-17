package payment

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

var paymeMerchantColumns = []string{
	"PaymeId",
	"OrderId",
	"TargetType",
	"AmountMinor",
	"State",
	"CreateTime",
	"PerformTime",
	"CancelTime",
	"Reason",
	"MerchantTx",
}

// paymeMerchantSpanner persists Payme Merchant API transactions in Spanner so
// multi-replica API pods share Create/Perform/Cancel/Check/GetStatement state.
type paymeMerchantSpanner struct {
	client *spanner.Client
}

func newPaymeMerchantSpanner(client *spanner.Client) *paymeMerchantSpanner {
	if client == nil {
		return nil
	}
	return &paymeMerchantSpanner{client: client}
}

func scanPaymeMerchantRow(row *spanner.Row) (paymeMerchantTx, error) {
	var (
		tx        paymeMerchantTx
		state64   int64
		reasonVal spanner.NullInt64
	)
	if err := row.Columns(
		&tx.PaymeID,
		&tx.OrderID,
		&tx.TargetType,
		&tx.AmountMinor,
		&state64,
		&tx.CreateTime,
		&tx.PerformTime,
		&tx.CancelTime,
		&reasonVal,
		&tx.MerchantTx,
	); err != nil {
		return paymeMerchantTx{}, err
	}
	tx.State = int(state64)
	tx.TargetType = normalizePaymeTargetType(tx.TargetType)
	if reasonVal.Valid {
		r := int(reasonVal.Int64)
		tx.Reason = &r
	}
	return tx, nil
}

func paymeMerchantMutation(tx paymeMerchantTx) *spanner.Mutation {
	var reasonVal spanner.NullInt64
	if tx.Reason != nil {
		reasonVal = spanner.NullInt64{Int64: int64(*tx.Reason), Valid: true}
	}
	return spanner.InsertOrUpdateMap("PaymeMerchantTransactions", map[string]any{
		"PaymeId":     tx.PaymeID,
		"OrderId":     tx.OrderID,
		"TargetType":  normalizePaymeTargetType(tx.TargetType),
		"AmountMinor": tx.AmountMinor,
		"State":       int64(tx.State),
		"CreateTime":  tx.CreateTime,
		"PerformTime": tx.PerformTime,
		"CancelTime":  tx.CancelTime,
		"Reason":      reasonVal,
		"MerchantTx":  tx.MerchantTx,
		"UpdatedAt":   spanner.CommitTimestamp,
	})
}

func (s *paymeMerchantSpanner) snapshot(ctx context.Context, id string) (paymeMerchantTx, bool, error) {
	if s == nil || s.client == nil {
		return paymeMerchantTx{}, false, fmt.Errorf("payme merchant spanner: nil client")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return paymeMerchantTx{}, false, nil
	}
	row, err := s.client.Single().ReadRow(ctx, "PaymeMerchantTransactions", spanner.Key{id}, paymeMerchantColumns)
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return paymeMerchantTx{}, false, nil
		}
		return paymeMerchantTx{}, false, fmt.Errorf("payme merchant snapshot %s: %w", id, err)
	}
	tx, err := scanPaymeMerchantRow(row)
	if err != nil {
		return paymeMerchantTx{}, false, fmt.Errorf("payme merchant scan %s: %w", id, err)
	}
	return tx, true, nil
}

func (s *paymeMerchantSpanner) snapshotByOrder(ctx context.Context, orderID string) (paymeMerchantTx, bool, error) {
	if s == nil || s.client == nil {
		return paymeMerchantTx{}, false, fmt.Errorf("payme merchant spanner: nil client")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return paymeMerchantTx{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT PaymeId, OrderId, TargetType, AmountMinor, State, CreateTime, PerformTime, CancelTime, Reason, MerchantTx
			FROM PaymeMerchantTransactions@{FORCE_INDEX=Idx_PaymeMerchantTx_ByOrder}
			WHERE OrderId = @orderId
			ORDER BY CreateTime DESC
			LIMIT 1`,
		Params: map[string]any{"orderId": orderID},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return paymeMerchantTx{}, false, nil
	}
	if err != nil {
		return paymeMerchantTx{}, false, fmt.Errorf("payme merchant query order %s: %w", orderID, err)
	}
	tx, err := scanPaymeMerchantRow(row)
	if err != nil {
		return paymeMerchantTx{}, false, fmt.Errorf("payme merchant scan order %s: %w", orderID, err)
	}
	return tx, true, nil
}

func (s *paymeMerchantSpanner) put(ctx context.Context, tx paymeMerchantTx) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("payme merchant spanner: nil client")
	}
	return spannerutils.RunReadWriteTransaction(ctx, s.client, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{paymeMerchantMutation(tx)})
	})
}

func (s *paymeMerchantSpanner) statement(ctx context.Context, from, to int64) ([]paymeMerchantTx, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("payme merchant spanner: nil client")
	}
	conds := make([]string, 0, 2)
	params := make(map[string]any, 2)
	if from > 0 {
		conds = append(conds, "CreateTime >= @from")
		params["from"] = from
	}
	if to > 0 {
		conds = append(conds, "CreateTime <= @to")
		params["to"] = to
	}
	whereClause := ""
	if len(conds) > 0 {
		whereClause = "WHERE " + strings.Join(conds, " AND ")
	}
	stmt := spanner.Statement{
		SQL: fmt.Sprintf(`SELECT PaymeId, OrderId, TargetType, AmountMinor, State, CreateTime, PerformTime, CancelTime, Reason, MerchantTx
			FROM PaymeMerchantTransactions@{FORCE_INDEX=Idx_PaymeMerchantTx_ByCreateTime}
			%s
			ORDER BY CreateTime ASC`, whereClause),
		Params: params,
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var out []paymeMerchantTx
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("payme merchant statement query: %w", err)
		}
		tx, err := scanPaymeMerchantRow(row)
		if err != nil {
			return nil, fmt.Errorf("payme merchant statement scan: %w", err)
		}
		out = append(out, tx)
	}
	if out == nil {
		out = []paymeMerchantTx{}
	}
	return out, nil
}
