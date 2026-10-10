package inventory

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AllocateWaveFEFOPure executes deterministic in-memory FEFO (First Expired, First Out)
// wave pick allocation against available, unexpired stock lots.
func AllocateWaveFEFOPure(
	now time.Time,
	req WavePickRequest,
	lotsByProduct map[string][]StockLotLocation,
) (*WavePickResult, error) {
	if strings.TrimSpace(req.WarehouseID) == "" {
		return nil, fmt.Errorf("warehouse_id required for wave pick")
	}
	if len(req.Items) == 0 {
		return &WavePickResult{
			WaveID:       req.WaveID,
			WarehouseID:  req.WarehouseID,
			Instructions: []PickInstruction{},
			Shortfalls:   nil,
			TotalLines:   0,
			TotalUnits:   0,
		}, nil
	}

	// Make deep copies of candidate lots and filter/sort by FEFO
	lotState := make(map[string][]*StockLotLocation)
	for pid, lotList := range lotsByProduct {
		var validLots []*StockLotLocation
		for i := range lotList {
			l := lotList[i]
			// Only consider AVAILABLE, unexpired lots with available stock
			if l.Lot.Status == "AVAILABLE" && l.Lot.ExpiryDate.After(now) && l.Lot.Available() > 0 {
				lotCopy := l
				validLots = append(validLots, &lotCopy)
			}
		}

		// Sort by FEFO: ExpiryDate ASC -> CreatedAt ASC -> LotID ASC
		sort.Slice(validLots, func(i, j int) bool {
			if !validLots[i].Lot.ExpiryDate.Equal(validLots[j].Lot.ExpiryDate) {
				return validLots[i].Lot.ExpiryDate.Before(validLots[j].Lot.ExpiryDate)
			}
			if !validLots[i].Lot.CreatedAt.Equal(validLots[j].Lot.CreatedAt) {
				return validLots[i].Lot.CreatedAt.Before(validLots[j].Lot.CreatedAt)
			}
			return validLots[i].Lot.LotID < validLots[j].Lot.LotID
		})

		lotState[pid] = validLots
	}

	var instructions []PickInstruction
	shortfallMap := make(map[string]*WaveShortfall)
	var shortfallOrder []string
	var totalUnits int64

	for _, itm := range req.Items {
		pid := strings.TrimSpace(itm.ProductID)
		if pid == "" || itm.Quantity <= 0 {
			continue
		}

		needed := itm.Quantity
		allocatedForLine := int64(0)
		candidateLots := lotState[pid]

		for _, lotLoc := range candidateLots {
			avail := lotLoc.Lot.Available()
			if avail <= 0 {
				continue
			}

			take := needed - allocatedForLine
			if take > avail {
				take = avail
			}

			instructions = append(instructions, PickInstruction{
				PickTaskID: uuid.NewString(),
				OrderID:    itm.OrderID,
				LineID:     itm.LineID,
				ProductID:  pid,
				LotID:      lotLoc.Lot.LotID,
				LotCode:    lotLoc.Lot.LotCode,
				ExpiryDate: lotLoc.Lot.ExpiryDate,
				LocationID: lotLoc.Location.LocationID,
				Aisle:      lotLoc.Location.Aisle,
				Rack:       lotLoc.Location.Rack,
				Shelf:      lotLoc.Location.Shelf,
				Bin:        lotLoc.Location.Bin,
				Quantity:   take,
			})

			lotLoc.Lot.QuantityAllocated += take
			allocatedForLine += take
			totalUnits += take

			if allocatedForLine >= needed {
				break
			}
		}

		if allocatedForLine < needed {
			short := needed - allocatedForLine
			if !req.AllowPartial {
				return nil, fmt.Errorf("insufficient FEFO stock for product %s (needed %d, allocated %d)", pid, needed, allocatedForLine)
			}

			if existing, ok := shortfallMap[pid]; ok {
				existing.Requested += needed
				existing.Allocated += allocatedForLine
				existing.Shortfall += short
			} else {
				shortfallMap[pid] = &WaveShortfall{
					ProductID: pid,
					Requested: needed,
					Allocated: allocatedForLine,
					Shortfall: short,
				}
				shortfallOrder = append(shortfallOrder, pid)
			}
		}
	}

	// Sort instructions by serpentine warehouse traversal: Aisle ASC -> Rack ASC -> Shelf ASC -> Bin ASC
	sort.Slice(instructions, func(i, j int) bool {
		if instructions[i].Aisle != instructions[j].Aisle {
			return instructions[i].Aisle < instructions[j].Aisle
		}
		if instructions[i].Rack != instructions[j].Rack {
			return instructions[i].Rack < instructions[j].Rack
		}
		if instructions[i].Shelf != instructions[j].Shelf {
			return instructions[i].Shelf < instructions[j].Shelf
		}
		if instructions[i].Bin != instructions[j].Bin {
			return instructions[i].Bin < instructions[j].Bin
		}
		if instructions[i].ProductID != instructions[j].ProductID {
			return instructions[i].ProductID < instructions[j].ProductID
		}
		if !instructions[i].ExpiryDate.Equal(instructions[j].ExpiryDate) {
			return instructions[i].ExpiryDate.Before(instructions[j].ExpiryDate)
		}
		return instructions[i].LotID < instructions[j].LotID
	})

	// Assign sequential 1-based pick sequence numbers
	for idx := range instructions {
		instructions[idx].PickSequence = idx + 1
	}

	var shortfalls []WaveShortfall
	for _, pid := range shortfallOrder {
		shortfalls = append(shortfalls, *shortfallMap[pid])
	}

	return &WavePickResult{
		WaveID:       req.WaveID,
		WarehouseID:  req.WarehouseID,
		Instructions: instructions,
		Shortfalls:   shortfalls,
		TotalLines:   len(instructions),
		TotalUnits:   totalUnits,
	}, nil
}
