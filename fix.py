import re

with open("the-lab-monorepo/apps/backend-go/order/unified_checkout.go", "r") as f:
    text = f.read()

# Remove the trailing else { ... } from the backorders block
text = re.sub(
    r'\} else \{\n\s*// Emit STOCK_BACKORDERED events per supplier\n\s*now := time\.Now\(\)\.UTC\(\)\n\s*for _, bp := range backordersBySup \{.*?go s\.PublishEvent\(context\.Background\(\), kafkaEvents\.EventStockBackordered.*?\}\n\s*\}\n\s*\}',
    '}\n        }',
    text,
    flags=re.DOTALL
)

# Remove the Step 5b section which was "Kafka fan-out for fulfilled orders — AFTER commit"
text = re.sub(
    r'\s*// ── Step 5b: Kafka fan-out for fulfilled orders — AFTER commit ─────────────.*?go s\.PublishEvent\(context\.Background\(\), kafkaEvents\.EventUnifiedCheckoutCompleted, struct \{.*?\).*?Timestamp:\s*now,\n\s*\}\)',
    '\n\n        // ── CACHE INVALIDATION After Commit ────────────────────────────────────────\n        cache.Invalidate(ctx, cache.KeyRetailerOrders(req.RetailerID))',
    text,
    flags=re.DOTALL
)


with open("the-lab-monorepo/apps/backend-go/order/unified_checkout.go", "w") as f:
    f.write(text)
