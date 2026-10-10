package inventory

import "time"

// Location represents a physical storage bin within a warehouse.
type Location struct {
	LocationID   string    `json:"location_id" spanner:"LocationId"`
	WarehouseID  string    `json:"warehouse_id" spanner:"WarehouseId"`
	SupplierID   string    `json:"supplier_id" spanner:"SupplierId"`
	Aisle        string    `json:"aisle" spanner:"Aisle"`
	Rack         string    `json:"rack" spanner:"Rack"`
	Shelf        string    `json:"shelf" spanner:"Shelf"`
	Bin          string    `json:"bin" spanner:"Bin"`
	Zone         string    `json:"zone" spanner:"Zone"`
	LocationType string    `json:"location_type" spanner:"LocationType"`
	IsActive     bool      `json:"is_active" spanner:"IsActive"`
	CreatedAt    time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt    time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// StockLot represents an identifiable batch of a SKU with expiration tracking.
type StockLot struct {
	LotID             string     `json:"lot_id" spanner:"LotId"`
	SupplierID        string     `json:"supplier_id" spanner:"SupplierId"`
	WarehouseID       string     `json:"warehouse_id" spanner:"WarehouseId"`
	ProductID         string     `json:"product_id" spanner:"ProductId"`
	LocationID        string     `json:"location_id" spanner:"LocationId"`
	LotCode           string     `json:"lot_code" spanner:"LotCode"`
	ManufacturedDate  *time.Time `json:"manufactured_date,omitempty" spanner:"ManufacturedDate"`
	ExpiryDate        time.Time  `json:"expiry_date" spanner:"ExpiryDate"`
	QuantityOnHand    int64      `json:"quantity_on_hand" spanner:"QuantityOnHand"`
	QuantityAllocated int64      `json:"quantity_allocated" spanner:"QuantityAllocated"`
	Status            string     `json:"status" spanner:"Status"` // AVAILABLE, QUARANTINE, EXPIRED, DEPLETED
	CreatedAt         time.Time  `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt         time.Time  `json:"updated_at" spanner:"UpdatedAt"`
}

// Available returns the unallocated stock available in this lot.
func (l StockLot) Available() int64 {
	avail := l.QuantityOnHand - l.QuantityAllocated
	if avail < 0 {
		return 0
	}
	return avail
}

// StockLotLocation joins lot data with physical coordinates for pick routing.
type StockLotLocation struct {
	Lot      StockLot `json:"lot"`
	Location Location `json:"location"`
}

// WavePickRequest defines the input to a FEFO wave pick allocation.
type WavePickRequest struct {
	WaveID       string         `json:"wave_id"`
	WarehouseID  string         `json:"warehouse_id"`
	SupplierID   string         `json:"supplier_id"`
	OrderIDs     []string       `json:"order_ids,omitempty"`
	Items        []WavePickItem `json:"items"`
	AllowPartial bool           `json:"allow_partial"`
}

// WavePickItem specifies an item requirement in a wave.
type WavePickItem struct {
	OrderID   string `json:"order_id,omitempty"`
	LineID    string `json:"line_id,omitempty"`
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

// PickInstruction gives detailed pick guidance to warehouse personnel.
type PickInstruction struct {
	PickTaskID   string    `json:"pick_task_id"`
	OrderID      string    `json:"order_id,omitempty"`
	LineID       string    `json:"line_id,omitempty"`
	ProductID    string    `json:"product_id"`
	LotID        string    `json:"lot_id"`
	LotCode      string    `json:"lot_code"`
	ExpiryDate   time.Time `json:"expiry_date"`
	LocationID   string    `json:"location_id"`
	Aisle        string    `json:"aisle"`
	Rack         string    `json:"rack"`
	Shelf        string    `json:"shelf"`
	Bin          string    `json:"bin"`
	Quantity     int64     `json:"quantity"`
	PickSequence int       `json:"pick_sequence"`
}

// WavePickResult represents the completed wave pick allocation plan.
type WavePickResult struct {
	WaveID       string            `json:"wave_id"`
	WarehouseID  string            `json:"warehouse_id"`
	Instructions []PickInstruction `json:"instructions"`
	Shortfalls   []WaveShortfall   `json:"shortfalls,omitempty"`
	TotalLines   int               `json:"total_lines"`
	TotalUnits   int64             `json:"total_units"`
}

// WaveShortfall details unfulfilled quantities when partial allocation occurs.
type WaveShortfall struct {
	ProductID string `json:"product_id"`
	Requested int64  `json:"requested"`
	Allocated int64  `json:"allocated"`
	Shortfall int64  `json:"shortfall"`
}
