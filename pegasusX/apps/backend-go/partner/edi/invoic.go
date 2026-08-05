package edi

import (
	"fmt"
	"strconv"
	"time"
)

// InvoiceSnapshot optional AR invoice fields for INVOIC.
type InvoiceSnapshot struct {
	InvoiceID     string
	PrincipalMinor int64
	Currency      string
	DueDate       string // YYYY-MM-DD
}

// BuildINVOIC encodes a commercial invoice.
func BuildINVOIC(o OrderSnapshot, inv *InvoiceSnapshot, externalDocID string) string {
	if externalDocID == "" {
		if inv != nil && inv.InvoiceID != "" {
			externalDocID = inv.InvoiceID
		} else {
			externalDocID = o.OrderID + ":INVOIC"
		}
	}
	amount := o.TotalMinor
	currency := o.Currency
	if inv != nil {
		if inv.PrincipalMinor > 0 {
			amount = inv.PrincipalMinor
		}
		if inv.Currency != "" {
			currency = inv.Currency
		}
	}
	if currency == "" {
		currency = "UZS"
	}
	ts := time.Now().UTC().Format("060102:1504")
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", o.RetailerID, ts, externalDocID}},
		{Tag: "UNH", Elements: []string{"1", "INVOIC:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"380", externalDocID}},
		{Tag: "RFF", Elements: []string{"ON:" + o.OrderID}},
		{Tag: "NAD", Elements: []string{"BY", o.RetailerID}},
		{Tag: "NAD", Elements: []string{"SU", o.SupplierID}},
		{Tag: "MOA", Elements: []string{fmt.Sprintf("86:%d:%s", amount, currency)}},
	}
	if inv != nil && inv.DueDate != "" {
		dd := digitsOnly(inv.DueDate)
		if len(dd) >= 8 {
			segs = append(segs, Segment{Tag: "DTM", Elements: []string{"13:" + dd[:8] + ":102"}})
		}
	}
	for i, ln := range o.Lines {
		segs = append(segs,
			Segment{Tag: "LIN", Elements: []string{strconv.Itoa(i + 1), "", ln.SKU + ":SA"}},
			Segment{Tag: "QTY", Elements: []string{fmt.Sprintf("47:%d", ln.Qty)}},
		)
	}
	segs = append(segs,
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 1), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", externalDocID}},
	)
	return WriteUNAFile(segs)
}
