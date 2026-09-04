import re

with open("apps/backend-go/order/settlement_hardening.go", "r") as f:
    content = f.read()

pattern = re.compile(r'_, sliceM, err := payout\.GenerateSettlementSlice\(ctx, s\.commissionResolver, s\.newID\(\), leg\.OrderID, supplierID, leg\.LegID, amount, currency, leg\.CapturedAt\.Time\)')

replacement = r"""
		sliceTime := leg.CapturedAt.Time
		if !leg.CapturedAt.Valid || sliceTime.IsZero() {
			if !leg.CreatedAt.IsZero() {
				sliceTime = leg.CreatedAt
			} else {
				sliceTime = time.Now().UTC()
			}
		}

		_, sliceM, err := payout.GenerateSettlementSlice(ctx, s.commissionResolver, s.newID(), leg.OrderID, supplierID, leg.LegID, amount, currency, sliceTime)"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/order/settlement_hardening.go", "w") as f:
    f.write(content)
