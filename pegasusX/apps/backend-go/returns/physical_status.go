package returns

// Physical status lifecycle for SupplierReturns rows.
const (
	PhysicalPending    = "PENDING"
	PhysicalOnTruck    = "ON_TRUCK"
	PhysicalArrived    = "ARRIVED"
	PhysicalReceiving  = "RECEIVING"
	PhysicalRestocked  = "RESTOCKED"
	PhysicalWrittenOff = "WRITTEN_OFF"
)

// Financial resolution statuses (existing SupplierReturns.Status column).
const (
	FinancialPending         = "PENDING"
	FinancialReturnedToStock = "RETURNED_TO_STOCK"
	FinancialWriteOff        = "WRITE_OFF"
)

// Disposition values for gate confirm.
const (
	DispositionRestock  = "RESTOCK"
	DispositionWriteOff = "WRITE_OFF"
)

// SessionStatusOpen marks an active bulk-receive session.
const SessionStatusOpen = "OPEN"

// SessionStatusCompleted marks a closed receive session.
const SessionStatusCompleted = "COMPLETED"
