
#### Phase 4.1: Retailer Shelf Intelligence & Promotions Lifecycle
- **Schema Changes**: Added `RetailerShelfAlerts`, `SupplierPromotionCampaigns`, and `RetailerPromotionEnrollments` to `spanner.ddl`.
- **Shelf Intelligence**: Created `retailer/shelf_intelligence.go` with core logic (`CheckAndGenerateOOSAlerts`, `ResolveShelfAlert`) to detect and alert on low capacity thresholds.
- **Promotions Lifecycle**: Created `promotion/lifecycle.go` to support creating Campaigns with `BudgetLimitMinor`, tracking budget usage, and letting Retailers opt-in via enrollments.
- **Evaluator Integration**: Updated `promotion/service.go` (`QuoteCheckout`) to invoke `FilterEligibleCampaignPromotions` so only active and enrolled campaigns apply their promotions.
- **Route Handlers**: Added standard endpoints in `promotionroutes/routes.go` and `retailerroutes/routes.go`.
- **Types**: Appended interfaces to `packages/types/index.ts`.
