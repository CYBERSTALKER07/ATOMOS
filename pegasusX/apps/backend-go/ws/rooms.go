package ws

import "strings"

// SupplierPromoRoom is the retailer-hub room for live promotion updates from a supplier.
func SupplierPromoRoom(supplierID string) string {
	return "supplier-promo:" + strings.TrimSpace(supplierID)
}

// PlatformAdminRoom is the shared governance room for PLATFORM_ADMIN consoles.
func PlatformAdminRoom() string {
	return "platform-admin"
}

// Live fleet telemetry for Dispatcher God-Mode
func SupplierTelemetryRoom(supplierID string) string {
    return "telemetry:supplier:" + strings.TrimSpace(supplierID)
}

// Specific driver command room for dynamic reroutes
func DriverCommandRoom(driverID string) string {
    return "cmd:driver:" + strings.TrimSpace(driverID)
}

// Retailer Live ETA room
func RetailerTrackingRoom(retailerID string) string {
    return "tracking:retailer:" + strings.TrimSpace(retailerID)
}
