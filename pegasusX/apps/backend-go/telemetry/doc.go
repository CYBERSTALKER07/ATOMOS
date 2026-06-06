// Package telemetry owns high-volume driver telemetry projections.
//
// Telemetry is intentionally cache-backed for live location reads; durable
// order, manifest, and payment state remain in their owning domain packages.
package telemetry
