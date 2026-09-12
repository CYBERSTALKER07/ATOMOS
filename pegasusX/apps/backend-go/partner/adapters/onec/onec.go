// Package onec is the G5.B 1C-first partner adapter (CommerceML-lite / journal pack).
// Not a certified 1C vendor integration — subset import + journals export dialect.
package onec

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// CommerceMLProduct is a minimal product node for catalog import.
type CommerceMLProduct struct {
	ID       string `xml:"Ид"`
	Name     string `xml:"Наименование"`
	Barcode  string `xml:"Штрихкод"`
	Price    int64  `xml:"Цена"` // minor units optional; often filled from prices package
	Currency string `xml:"Валюта"`
}

// CommerceMLCatalog is a minimal catalog wrapper.
type CommerceMLCatalog struct {
	XMLName  xml.Name            `xml:"Каталог"`
	Products []CommerceMLProduct `xml:"Товары>Товар"`
}

// ImportBatch is the adapter-facing import payload (JSON or mapped from XML).
type ImportBatch struct {
	ExternalDocID string        `json:"external_doc_id"`
	Products      []ImportProduct `json:"products"`
}

// ImportProduct maps to partner ProductUpsertItem fields.
type ImportProduct struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Barcode    string `json:"barcode"`
	PriceMinor int64  `json:"price_minor"`
	Currency   string `json:"currency"`
}

// ParseCommerceMLCatalog parses a minimal CommerceML-like catalog XML.
func ParseCommerceMLCatalog(raw []byte) (ImportBatch, error) {
	var cat CommerceMLCatalog
	if err := xml.Unmarshal(raw, &cat); err != nil {
		return ImportBatch{}, fmt.Errorf("commerceml_parse: %w", err)
	}
	batch := ImportBatch{ExternalDocID: "catalog"}
	for _, p := range cat.Products {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		cur := strings.TrimSpace(p.Currency)
		batch.Products = append(batch.Products, ImportProduct{
			ExternalID: id,
			Name:       strings.TrimSpace(p.Name),
			Barcode:    strings.TrimSpace(p.Barcode),
			PriceMinor: p.Price,
			Currency:   cur,
		})
	}
	if len(batch.Products) == 0 {
		return ImportBatch{}, fmt.Errorf("no_products")
	}
	return batch, nil
}

// JournalDialect is the export dialect label for 1C-friendly journals.
const JournalDialect = "1c"

// AdapterName for PartnerExternalDocuments.
const AdapterName = "onec"
