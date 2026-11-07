package retailer

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type ReturnRequest struct {
	RequestID  string    `json:"request_id"`
	RetailerID string    `json:"retailer_id"`
	OrderID    string    `json:"order_id"`
	Status     string    `json:"status"`
	LinesJSON  string    `json:"lines_json"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

func (r *SpannerRepository) InsertReturnRequest(ctx context.Context, req ReturnRequest) error {
	m := spanner.Insert("RetailerReturnRequests",
		[]string{"RequestId", "RetailerId", "OrderId", "Status", "LinesJson", "Reason", "CreatedAt", "UpdatedAt"},
		[]interface{}{req.RequestID, req.RetailerID, req.OrderID, req.Status, spanner.NullJSON{Value: req.LinesJSON, Valid: true}, req.Reason, spanner.CommitTimestamp, nil},
	)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (r *SpannerRepository) ListReturnRequestsByRetailer(ctx context.Context, retailerID string) ([]ReturnRequest, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RequestId, RetailerId, OrderId, Status, CAST(LinesJson AS STRING), Reason, CreatedAt, UpdatedAt 
		      FROM RetailerReturnRequests 
		      WHERE RetailerId = @retId ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"retId": retailerID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var reqs []ReturnRequest
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var req ReturnRequest
		var updAt spanner.NullTime
		var lines spanner.NullString
		if err := row.Columns(
			&req.RequestID, &req.RetailerID, &req.OrderID, &req.Status, &lines, &req.Reason,
			&req.CreatedAt, &updAt,
		); err != nil {
			return nil, err
		}
		if updAt.Valid {
			req.UpdatedAt = updAt.Time
		}
		if lines.Valid {
			req.LinesJSON = lines.StringVal
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}
