#\!/bin/bash
set -e
FILES=(
  "the-lab-monorepo/tests/supplier/auth.spec.ts"
  "the-lab-monorepo/tests/retailer/checkout.spec.ts"
  "the-lab-monorepo/packages/validation/src/payment.ts"
  "the-lab-monorepo/apps/backend-go/order/service.go"
  "the-lab-monorepo/apps/backend-go/order/service_test.go"
  "the-lab-monorepo/apps/backend-go/order/unified_checkout.go"
  "the-lab-monorepo/apps/backend-go/order/checkout.go"
  "the-lab-monorepo/apps/backend-go/cmd/seed/main.go"
  "the-lab-monorepo/apps/backend-go/cmd/simulate/main.go"
  "the-lab-monorepo/apps/backend-go/cmd/setup/main.go"
  "the-lab-monorepo/apps/backend-go/paymentroutes/routes.go"
  "the-lab-monorepo/apps/backend-go/payment/gateway_client.go"
  "the-lab-monorepo/apps/backend-go/tests/e2e/core_flow_test.go"
  "the-lab-monorepo/apps/backend-go/tests/stress/load_shedding.js"
  "the-lab-monorepo/apps/backend-go/admin/audit_cron.go"
  "the-lab-monorepo/apps/backend-go/schema/spanner.ddl"
  "the-lab-monorepo/apps/backend-go/countrycfg/service.go"
  "the-lab-monorepo/apps/backend-go/supplier/registration.go"
  "the-lab-monorepo/apps/backend-go/kafka/kafka_integration_test.go"
  "the-lab-monorepo/apps/backend-go/warehouse/payment_config.go"
  "the-lab-monorepo/apps/backend-go/main.go"
  "the-lab-monorepo/apps/backend-go/notifications/formatter_test.go"
  "the-lab-monorepo/apps/backend-go/vault/vault_test.go"
  "the-lab-monorepo/apps/backend-go/vault/capabilities.go"
  "the-lab-monorepo/apps/backend-go/vault/vault.go"
  "the-lab-monorepo/apps/backend-go/analytics/revenue.go"
  "the-lab-monorepo/apps/admin-portal/app/setup/billing/page.tsx"
  "the-lab-monorepo/apps/admin-portal/app/supplier/payment-config/page.tsx"
  "the-lab-monorepo/apps/admin-portal/app/supplier/country-overrides/page.tsx"
  "the-lab-monorepo/apps/admin-portal/app/treasury/chargebacks/page.tsx"
  "the-lab-monorepo/apps/retailer-app-android/app/src/test/java/com/thelab/retailer/ui/screens/cart/CartUiStateTest.kt"
  "the-lab-monorepo/apps/retailer-app-android/app/src/test/java/com/thelab/retailer/ui/screens/cart/CartUiStateComputedTest.kt"
  "the-lab-monorepo/apps/retailer-app-android/app/src/test/java/com/thelab/retailer/data/api/RetailerWSMessageTest.kt"
  "the-lab-monorepo/apps/retailer-app-android/app/src/main/java/com/thelab/retailer/ui/screens/cart/CartViewModel.kt"
  "the-lab-monorepo/apps/retailer-app-ios/retailerapp/reatilerappTests/RetailerServiceTests.swift"
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
