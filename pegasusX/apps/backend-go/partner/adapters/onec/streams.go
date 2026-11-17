package onec

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
)

// CommerceMLBundle contains typed catalog/offer records unpacked from a 1C CommerceML v2/v3 export.
type CommerceMLBundle struct {
	Streams           []parser.ParsedStream
	Batch             ImportBatch
	StockByExternalID map[string]float64
	Customers         []map[string]any
	Documents         []map[string]any
}

// ParseCommerceMLStreams parses multi-entity 1C CommerceML XML (products, offers/stock,
// counterparties, and documents) using the shared streaming migration parser.
func ParseCommerceMLStreams(xmlBytes []byte, onChunk parser.StreamRowHandler) ([]parser.ParsedStream, error) {
	return parser.StreamCommerceML(xmlBytes, onChunk)
}

// ExtractCommerceMLBundle converts CommerceML streams into typed catalog/offer records
// plus normalized customer and document rows for both partner sync and migration onboarding.
func ExtractCommerceMLBundle(xmlBytes []byte) (CommerceMLBundle, error) {
	bundle := CommerceMLBundle{
		Batch:             ImportBatch{ExternalDocID: "commerceml"},
		StockByExternalID: make(map[string]float64),
	}
	productMap := make(map[string]ImportProduct)
	productOrder := make([]string, 0)

	ensureProduct := func(extID string) ImportProduct {
		p, ok := productMap[extID]
		if !ok {
			productOrder = append(productOrder, extID)
			p.ExternalID = extID
		}
		return p
	}

	streams, err := parser.StreamCommerceML(xmlBytes, func(streamName string, _ []string, rows []parser.ParsedRow) error {
		for _, r := range rows {
			f := r.Fields
			switch streamName {
			case "commerceml_products":
				extID := firstFieldString(f, "external_sku_id", "sku", "name")
				if extID == "" {
					continue
				}
				p := ensureProduct(extID)
				p.Name = firstNonEmpty(fieldString(f, "name"), p.Name)
				p.Barcode = firstNonEmpty(fieldString(f, "barcode"), p.Barcode)
				productMap[extID] = p
			case "commerceml_offers_stock":
				extID := firstFieldString(f, "external_sku_id", "sku")
				if extID == "" {
					continue
				}
				p := ensureProduct(extID)
				p.Name = firstNonEmpty(p.Name, fieldString(f, "name"))
				p.Currency = firstNonEmpty(fieldString(f, "currency"), p.Currency)
				if q, parseErr := strconv.ParseFloat(firstNonEmpty(fieldString(f, "quantity_on_hand"), "0"), 64); parseErr == nil {
					bundle.StockByExternalID[extID] += q
				}
				if pr, parseErr := strconv.ParseFloat(firstNonEmpty(fieldString(f, "price"), "0"), 64); parseErr == nil && pr > 0 {
					p.PriceMinor = int64(pr * 100)
				}
				productMap[extID] = p
			case "commerceml_customers":
				bundle.Customers = append(bundle.Customers, f)
			case "commerceml_documents":
				bundle.Documents = append(bundle.Documents, f)
			}
		}
		return nil
	})
	if err != nil {
		return CommerceMLBundle{}, err
	}
	bundle.Streams = streams
	for _, extID := range productOrder {
		bundle.Batch.Products = append(bundle.Batch.Products, productMap[extID])
	}
	return bundle, nil
}

func fieldString(fields map[string]any, key string) string {
	v, ok := fields[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func firstFieldString(fields map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := fieldString(fields, k); s != "" {
			return s
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
