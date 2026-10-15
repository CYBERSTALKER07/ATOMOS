package order

import "github.com/pegasusx/pegasusx/apps/backend-go/orderstage"

// CustomerStage represents the high-level, simplified progress stage exposed to
// retailers, drivers, and consumer-facing mobile clients.
type CustomerStage = orderstage.Stage

const (
	CustomerStagePlaced                  = orderstage.Placed
	CustomerStageProcessing              = orderstage.Processing
	CustomerStageOutForDelivery          = orderstage.OutForDelivery
	CustomerStageDelivered               = orderstage.Delivered
	CustomerStageCancelled               = orderstage.Cancelled
	CustomerStageExceptionActionRequired = orderstage.ExceptionActionRequired
)

// ToCustomerStage projects any canonical backend order status into a customer stage.
// Mapping is pinned by contracts/order-customer-stage.json via package orderstage.
func ToCustomerStage(status Status) CustomerStage {
	return orderstage.FromString(string(status))
}

// CustomerStageProgressPercent returns the UI progress bar percentage (0-100).
func CustomerStageProgressPercent(stage CustomerStage) int {
	return orderstage.ProgressPercent(stage)
}
