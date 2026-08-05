package edi

import (
	"fmt"
	"strconv"
	"time"
)

// OrderSnapshot is the minimal order view for outbound EDI.
type OrderSnapshot struct {
	OrderID    string
	RetailerID string
	SupplierID string
	Status     string
	Currency   string
	TotalMinor int64
	Lines      []Line
}

// BuildORDRSP encodes an order response for the given status.
func BuildORDRSP(o OrderSnapshot, externalDocID string) string {
	if externalDocID == "" {
		externalDocID = o.OrderID + ":" + o.Status
	}
	respCode := "29" // accepted
	switch o.Status {
	case "CANCELLED", "CANCEL_REQUESTED", "REJECTED":
		respCode = "27" // not accepted
	case "BACKORDERED":
		respCode = "4" // change
	case "SCHEDULED", "PENDING", "CONFIRMED", "AUTO_ACCEPTED":
		respCode = "29"
	}
	ts := time.Now().UTC().Format("060102:1504")
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", o.RetailerID, ts, externalDocID}},
		{Tag: "UNH", Elements: []string{"1", "ORDRSP:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{respCode, externalDocID}},
		{Tag: "RFF", Elements: []string{"ON:" + o.OrderID}},
		{Tag: "NAD", Elements: []string{"BY", o.RetailerID}},
		{Tag: "NAD", Elements: []string{"SU", o.SupplierID}},
		{Tag: "FTX", Elements: []string{"AAI", "", "", o.Status}},
	}
	for i, ln := range o.Lines {
		segs = append(segs,
			Segment{Tag: "LIN", Elements: []string{strconv.Itoa(i + 1), "", ln.SKU + ":SA"}},
			Segment{Tag: "QTY", Elements: []string{fmt.Sprintf("21:%d", ln.Qty)}},
		)
	}
	segs = append(segs,
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 1), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", externalDocID}},
	)
	return WriteUNAFile(segs)
}
