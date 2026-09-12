package edi

import (
	"fmt"
	"strconv"
	"strings"
)

// Line is one ORDERS line item.
type Line struct {
	SKU string
	Qty int64
}

// OrdersMessage is the parsed inbound ORDERS payload.
type OrdersMessage struct {
	ExternalDocID string
	BuyerRef      string // retailer id (NAD+BY)
	SellerRef     string // supplier id (NAD+SU)
	Lat           float64
	Lng           float64
	H3Cell        string
	DeliveryDate  string // YYYY-MM-DD optional
	Lines         []Line
}

// ParseORDERS parses an EDI-lite ORDERS file.
func ParseORDERS(raw string) (OrdersMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return OrdersMessage{}, err
	}
	var msg OrdersMessage
	var unhType string
	for _, s := range segs {
		switch s.Tag {
		case "UNH":
			unhType = Comp(s.Elem(1), 0)
			if unhType != "" && !strings.EqualFold(unhType, DocTypeORDERS) {
				return OrdersMessage{}, fmt.Errorf("unexpected_msg_type:%s", unhType)
			}
		case "BGM":
			// BGM+220+{docId}
			msg.ExternalDocID = Comp(s.Elem(1), 0)
			if msg.ExternalDocID == "" {
				msg.ExternalDocID = s.Elem(1)
			}
		case "NAD":
			role := strings.ToUpper(s.Elem(0))
			ref := Comp(s.Elem(1), 0)
			if ref == "" {
				ref = s.Elem(1)
			}
			switch role {
			case "BY":
				msg.BuyerRef = ref
			case "SU":
				msg.SellerRef = ref
			}
		case "LOC":
			// LOC+7+{h3} or LOC+DEL+lat:lng:h3
			code := strings.ToUpper(s.Elem(0))
			if code == "7" || code == "DEL" {
				comp := s.Elem(1)
				if lat, err := strconv.ParseFloat(Comp(comp, 0), 64); err == nil {
					msg.Lat = lat
				}
				if lng, err := strconv.ParseFloat(Comp(comp, 1), 64); err == nil {
					msg.Lng = lng
				}
				h3 := Comp(comp, 2)
				if h3 == "" && len(comp) == 15 {
					h3 = comp
				}
				if h3 != "" {
					msg.H3Cell = h3
				}
			}
		case "DTM":
			// DTM+2:YYYYMMDD:102
			comp := s.Elem(0)
			if Comp(comp, 0) == "2" {
				d := Comp(comp, 1)
				if len(d) == 8 {
					msg.DeliveryDate = d[:4] + "-" + d[4:6] + "-" + d[6:8]
				} else if len(d) == 10 {
					msg.DeliveryDate = d
				}
			}
		case "LIN":
			sku := Comp(s.Elem(2), 0)
			if sku == "" {
				sku = s.Elem(2)
			}
			if sku == "" {
				sku = s.Elem(1)
			}
			msg.Lines = append(msg.Lines, Line{SKU: strings.TrimSpace(sku)})
		case "QTY":
			// QTY+21:{qty}
			comp := s.Elem(0)
			qStr := Comp(comp, 1)
			if qStr == "" {
				qStr = digitsOnly(comp)
			}
			q, _ := strconv.ParseInt(qStr, 10, 64)
			if len(msg.Lines) > 0 {
				msg.Lines[len(msg.Lines)-1].Qty = q
			}
		}
	}
	if msg.ExternalDocID == "" {
		return OrdersMessage{}, fmt.Errorf("missing_bgm")
	}
	if len(msg.Lines) == 0 {
		return OrdersMessage{}, fmt.Errorf("no_lines")
	}
	for i, ln := range msg.Lines {
		if ln.SKU == "" || ln.Qty <= 0 {
			return OrdersMessage{}, fmt.Errorf("invalid_line:%d", i)
		}
	}
	return msg, nil
}

// BuildORDERS encodes an ORDERS message (test / round-trip helper).
func BuildORDERS(m OrdersMessage) string {
	segs := []Segment{
		{Tag: "UNB", Elements: []string{"UNOC:3", "PEGASUS", "PARTNER", "00000000:0000", m.ExternalDocID}},
		{Tag: "UNH", Elements: []string{"1", "ORDERS:D:96A:UN"}},
		{Tag: "BGM", Elements: []string{"220", m.ExternalDocID}},
	}
	if m.BuyerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"BY", m.BuyerRef}})
	}
	if m.SellerRef != "" {
		segs = append(segs, Segment{Tag: "NAD", Elements: []string{"SU", m.SellerRef}})
	}
	if m.Lat != 0 || m.Lng != 0 || m.H3Cell != "" {
		segs = append(segs, Segment{Tag: "LOC", Elements: []string{"DEL",
			fmt.Sprintf("%g:%g:%s", m.Lat, m.Lng, m.H3Cell)}})
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
