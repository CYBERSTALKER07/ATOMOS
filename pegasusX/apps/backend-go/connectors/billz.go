package connectors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBillzHTTPTimeout = 10 * time.Second

// BillzConnector implements the read-only Billz v1/v2 REST integration under Connector.
type BillzConnector struct {
	httpClient *http.Client
	secrets    SecretResolver
	envBaseURL func() string
	envToken   func() string
	now        func() time.Time
}

// NewBillzConnector constructs a BillzConnector backed by SecretResolver and http.Client.
func NewBillzConnector(secrets SecretResolver, httpClient *http.Client) *BillzConnector {
	if secrets == nil {
		secrets = NewDefaultSecretResolver(nil)
	}
	return &BillzConnector{
		httpClient: httpClient,
		secrets:    secrets,
		envBaseURL: func() string { return os.Getenv("BILLZ_API_BASE_URL") },
		envToken:   func() string { return os.Getenv("BILLZ_API_TOKEN") },
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (b *BillzConnector) Provider() string {
	return ProviderBillz
}

func (b *BillzConnector) resolveConfig(ctx context.Context, cred ExternalCredential) (string, string, error) {
	return resolveEndpointConfig(ctx, cred, b.secrets, b.envBaseURL, b.envToken, ProviderBillz, "BILLZ_API_BASE_URL", "BILLZ_API_TOKEN", b != nil && b.httpClient != nil)
}

func (b *BillzConnector) Health(ctx context.Context, cred ExternalCredential) (HealthResult, error) {
	started := b.now()
	res := HealthResult{
		Provider:     ProviderBillz,
		AccountLabel: normalizeAccountLabel(cred.AccountLabel),
		CheckedAt:    started,
	}
	baseURL, token, err := b.resolveConfig(ctx, cred)
	if err != nil {
		res.Message = err.Error()
		return res, err
	}

	statusCode, body, callErr := b.doGET(ctx, baseURL+"/v1/shops?limit=1", cred.AuthType, token)
	res.LatencyMs = b.now().Sub(started).Milliseconds()
	res.StatusCode = statusCode
	if callErr != nil {
		res.Message = scrubSecrets(callErr.Error(), token)
		return res, callErr
	}
	if _, parseErr := extractBillzRows(body, StreamShops); parseErr != nil {
		res.Message = parseErr.Error()
		return res, parseErr
	}
	res.Healthy = true
	res.Message = "billz api reachable"
	return res, nil
}

func (b *BillzConnector) Pull(ctx context.Context, cred ExternalCredential, req PullRequest) (PullResult, error) {
	baseURL, token, err := b.resolveConfig(ctx, cred)
	if err != nil {
		return PullResult{}, err
	}

	stream := normalizeBillzStream(req.Stream)
	targets := []string{stream}
	if stream == StreamAll {
		targets = []string{StreamProducts, StreamStocks, StreamShops, StreamOrders, StreamClients}
	}

	out := PullResult{
		Provider:     ProviderBillz,
		AccountLabel: normalizeAccountLabel(cred.AccountLabel),
		Stream:       stream,
		Streams:      make(map[string][]map[string]any, len(targets)),
		StreamCounts: make(map[string]int64, len(targets)),
	}

	for _, st := range targets {
		endpoint, ok := billzEndpointForStream(st)
		if !ok {
			return PullResult{}, fmt.Errorf("%w: unsupported billz stream %q", ErrInvalidInput, st)
		}
		if req.Limit > 0 {
			endpoint = fmt.Sprintf("%s?limit=%d", endpoint, req.Limit)
		}
		_, body, callErr := b.doGET(ctx, baseURL+endpoint, cred.AuthType, token)
		if callErr != nil {
			return PullResult{}, callErr
		}
		rows, parseErr := extractBillzRows(body, st)
		if parseErr != nil {
			return PullResult{}, parseErr
		}
		out.Streams[st] = rows
		out.StreamCounts[st] = int64(len(rows))
		out.RawCount += int64(len(rows))

		mapped, mapErr := b.MapSchema(ctx, st, rows)
		if mapErr != nil {
			return PullResult{}, mapErr
		}
		out.Canonical = append(out.Canonical, mapped...)
	}
	out.RecordsCount = int64(len(out.Canonical))
	return out, nil
}

// Push performs an approval-gated write-back to Billz (goods receipt, inventory adjustment, or products).
func (b *BillzConnector) Push(ctx context.Context, cred ExternalCredential, req PushRequest) (PushResult, error) {
	baseURL, token, err := b.resolveConfig(ctx, cred)
	if err != nil {
		return PushResult{}, err
	}

	stream := normalizeBillzPushStream(req.Stream)
	endpoint, ok := billzPushEndpointForStream(stream)
	if !ok {
		return PushResult{}, fmt.Errorf("%w: unsupported billz push stream %q", ErrInvalidInput, req.Stream)
	}

	body, count, err := prepareJSONPushBody(ProviderBillz, cred, req, stream)
	if err != nil {
		return PushResult{}, err
	}

	headers := map[string]string{}
	if aid := strings.TrimSpace(req.ApprovalID); aid != "" {
		headers["X-Atomos-Approval-Id"] = aid
	}
	_, respBody, callErr := doHTTPRequest(ctx, b.httpClient, ProviderBillz, http.MethodPost, baseURL+endpoint, cred.AuthType, token, "application/json", body, headers)
	if callErr != nil {
		return PushResult{}, callErr
	}

	now := time.Now().UTC()
	if b.now != nil {
		now = b.now()
	}
	return PushResult{
		Provider:      ProviderBillz,
		AccountLabel:  normalizeAccountLabel(cred.AccountLabel),
		Stream:        stream,
		RecordsPushed: count,
		ExternalDocID: extractExternalDocID(respBody, "billz-push", now),
		Status:        RunStatusCompleted,
		Message:       "write-back pushed to billz",
	}, nil
}

func (b *BillzConnector) MapSchema(_ context.Context, stream string, rawRows []map[string]any) ([]CanonicalRecord, error) {
	st := normalizeBillzStream(stream)
	out := make([]CanonicalRecord, 0, len(rawRows))
	for _, row := range rawRows {
		if row == nil {
			continue
		}
		switch st {
		case StreamProducts:
			extID := firstMapStr(row, "id", "product_id", "sku_id", "sku", "external_id")
			if extID == "" {
				continue
			}
			priceMinor, ok := extractMinorField(row, "UZS", "price_minor", "sale_price_minor", "price", "sale_price")
			if !ok {
				priceMinor = 0
			}
			qty, _ := extractIntField(row, "quantity_on_hand", "qty", "quantity", "stock")
			rec := CanonicalRecord{
				ObjectType:     ObjectTypeProduct,
				ExternalID:     extID,
				SKU:            firstNonEmpty(firstMapStr(row, "sku", "article", "code"), extID),
				Barcode:        firstMapStr(row, "barcode", "ean"),
				Name:           firstMapStr(row, "name", "title", "product_name"),
				Currency:       firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
				Status:         firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "ACTIVE"),
				PriceMinor:     priceMinor,
				QuantityOnHand: qty,
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		case StreamStocks:
			prodID := firstMapStr(row, "product_id", "sku_id", "sku", "id", "external_id")
			if prodID == "" {
				continue
			}
			whID := firstMapStr(row, "warehouse_id", "shop_id", "branch_id", "store_id")
			qty, _ := extractIntField(row, "quantity_on_hand", "quantity", "qty", "stock", "available", "balance")
			extID := prodID
			if whID != "" {
				extID = prodID + "@" + whID
			}
			rec := CanonicalRecord{
				ObjectType:     ObjectTypeInventory,
				ExternalID:     extID,
				SKU:            prodID,
				WarehouseID:    whID,
				QuantityOnHand: qty,
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		case StreamShops:
			extID := firstMapStr(row, "id", "shop_id", "retailer_id", "branch_id")
			if extID == "" {
				continue
			}
			rec := CanonicalRecord{
				ObjectType:  ObjectTypeShop,
				ExternalID:  extID,
				WarehouseID: extID,
				RetailerID:  extID,
				Name:        firstMapStr(row, "name", "title", "shop_name"),
				Status:      firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "ACTIVE"),
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		case StreamOrders:
			extID := firstMapStr(row, "id", "order_id", "external_id", "number")
			if extID == "" {
				continue
			}
			totalMinor, _ := extractMinorField(row, "UZS", "total_minor", "amount_minor", "total", "amount", "sum")
			rec := CanonicalRecord{
				ObjectType: ObjectTypeOrder,
				ExternalID: extID,
				OrderID:    firstNonEmpty(firstMapStr(row, "order_id"), extID),
				RetailerID: firstMapStr(row, "client_id", "retailer_id", "customer_id", "shop_id"),
				Currency:   firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
				Status:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "OPEN"),
				TotalMinor: totalMinor,
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		case StreamClients:
			extID := firstMapStr(row, "id", "client_id", "retailer_id", "customer_id")
			if extID == "" {
				continue
			}
			balanceMinor, _ := extractMinorField(row, "UZS", "balance_minor", "debt_minor", "ar_balance_minor", "balance", "debt")
			rec := CanonicalRecord{
				ObjectType:   ObjectTypeClient,
				ExternalID:   extID,
				RetailerID:   firstNonEmpty(firstMapStr(row, "retailer_id"), extID),
				OrderID:      firstMapStr(row, "invoice_id", "order_id"),
				Name:         firstMapStr(row, "name", "full_name", "company_name"),
				Currency:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
				BalanceMinor: balanceMinor,
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		default:
			return nil, fmt.Errorf("%w: unsupported billz stream %q", ErrInvalidInput, stream)
		}
	}
	return out, nil
}

func (b *BillzConnector) doGET(ctx context.Context, rawURL, authType, token string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: build billz request: %v", ErrInvalidInput, err)
	}
	req.Header.Set("Accept", "application/json")
	applyAuthHeader(req, authType, token)

	client, _ := selectConnectorHTTPClient(b.httpClient, defaultBillzHTTPTimeout)
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, nil, fmt.Errorf("%w: billz request timed out", ErrConnectorTimeout)
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return 0, nil, fmt.Errorf("%w: billz network timeout", ErrConnectorTimeout)
		}
		return 0, nil, fmt.Errorf("%w: billz request error: %s", ErrConnectorUpstreamError, scrubSecrets(err.Error(), token))
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if readErr != nil {
		return resp.StatusCode, nil, fmt.Errorf("%w: read billz body: %v", ErrConnectorUpstreamError, readErr)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return resp.StatusCode, body, fmt.Errorf("%w: billz returned HTTP %d", ErrConnectorAuthFailed, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, body, fmt.Errorf("%w: billz returned HTTP %d", ErrConnectorUpstreamError, resp.StatusCode)
	}
	return resp.StatusCode, body, nil
}

func applyAuthHeader(req *http.Request, authType, token string) {
	switch normalizeAuthType(authType) {
	case AuthTypeAPIKey:
		req.Header.Set("X-Api-Key", token)
	case AuthTypeBasic:
		if strings.Contains(token, ":") {
			parts := strings.SplitN(token, ":", 2)
			req.SetBasicAuth(parts[0], parts[1])
		} else {
			req.Header.Set("Authorization", "Basic "+token)
		}
	case AuthTypeNone:
		// Intentionally omit Authorization header only when auth_type=none.
	default:
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func billzEndpointForStream(stream string) (string, bool) {
	switch normalizeBillzStream(stream) {
	case StreamProducts:
		return "/v1/products", true
	case StreamStocks:
		return "/v1/stocks", true
	case StreamShops:
		return "/v1/shops", true
	case StreamOrders:
		return "/v1/orders", true
	case StreamClients:
		return "/v1/clients", true
	default:
		return "", false
	}
}

func normalizeBillzPushStream(stream string) string {
	s := strings.ToLower(strings.TrimSpace(stream))
	if s == "" || s == StreamAll {
		return "inventory_adjustment"
	}
	return s
}

func billzPushEndpointForStream(stream string) (string, bool) {
	switch normalizeBillzPushStream(stream) {
	case "goods_receipt", "goods_receipts", "receipts", "receipt", "inbound":
		return "/v1/goods-receipts", true
	case "inventory_adjustment", "inventory_adjustments", "inventory", StreamStocks:
		return "/v1/inventory-adjustments", true
	case StreamProducts, "catalog":
		return "/v1/products", true
	default:
		return "", false
	}
}

func normalizeBillzStream(stream string) string {
	s := strings.ToLower(strings.TrimSpace(stream))
	if s == "" {
		return StreamAll
	}
	return s
}

func extractBillzRows(body []byte, stream string) ([]map[string]any, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, fmt.Errorf("%w: empty billz response body", ErrConnectorBadPayload)
	}
	var raw any
	if err := decodeJSONWithNumbers([]byte(trimmed), &raw); err != nil {
		return nil, fmt.Errorf("%w: decode billz json: %v", ErrConnectorBadPayload, err)
	}
	switch tv := raw.(type) {
	case []any:
		return toMapSlice(tv)
	case map[string]any:
		keys := []string{stream, "items", "data", "rows", "results"}
		for _, k := range keys {
			if sub, ok := tv[k]; ok && sub != nil {
				if arr, isArr := sub.([]any); isArr {
					return toMapSlice(arr)
				}
			}
		}
		if len(tv) == 0 {
			return []map[string]any{}, nil
		}
		// Single object envelope with no list key: wrap if it looks like a single record, else empty.
		if _, hasErr := tv["error"]; hasErr {
			return nil, fmt.Errorf("%w: upstream error payload", ErrConnectorUpstreamError)
		}
		return []map[string]any{}, nil
	default:
		return nil, fmt.Errorf("%w: unexpected billz json root %T", ErrConnectorBadPayload, raw)
	}
}

func toMapSlice(arr []any) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: expected object row, got %T", ErrConnectorBadPayload, item)
		}
		out = append(out, m)
	}
	return out, nil
}

func firstMapStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}

func extractIntField(m map[string]any, keys ...string) (int64, bool) {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		if n, parsed := parseStrictInt64(v); parsed {
			return n, true
		}
	}
	return 0, false
}

func extractMinorField(m map[string]any, _ string, keys ...string) (int64, bool) {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		if strings.HasSuffix(k, "_minor") {
			if n, parsed := parseStrictInt64(v); parsed {
				return n, true
			}
			continue
		}
		if n, parsed := parseDecimalToMinorUnits(v, 2); parsed {
			return n, true
		}
	}
	return 0, false
}

func canonicalContentHash(rec CanonicalRecord) string {
	return sha256HexString(
		rec.ObjectType,
		rec.ExternalID,
		rec.SKU,
		rec.Barcode,
		rec.Name,
		rec.WarehouseID,
		rec.OrderID,
		rec.RetailerID,
		rec.Currency,
		rec.Status,
		fmt.Sprintf("%d", rec.QuantityOnHand),
		fmt.Sprintf("%d", rec.PriceMinor),
		fmt.Sprintf("%d", rec.TotalMinor),
		fmt.Sprintf("%d", rec.BalanceMinor),
	)
}
