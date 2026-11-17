package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner/adapters/onec"
)

const defaultOnecHTTPTimeout = 15 * time.Second

// OnecConnector adapts partner/adapters/onec CommerceML and multi-account 1C credentials
// onto the canonical M1.9 Connector interface.
type OnecConnector struct {
	httpClient   *http.Client
	secrets      SecretResolver
	envBaseURL   func() string
	envToken     func() string
	isProduction func() bool
	now          func() time.Time
}

// NewOnecConnector constructs a multi-account 1C connector backed by SecretResolver.
func NewOnecConnector(secrets SecretResolver, httpClient *http.Client) *OnecConnector {
	if secrets == nil {
		secrets = NewDefaultSecretResolver(nil)
	}
	return &OnecConnector{
		httpClient:   httpClient,
		secrets:      secrets,
		envBaseURL:   func() string { return os.Getenv("ONEC_API_BASE_URL") },
		envToken:     func() string { return os.Getenv("ONEC_API_TOKEN") },
		isProduction: auth.IsProduction,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func (c *OnecConnector) Provider() string {
	return ProviderOnec
}

func (c *OnecConnector) resolveToken(ctx context.Context, cred ExternalCredential) (string, error) {
	if strings.TrimSpace(cred.SecretRef) != "" {
		if scoped, ok := c.secrets.(interface {
			ResolveForTenant(ctx context.Context, tenantID, secretRef string) (string, error)
		}); ok {
			return scoped.ResolveForTenant(ctx, cred.TenantID, cred.SecretRef)
		}
		return c.secrets.Resolve(ctx, cred.SecretRef)
	}
	if c.envToken != nil {
		if t := strings.TrimSpace(c.envToken()); t != "" {
			return t, nil
		}
	}
	return "", fmt.Errorf("%w: 1c secret_ref or ONEC_API_TOKEN is required", ErrConnectorNotConfigured)
}

func (c *OnecConnector) resolveRemoteConfig(ctx context.Context, cred ExternalCredential) (string, string, error) {
	baseEnv := ""
	if c.envBaseURL != nil {
		baseEnv = c.envBaseURL()
	}
	baseURL := strings.TrimRight(firstNonEmpty(cred.BaseURL, baseEnv), "/")
	if baseURL == "" {
		return "", "", fmt.Errorf("%w: 1c base_url or ONEC_API_BASE_URL is required", ErrConnectorNotConfigured)
	}
	allowPriv := (c != nil && c.httpClient != nil) || allowPrivateConnectorHostsFromEnv()
	if err := ValidateConnectorBaseURL(baseURL, allowPriv); err != nil {
		return "", "", err
	}
	token, err := c.resolveToken(ctx, cred)
	if err != nil {
		return "", "", err
	}
	return baseURL, token, nil
}

func (c *OnecConnector) Health(ctx context.Context, cred ExternalCredential) (HealthResult, error) {
	started := c.now()
	res := HealthResult{
		Provider:     ProviderOnec,
		AccountLabel: normalizeAccountLabel(cred.AccountLabel),
		CheckedAt:    started,
	}
	baseURL, token, err := c.resolveRemoteConfig(ctx, cred)
	if err != nil {
		res.Message = err.Error()
		return res, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		res.Message = err.Error()
		return res, fmt.Errorf("%w: build 1c health request: %v", ErrInvalidInput, err)
	}
	applyAuthHeader(req, firstNonEmpty(cred.AuthType, AuthTypeBasic), token)

	client, _ := selectConnectorHTTPClient(c.httpClient, defaultOnecHTTPTimeout)
	resp, err := client.Do(req)
	res.LatencyMs = c.now().Sub(started).Milliseconds()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			res.Message = ErrConnectorTimeout.Error()
			return res, fmt.Errorf("%w: 1c health check timed out", ErrConnectorTimeout)
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			res.Message = ErrConnectorTimeout.Error()
			return res, fmt.Errorf("%w: 1c health check timed out", ErrConnectorTimeout)
		}
		res.Message = scrubSecrets(err.Error(), token)
		return res, fmt.Errorf("%w: 1c health request failed: %s", ErrConnectorUpstreamError, res.Message)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	res.StatusCode = resp.StatusCode
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		res.Message = fmt.Sprintf("1c returned HTTP %d", resp.StatusCode)
		return res, fmt.Errorf("%w: %s", ErrConnectorAuthFailed, res.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		res.Message = fmt.Sprintf("1c returned HTTP %d", resp.StatusCode)
		return res, fmt.Errorf("%w: %s", ErrConnectorUpstreamError, res.Message)
	}
	res.Healthy = true
	res.Message = "1c endpoint reachable"
	return res, nil
}

func (c *OnecConnector) Pull(ctx context.Context, cred ExternalCredential, req PullRequest) (PullResult, error) {
	var rawXML []byte
	if len(bytes.TrimSpace(req.RawPayload)) > 0 {
		if c.isProduction != nil && c.isProduction() {
			if _, err := c.resolveToken(ctx, cred); err != nil {
				return PullResult{}, err
			}
		}
		rawXML = req.RawPayload
	} else {
		baseURL, token, err := c.resolveRemoteConfig(ctx, cred)
		if err != nil {
			return PullResult{}, err
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
		if err != nil {
			return PullResult{}, fmt.Errorf("%w: build 1c pull request: %v", ErrInvalidInput, err)
		}
		applyAuthHeader(httpReq, firstNonEmpty(cred.AuthType, AuthTypeBasic), token)
		client, _ := selectConnectorHTTPClient(c.httpClient, defaultOnecHTTPTimeout)
		resp, err := client.Do(httpReq)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return PullResult{}, fmt.Errorf("%w: 1c pull timed out", ErrConnectorTimeout)
			}
			return PullResult{}, fmt.Errorf("%w: 1c pull failed: %s", ErrConnectorUpstreamError, scrubSecrets(err.Error(), token))
		}
		defer resp.Body.Close()
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		if readErr != nil {
			return PullResult{}, fmt.Errorf("%w: read 1c body: %v", ErrConnectorUpstreamError, readErr)
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return PullResult{}, fmt.Errorf("%w: 1c returned HTTP %d", ErrConnectorAuthFailed, resp.StatusCode)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return PullResult{}, fmt.Errorf("%w: 1c returned HTTP %d", ErrConnectorUpstreamError, resp.StatusCode)
		}
		rawXML = body
	}

	bundle, err := onec.ExtractCommerceMLBundle(rawXML)
	if err != nil {
		if batch, catErr := onec.ParseCommerceMLCatalog(rawXML); catErr == nil {
			rows := make([]map[string]any, 0, len(batch.Products))
			for _, p := range batch.Products {
				rows = append(rows, map[string]any{
					"external_sku_id": p.ExternalID,
					"name":            p.Name,
					"barcode":         p.Barcode,
					"price_minor":     p.PriceMinor,
					"currency":        p.Currency,
				})
			}
			canonical, mapErr := c.MapSchema(ctx, StreamProducts, rows)
			if mapErr != nil {
				return PullResult{}, mapErr
			}
			return PullResult{
				Provider:     ProviderOnec,
				AccountLabel: normalizeAccountLabel(cred.AccountLabel),
				Stream:       StreamProducts,
				Streams:      map[string][]map[string]any{StreamProducts: rows},
				Canonical:    canonical,
				RecordsCount: int64(len(canonical)),
				StreamCounts: map[string]int64{StreamProducts: int64(len(rows))},
			}, nil
		}
		return PullResult{}, fmt.Errorf("%w: parse 1c commerceml: %v", ErrConnectorBadPayload, err)
	}

	out := PullResult{
		Provider:     ProviderOnec,
		AccountLabel: normalizeAccountLabel(cred.AccountLabel),
		Stream:       firstNonEmpty(strings.ToLower(strings.TrimSpace(req.Stream)), StreamAll),
		Streams:      make(map[string][]map[string]any),
		StreamCounts: make(map[string]int64),
	}
	if err := collectFullCommerceMLStreams(ctx, c, rawXML, &out); err != nil {
		return PullResult{}, err
	}
	_ = bundle
	return out, nil
}

// Push sends approval-gated CommerceML XML or OData JSON journal/catalog write-backs to 1C.
func (c *OnecConnector) Push(ctx context.Context, cred ExternalCredential, req PushRequest) (PushResult, error) {
	baseURL, token, err := c.resolveRemoteConfig(ctx, cred)
	if err != nil {
		return PushResult{}, err
	}

	normStream := strings.ToLower(strings.TrimSpace(req.Stream))
	proto := strings.ToLower(strings.TrimSpace(firstMapStr(cred.Metadata, "protocol")))
	isOData := proto == "odata" || normStream == "odata" || normStream == "odata_journals"
	isJournalXML := !isOData && (normStream == "journals" || normStream == "journal" || normStream == "documents" || normStream == "commerceml_documents" || normStream == "goods_receipt" || normStream == "goods_receipts" || normStream == "inventory_adjustment" || normStream == "inventory_adjustments")

	payload := req.RawPayload
	contentType := "application/xml; charset=utf-8"
	targetURL := baseURL
	recordsPushed := int64(len(req.Records))

	if len(bytes.TrimSpace(payload)) > 0 {
		trimmed := bytes.TrimSpace(payload)
		if trimmed[0] == '{' || trimmed[0] == '[' {
			var generic any
			if err := decodeJSONWithNumbers(trimmed, &generic); err != nil {
				return PushResult{}, fmt.Errorf("%w: invalid 1c json payload: %v", ErrInvalidInput, err)
			}
			if err := validateNoFloats(generic); err != nil {
				return PushResult{}, err
			}
			contentType = "application/json"
		}
		if recordsPushed == 0 {
			recordsPushed = 1
		}
	} else if isOData {
		if len(req.Records) == 0 {
			return PushResult{}, fmt.Errorf("%w: 1c odata push requires records or raw_payload", ErrInvalidInput)
		}
		entries := make([]map[string]any, 0, len(req.Records))
		for _, rec := range req.Records {
			entries = append(entries, map[string]any{
				"Ref_Key":        firstNonEmpty(rec.ExternalID, rec.OrderID, rec.SKU),
				"Description":    firstNonEmpty(rec.Name, rec.SKU, rec.ExternalID),
				"Sku":            rec.SKU,
				"WarehouseId":    rec.WarehouseID,
				"CounterpartyId": rec.RetailerID,
				"Quantity":       rec.QuantityOnHand,
				"AmountMinor":    firstNonZeroInt64(rec.TotalMinor, rec.BalanceMinor, rec.PriceMinor),
				"Currency":       firstNonEmpty(rec.Currency, "UZS"),
				"Status":         firstNonEmpty(rec.Status, "POSTED"),
				"ApprovalId":     strings.TrimSpace(req.ApprovalID),
				"ContentHash":    rec.ContentHash,
			})
		}
		odataBody := map[string]any{
			"@odata.context": "1c.odata.journals.v1",
			"account_label":  normalizeAccountLabel(cred.AccountLabel),
			"stream":         firstNonEmpty(normStream, "odata_journals"),
			"approval_id":    strings.TrimSpace(req.ApprovalID),
			"value":          entries,
		}
		encoded, encErr := json.Marshal(odataBody)
		if encErr != nil {
			return PushResult{}, fmt.Errorf("%w: marshal 1c odata json: %v", ErrInvalidInput, encErr)
		}
		payload = encoded
		contentType = "application/json"
		if odataPath := strings.TrimSpace(firstMapStr(cred.Metadata, "odata_path")); odataPath != "" {
			if !strings.HasPrefix(odataPath, "/") {
				odataPath = "/" + odataPath
			}
			targetURL = baseURL + odataPath
		} else {
			targetURL = baseURL + "/odata/journals"
		}
	} else if isJournalXML {
		if len(req.Records) == 0 {
			return PushResult{}, fmt.Errorf("%w: 1c journal push requires records or raw_payload", ErrInvalidInput)
		}
		type cmlCounterparty struct {
			ID string `xml:"Ид"`
		}
		type cmlDoc struct {
			ID           string            `xml:"Ид"`
			Number       string            `xml:"Номер"`
			Operation    string            `xml:"ХозОперация"`
			Currency     string            `xml:"Валюта"`
			AmountMinor  int64             `xml:"Сумма"`
			Comment      string            `xml:"Комментарий,omitempty"`
			Counterparty []cmlCounterparty `xml:"Контрагенты>Контрагент,omitempty"`
		}
		type cmlInfo struct {
			XMLName       xml.Name `xml:"КоммерческаяИнформация"`
			SchemaVersion string   `xml:"ВерсияСхемы,attr"`
			Documents     []cmlDoc `xml:"Документ"`
		}
		info := cmlInfo{SchemaVersion: "2.10"}
		for _, rec := range req.Records {
			docID := firstNonEmpty(rec.OrderID, rec.ExternalID, rec.SKU)
			var cps []cmlCounterparty
			if rec.RetailerID != "" {
				cps = append(cps, cmlCounterparty{ID: rec.RetailerID})
			}
			info.Documents = append(info.Documents, cmlDoc{
				ID:           docID,
				Number:       docID,
				Operation:    firstNonEmpty(rec.Status, normStream),
				Currency:     firstNonEmpty(rec.Currency, "UZS"),
				AmountMinor:  firstNonZeroInt64(rec.TotalMinor, rec.BalanceMinor, rec.PriceMinor),
				Comment:      firstNonEmpty(rec.Name, "approval:"+strings.TrimSpace(req.ApprovalID)),
				Counterparty: cps,
			})
		}
		encoded, encErr := xml.MarshalIndent(info, "", "  ")
		if encErr != nil {
			return PushResult{}, fmt.Errorf("%w: marshal 1c journal xml: %v", ErrInvalidInput, encErr)
		}
		payload = append([]byte(xml.Header), encoded...)
	} else {
		if len(req.Records) == 0 {
			return PushResult{}, fmt.Errorf("%w: 1c push requires records or raw_payload", ErrInvalidInput)
		}
		type xmlItem struct {
			ID         string `xml:"Ид"`
			Name       string `xml:"Наименование"`
			Barcode    string `xml:"Штрихкод,omitempty"`
			PriceMinor int64  `xml:"Цена"`
			Currency   string `xml:"Валюта,omitempty"`
		}
		type xmlDoc struct {
			XMLName  xml.Name  `xml:"Каталог"`
			Products []xmlItem `xml:"Товары>Товар"`
		}
		doc := xmlDoc{}
		for _, rec := range req.Records {
			doc.Products = append(doc.Products, xmlItem{
				ID:         firstNonEmpty(rec.ExternalID, rec.SKU),
				Name:       rec.Name,
				Barcode:    rec.Barcode,
				PriceMinor: rec.PriceMinor,
				Currency:   firstNonEmpty(rec.Currency, "UZS"),
			})
		}
		encoded, encErr := xml.MarshalIndent(doc, "", "  ")
		if encErr != nil {
			return PushResult{}, fmt.Errorf("%w: marshal 1c xml: %v", ErrInvalidInput, encErr)
		}
		payload = append([]byte(xml.Header), encoded...)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return PushResult{}, fmt.Errorf("%w: build 1c push request: %v", ErrInvalidInput, err)
	}
	httpReq.Header.Set("Content-Type", contentType)
	if aid := strings.TrimSpace(req.ApprovalID); aid != "" {
		httpReq.Header.Set("X-Atomos-Approval-Id", aid)
	}
	applyAuthHeader(httpReq, firstNonEmpty(cred.AuthType, AuthTypeBasic), token)

	client, _ := selectConnectorHTTPClient(c.httpClient, defaultOnecHTTPTimeout)
	resp, err := client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return PushResult{}, fmt.Errorf("%w: 1c push timed out", ErrConnectorTimeout)
		}
		return PushResult{}, fmt.Errorf("%w: 1c push failed: %s", ErrConnectorUpstreamError, scrubSecrets(err.Error(), token))
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return PushResult{}, fmt.Errorf("%w: 1c returned HTTP %d", ErrConnectorAuthFailed, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PushResult{}, fmt.Errorf("%w: 1c returned HTTP %d", ErrConnectorUpstreamError, resp.StatusCode)
	}

	now := time.Now().UTC()
	if c.now != nil {
		now = c.now()
	}
	docID := extractExternalDocID(respBody, "1c-push", now)
	msg := "commerceml payload pushed to 1c"
	if isOData {
		msg = "odata journal payload pushed to 1c"
	} else if isJournalXML {
		msg = "commerceml journal payload pushed to 1c"
	}
	return PushResult{
		Provider:      ProviderOnec,
		AccountLabel:  normalizeAccountLabel(cred.AccountLabel),
		Stream:        firstNonEmpty(normStream, onec.JournalDialect),
		RecordsPushed: recordsPushed,
		ExternalDocID: docID,
		Status:        RunStatusCompleted,
		Message:       msg,
	}, nil
}

func firstNonZeroInt64(vals ...int64) int64 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}

func (c *OnecConnector) MapSchema(_ context.Context, stream string, rawRows []map[string]any) ([]CanonicalRecord, error) {
	st := mapCommerceMLStreamName(stream)
	if st == "" {
		st = strings.ToLower(strings.TrimSpace(stream))
	}
	out := make([]CanonicalRecord, 0, len(rawRows))
	for _, row := range rawRows {
		if row == nil {
			continue
		}
		switch st {
		case StreamProducts:
			extID := firstMapStr(row, "external_sku_id", "sku", "id", "product_id")
			if extID == "" {
				continue
			}
			priceMinor, _ := extractMinorField(row, "UZS", "price_minor", "price")
			qty, _ := extractIntField(row, "quantity_on_hand", "qty")
			rec := CanonicalRecord{
				ObjectType:     ObjectTypeProduct,
				ExternalID:     extID,
				SKU:            firstNonEmpty(firstMapStr(row, "sku", "external_sku_id"), extID),
				Barcode:        firstMapStr(row, "barcode"),
				Name:           firstMapStr(row, "name"),
				Currency:       firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
				Status:         "ACTIVE",
				PriceMinor:     priceMinor,
				QuantityOnHand: qty,
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		case StreamStocks:
			extSKU := firstMapStr(row, "external_sku_id", "sku", "product_id", "id")
			if extSKU == "" {
				continue
			}
			whID := firstMapStr(row, "warehouse_id", "store_id")
			qty, _ := extractIntField(row, "quantity_on_hand", "qty", "stock")
			extID := extSKU
			if whID != "" {
				extID = extSKU + "@" + whID
			}
			rec := CanonicalRecord{
				ObjectType:     ObjectTypeInventory,
				ExternalID:     extID,
				SKU:            extSKU,
				WarehouseID:    whID,
				QuantityOnHand: qty,
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		case StreamClients:
			extID := firstMapStr(row, "external_customer_id", "inn", "id", "client_id", "name")
			if extID == "" {
				continue
			}
			balanceMinor, _ := extractMinorField(row, "UZS", "balance_minor", "balance", "debt")
			rec := CanonicalRecord{
				ObjectType:   ObjectTypeClient,
				ExternalID:   extID,
				RetailerID:   firstNonEmpty(firstMapStr(row, "retailer_id", "inn"), extID),
				Name:         firstMapStr(row, "name", "official_name"),
				Currency:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
				BalanceMinor: balanceMinor,
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		case StreamOrders:
			extID := firstMapStr(row, "external_order_id", "external_order_no", "id", "order_id")
			if extID == "" {
				continue
			}
			totalMinor, _ := extractMinorField(row, "UZS", "total_minor", "total", "amount", "sum")
			rec := CanonicalRecord{
				ObjectType: ObjectTypeOrder,
				ExternalID: extID,
				OrderID:    firstNonEmpty(firstMapStr(row, "external_order_no", "order_id"), extID),
				RetailerID: firstMapStr(row, "external_customer_id", "inn", "retailer_id"),
				Currency:   firstNonEmpty(strings.ToUpper(firstMapStr(row, "currency")), "UZS"),
				Status:     firstNonEmpty(strings.ToUpper(firstMapStr(row, "status")), "OPEN"),
				TotalMinor: totalMinor,
			}
			rec.ContentHash = canonicalContentHash(rec)
			out = append(out, rec)

		default:
			return nil, fmt.Errorf("%w: unsupported 1c stream %q", ErrInvalidInput, stream)
		}
	}
	return out, nil
}

func mapCommerceMLStreamName(streamName string) string {
	switch strings.ToLower(strings.TrimSpace(streamName)) {
	case "commerceml_products", StreamProducts:
		return StreamProducts
	case "commerceml_offers_stock", StreamStocks:
		return StreamStocks
	case "commerceml_customers", StreamClients:
		return StreamClients
	case "commerceml_documents", StreamOrders:
		return StreamOrders
	default:
		return ""
	}
}
