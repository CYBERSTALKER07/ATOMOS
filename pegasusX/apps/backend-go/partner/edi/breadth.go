// Package edi — additional EDI-lite message types (PRICAT/INVRPT/SLSRPT/RECADV/
// ORDCHG/DELFOR/REMADV). Still not certified EDIFACT; dialect matches ORDERS.
package edi

import (
	"fmt"
	"strconv"
	"strings"
)

// CatalogLine is one PRICAT / price-catalog row.
type CatalogLine struct {
	SKU        string
	Name       string
	PriceMinor int64
	Currency   string
}

// PricatMessage is an inbound price/catalog feed.
type PricatMessage struct {
	ExternalDocID string
	SellerRef     string
	Lines         []CatalogLine
}

// StockLine is one INVRPT inventory row.
type StockLine struct {
	SKU       string
	QtyOnHand int64
	Warehouse string
}

// InvrptMessage is an inbound inventory report.
type InvrptMessage struct {
	ExternalDocID string
	SellerRef     string
	Lines         []StockLine
}

// SalesLine is one SLSRPT sell-through row.
type SalesLine struct {
	SKU string
	Qty int64
}

// SlsrptMessage is an inbound sales report (demand signal).
type SlsrptMessage struct {
	ExternalDocID string
	BuyerRef      string
	SellerRef     string
	ReportDate    string // YYYY-MM-DD
	Lines         []SalesLine
}

// RecadvMessage is a receiving advice (goods receipt).
type RecadvMessage struct {
	ExternalDocID string
	RefOrderID    string
	BuyerRef      string
	SellerRef     string
	AcceptedQty   int64
}

// OrdchgMessage is an order change request.
type OrdchgMessage struct {
	ExternalDocID string
	RefOrderID    string
	BuyerRef      string
	SellerRef     string
	Lines         []Line
}

// DelforMessage is a delivery forecast / schedule.
type DelforMessage struct {
	ExternalDocID string
	BuyerRef      string
	SellerRef     string
	DeliveryDate  string
	Lines         []Line
}

// RemadvMessage is a remittance advice.
type RemadvMessage struct {
	ExternalDocID  string
	RefInvoiceID   string
	BuyerRef       string
	SellerRef      string
	PaidMinor      int64
	Currency       string
}

func parseBGMDocID(s Segment) string {
	id := Comp(s.Elem(1), 0)
	if id == "" {
		id = s.Elem(1)
	}
	return strings.TrimSpace(id)
}

func parseNAD(s Segment, msgBuyer, msgSeller *string) {
	role := strings.ToUpper(s.Elem(0))
	ref := Comp(s.Elem(1), 0)
	if ref == "" {
		ref = s.Elem(1)
	}
	switch role {
	case "BY":
		*msgBuyer = ref
	case "SU":
		*msgSeller = ref
	}
}

func requireUNH(segs []Segment, want string) error {
	for _, s := range segs {
		if s.Tag == "UNH" {
			t := Comp(s.Elem(1), 0)
			if t != "" && !strings.EqualFold(t, want) {
				return fmt.Errorf("unexpected_msg_type:%s", t)
			}
			return nil
		}
	}
	return fmt.Errorf("unh_missing")
}

// ParsePRICAT parses an EDI-lite price catalog.
func ParsePRICAT(raw string) (PricatMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return PricatMessage{}, err
	}
	if err := requireUNH(segs, DocTypePRICAT); err != nil {
		return PricatMessage{}, err
	}
	var msg PricatMessage
	var buyer string
	for _, s := range segs {
		switch s.Tag {
		case "BGM":
			msg.ExternalDocID = parseBGMDocID(s)
		case "NAD":
			parseNAD(s, &buyer, &msg.SellerRef)
		case "LIN":
			sku := Comp(s.Elem(2), 0)
			if sku == "" {
				sku = s.Elem(2)
			}
			msg.Lines = append(msg.Lines, CatalogLine{SKU: strings.TrimSpace(sku)})
		case "IMD":
			if len(msg.Lines) > 0 {
				name := s.Elem(2)
				if name == "" {
					name = s.Elem(1)
				}
				msg.Lines[len(msg.Lines)-1].Name = strings.TrimSpace(name)
			}
		case "PRI":
			comp := s.Elem(0)
			pStr := Comp(comp, 1)
			if pStr == "" {
				pStr = digitsOnly(comp)
			}
			p, _ := strconv.ParseInt(pStr, 10, 64)
			cur := Comp(comp, 2)
			if len(msg.Lines) > 0 {
				msg.Lines[len(msg.Lines)-1].PriceMinor = p
				if cur != "" {
					msg.Lines[len(msg.Lines)-1].Currency = cur
				}
			}
		}
	}
	_ = buyer
	if msg.ExternalDocID == "" {
		return PricatMessage{}, fmt.Errorf("missing_bgm")
	}
	if len(msg.Lines) == 0 {
		return PricatMessage{}, fmt.Errorf("no_lines")
	}
	return msg, nil
}

// BuildPRICAT encodes a PRICAT message.
func BuildPRICAT(m PricatMessage) string {
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", "PARTNER", "00000000:0000", m.ExternalDocID}},
		{Tag: "UNH", Elements: []string{"1", "PRICAT:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"9", m.ExternalDocID}},
	}
	if m.SellerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"SU", m.SellerRef}})
	}
	for i, ln := range m.Lines {
		segs = append(segs, Segment{Tag: "LIN", Elements: []string{strconv.Itoa(i + 1), "", ln.SKU + ":SA"}})
		if ln.Name != "" {
			segs = append(segs, Segment{Tag: "IMD", Elements: []string{"F", "", ln.Name}})
		}
		cur := ediCurrency(m.SellerRef, ln.Currency)
		segs = append(segs, Segment{Tag: "PRI", Elements: []string{fmt.Sprintf("AAA:%d:%s", ln.PriceMinor, cur)}})
	}
	segs = append(segs,
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 1), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", m.ExternalDocID}},
	)
	return WriteUNAFile(segs)
}

// ParseINVRPT parses an EDI-lite inventory report.
func ParseINVRPT(raw string) (InvrptMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return InvrptMessage{}, err
	}
	if err := requireUNH(segs, DocTypeINVRPT); err != nil {
		return InvrptMessage{}, err
	}
	var msg InvrptMessage
	var buyer string
	for _, s := range segs {
		switch s.Tag {
		case "BGM":
			msg.ExternalDocID = parseBGMDocID(s)
		case "NAD":
			parseNAD(s, &buyer, &msg.SellerRef)
		case "LIN":
			sku := Comp(s.Elem(2), 0)
			if sku == "" {
				sku = s.Elem(2)
			}
			msg.Lines = append(msg.Lines, StockLine{SKU: strings.TrimSpace(sku)})
		case "QTY":
			comp := s.Elem(0)
			qStr := Comp(comp, 1)
			if qStr == "" {
				qStr = digitsOnly(comp)
			}
			q, _ := strconv.ParseInt(qStr, 10, 64)
			if len(msg.Lines) > 0 {
				msg.Lines[len(msg.Lines)-1].QtyOnHand = q
			}
		case "LOC":
			wh := Comp(s.Elem(1), 0)
			if wh == "" {
				wh = s.Elem(1)
			}
			if len(msg.Lines) > 0 {
				msg.Lines[len(msg.Lines)-1].Warehouse = strings.TrimSpace(wh)
			}
		}
	}
	_ = buyer
	if msg.ExternalDocID == "" {
		return InvrptMessage{}, fmt.Errorf("missing_bgm")
	}
	if len(msg.Lines) == 0 {
		return InvrptMessage{}, fmt.Errorf("no_lines")
	}
	return msg, nil
}

// BuildINVRPT encodes an INVRPT message.
func BuildINVRPT(m InvrptMessage) string {
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", "PARTNER", "00000000:0000", m.ExternalDocID}},
		{Tag: "UNH", Elements: []string{"1", "INVRPT:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"35", m.ExternalDocID}},
	}
	if m.SellerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"SU", m.SellerRef}})
	}
	for i, ln := range m.Lines {
		segs = append(segs,
			Segment{Tag: "LIN", Elements: []string{strconv.Itoa(i + 1), "", ln.SKU + ":SA"}},
			Segment{Tag: "QTY", Elements: []string{fmt.Sprintf("145:%d", ln.QtyOnHand)}},
		)
		if ln.Warehouse != "" {
			segs = append(segs, Segment{Tag: "LOC", Elements: []string{"14", ln.Warehouse}})
		}
	}
	segs = append(segs,
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 1), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", m.ExternalDocID}},
	)
	return WriteUNAFile(segs)
}

// ParseSLSRPT parses an EDI-lite sales report.
func ParseSLSRPT(raw string) (SlsrptMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return SlsrptMessage{}, err
	}
	if err := requireUNH(segs, DocTypeSLSRPT); err != nil {
		return SlsrptMessage{}, err
	}
	var msg SlsrptMessage
	for _, s := range segs {
		switch s.Tag {
		case "BGM":
			msg.ExternalDocID = parseBGMDocID(s)
		case "NAD":
			parseNAD(s, &msg.BuyerRef, &msg.SellerRef)
		case "DTM":
			comp := s.Elem(0)
			if Comp(comp, 0) == "137" || Comp(comp, 0) == "2" {
				d := Comp(comp, 1)
				if len(d) == 8 {
					msg.ReportDate = d[:4] + "-" + d[4:6] + "-" + d[6:8]
				} else {
					msg.ReportDate = d
				}
			}
		case "LIN":
			sku := Comp(s.Elem(2), 0)
			if sku == "" {
				sku = s.Elem(2)
			}
			msg.Lines = append(msg.Lines, SalesLine{SKU: strings.TrimSpace(sku)})
		case "QTY":
			comp := s.Elem(0)
			qStr := Comp(comp, 1)
			q, _ := strconv.ParseInt(qStr, 10, 64)
			if len(msg.Lines) > 0 {
				msg.Lines[len(msg.Lines)-1].Qty = q
			}
		}
	}
	if msg.ExternalDocID == "" {
		return SlsrptMessage{}, fmt.Errorf("missing_bgm")
	}
	return msg, nil
}

// BuildSLSRPT encodes a SLSRPT message.
func BuildSLSRPT(m SlsrptMessage) string {
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", "PARTNER", "00000000:0000", m.ExternalDocID}},
		{Tag: "UNH", Elements: []string{"1", "SLSRPT:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"73", m.ExternalDocID}},
	}
	if m.BuyerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"BY", m.BuyerRef}})
	}
	if m.SellerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"SU", m.SellerRef}})
	}
	if m.ReportDate != "" {
		d := strings.ReplaceAll(m.ReportDate, "-", "")
		segs = append(segs, Segment{Tag: "DTM", Elements: []string{"137:" + d + ":102"}})
	}
	for i, ln := range m.Lines {
		segs = append(segs,
			Segment{Tag: "LIN", Elements: []string{strconv.Itoa(i + 1), "", ln.SKU + ":SA"}},
			Segment{Tag: "QTY", Elements: []string{fmt.Sprintf("153:%d", ln.Qty)}},
		)
	}
	segs = append(segs,
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 1), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", m.ExternalDocID}},
	)
	return WriteUNAFile(segs)
}

// ParseRECADV parses a receiving advice.
func ParseRECADV(raw string) (RecadvMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return RecadvMessage{}, err
	}
	if err := requireUNH(segs, DocTypeRECADV); err != nil {
		return RecadvMessage{}, err
	}
	var msg RecadvMessage
	for _, s := range segs {
		switch s.Tag {
		case "BGM":
			msg.ExternalDocID = parseBGMDocID(s)
		case "RFF":
			comp := s.Elem(0)
			if Comp(comp, 0) == "ON" {
				msg.RefOrderID = Comp(comp, 1)
			}
		case "NAD":
			parseNAD(s, &msg.BuyerRef, &msg.SellerRef)
		case "QTY":
			comp := s.Elem(0)
			qStr := Comp(comp, 1)
			q, _ := strconv.ParseInt(qStr, 10, 64)
			msg.AcceptedQty += q
		}
	}
	if msg.ExternalDocID == "" {
		return RecadvMessage{}, fmt.Errorf("missing_bgm")
	}
	if msg.RefOrderID == "" {
		msg.RefOrderID = msg.ExternalDocID
	}
	return msg, nil
}

// BuildRECADV encodes a RECADV message.
func BuildRECADV(m RecadvMessage) string {
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", "PARTNER", "00000000:0000", m.ExternalDocID}},
		{Tag: "UNH", Elements: []string{"1", "RECADV:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"632", m.ExternalDocID}},
		{Tag: "RFF", Elements: []string{"ON:" + m.RefOrderID}},
	}
	if m.BuyerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"BY", m.BuyerRef}})
	}
	if m.SellerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"SU", m.SellerRef}})
	}
	segs = append(segs,
		Segment{Tag: "QTY", Elements: []string{fmt.Sprintf("194:%d", m.AcceptedQty)}},
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 2), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", m.ExternalDocID}},
	)
	return WriteUNAFile(segs)
}

// ParseORDCHG parses an order change.
func ParseORDCHG(raw string) (OrdchgMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return OrdchgMessage{}, err
	}
	if err := requireUNH(segs, DocTypeORDCHG); err != nil {
		return OrdchgMessage{}, err
	}
	var msg OrdchgMessage
	for _, s := range segs {
		switch s.Tag {
		case "BGM":
			msg.ExternalDocID = parseBGMDocID(s)
		case "RFF":
			comp := s.Elem(0)
			if Comp(comp, 0) == "ON" {
				msg.RefOrderID = Comp(comp, 1)
			}
		case "NAD":
			parseNAD(s, &msg.BuyerRef, &msg.SellerRef)
		case "LIN":
			sku := Comp(s.Elem(2), 0)
			if sku == "" {
				sku = s.Elem(2)
			}
			msg.Lines = append(msg.Lines, Line{SKU: strings.TrimSpace(sku)})
		case "QTY":
			comp := s.Elem(0)
			qStr := Comp(comp, 1)
			q, _ := strconv.ParseInt(qStr, 10, 64)
			if len(msg.Lines) > 0 {
				msg.Lines[len(msg.Lines)-1].Qty = q
			}
		}
	}
	if msg.ExternalDocID == "" {
		return OrdchgMessage{}, fmt.Errorf("missing_bgm")
	}
	if msg.RefOrderID == "" {
		msg.RefOrderID = msg.ExternalDocID
	}
	return msg, nil
}

// BuildORDCHG encodes an ORDCHG message.
func BuildORDCHG(m OrdchgMessage) string {
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", "PARTNER", "00000000:0000", m.ExternalDocID}},
		{Tag: "UNH", Elements: []string{"1", "ORDCHG:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"230", m.ExternalDocID}},
		{Tag: "RFF", Elements: []string{"ON:" + m.RefOrderID}},
	}
	if m.BuyerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"BY", m.BuyerRef}})
	}
	if m.SellerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"SU", m.SellerRef}})
	}
	for i, ln := range m.Lines {
		segs = append(segs,
			Segment{Tag: "LIN", Elements: []string{strconv.Itoa(i + 1), "", ln.SKU + ":SA"}},
			Segment{Tag: "QTY", Elements: []string{fmt.Sprintf("21:%d", ln.Qty)}},
		)
	}
	segs = append(segs,
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 1), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", m.ExternalDocID}},
	)
	return WriteUNAFile(segs)
}

// ParseDELFOR parses a delivery forecast.
func ParseDELFOR(raw string) (DelforMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return DelforMessage{}, err
	}
	if err := requireUNH(segs, DocTypeDELFOR); err != nil {
		return DelforMessage{}, err
	}
	var msg DelforMessage
	for _, s := range segs {
		switch s.Tag {
		case "BGM":
			msg.ExternalDocID = parseBGMDocID(s)
		case "NAD":
			parseNAD(s, &msg.BuyerRef, &msg.SellerRef)
		case "DTM":
			comp := s.Elem(0)
			if Comp(comp, 0) == "2" {
				d := Comp(comp, 1)
				if len(d) == 8 {
					msg.DeliveryDate = d[:4] + "-" + d[4:6] + "-" + d[6:8]
				} else {
					msg.DeliveryDate = d
				}
			}
		case "LIN":
			sku := Comp(s.Elem(2), 0)
			if sku == "" {
				sku = s.Elem(2)
			}
			msg.Lines = append(msg.Lines, Line{SKU: strings.TrimSpace(sku)})
		case "QTY":
			comp := s.Elem(0)
			qStr := Comp(comp, 1)
			q, _ := strconv.ParseInt(qStr, 10, 64)
			if len(msg.Lines) > 0 {
				msg.Lines[len(msg.Lines)-1].Qty = q
			}
		}
	}
	if msg.ExternalDocID == "" {
		return DelforMessage{}, fmt.Errorf("missing_bgm")
	}
	return msg, nil
}

// BuildDELFOR encodes a DELFOR message.
func BuildDELFOR(m DelforMessage) string {
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", "PARTNER", "00000000:0000", m.ExternalDocID}},
		{Tag: "UNH", Elements: []string{"1", "DELFOR:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"241", m.ExternalDocID}},
	}
	if m.BuyerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"BY", m.BuyerRef}})
	}
	if m.SellerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"SU", m.SellerRef}})
	}
	if m.DeliveryDate != "" {
		d := strings.ReplaceAll(m.DeliveryDate, "-", "")
		segs = append(segs, Segment{Tag: "DTM", Elements: []string{"2:" + d + ":102"}})
	}
	for i, ln := range m.Lines {
		segs = append(segs,
			Segment{Tag: "LIN", Elements: []string{strconv.Itoa(i + 1), "", ln.SKU + ":SA"}},
			Segment{Tag: "QTY", Elements: []string{fmt.Sprintf("21:%d", ln.Qty)}},
		)
	}
	segs = append(segs,
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 1), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", m.ExternalDocID}},
	)
	return WriteUNAFile(segs)
}

// ParseREMADV parses a remittance advice.
func ParseREMADV(raw string) (RemadvMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return RemadvMessage{}, err
	}
	if err := requireUNH(segs, DocTypeREMADV); err != nil {
		return RemadvMessage{}, err
	}
	var msg RemadvMessage
	for _, s := range segs {
		switch s.Tag {
		case "BGM":
			msg.ExternalDocID = parseBGMDocID(s)
		case "RFF":
			comp := s.Elem(0)
			if Comp(comp, 0) == "IV" {
				msg.RefInvoiceID = Comp(comp, 1)
			}
		case "NAD":
			parseNAD(s, &msg.BuyerRef, &msg.SellerRef)
		case "MOA":
			comp := s.Elem(0)
			pStr := Comp(comp, 1)
			p, _ := strconv.ParseInt(pStr, 10, 64)
			msg.PaidMinor = p
			if c := Comp(comp, 2); c != "" {
				msg.Currency = c
			}
		}
	}
	if msg.ExternalDocID == "" {
		return RemadvMessage{}, fmt.Errorf("missing_bgm")
	}
	return msg, nil
}

// BuildREMADV encodes a REMADV message.
func BuildREMADV(m RemadvMessage) string {
	cur := ediCurrency(m.SellerRef, m.Currency)
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", "PARTNER", "00000000:0000", m.ExternalDocID}},
		{Tag: "UNH", Elements: []string{"1", "REMADV:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"481", m.ExternalDocID}},
		{Tag: "RFF", Elements: []string{"IV:" + m.RefInvoiceID}},
	}
	if m.BuyerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"BY", m.BuyerRef}})
	}
	if m.SellerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"SU", m.SellerRef}})
	}
	segs = append(segs,
		Segment{Tag: "MOA", Elements: []string{fmt.Sprintf("12:%d:%s", m.PaidMinor, cur)}},
		Segment{Tag: "UNT", Elements: []string{strconv.Itoa(len(segs) + 2), "1"}},
		Segment{Tag: "UNZ", Elements: []string{"1", m.ExternalDocID}},
	)
	return WriteUNAFile(segs)
}
