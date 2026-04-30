#\!/bin/bash
set -e
FILES=(
  "pegasus/tests/supplier/auth.spec.ts"
  "pegasus/tests/retailer/checkout.spec.ts"
  "pegasus/packages/validation/src/payment.ts"
  "pegasus/apps/backend-go/order/service.go"
  "pegasus/apps/backend-go/order/service_test.go"
  "pegasus/apps/backend-go/order/unified_checkout.go"
  "pegasus/apps/backend-go/order/checkout.go"
  "pegasus/apps/backend-go/cmd/seed/main.go"
  "pegasus/apps/backend-go/cmd/simulate/main.go"
  "pegasus/apps/backend-go/cmd/setup/main.go"
  "pegasus/apps/backend-go/paymentroutes/routes.go"
  "pegasus/apps/backend-go/payment/gateway_client.go"
  "pegasus/apps/backend-go/tests/e2e/core_flow_test.go"
  "pegasus/apps/backend-go/tests/stress/load_shedding.js"
  "pegasus/apps/backend-go/admin/audit_cron.go"
  "pegasus/apps/backend-go/schema/spanner.ddl"
  "pegasus/apps/backend-go/countrycfg/service.go"
  "pegasus/apps/backend-go/supplier/registration.go"
  "pegasus/apps/backend-go/kafka/kafka_integration_test.go"
  "pegasus/apps/backend-go/warehouse/payment_config.go"
  "pegasus/apps/backend-go/main.go"
  "pegasus/apps/backend-go/notifications/formatter_test.go"
  "pegasus/apps/backend-go/vault/vault_test.go"
  "pegasus/apps/backend-go/vault/capabilities.go"
  "pegasus/apps/backend-go/vault/vault.go"
  "pegasus/apps/backend-go/analytics/revenue.go"
  "pegasus/apps/admin-portal/app/setup/billing/page.tsx"
  "pegasus/apps/admin-portal/app/supplier/payment-config/page.tsx"
  "pegasus/apps/admin-portal/app/supplier/country-overrides/page.tsx"
  "pegasus/apps/admin-portal/app/treasury/chargebacks/page.tsx"
  "pegasus/apps/retailer-app-android/app/src/test/java/com/thelab/retailer/ui/screens/cart/CartUiStateTest.kt"
  "pegasus/apps/retailer-app-android/app/src/test/java/com/thelab/retailer/ui/screens/cart/CartUiStateComputedTest.kt"
  "pegasus/apps/retailer-app-android/app/src/test/java/com/thelab/retailer/data/api/RetailerWSMessageTest.kt"
  "pegasus/apps/retailer-app-android/app/src/main/java/com/thelab/retailer/ui/screens/cart/CartViewModel.kt"
  "pegasus/apps/retailer-app-ios/retailerapp/reatilerappTests/RetailerServiceTests.swift"
)

for file in "${FILES[@]}"; do
  if [ -f "$file" ]; then
    perl -pi -e 's/PAYME/GLOBAL_PAY/g' "$file"
    perl -pi -e 's/Payme/GlobalPay/g' "$file"
    perl -pi -e 's/payme/global_pay/g' "$file"
    
    perl -pi -e 's/CLICK/CASH/g' "$file"
    perl -pi -e 's/Click/Cash/g' "$file"
    perl -pi -e 's/click/cash/g' "$file"
  fi
done
