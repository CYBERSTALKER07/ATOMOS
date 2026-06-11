// Package handoff owns delivery QR token minting, exposure rules, and validation.
// Tokens are persisted on Orders.DeliveryToken and surfaced to retailer clients as
// delivery_token while driver clients continue to use qr_token on the wire.
package handoff
