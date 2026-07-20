// Package warehouseops is the stable facade for warehouse operations used by
// route packages and future service extraction.
//
// Historically this directory was empty while handlers lived under warehouse/.
// Enterprise layout keeps warehouse domain logic in package warehouse and
// re-exports the ops-facing contracts here so call sites can depend on a
// narrow boundary without importing the full warehouse surface.
//
// Usage:
//
//	import "github.com/pegasusx/pegasusx/apps/backend-go/warehouseops"
//
//	// Prefer warehouseops.Service / warehouseops.Repository aliases when wiring
//	// ops-only consumers (dashboards, dispatch, inbound).
package warehouseops
