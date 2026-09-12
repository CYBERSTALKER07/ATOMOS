package edi

import (
	"fmt"
	"strconv"
	"strings"
)

// OrdrspMessage is a parsed inbound ORDRSP (order response from trading partner).
type OrdrspMessage struct {
	ExternalDocID string
	RefOrderID    string // RFF+ON or BGM doc id
	ResponseCode  string // 27/29/etc from BGM
	BuyerRef      string
	SellerRef     string
	Accepted      bool
}

// InvoicMessage is a parsed inbound INVOIC (commercial invoice from trading partner).
type InvoicMessage struct {
	ExternalDocID  string
	RefOrderID     string
	BuyerRef       string
	SellerRef      string
	PrincipalMinor int64
	Currency       string
}

// DetectDocType returns the UNH message type (ORDERS/ORDRSP/DESADV/INVOIC/CONTRL/APERAK).
func DetectDocType(raw string) (string, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return "", err
	}
	for _, s := range segs {
		if s.Tag == "UNH" {
			t := strings.ToUpper(Comp(s.Elem(1), 0))
			if t == "" {
				return "", fmt.Errorf("unh_type_missing")
			}
			return t, nil
		}
	}
	return "", fmt.Errorf("unh_missing")
}

// ParseORDRSP parses an EDI-lite ORDRSP file.
func ParseORDRSP(raw string) (OrdrspMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return OrdrspMessage{}, err
	}
	var msg OrdrspMessage
	for _, s := range segs {
		switch s.Tag {
		case "UNH":
			if t := Comp(s.Elem(1), 0); t != "" && !strings.EqualFold(t, DocTypeORDRSP) {
				return OrdrspMessage{}, fmt.Errorf("unexpected_msg_type:%s", t)
			}
		case "BGM":
			msg.ExternalDocID = Comp(s.Elem(1), 0)
			if msg.ExternalDocID == "" {
				msg.ExternalDocID = s.Elem(1)
			}
			msg.ResponseCode = Comp(s.Elem(2), 0)
			if msg.ResponseCode == "" {
				msg.ResponseCode = s.Elem(2)
			}
		case "RFF":
			qual := Comp(s.Elem(0), 0)
			if strings.EqualFold(qual, "ON") {
				msg.RefOrderID = Comp(s.Elem(0), 1)
				if msg.RefOrderID == "" {
					msg.RefOrderID = s.Elem(1)
				}
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
		}
	}
	if msg.ExternalDocID == "" {
		return OrdrspMessage{}, fmt.Errorf("external_doc_id_required")
	}
	if msg.RefOrderID == "" {
		msg.RefOrderID = msg.ExternalDocID
	}
	// PegasusX EDI-lite ORDRSP dialect (matches BuildORDRSP): 29=accepted, 27=rejected.
	code := strings.ToUpper(strings.TrimSpace(msg.ResponseCode))
	switch code {
	case "27", "REJECTED", "REJ":
		msg.Accepted = false
	default:
		msg.Accepted = true
	}
	return msg, nil
}

// ParseINVOIC parses an EDI-lite INVOIC file (minimal commercial fields).
func ParseINVOIC(raw string) (InvoicMessage, error) {
	_, segs, err := ParseUNAFile(raw)
	if err != nil {
		return InvoicMessage{}, err
	}
	var msg InvoicMessage
	for _, s := range segs {
		switch s.Tag {
		case "UNH":
			if t := Comp(s.Elem(1), 0); t != "" && !strings.EqualFold(t, DocTypeINVOIC) {
				return InvoicMessage{}, fmt.Errorf("unexpected_msg_type:%s", t)
			}
		case "BGM":
			msg.ExternalDocID = Comp(s.Elem(1), 0)
			if msg.ExternalDocID == "" {
				msg.ExternalDocID = s.Elem(1)
			}
		case "RFF":
			qual := Comp(s.Elem(0), 0)
			if strings.EqualFold(qual, "ON") {
				msg.RefOrderID = Comp(s.Elem(0), 1)
				if msg.RefOrderID == "" {
					msg.RefOrderID = s.Elem(1)
				}
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
		case "MOA":
			// MOA+86:{amount}:{currency} or MOA+9:{amount}
			qual := Comp(s.Elem(0), 0)
			amtStr := Comp(s.Elem(0), 1)
			if amtStr == "" {
				amtStr = s.Elem(1)
			}
			if qual == "86" || qual == "9" || qual == "77" {
				if v, err := strconv.ParseInt(strings.ReplaceAll(amtStr, ".", ""), 10, 64); err == nil {
					msg.PrincipalMinor = v
				}
			}
			if msg.Currency == "" {
				if c := Comp(s.Elem(0), 2); c != "" {
					msg.Currency = c
				}
			}
		case "CUX":
			msg.Currency = Comp(s.Elem(0), 1)
			if msg.Currency == "" {
				msg.Currency = s.Elem(1)
			}
		}
	}
	if msg.ExternalDocID == "" {
		return InvoicMessage{}, fmt.Errorf("external_doc_id_required")
	}
	if msg.RefOrderID == "" {
		msg.RefOrderID = msg.ExternalDocID
	}
	return msg, nil
}
