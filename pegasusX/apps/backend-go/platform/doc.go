// Package platform owns client version policy, safe-update deferral, and
// device-token registration for push fallback. It does not own payment or
// order lifecycle — it reads those aggregates to decide update deferral only.
package platform
