package predictivepush

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// DemandEvent represents a predicted future order from a retailer.
type DemandEvent struct {
	RetailerId   string
	ProductId    string
	SupplierId   string
	TargetDate   time.Time
	Quantity     int64
	Confidence   float64
	PatternDays  int
}

// Analyzer analyzes historical orders to find consistent ordering patterns.
type Analyzer struct {
	spannerClient *spanner.Client
	weeksToAnalyze int
	confidenceThreshold float64
}

// NewAnalyzer creates a new Predictive Push Analyzer.
func NewAnalyzer(client *spanner.Client) *Analyzer {
	return &Analyzer{
		spannerClient: client,
		weeksToAnalyze: 8,
		confidenceThreshold: 0.75, // Need 75% consistency to trigger
	}
}

// Analyze historical ordering patterns for all retailers.
func (a *Analyzer) Analyze(ctx context.Context, targetDay time.Time) ([]*DemandEvent, error) {
	// Look back N weeks from today
	startTime := targetDay.AddDate(0, 0, -a.weeksToAnalyze*7)
	
	// Query historical orders and their line items
	// Group by Retailer, Product, Supplier, and Day of Week
	// For simplicity in this v1 agent, we use a single query that calculates the frequency.
	stmt := spanner.Statement{
		SQL: `
		SELECT 
			o.RetailerId,
			o.SupplierId,
			ci.ProductId,
			EXTRACT(DAYOFWEEK FROM o.CreatedAt) as OrderDayOfWeek,
			COUNT(*) as OrderCount,
			AVG(ci.Quantity) as AvgQuantity
		FROM Orders o
		JOIN CartItems ci ON o.OrderId = ci.OrderId
		WHERE o.CreatedAt >= @startTime AND o.CreatedAt <= @endTime
		  AND o.Status IN ('COMPLETED', 'DELIVERED', 'SHIPPED', 'PROCESSING')
		GROUP BY o.RetailerId, o.SupplierId, ci.ProductId, EXTRACT(DAYOFWEEK FROM o.CreatedAt)
		HAVING COUNT(*) >= @minOrders
		`,
		Params: map[string]interface{}{
			"startTime": startTime,
			"endTime":   targetDay,
			"minOrders": 4, // Must have ordered at least 4 times in the past 8 weeks on this day
		},
	}

	iter := a.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var events []*DemandEvent
	targetDayOfWeek := int(targetDay.Weekday()) + 1 // Spanner DAYOFWEEK is 1-7 (Sunday=1)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var retailerId, supplierId, productId string
		var orderDayOfWeek int64
		var orderCount int64
		var avgQuantity float64

		if err := row.Columns(&retailerId, &supplierId, &productId, &orderDayOfWeek, &orderCount, &avgQuantity); err != nil {
			return nil, err
		}

		// Calculate confidence based on weeks to analyze
		confidence := float64(orderCount) / float64(a.weeksToAnalyze)

		if confidence >= a.confidenceThreshold && int(orderDayOfWeek) == targetDayOfWeek {
			events = append(events, &DemandEvent{
				RetailerId:  retailerId,
				ProductId:   productId,
				SupplierId:  supplierId,
				TargetDate:  targetDay,
				Quantity:    int64(avgQuantity),
				Confidence:  confidence,
				PatternDays: int(orderCount),
			})
		}
	}

	return events, nil
}
