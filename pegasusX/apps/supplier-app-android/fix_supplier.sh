#!/bin/bash
sed -i '' 's/val resp = api.issuePaymentBypass(mapOf("order_id" to orderId))/val resp = ops.issuePaymentBypass(com.pegasusx.supplier.data.model.PaymentBypassRequest(orderId = orderId), java.util.UUID.randomUUID().toString())/g' app/src/main/java/com/pegasusx/supplier/ui/screens/orders/OrderDetailScreen.kt
