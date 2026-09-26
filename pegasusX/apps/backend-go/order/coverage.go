package order

import (
	"fmt"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/uber/h3-go/v4"
)

const coverageCityDiskK = 4

func coverageH3Cell(lat, lng float64) string {
	cell, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, orderH3Resolution)
	if err != nil {
		return fmt.Sprintf("%.4f,%.4f", lat, lng)
	}
	return cell.String()
}

// CoverageCity is a supplier-selected city used to generate compacted H3 cells.
type CoverageCity struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// CellsForCity returns compacted H3 cells (dispatch res 7 disk) covering a city point.
func CellsForCity(lat, lng float64) []string {
	cellStr := coverageH3Cell(lat, lng)
	cell := h3.Cell(h3.IndexFromString(cellStr))
	if !cell.IsValid() {
		if cellStr != "" {
			return []string{cellStr}
		}
		return nil
	}
	disk, err := h3.GridDisk(cell, coverageCityDiskK)
	if err != nil || len(disk) == 0 {
		return []string{cellStr}
	}
	compacted, err := proximity.CompactCells(disk)
	if err != nil || len(compacted) == 0 {
		out := make([]string, 0, len(disk))
		for _, c := range disk {
			out = append(out, c.String())
		}
		return out
	}
	out := make([]string, 0, len(compacted))
	for _, c := range compacted {
		out = append(out, c.String())
	}
	return out
}

// CellInCoverage is the GS-L0 membership helper. Implementation lives on the engine.
func CellInCoverage(retailerCell string, stored []string) bool {
	return proximity.CellInCoverage(retailerCell, stored)
}

// WarehouseCoversRetailer is the locked hybrid (GS-L0). One implementation: proximity.CoversRetailer.
func WarehouseCoversRetailer(warehouseCountry string, coverageCells []string, retailerCountry, retailerCell string) bool {
	return proximity.CoversRetailer(warehouseCountry, coverageCells, retailerCountry, retailerCell)
}
