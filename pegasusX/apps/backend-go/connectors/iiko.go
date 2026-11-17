package connectors

import (
	"net/http"
	"strings"
)

// IikoConnector implements the Connector interface against iiko HORECA POS/ERP REST API.
type IikoConnector struct {
	*restERPAdapter
}

// NewIikoConnector constructs an IikoConnector backed by SecretResolver and http.Client.
func NewIikoConnector(secrets SecretResolver, httpClient *http.Client) *IikoConnector {
	cfg := restERPConfig{
		provider:       ProviderIiko,
		baseEnvName:    "IIKO_API_BASE_URL",
		tokenEnvName:   "IIKO_API_TOKEN",
		healthPath:     "/api/1/organizations",
		healthEnvelope: []string{"organizations"},
		pullPaths: map[string]string{
			StreamProducts: "/api/1/nomenclature",
			StreamStocks:   "/api/1/stocks",
			StreamShops:    "/api/1/organizations",
			StreamOrders:   "/api/1/orders",
			StreamClients:  "/api/1/customers",
		},
		pullEnvelopes: map[string][]string{
			StreamProducts: {"products", "nomenclature"},
			StreamStocks:   {"stocks"},
			StreamShops:    {"organizations", "shops"},
			StreamOrders:   {"orders"},
			StreamClients:  {"customers", "clients"},
		},
		pushPaths: map[string]string{
			"goods_receipt":         "/api/1/documents/incoming_invoice",
			"goods_receipts":        "/api/1/documents/incoming_invoice",
			"incoming_invoice":      "/api/1/documents/incoming_invoice",
			"receipt":               "/api/1/documents/incoming_invoice",
			"receipts":              "/api/1/documents/incoming_invoice",
			"inventory_adjustment":  "/api/1/inventory/writeoff",
			"inventory_adjustments": "/api/1/inventory/writeoff",
			"writeoff":              "/api/1/inventory/writeoff",
			"inventory":             "/api/1/inventory/writeoff",
			StreamStocks:            "/api/1/inventory/writeoff",
			StreamOrders:            "/api/1/orders/create",
		},
		defaultPushKey: "goods_receipt",
		mapRow:         mapIikoRow,
	}
	return &IikoConnector{restERPAdapter: newRESTERPAdapter(cfg, secrets, httpClient)}
}

func mapIikoRow(stream string, row map[string]any) (CanonicalRecord, bool) {
	switch stream {
	case StreamProducts:
		extID := firstMapStr(row, "id", "productId", "code", "sku")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		priceMinor, _ := extractMinorField(row, "UZS", "price_minor", "defaultSalePrice_minor", "price")
		qty, _ := extractIntField(row, "quantity_on_hand", "amount", "quantity", "stock")
		return CanonicalRecord{
			ObjectType:     ObjectTypeProduct,
			ExternalID:     extID,
			SKU:            firstNonEmpty(firstMapStr(row, "sku", "code"), extID),
			Barcode:        firstMapStr(row, "barcode", "ean"),
			Name:           firstMapStr(row, "name", "title"),
			Currency:       firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			Status:         "ACTIVE",
			PriceMinor:     priceMinor,
			QuantityOnHand: qty,
		}, true

	case StreamStocks:
		prodID := firstMapStr(row, "productId", "product_id", "sku", "code", "id")
		if prodID == "" {
			return CanonicalRecord{}, false
		}
		whID := firstMapStr(row, "warehouse_id", "storeId", "terminalGroupId", "organizationId")
		qty, _ := extractIntField(row, "quantity_on_hand", "balance", "amount", "quantity", "stock")
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
		extID := firstMapStr(row, "id", "organizationId", "code")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		return CanonicalRecord{
			ObjectType:  ObjectTypeShop,
			ExternalID:  extID,
			WarehouseID: extID,
			RetailerID:  extID,
			Name:        firstMapStr(row, "name", "restaurantAddress"),
			Status:      "ACTIVE",
		}, true

	case StreamOrders:
		extID := firstMapStr(row, "id", "orderId", "externalNumber", "number")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		totalMinor, _ := extractMinorField(row, "UZS", "total_minor", "sum_minor", "sum", "total")
		return CanonicalRecord{
			ObjectType: ObjectTypeOrder,
			ExternalID: extID,
			OrderID:    firstNonEmpty(firstMapStr(row, "orderId", "externalNumber"), extID),
			RetailerID: firstMapStr(row, "customerId", "client_id", "retailer_id"),
			Currency:   firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			Status:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "OPEN"),
			TotalMinor: totalMinor,
		}, true

	case StreamClients:
		extID := firstMapStr(row, "id", "customerId", "client_id", "phone")
		if extID == "" {
			return CanonicalRecord{}, false
		}
		balanceMinor, _ := extractMinorField(row, "UZS", "balance_minor", "walletBalance_minor", "balance")
		return CanonicalRecord{
			ObjectType:   ObjectTypeClient,
			ExternalID:   extID,
			RetailerID:   firstNonEmpty(firstMapStr(row, "retailer_id", "customerId"), extID),
			Name:         firstMapStr(row, "name", "surname"),
			Currency:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
			BalanceMinor: balanceMinor,
		}, true
	}
	return CanonicalRecord{}, false
}
