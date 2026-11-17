package dispatch

// dispatchableEligibilitySQL selects orders ready for warehouse dispatch.
// Payment is collected at delivery (offload), not as a pre-dispatch gate.
// Migrated historical and held MIGRATION_OPEN orders are strictly excluded.
const dispatchableEligibilitySQL = `
	          AND o.Status = 'PENDING'
	          AND o.Status != 'MIGRATION_OPEN'
	          AND COALESCE(o.OrderSource, '') != 'MIGRATION'
	          AND o.ConfirmationStatus IN ('CONFIRMED', 'AUTO_CONFIRMED')
	          AND (o.DriverId IS NULL OR o.DriverId = '')`
