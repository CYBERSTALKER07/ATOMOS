package admin

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/robfig/cron/v3"
	// "github.com/lab-industries/backend/packages/gateways" // Your GlobalPay/Cash API clients
)

// StartReconciliationCron boots the background audit worker
func StartReconciliationCron(ctx context.Context, client *spanner.Client) {
	// Initialize cron with Tashkent timezone (UTC+5)
	tashkentLoc, _ := time.LoadLocation("Asia/Tashkent")
	c := cron.New(cron.WithLocation(tashkentLoc))

	// Execute at the top of every hour
	_, err := c.AddFunc("0 * * * *", func() {
		fmt.Println("[AUDIT_CRON] Initiating automated ledger reconciliation...")
		runAuditCycle(ctx, client)
	})

	if err != nil {
		fmt.Printf("[CRITICAL] Failed to arm audit cron: %v\n", err)
		return
	}

	c.Start()
}

func runAuditCycle(ctx context.Context, client *spanner.Client) {
	// 1. Pull all orders marked 'FUNDS_SECURED' in the last 24 hours
	// (Simulated query for brevity)
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, RetailerId, Amount, PaymentGateway 
		      FROM Orders 
		      WHERE State = 'COMPLETED' 
		      AND CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)`,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var mutations []*spanner.Mutation

	for {
		row, err := iter.Next()
		if err != nil {
			break // EOF or query fault
		}

		var orderId, retailerId, gateway string
		var expectedAmount int64
		row.Columns(&orderId, &retailerId, &expectedAmount, &gateway)

		// 2. Query the External Gateway for the exact captured amount
		var actualAmount int64
		// if gateway == "GLOBAL_PAY" {
		// 	actualAmount = gateways.GlobalPayClient.CheckTransaction(orderId)
		// } else {
		// 	actualAmount = gateways.CashClient.CheckTransaction(orderId)
		// }

		// *Simulation: Force a delta for testing if expectedAmount > 5,000,000*
		actualAmount = expectedAmount
		if expectedAmount > 5000000 {
			actualAmount = expectedAmount - 200000 // 200k simulated missing
		}

		// 3. Detect Anomaly
		if actualAmount != expectedAmount {
			fmt.Printf("[ANOMALY_DETECTED] Order %s: Spanner expected %d, Gateway reports %d\n", orderId, expectedAmount, actualAmount)

			m := spanner.InsertOrUpdate("LedgerAnomalies",
				[]string{"OrderId", "RetailerId", "SpannerAmount", "GatewayAmount", "GatewayProvider", "Status", "DetectedAt"},
				[]interface{}{orderId, retailerId, expectedAmount, actualAmount, gateway, "DELTA", spanner.CommitTimestamp},
			)
			mutations = append(mutations, m)
		}
	}

	// 4. Batch commit all newly discovered anomalies to the ledger
	if len(mutations) > 0 {
		_, err := client.Apply(ctx, mutations)
		if err != nil {
			fmt.Printf("[AUDIT_CRON_FAULT] Failed to write anomalies: %v\n", err)
		} else {
			fmt.Printf("[AUDIT_CRON] Successfully logged %d discrepancies.\n", len(mutations))
		}
	} else {
		fmt.Println("[AUDIT_CRON] Cycle complete. Zero deltas detected.")
	}
}
