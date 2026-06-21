package dispatch

// dispatchableEligibilitySQL selects orders ready for warehouse dispatch.
// Payment is collected at delivery (offload), not as a pre-dispatch gate.
const dispatchableEligibilitySQL = `
	          AND o.Status = 'PENDING'
	          AND o.ConfirmationStatus IN ('CONFIRMED', 'AUTO_CONFIRMED')
	          AND (o.DriverId IS NULL OR o.DriverId = '')`
