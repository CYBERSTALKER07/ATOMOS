package parser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// StreamCommerceML unpacks 1C CommerceML XML exports into separate entity streams:
// - `commerceml_products` (<Каталог><Товары><Товар>)
// - `commerceml_offers_stock` (<ПакетПредложений><Предложения><Предложение> + <Остатки>)
// - `commerceml_customers` (<Контрагенты><Контрагент>)
// - `commerceml_documents` (<Документ>)
func StreamCommerceML(data []byte, onChunk StreamRowHandler) ([]ParsedStream, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	// Handle 1C XML exports declared with windows-1251 encoding natively.
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(strings.TrimSpace(charset)) {
		case "windows-1251", "cp1251":
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		case "cp866", "ibm866":
			return charmap.CodePage866.NewDecoder().Reader(input), nil
		default:
			return input, nil
		}
	}

	buckets := map[string][]map[string]any{}
	appendRow := func(stream string, row map[string]any) {
		if len(row) == 0 {
			return
		}
		buckets[stream] = append(buckets[stream], row)
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: parse commerceml xml: %v", ErrMalformedArtifact, err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "Товар":
			var item cmlProduct
			if err := dec.DecodeElement(&item, &se); err != nil {
				return nil, fmt.Errorf("decode CommerceML Товар: %w", err)
			}
			row := map[string]any{
				"external_sku_id": nonEmpty(item.ID, item.VendorCode),
				"sku":             nonEmpty(item.VendorCode, item.ID),
				"name":            strings.TrimSpace(item.Name),
				"barcode":         strings.TrimSpace(item.Barcode),
				"unit":            strings.TrimSpace(item.BaseUnit),
				"description":     strings.TrimSpace(item.Description),
			}
			appendRow("commerceml_products", row)

		case "Предложение":
			var offer cmlOffer
			if err := dec.DecodeElement(&offer, &se); err != nil {
				return nil, fmt.Errorf("decode CommerceML Предложение: %w", err)
			}
			price := ""
			currency := ""
			if len(offer.Prices) > 0 {
				price = strings.TrimSpace(offer.Prices[0].PricePerUnit)
				currency = strings.TrimSpace(offer.Prices[0].Currency)
			}
			warehouses := offer.collectRests()
			if len(warehouses) > 0 {
				for _, r := range warehouses {
					appendRow("commerceml_offers_stock", map[string]any{
						"external_sku_id":  strings.TrimSpace(offer.ID),
						"name":             strings.TrimSpace(offer.Name),
						"barcode":          strings.TrimSpace(offer.Barcode),
						"price":            price,
						"currency":         currency,
						"warehouse_id":     strings.TrimSpace(r.WarehouseID),
						"quantity_on_hand": strings.TrimSpace(nonEmpty(r.Qty, offer.Quantity)),
					})
				}
			} else {
				appendRow("commerceml_offers_stock", map[string]any{
					"external_sku_id":  strings.TrimSpace(offer.ID),
					"name":             strings.TrimSpace(offer.Name),
					"barcode":          strings.TrimSpace(offer.Barcode),
					"price":            price,
					"currency":         currency,
					"quantity_on_hand": strings.TrimSpace(offer.Quantity),
				})
			}

		case "Контрагент":
			var cp cmlCounterparty
			if err := dec.DecodeElement(&cp, &se); err != nil {
				return nil, fmt.Errorf("decode CommerceML Контрагент: %w", err)
			}
			appendRow("commerceml_customers", map[string]any{
				"external_customer_id": strings.TrimSpace(cp.ID),
				"customer_name":        nonEmpty(cp.FullName, cp.Name),
				"inn":                  strings.TrimSpace(cp.INN),
				"kpp":                  strings.TrimSpace(cp.KPP),
				"phone":                cp.findContact("Телефон"),
				"email":                cp.findContact("Почта"),
				"address":              strings.TrimSpace(cp.Address.Presentation),
			})

		case "Документ":
			var doc cmlDocument
			if err := dec.DecodeElement(&doc, &se); err != nil {
				return nil, fmt.Errorf("decode CommerceML Документ: %w", err)
			}
			customerID := ""
			customerName := ""
			customerINN := ""
			if len(doc.Counterparties) > 0 {
				customerID = strings.TrimSpace(doc.Counterparties[0].ID)
				customerName = strings.TrimSpace(nonEmpty(doc.Counterparties[0].FullName, doc.Counterparties[0].Name))
				customerINN = strings.TrimSpace(doc.Counterparties[0].INN)
			}
			itemsSummary := make([]map[string]any, 0, len(doc.Products))
			for _, p := range doc.Products {
				itemsSummary = append(itemsSummary, map[string]any{
					"external_sku_id": strings.TrimSpace(p.ID),
					"name":            strings.TrimSpace(p.Name),
					"quantity":        strings.TrimSpace(p.Quantity),
					"price":           strings.TrimSpace(p.Price),
					"total":           strings.TrimSpace(p.Total),
				})
			}
			appendRow("commerceml_documents", map[string]any{
				"external_order_no": strings.TrimSpace(nonEmpty(doc.Number, doc.ID)),
				"operation":         strings.TrimSpace(doc.Operation),
				"doc_date":          strings.TrimSpace(doc.Date),
				"currency":          strings.TrimSpace(doc.Currency),
				"total_amount":      strings.TrimSpace(doc.Total),
				"status":            strings.TrimSpace(doc.Status),
				"customer_id":       customerID,
				"customer_name":     customerName,
				"inn":               customerINN,
				"line_items":        itemsSummary,
			})
		}
	}

	if len(buckets) == 0 {
		return nil, fmt.Errorf("%w: no recognized CommerceML elements found", ErrMalformedArtifact)
	}

	orderedNames := []string{
		"commerceml_products",
		"commerceml_offers_stock",
		"commerceml_customers",
		"commerceml_documents",
	}
	streams := make([]ParsedStream, 0, len(buckets))
	for _, streamName := range orderedNames {
		rows := buckets[streamName]
		if len(rows) == 0 {
			continue
		}
		headerMap := map[string]struct{}{}
		for _, r := range rows {
			for k := range r {
				headerMap[k] = struct{}{}
			}
		}
		headers := make([]string, 0, len(headerMap))
		for k := range headerMap {
			headers = append(headers, k)
		}
		meta := ParsedStream{
			StreamName: streamName,
			Format:     FormatCommerceML,
			Encoding:   "utf-8",
			Headers:    headers,
			TotalRows:  int64(len(rows)),
		}
		chunk := make([]ParsedRow, 0, defaultStreamChunkSize)
		for idx, r := range rows {
			pr := ParsedRow{RowIndex: int64(idx), Fields: r}
			if len(meta.SampleRows) < 20 {
				meta.SampleRows = append(meta.SampleRows, pr)
			}
			if onChunk != nil {
				chunk = append(chunk, pr)
				if len(chunk) >= defaultStreamChunkSize {
					if err := onChunk(streamName, headers, chunk); err != nil {
						return nil, err
					}
					chunk = chunk[:0]
				}
			}
		}
		if onChunk != nil && len(chunk) > 0 {
			if err := onChunk(streamName, headers, chunk); err != nil {
				return nil, err
			}
		}
		streams = append(streams, meta)
	}
	return streams, nil
}

type cmlProduct struct {
	ID          string `xml:"Ид"`
	Barcode     string `xml:"Штрихкод"`
	VendorCode  string `xml:"Артикул"`
	Name        string `xml:"Наименование"`
	BaseUnit    string `xml:"БазоваяЕдиница"`
	Description string `xml:"Описание"`
}

type cmlWarehouseRest struct {
	WarehouseIDAttr string `xml:"ИдСклада,attr"`
	QtyAttr         string `xml:"КоличествоНаСкладе,attr"`
	WarehouseID     string `xml:"Ид"`
	Qty             string `xml:"КоличествоНаСкладе"`
}

type cmlRestContainer struct {
	DirectWarehouse cmlWarehouseRest `xml:"Склад"`
	RestNodes       []struct {
		Warehouse cmlWarehouseRest `xml:"Склад"`
		Qty       string           `xml:"Количество"`
	} `xml:"Остаток"`
}

type cmlOffer struct {
	ID       string `xml:"Ид"`
	Barcode  string `xml:"Штрихкод"`
	Name     string `xml:"Наименование"`
	Quantity string `xml:"Количество"`
	Prices   []struct {
		PricePerUnit string `xml:"ЦенаЗаЕдиницу"`
		Currency     string `xml:"Валюта"`
	} `xml:"Цены>Цена"`
	Rests            []cmlRestContainer `xml:"Остатки"`
	DirectWarehouses []cmlWarehouseRest `xml:"Склад"`
}

func (o cmlOffer) collectRests() []struct {
	WarehouseID string
	Qty         string
} {
	var out []struct {
		WarehouseID string
		Qty         string
	}
	appendWR := func(w cmlWarehouseRest, fallbackQty string) {
		wh := nonEmpty(w.WarehouseIDAttr, w.WarehouseID)
		qty := nonEmpty(nonEmpty(w.QtyAttr, w.Qty), fallbackQty)
		if wh != "" || qty != "" {
			out = append(out, struct {
				WarehouseID string
				Qty         string
			}{WarehouseID: wh, Qty: qty})
		}
	}
	for _, r := range o.Rests {
		appendWR(r.DirectWarehouse, "")
		for _, rn := range r.RestNodes {
			appendWR(rn.Warehouse, rn.Qty)
		}
	}
	for _, w := range o.DirectWarehouses {
		appendWR(w, "")
	}
	return out
}

type cmlCounterparty struct {
	ID       string `xml:"Ид"`
	Name     string `xml:"Наименование"`
	FullName string `xml:"ОфициальноеНаименование"`
	INN      string `xml:"ИНН"`
	KPP      string `xml:"КПП"`
	Address  struct {
		Presentation string `xml:"Представление"`
	} `xml:"ЮридическийАдрес"`
	Contacts []struct {
		Type  string `xml:"Тип"`
		Value string `xml:"Значение"`
	} `xml:"Контакты>Контакт"`
}

func (c cmlCounterparty) findContact(typ string) string {
	for _, ct := range c.Contacts {
		if strings.Contains(strings.ToLower(ct.Type), strings.ToLower(typ)) {
			return strings.TrimSpace(ct.Value)
		}
	}
	return ""
}

type cmlDocument struct {
	ID             string            `xml:"Ид"`
	Number         string            `xml:"Номер"`
	Date           string            `xml:"Дата"`
	Operation      string            `xml:"ХозОперация"`
	Currency       string            `xml:"Валюта"`
	Total          string            `xml:"Сумма"`
	Status         string            `xml:"Статус"`
	Counterparties []cmlCounterparty `xml:"Контрагенты>Контрагент"`
	Products       []struct {
		ID       string `xml:"Ид"`
		Name     string `xml:"Наименование"`
		Price    string `xml:"ЦенаЗаЕдиницу"`
		Quantity string `xml:"Количество"`
		Total    string `xml:"Сумма"`
	} `xml:"Товары>Товар"`
}

func nonEmpty(a, b string) string {
	if t := strings.TrimSpace(a); t != "" {
		return t
	}
	return strings.TrimSpace(b)
}
