package dispatch

// Cleared ledger entry types that satisfy dispatch payment gate.
var clearedPaymentEntryTypes = []string{
	"WEBHOOK_PAID",
	"CASH_COLLECTED",
	"SETTLEMENT_CREDIT",
}

// Cleared payment-session statuses that satisfy dispatch payment gate.
var clearedPaymentSessionStatuses = []string{
	"PAID",
	"CAPTURED",
	"SETTLED",
	"SUCCESS",
	"AUTHORIZED",
}

const dispatchableEligibilitySQL = `
	          AND o.Status = 'PENDING'
	          AND o.ConfirmationStatus = 'CONFIRMED'
	          AND (o.DriverId IS NULL OR o.DriverId = '')
	          AND (
	            EXISTS (
	              SELECT 1
	              FROM PaymentLedgerEntries ple
	              WHERE ple.OrderId = o.OrderId
	                AND ple.EntryType IN UNNEST(@clearedEntryTypes)
	            )
	            OR EXISTS (
	              SELECT 1
	              FROM PaymentSessions ps
	              WHERE ps.OrderId = o.OrderId
	                AND ps.Status IN UNNEST(@clearedSessionStatuses)
	            )
	          )`
