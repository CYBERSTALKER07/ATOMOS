#!/bin/bash
sed -i '' 's/error = "Token: ${resp.body()!!\["bypass_token"\]}"/error = "Token: ${resp.body()!!.bypassToken}"/g' app/src/main/java/com/pegasusx/supplier/ui/screens/orders/OrderDetailScreen.kt
