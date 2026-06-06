import re

with open("apps/backend-go/order/external_payment.go", "r") as f:
    content = f.read()

content = content.replace(
    """if err := emitOrderStatusChanged(ctx, txn, orderRecord, previousStatus, "external_payment_cleared"); err != nil {""",
    """if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "external_payment_cleared",
		}); err != nil {"""
)

with open("apps/backend-go/order/external_payment.go", "w") as f:
    f.write(content)
