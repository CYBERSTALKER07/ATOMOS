package order

import (
	"fmt"
	"strings"

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

// CellInCoverage is true when the retailer H3 cell is the stored cell or a child of it.
func CellInCoverage(retailerCell string, stored []string) bool {
	retailerCell = strings.TrimSpace(retailerCell)
	if retailerCell == "" || len(stored) == 0 {
		return false
	}
	r := h3.Cell(h3.IndexFromString(retailerCell))
	if !r.IsValid() {
		for _, s := range stored {
			if strings.TrimSpace(s) == retailerCell {
				return true
			}
		}
		return false
	}
	for _, raw := range stored {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if s == retailerCell {
			return true
		}
		sc := h3.Cell(h3.IndexFromString(s))
		if !sc.IsValid() {
			continue
		}
		if r == sc {
			return true
		}
		res := sc.Resolution()
		if res < 0 || res > 15 {
			continue
		}
		parent, err := r.Parent(res)
		if err == nil && parent == sc {
			return true
		}
	}
	return false
}

// WarehouseCoversRetailer is the locked hybrid:
// cells set → H3 membership; no cells → whole warehouse country (closest is caller's job).
func WarehouseCoversRetailer(warehouseCountry string, coverageCells []string, retailerCountry, retailerCell string) bool {
	if len(coverageCells) > 0 {
		return CellInCoverage(retailerCell, coverageCells)
	}
	whC := strings.ToUpper(strings.TrimSpace(warehouseCountry))
	rtC := strings.ToUpper(strings.TrimSpace(retailerCountry))
	if whC == "" || rtC == "" {
		return true
	}
	return whC == rtC
}
