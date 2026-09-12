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
