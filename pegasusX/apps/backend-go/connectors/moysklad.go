package connectors

import (
	"net/http"
	"strings"
	"time"
)

// MoySkladConnector implements the Connector interface against MoySklad JSON REST API.
type MoySkladConnector struct {
	*restERPAdapter
}

// NewMoySkladConnector constructs a MoySkladConnector backed by SecretResolver and http.Client.
func NewMoySkladConnector(secrets SecretResolver, httpClient *http.Client) *MoySkladConnector {
	cfg := restERPConfig{
		provider:       ProviderMoySklad,
		baseEnvName:    "MOYSKLAD_API_BASE_URL",
		tokenEnvName:   "MOYSKLAD_API_TOKEN",
		healthPath:     "/entity/organization?limit=1",
		healthEnvelope: []string{"rows"},
		pullPaths: map[string]string{
			StreamProducts: "/entity/assortment",
			StreamStocks:   "/report/stock/all",
			StreamShops:    "/entity/store",
			StreamOrders:   "/entity/customerorder",
			StreamClients:  "/entity/counterparty",
		},
		pullEnvelopes: map[string][]string{
			StreamProducts: {"rows", "products"},
			StreamStocks:   {"rows", "stocks"},
			StreamShops:    {"rows", "stores"},
			StreamOrders:   {"rows", "orders"},
			StreamClients:  {"rows", "counterparties"},
		},
		pushPaths: map[string]string{
			"goods_receipt":         "/entity/supply",
			"goods_receipts":        "/entity/supply",
			"supply":                "/entity/supply",
			"receipt":               "/entity/supply",
			"receipts":              "/entity/supply",
			"inventory_adjustment":  "/entity/enter",
			"inventory_adjustments": "/entity/enter",
			"inventory":             "/entity/enter",
			StreamStocks:            "/entity/enter",
			"enter":                 "/entity/enter",
			"writeoff":              "/entity/loss",
			"loss":                  "/entity/loss",
			StreamOrders:            "/entity/customerorder",
		},
		defaultPushKey: "inventory_adjustment",
		mapRow:         mapMoySkladRow,
		extractPushID: func(body []byte, prefix string, now time.Time) string {
			if id := extractExternalDocID(body, "", now); id != "" && !strings.HasPrefix(id, "-") {
				return id
			}
			var m map[string]any
			if len(body) > 0 && decodeJSONWithNumbers(body, &m) == nil && m != nil {
				if meta, ok := m["meta"].(map[string]any); ok {
					if href := firstMapStr(meta, "href"); href != "" {
						parts := strings.Split(strings.TrimRight(href, "/"), "/")
						if last := parts[len(parts)-1]; last != "" {
							return last
						}
					}
				}
			}
			return extractExternalDocID(nil, prefix, now)
		},
	}
	return &MoySkladConnector{restERPAdapter: newRESTERPAdapter(cfg, secrets, httpClient)}
}

func mapMoySkladRow(stream string, row map[string]any) (CanonicalRecord, bool) {
	switch stream {
	case StreamProducts:
		extID := firstMapStr(row, "id", "externalCode", "code", "article", "sku")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		priceMinor, ok := extractMinorField(row, "UZS", "price_minor", "sale_price_minor", "price")
		if !ok {
			if sp, has := row["salePrices"].([]any); has && len(sp) > 0 {
				if first, isMap := sp[0].(map[string]any); isMap {
					priceMinor, _ = parseStrictInt64(first["value"])
				}
			}
		}
		barcode := firstMapStr(row, "barcode", "ean13")
		if barcode == "" {
			if bcs, has := row["barcodes"].([]any); has && len(bcs) > 0 {
				if first, isMap := bcs[0].(map[string]any); isMap {
					barcode = firstMapStr(first, "ean13", "ean8", "code128", "gtin")
				}
			}
		}
		qty, _ := extractIntField(row, "quantity_on_hand", "quantity", "stock")
		return CanonicalRecord{
			ObjectType:     ObjectTypeProduct,
			ExternalID:     extID,
			SKU:            firstNonEmpty(firstMapStr(row, "article", "code", "sku", "externalCode"), extID),
			Barcode:        barcode,
			Name:           firstMapStr(row, "name", "title"),
			Currency:       firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			Status:         "ACTIVE",
			PriceMinor:     priceMinor,
			QuantityOnHand: qty,
		}, true

	case StreamStocks:
		prodID := firstMapStr(row, "product_id", "assortmentId", "sku", "article", "code", "externalCode", "id")
		if prodID == "" {
			return CanonicalRecord{}, false
		}
		whID := firstMapStr(row, "warehouse_id", "storeId", "store_id")
		qty, _ := extractIntField(row, "quantity_on_hand", "stock", "quantity", "available")
		extID := prodID
		if whID != "" {
			extID = prodID + "@" + whID
		}
		return CanonicalRecord{
			ObjectType:     ObjectTypeInventory,
			ExternalID:     extID,
			SKU:            prodID,
			WarehouseID:    whID,
			QuantityOnHand: qty,
		}, true

	case StreamShops:
		extID := firstMapStr(row, "id", "externalCode", "code")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		return CanonicalRecord{
			ObjectType:  ObjectTypeShop,
			ExternalID:  extID,
			WarehouseID: extID,
			RetailerID:  extID,
			Name:        firstMapStr(row, "name", "title"),
			Status:      "ACTIVE",
		}, true

	case StreamOrders:
		extID := firstMapStr(row, "id", "externalCode", "name", "order_id")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		totalMinor, ok := extractIntField(row, "total_minor", "sum")
		if !ok {
			totalMinor, _ = extractMinorField(row, "UZS", "amount", "total")
		}
		retID := firstMapStr(row, "client_id", "retailer_id", "agent_id")
		if retID == "" {
			if ag, isMap := row["agent"].(map[string]any); isMap {
				retID = firstMapStr(ag, "id", "externalCode")
			}
		}
		return CanonicalRecord{
			ObjectType: ObjectTypeOrder,
			ExternalID: extID,
			OrderID:    firstNonEmpty(firstMapStr(row, "order_id", "name"), extID),
			RetailerID: retID,
			Currency:   firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			Status:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "OPEN"),
			TotalMinor: totalMinor,
		}, true

	case StreamClients:
		extID := firstMapStr(row, "id", "externalCode", "inn", "code")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		balanceMinor, ok := extractIntField(row, "balance_minor", "balance")
		if !ok {
			balanceMinor, _ = extractMinorField(row, "UZS", "debt")
		}
		return CanonicalRecord{
			ObjectType:   ObjectTypeClient,
			ExternalID:   extID,
			RetailerID:   firstNonEmpty(firstMapStr(row, "retailer_id", "inn"), extID),
			Name:         firstMapStr(row, "name", "legalTitle"),
			Currency:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			BalanceMinor: balanceMinor,
		}, true
	}
	return CanonicalRecord{}, false
}
