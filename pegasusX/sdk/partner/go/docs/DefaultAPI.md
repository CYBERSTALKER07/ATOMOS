# \DefaultAPI

All URIs are relative to *https://api-ssmr.pegasusx.app*

Method | HTTP request | Description
------------- | ------------- | -------------
[**IssuePartnerApiKey**](DefaultAPI.md#IssuePartnerApiKey) | **Post** /v1/admin/partner-keys | Issue partner API key (human JWT)
[**ListPartnerApiKeys**](DefaultAPI.md#ListPartnerApiKeys) | **Get** /v1/admin/partner-keys | List partner keys for caller tenant
[**PartnerAs2Receive**](DefaultAPI.md#PartnerAs2Receive) | **Post** /partner/v1/as2 | AS2 receive (EDI-lite ORDERS)
[**PartnerCreateExport**](DefaultAPI.md#PartnerCreateExport) | **Post** /partner/v1/exports | Create async bulk export job
[**PartnerCreateOrder**](DefaultAPI.md#PartnerCreateOrder) | **Post** /partner/v1/orders | Create order
[**PartnerCreateWebhook**](DefaultAPI.md#PartnerCreateWebhook) | **Post** /partner/v1/webhooks | Create outbound webhook subscription
[**PartnerDeactivateWebhook**](DefaultAPI.md#PartnerDeactivateWebhook) | **Delete** /partner/v1/webhooks/{subscriptionID} | Deactivate webhook subscription
[**PartnerGetAs2Config**](DefaultAPI.md#PartnerGetAs2Config) | **Get** /partner/v1/as2/config | Get tenant AS2 station config
[**PartnerGetCoa**](DefaultAPI.md#PartnerGetCoa) | **Get** /partner/v1/coa | Get tenant chart of accounts for journals exports
[**PartnerGetEdiDocument**](DefaultAPI.md#PartnerGetEdiDocument) | **Get** /partner/v1/edi/documents/{documentID} | Get EDI document status
[**PartnerGetExport**](DefaultAPI.md#PartnerGetExport) | **Get** /partner/v1/exports/{jobID} | Get export job status (+ signed download_url when SUCCEEDED)
[**PartnerGetOrder**](DefaultAPI.md#PartnerGetOrder) | **Get** /partner/v1/orders/{orderID} | Get order (tenant-scoped; IDOR fail-closed)
[**PartnerInventoryAvailability**](DefaultAPI.md#PartnerInventoryAvailability) | **Get** /partner/v1/inventory/availability | On-hand minus reserved availability
[**PartnerListCatalog**](DefaultAPI.md#PartnerListCatalog) | **Get** /partner/v1/catalog | Catalog with resolved prices / stock enrichment
[**PartnerListDeadLetter**](DefaultAPI.md#PartnerListDeadLetter) | **Get** /partner/v1/webhooks/dead-letter | List dead-letter delivery attempts
[**PartnerListEdiDocuments**](DefaultAPI.md#PartnerListEdiDocuments) | **Get** /partner/v1/edi/documents | List recent EDI-lite documents for the key tenant
[**PartnerListExports**](DefaultAPI.md#PartnerListExports) | **Get** /partner/v1/exports | List recent export jobs for the key tenant
[**PartnerListWebhooks**](DefaultAPI.md#PartnerListWebhooks) | **Get** /partner/v1/webhooks | List webhook subscriptions (secret omitted; secret_prefix only)
[**PartnerOAuthToken**](DefaultAPI.md#PartnerOAuthToken) | **Post** /partner/v1/oauth/token | OAuth2 client_credentials token
[**PartnerPOSDemandFeed**](DefaultAPI.md#PartnerPOSDemandFeed) | **Post** /partner/v1/demand/pos-feed | Ingest POS sell-through demand signals (retail chain → planning)
[**PartnerPingWebhook**](DefaultAPI.md#PartnerPingWebhook) | **Post** /partner/v1/webhooks/{subscriptionID}/ping | Send signed test delivery
[**PartnerPutAs2Config**](DefaultAPI.md#PartnerPutAs2Config) | **Put** /partner/v1/as2/config | Upsert tenant AS2 station config
[**PartnerPutCoa**](DefaultAPI.md#PartnerPutCoa) | **Put** /partner/v1/coa | Upsert tenant chart of accounts for journals exports
[**PartnerReplayDeadLetter**](DefaultAPI.md#PartnerReplayDeadLetter) | **Post** /partner/v1/webhooks/dead-letter/{attemptID}/replay | Requeue a DEAD delivery attempt (full retry budget)
[**PartnerReplayEdiDocument**](DefaultAPI.md#PartnerReplayEdiDocument) | **Post** /partner/v1/edi/documents/{documentID}/replay | Requeue FAILED/EMITTED EDI document
[**PartnerRotateWebhookSecret**](DefaultAPI.md#PartnerRotateWebhookSecret) | **Post** /partner/v1/webhooks/{subscriptionID}/rotate-secret | Rotate HMAC signing secret (returned once)
[**PartnerUpsertPrices**](DefaultAPI.md#PartnerUpsertPrices) | **Put** /partner/v1/catalog/prices | Batch upsert list prices by external_id
[**PartnerUpsertProducts**](DefaultAPI.md#PartnerUpsertProducts) | **Put** /partner/v1/catalog/products | Batch upsert products by external_id (ProductId &#x3D;&#x3D; external_id)
[**PartnerUpsertStock**](DefaultAPI.md#PartnerUpsertStock) | **Put** /partner/v1/inventory/stock | Batch set absolute on-hand stock by external_id × warehouse



## IssuePartnerApiKey

> IssuePartnerApiKey(ctx).IssuePartnerApiKeyRequest(issuePartnerApiKeyRequest).Execute()

Issue partner API key (human JWT)

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	issuePartnerApiKeyRequest := *openapiclient.NewIssuePartnerApiKeyRequest() // IssuePartnerApiKeyRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.IssuePartnerApiKey(context.Background()).IssuePartnerApiKeyRequest(issuePartnerApiKeyRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.IssuePartnerApiKey``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiIssuePartnerApiKeyRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **issuePartnerApiKeyRequest** | [**IssuePartnerApiKeyRequest**](IssuePartnerApiKeyRequest.md) |  | 

### Return type

 (empty response body)

### Authorization

[BearerJWT](../README.md#BearerJWT)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListPartnerApiKeys

> ListPartnerApiKeys(ctx).Execute()

List partner keys for caller tenant

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.ListPartnerApiKeys(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ListPartnerApiKeys``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListPartnerApiKeysRequest struct via the builder pattern


### Return type

 (empty response body)

### Authorization

[BearerJWT](../README.md#BearerJWT)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerAs2Receive

> PartnerAs2Receive(ctx).Execute()

AS2 receive (EDI-lite ORDERS)



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerAs2Receive(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerAs2Receive``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerAs2ReceiveRequest struct via the builder pattern


### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerCreateExport

> PartnerExportJob PartnerCreateExport(ctx).PartnerCreateExportRequest(partnerCreateExportRequest).Execute()

Create async bulk export job

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	partnerCreateExportRequest := *openapiclient.NewPartnerCreateExportRequest("Resource_example") // PartnerCreateExportRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerCreateExport(context.Background()).PartnerCreateExportRequest(partnerCreateExportRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerCreateExport``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerCreateExport`: PartnerExportJob
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerCreateExport`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerCreateExportRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **partnerCreateExportRequest** | [**PartnerCreateExportRequest**](PartnerCreateExportRequest.md) |  | 

### Return type

[**PartnerExportJob**](PartnerExportJob.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerCreateOrder

> PartnerCreateOrderResponse PartnerCreateOrder(ctx).IdempotencyKey(idempotencyKey).PartnerCreateOrderRequest(partnerCreateOrderRequest).Execute()

Create order

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	idempotencyKey := "idempotencyKey_example" // string | 
	partnerCreateOrderRequest := *openapiclient.NewPartnerCreateOrderRequest([]openapiclient.PartnerCreateOrderRequestLineItemsInner{*openapiclient.NewPartnerCreateOrderRequestLineItemsInner()}, float32(123), float32(123)) // PartnerCreateOrderRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerCreateOrder(context.Background()).IdempotencyKey(idempotencyKey).PartnerCreateOrderRequest(partnerCreateOrderRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerCreateOrder``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerCreateOrder`: PartnerCreateOrderResponse
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerCreateOrder`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerCreateOrderRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **idempotencyKey** | **string** |  | 
 **partnerCreateOrderRequest** | [**PartnerCreateOrderRequest**](PartnerCreateOrderRequest.md) |  | 

### Return type

[**PartnerCreateOrderResponse**](PartnerCreateOrderResponse.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerCreateWebhook

> PartnerCreateWebhook(ctx).PartnerCreateWebhookRequest(partnerCreateWebhookRequest).Execute()

Create outbound webhook subscription

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	partnerCreateWebhookRequest := *openapiclient.NewPartnerCreateWebhookRequest("Url_example") // PartnerCreateWebhookRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerCreateWebhook(context.Background()).PartnerCreateWebhookRequest(partnerCreateWebhookRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerCreateWebhook``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerCreateWebhookRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **partnerCreateWebhookRequest** | [**PartnerCreateWebhookRequest**](PartnerCreateWebhookRequest.md) |  | 

### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerDeactivateWebhook

> PartnerDeactivateWebhook(ctx, subscriptionID).Execute()

Deactivate webhook subscription

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	subscriptionID := "subscriptionID_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerDeactivateWebhook(context.Background(), subscriptionID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerDeactivateWebhook``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**subscriptionID** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerDeactivateWebhookRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerGetAs2Config

> PartnerAs2Config PartnerGetAs2Config(ctx).Execute()

Get tenant AS2 station config

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerGetAs2Config(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerGetAs2Config``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerGetAs2Config`: PartnerAs2Config
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerGetAs2Config`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerGetAs2ConfigRequest struct via the builder pattern


### Return type

[**PartnerAs2Config**](PartnerAs2Config.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerGetCoa

> PartnerCoaMap PartnerGetCoa(ctx).Execute()

Get tenant chart of accounts for journals exports

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerGetCoa(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerGetCoa``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerGetCoa`: PartnerCoaMap
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerGetCoa`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerGetCoaRequest struct via the builder pattern


### Return type

[**PartnerCoaMap**](PartnerCoaMap.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerGetEdiDocument

> PartnerEdiDocument PartnerGetEdiDocument(ctx, documentID).Execute()

Get EDI document status

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	documentID := "documentID_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerGetEdiDocument(context.Background(), documentID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerGetEdiDocument``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerGetEdiDocument`: PartnerEdiDocument
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerGetEdiDocument`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**documentID** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerGetEdiDocumentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**PartnerEdiDocument**](PartnerEdiDocument.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerGetExport

> PartnerExportJob PartnerGetExport(ctx, jobID).Execute()

Get export job status (+ signed download_url when SUCCEEDED)

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	jobID := "jobID_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerGetExport(context.Background(), jobID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerGetExport``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerGetExport`: PartnerExportJob
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerGetExport`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**jobID** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerGetExportRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**PartnerExportJob**](PartnerExportJob.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerGetOrder

> PartnerGetOrder(ctx, orderID).Execute()

Get order (tenant-scoped; IDOR fail-closed)

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	orderID := "orderID_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerGetOrder(context.Background(), orderID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerGetOrder``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orderID** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerGetOrderRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerInventoryAvailability

> PartnerInventoryAvailability(ctx).SupplierId(supplierId).RetailerId(retailerId).ProductIds(productIds).Execute()

On-hand minus reserved availability

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	supplierId := "supplierId_example" // string |  (optional)
	retailerId := "retailerId_example" // string |  (optional)
	productIds := "productIds_example" // string | Comma-separated product IDs (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerInventoryAvailability(context.Background()).SupplierId(supplierId).RetailerId(retailerId).ProductIds(productIds).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerInventoryAvailability``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerInventoryAvailabilityRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **supplierId** | **string** |  | 
 **retailerId** | **string** |  | 
 **productIds** | **string** | Comma-separated product IDs | 

### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerListCatalog

> PartnerListCatalog(ctx).SupplierId(supplierId).RetailerId(retailerId).CategoryId(categoryId).Execute()

Catalog with resolved prices / stock enrichment

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	supplierId := "supplierId_example" // string |  (optional)
	retailerId := "retailerId_example" // string |  (optional)
	categoryId := "categoryId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerListCatalog(context.Background()).SupplierId(supplierId).RetailerId(retailerId).CategoryId(categoryId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerListCatalog``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerListCatalogRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **supplierId** | **string** |  | 
 **retailerId** | **string** |  | 
 **categoryId** | **string** |  | 

### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerListDeadLetter

> PartnerListDeadLetter(ctx).Execute()

List dead-letter delivery attempts

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerListDeadLetter(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerListDeadLetter``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerListDeadLetterRequest struct via the builder pattern


### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerListEdiDocuments

> PartnerListEdiDocuments200Response PartnerListEdiDocuments(ctx).Execute()

List recent EDI-lite documents for the key tenant

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerListEdiDocuments(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerListEdiDocuments``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerListEdiDocuments`: PartnerListEdiDocuments200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerListEdiDocuments`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerListEdiDocumentsRequest struct via the builder pattern


### Return type

[**PartnerListEdiDocuments200Response**](PartnerListEdiDocuments200Response.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerListExports

> PartnerListExports200Response PartnerListExports(ctx).Execute()

List recent export jobs for the key tenant

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerListExports(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerListExports``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerListExports`: PartnerListExports200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerListExports`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerListExportsRequest struct via the builder pattern


### Return type

[**PartnerListExports200Response**](PartnerListExports200Response.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerListWebhooks

> PartnerListWebhooks200Response PartnerListWebhooks(ctx).Execute()

List webhook subscriptions (secret omitted; secret_prefix only)

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerListWebhooks(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerListWebhooks``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerListWebhooks`: PartnerListWebhooks200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerListWebhooks`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerListWebhooksRequest struct via the builder pattern


### Return type

[**PartnerListWebhooks200Response**](PartnerListWebhooks200Response.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerOAuthToken

> PartnerOAuthTokenResponse PartnerOAuthToken(ctx).PartnerOAuthTokenRequest(partnerOAuthTokenRequest).Execute()

OAuth2 client_credentials token

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	partnerOAuthTokenRequest := *openapiclient.NewPartnerOAuthTokenRequest("GrantType_example", "ClientSecret_example") // PartnerOAuthTokenRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerOAuthToken(context.Background()).PartnerOAuthTokenRequest(partnerOAuthTokenRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerOAuthToken``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerOAuthToken`: PartnerOAuthTokenResponse
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerOAuthToken`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerOAuthTokenRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **partnerOAuthTokenRequest** | [**PartnerOAuthTokenRequest**](PartnerOAuthTokenRequest.md) |  | 

### Return type

[**PartnerOAuthTokenResponse**](PartnerOAuthTokenResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json, application/x-www-form-urlencoded
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerPOSDemandFeed

> PartnerPOSDemandFeed(ctx).IdempotencyKey(idempotencyKey).PartnerPOSDemandFeedRequest(partnerPOSDemandFeedRequest).Execute()

Ingest POS sell-through demand signals (retail chain → planning)

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	idempotencyKey := "idempotencyKey_example" // string | 
	partnerPOSDemandFeedRequest := *openapiclient.NewPartnerPOSDemandFeedRequest([]openapiclient.PartnerPOSDemandFeedRequestLinesInner{*openapiclient.NewPartnerPOSDemandFeedRequestLinesInner("ExternalId_example", float32(123))}) // PartnerPOSDemandFeedRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerPOSDemandFeed(context.Background()).IdempotencyKey(idempotencyKey).PartnerPOSDemandFeedRequest(partnerPOSDemandFeedRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerPOSDemandFeed``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerPOSDemandFeedRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **idempotencyKey** | **string** |  | 
 **partnerPOSDemandFeedRequest** | [**PartnerPOSDemandFeedRequest**](PartnerPOSDemandFeedRequest.md) |  | 

### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerPingWebhook

> PartnerPingWebhook(ctx, subscriptionID).Execute()

Send signed test delivery

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	subscriptionID := "subscriptionID_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerPingWebhook(context.Background(), subscriptionID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerPingWebhook``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**subscriptionID** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerPingWebhookRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerPutAs2Config

> PartnerAs2Config PartnerPutAs2Config(ctx).PartnerAs2ConfigUpdate(partnerAs2ConfigUpdate).Execute()

Upsert tenant AS2 station config

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	partnerAs2ConfigUpdate := *openapiclient.NewPartnerAs2ConfigUpdate() // PartnerAs2ConfigUpdate | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerPutAs2Config(context.Background()).PartnerAs2ConfigUpdate(partnerAs2ConfigUpdate).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerPutAs2Config``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerPutAs2Config`: PartnerAs2Config
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerPutAs2Config`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerPutAs2ConfigRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **partnerAs2ConfigUpdate** | [**PartnerAs2ConfigUpdate**](PartnerAs2ConfigUpdate.md) |  | 

### Return type

[**PartnerAs2Config**](PartnerAs2Config.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerPutCoa

> PartnerCoaMap PartnerPutCoa(ctx).PartnerCoaMapUpdate(partnerCoaMapUpdate).Execute()

Upsert tenant chart of accounts for journals exports

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	partnerCoaMapUpdate := *openapiclient.NewPartnerCoaMapUpdate() // PartnerCoaMapUpdate | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.PartnerPutCoa(context.Background()).PartnerCoaMapUpdate(partnerCoaMapUpdate).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerPutCoa``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartnerPutCoa`: PartnerCoaMap
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.PartnerPutCoa`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerPutCoaRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **partnerCoaMapUpdate** | [**PartnerCoaMapUpdate**](PartnerCoaMapUpdate.md) |  | 

### Return type

[**PartnerCoaMap**](PartnerCoaMap.md)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerReplayDeadLetter

> PartnerReplayDeadLetter(ctx, attemptID).Execute()

Requeue a DEAD delivery attempt (full retry budget)

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	attemptID := "attemptID_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerReplayDeadLetter(context.Background(), attemptID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerReplayDeadLetter``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**attemptID** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerReplayDeadLetterRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerReplayEdiDocument

> PartnerReplayEdiDocument(ctx, documentID).Execute()

Requeue FAILED/EMITTED EDI document

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	documentID := "documentID_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerReplayEdiDocument(context.Background(), documentID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerReplayEdiDocument``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**documentID** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerReplayEdiDocumentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerRotateWebhookSecret

> PartnerRotateWebhookSecret(ctx, subscriptionID).Execute()

Rotate HMAC signing secret (returned once)

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	subscriptionID := "subscriptionID_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerRotateWebhookSecret(context.Background(), subscriptionID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerRotateWebhookSecret``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**subscriptionID** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPartnerRotateWebhookSecretRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerUpsertPrices

> PartnerUpsertPrices(ctx).IdempotencyKey(idempotencyKey).PartnerPriceUpsertRequest(partnerPriceUpsertRequest).Execute()

Batch upsert list prices by external_id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	idempotencyKey := "idempotencyKey_example" // string | 
	partnerPriceUpsertRequest := *openapiclient.NewPartnerPriceUpsertRequest([]openapiclient.PartnerPriceUpsertRequestItemsInner{*openapiclient.NewPartnerPriceUpsertRequestItemsInner("ExternalId_example", int32(123))}) // PartnerPriceUpsertRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerUpsertPrices(context.Background()).IdempotencyKey(idempotencyKey).PartnerPriceUpsertRequest(partnerPriceUpsertRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerUpsertPrices``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerUpsertPricesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **idempotencyKey** | **string** |  | 
 **partnerPriceUpsertRequest** | [**PartnerPriceUpsertRequest**](PartnerPriceUpsertRequest.md) |  | 

### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerUpsertProducts

> PartnerUpsertProducts(ctx).IdempotencyKey(idempotencyKey).PartnerProductUpsertRequest(partnerProductUpsertRequest).Execute()

Batch upsert products by external_id (ProductId == external_id)

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	idempotencyKey := "idempotencyKey_example" // string | 
	partnerProductUpsertRequest := *openapiclient.NewPartnerProductUpsertRequest([]openapiclient.PartnerProductUpsertRequestItemsInner{*openapiclient.NewPartnerProductUpsertRequestItemsInner("ExternalId_example", "Name_example")}) // PartnerProductUpsertRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerUpsertProducts(context.Background()).IdempotencyKey(idempotencyKey).PartnerProductUpsertRequest(partnerProductUpsertRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerUpsertProducts``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerUpsertProductsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **idempotencyKey** | **string** |  | 
 **partnerProductUpsertRequest** | [**PartnerProductUpsertRequest**](PartnerProductUpsertRequest.md) |  | 

### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartnerUpsertStock

> PartnerUpsertStock(ctx).IdempotencyKey(idempotencyKey).PartnerStockUpsertRequest(partnerStockUpsertRequest).Execute()

Batch set absolute on-hand stock by external_id × warehouse

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/pegasusx/pegasusx/sdk/partner/go"
)

func main() {
	idempotencyKey := "idempotencyKey_example" // string | 
	partnerStockUpsertRequest := *openapiclient.NewPartnerStockUpsertRequest([]openapiclient.PartnerStockUpsertRequestItemsInner{*openapiclient.NewPartnerStockUpsertRequestItemsInner("ExternalId_example", "WarehouseId_example", int32(123))}) // PartnerStockUpsertRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.PartnerUpsertStock(context.Background()).IdempotencyKey(idempotencyKey).PartnerStockUpsertRequest(partnerStockUpsertRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.PartnerUpsertStock``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartnerUpsertStockRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **idempotencyKey** | **string** |  | 
 **partnerStockUpsertRequest** | [**PartnerStockUpsertRequest**](PartnerStockUpsertRequest.md) |  | 

### Return type

 (empty response body)

### Authorization

[PartnerApiKey](../README.md#PartnerApiKey)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

