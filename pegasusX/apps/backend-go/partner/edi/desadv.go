package edi

import (
	"fmt"
	"strconv"
	"time"
)

// BuildDESADV encodes a despatch advice for LOADED / IN_TRANSIT.
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
