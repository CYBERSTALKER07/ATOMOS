package connectors

import (
	"net/http"
	"strings"
)

// SmartupConnector implements the Connector interface against Smartup CIS distribution ERP REST API.
type SmartupConnector struct {
	*restERPAdapter
}

// NewSmartupConnector constructs a SmartupConnector backed by SecretResolver and http.Client.
func NewSmartupConnector(secrets SecretResolver, httpClient *http.Client) *SmartupConnector {
	cfg := restERPConfig{
		provider:       ProviderSmartup,
		baseEnvName:    "SMARTUP_API_BASE_URL",
		tokenEnvName:   "SMARTUP_API_TOKEN",
		healthPath:     "/v1/warehouses?limit=1",
		healthEnvelope: []string{"warehouses", "shops"},
		pullPaths: map[string]string{
			StreamProducts: "/v1/products",
			StreamStocks:   "/v1/stocks",
			StreamShops:    "/v1/warehouses",
			StreamOrders:   "/v1/orders",
			StreamClients:  "/v1/clients",
		},
		pullEnvelopes: map[string][]string{
			StreamProducts: {"products"},
			StreamStocks:   {"stocks"},
			StreamShops:    {"warehouses", "shops"},
			StreamOrders:   {"orders", "deals"},
			StreamClients:  {"clients", "persons"},
		},
		pushPaths: map[string]string{
			"goods_receipt":         "/v1/goods-receipts",
			"goods_receipts":        "/v1/goods-receipts",
			"receipt":               "/v1/goods-receipts",
			"receipts":              "/v1/goods-receipts",
			"inventory_adjustment":  "/v1/inventory-adjustments",
			"inventory_adjustments": "/v1/inventory-adjustments",
			"inventory":             "/v1/inventory-adjustments",
			StreamStocks:            "/v1/inventory-adjustments",
			"journals":              "/v1/journals",
			"journal":               "/v1/journals",
			StreamProducts:          "/v1/products",
		},
		defaultPushKey: "goods_receipt",
		mapRow:         mapSmartupRow,
	}
	return &SmartupConnector{restERPAdapter: newRESTERPAdapter(cfg, secrets, httpClient)}
}

func mapSmartupRow(stream string, row map[string]any) (CanonicalRecord, bool) {
	switch stream {
	case StreamProducts:
		extID := firstMapStr(row, "product_id", "id", "code", "sku")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		priceMinor, _ := extractMinorField(row, "UZS", "price_minor", "sale_price_minor", "price")
		qty, _ := extractIntField(row, "quantity_on_hand", "balance", "qty", "stock")
		return CanonicalRecord{
			ObjectType:     ObjectTypeProduct,
			ExternalID:     extID,
			SKU:            firstNonEmpty(firstMapStr(row, "sku", "code", "article"), extID),
			Barcode:        firstMapStr(row, "barcode", "ean"),
			Name:           firstMapStr(row, "name", "short_name", "title"),
			Currency:       firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			Status:         firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "ACTIVE"),
			PriceMinor:     priceMinor,
			QuantityOnHand: qty,
		}, true

	case StreamStocks:
		prodID := firstMapStr(row, "product_id", "sku", "code", "id")
		if prodID == "" {
			return CanonicalRecord{}, false
		}
		whID := firstMapStr(row, "warehouse_id", "filial_id", "store_id")
		qty, _ := extractIntField(row, "quantity_on_hand", "balance", "qty", "quantity", "stock")
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
		extID := firstMapStr(row, "warehouse_id", "filial_id", "id", "code")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		return CanonicalRecord{
			ObjectType:  ObjectTypeShop,
			ExternalID:  extID,
			WarehouseID: extID,
			RetailerID:  extID,
			Name:        firstMapStr(row, "name", "filial_name", "title"),
			Status:      firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "ACTIVE"),
		}, true

	case StreamOrders:
		extID := firstMapStr(row, "deal_id", "order_id", "id", "deal_number")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		totalMinor, _ := extractMinorField(row, "UZS", "total_minor", "amount_minor", "total_amount", "sum")
		return CanonicalRecord{
			ObjectType: ObjectTypeOrder,
			ExternalID: extID,
			OrderID:    firstNonEmpty(firstMapStr(row, "order_id", "deal_id"), extID),
			RetailerID: firstMapStr(row, "person_id", "client_id", "retailer_id"),
			Currency:   firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			Status:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "OPEN"),
			TotalMinor: totalMinor,
		}, true

	case StreamClients:
		extID := firstMapStr(row, "person_id", "client_id", "id", "inn", "tin")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		balanceMinor, _ := extractMinorField(row, "UZS", "balance_minor", "debt_minor", "balance", "debt")
		return CanonicalRecord{
			ObjectType:   ObjectTypeClient,
			ExternalID:   extID,
			RetailerID:   firstNonEmpty(firstMapStr(row, "retailer_id", "inn", "tin"), extID),
			Name:         firstMapStr(row, "name", "legal_name"),
			Currency:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			BalanceMinor: balanceMinor,
		}, true
	}
	return CanonicalRecord{}, false
}
