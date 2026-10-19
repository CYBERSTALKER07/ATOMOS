package edi

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// BuildDESADV encodes a despatch advice for LOADED / IN_TRANSIT.
// When ShipUnits are present, emits CPS/PAC/GIN packing hierarchy with SSCC (BJ).
func BuildDESADV(o OrderSnapshot, externalDocID string) string {
	if externalDocID == "" {
		externalDocID = o.OrderID + ":" + o.Status
	}
	ts := time.Now().UTC().Format("060102:1504")
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", o.RetailerID, ts, externalDocID}},
		{Tag: "UNH", Elements: []string{"1", "DESADV:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"351", externalDocID}},
		{Tag: "RFF", Elements: []string{"ON:" + o.OrderID}},
		{Tag: "NAD", Elements: []string{"BY", o.RetailerID}},
		{Tag: "NAD", Elements: []string{"SU", o.SupplierID}},
		{Tag: "FTX", Elements: []string{"AAI", "", "", o.Status}},
	}
	segs = append(segs, desadvPackingSegments(o)...)
	for i, ln := range o.Lines {
		segs = append(segs,
			Segment{Tag: "LIN", Elements: []string{strconv.Itoa(i + 1), "", ln.SKU + ":SA"}},
			Segment{Tag: "QTY", Elements: []string{fmt.Sprintf("12:%d", ln.Qty)}}, // despatched
		)
	}
	segs = append(segs,
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 1), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", externalDocID}},
	)
	return WriteUNAFile(segs)
}

// desadvPackingSegments builds CPS/PAC/GIN for ManifestShipUnits SSCCs.
// Dialect (EDI-lite, not certified EDIFACT):
//
//	CPS+1'                 root packing node
//	PAC+n++CT'             n ship units
//	CPS+{i}+1'             child of root
//	PAC+1++CT'
//	GIN+BJ+{sscc}'         BJ = Serial Shipping Container Code
func desadvPackingSegments(o OrderSnapshot) []Segment {
	units := filterShipUnits(o.ShipUnits)
	if len(units) == 0 {
		return nil
	}
	out := make([]Segment, 0, 3+len(units)*3)
	if mid := firstManifestID(o); mid != "" {
		out = append(out, Segment{Tag: "RFF", Elements: []string{"PK:" + mid}})
	}
	out = append(out,
		Segment{Tag: "CPS", Elements: []string{"1"}},
		Segment{Tag: "PAC", Elements: []string{strconv.Itoa(len(units)), "", "CT"}},
	)
	for i, u := range units {
		child := strconv.Itoa(i + 2)
		out = append(out,
			Segment{Tag: "CPS", Elements: []string{child, "1"}},
			Segment{Tag: "PAC", Elements: []string{"1", "", "CT"}},
			Segment{Tag: "GIN", Elements: []string{"BJ", strings.TrimSpace(u.SSCC)}},
		)
		if gtin := strings.TrimSpace(u.GTIN); gtin != "" {
			out = append(out, Segment{Tag: "GIN", Elements: []string{"BN", gtin}})
		}
	}
	return out
}

func firstManifestID(o OrderSnapshot) string {
	if mid := strings.TrimSpace(o.ManifestID); mid != "" {
		return mid
	}
	for _, u := range o.ShipUnits {
		if mid := strings.TrimSpace(u.ManifestID); mid != "" {
			return mid
		}
	}
	return ""
}

func filterShipUnits(in []ShipUnit) []ShipUnit {
	out := make([]ShipUnit, 0, len(in))
	for _, u := range in {
		if strings.TrimSpace(u.SSCC) == "" {
			continue
		}
		out = append(out, u)
	}
	return out
}
