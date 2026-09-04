#!/bin/bash
sed -i '' 's/OrderMutationAction.Reject -> "Cancel order?"/OrderMutationAction.Reject -> "Cancel order?"\n            OrderMutationAction.PaymentBypass -> "Bypass Payment?"/g' app/src/main/java/com/pegasusx/warehouse/ui/screens/orders/OrderDetailScreen.kt
sed -i '' 's/OrderMutationAction.Reject -> "Cancel order"/OrderMutationAction.Reject -> "Cancel order"\n            OrderMutationAction.PaymentBypass -> "Bypass"/g' app/src/main/java/com/pegasusx/warehouse/ui/screens/orders/OrderDetailScreen.kt
