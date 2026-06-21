package proximity

// Delivery approach radius: retailer QR auto-open, driver scan enable, telemetry DRIVER_APPROACHING.
const (
	DeliveryApproachRadiusKm = 0.5
	DeliveryApproachRadiusM  = 500.0
)

// WithinDeliveryApproach reports whether distKm is inside the delivery approach geofence.
func WithinDeliveryApproach(distKm float64) bool {
	return distKm < DeliveryApproachRadiusKm
}
