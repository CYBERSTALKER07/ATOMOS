with open('pegasus/apps/backend-go/kafka/billing_tier_worker.go', 'r') as f:
    content = f.read()

content = content.replace("pool.Start(ctx)", "if err := pool.Run(ctx); err != nil && ctx.Err() == nil { slog.ErrorContext(ctx, \"billing_tier_worker: pool exited\", \"err\", err) }")

content = content.replace("""	var event struct {
		Event struct {
			SupplierID string `json:"supplier_id"`
			Amount     int64  `json:"amount"`
		} `json:"event"`
	}
	
""", "")

with open('pegasus/apps/backend-go/kafka/billing_tier_worker.go', 'w') as f:
    f.write(content)
