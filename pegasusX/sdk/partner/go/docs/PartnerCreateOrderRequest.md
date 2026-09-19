# PartnerCreateOrderRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SupplierId** | Pointer to **string** | Required for RETAILER keys | [optional] 
**RetailerId** | Pointer to **string** | Required for SUPPLIER keys | [optional] 
**LineItems** | [**[]PartnerCreateOrderRequestLineItemsInner**](PartnerCreateOrderRequestLineItemsInner.md) |  | 
**H3Cell** | Pointer to **string** |  | [optional] 
**Lat** | **float32** |  | 
**Lng** | **float32** |  | 

## Methods

### NewPartnerCreateOrderRequest

`func NewPartnerCreateOrderRequest(lineItems []PartnerCreateOrderRequestLineItemsInner, lat float32, lng float32, ) *PartnerCreateOrderRequest`

NewPartnerCreateOrderRequest instantiates a new PartnerCreateOrderRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerCreateOrderRequestWithDefaults

`func NewPartnerCreateOrderRequestWithDefaults() *PartnerCreateOrderRequest`

NewPartnerCreateOrderRequestWithDefaults instantiates a new PartnerCreateOrderRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSupplierId

`func (o *PartnerCreateOrderRequest) GetSupplierId() string`

GetSupplierId returns the SupplierId field if non-nil, zero value otherwise.

### GetSupplierIdOk

`func (o *PartnerCreateOrderRequest) GetSupplierIdOk() (*string, bool)`

GetSupplierIdOk returns a tuple with the SupplierId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSupplierId

`func (o *PartnerCreateOrderRequest) SetSupplierId(v string)`

SetSupplierId sets SupplierId field to given value.

### HasSupplierId

`func (o *PartnerCreateOrderRequest) HasSupplierId() bool`

HasSupplierId returns a boolean if a field has been set.

### GetRetailerId

`func (o *PartnerCreateOrderRequest) GetRetailerId() string`

GetRetailerId returns the RetailerId field if non-nil, zero value otherwise.

### GetRetailerIdOk

`func (o *PartnerCreateOrderRequest) GetRetailerIdOk() (*string, bool)`

GetRetailerIdOk returns a tuple with the RetailerId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRetailerId

`func (o *PartnerCreateOrderRequest) SetRetailerId(v string)`

SetRetailerId sets RetailerId field to given value.

### HasRetailerId

`func (o *PartnerCreateOrderRequest) HasRetailerId() bool`

HasRetailerId returns a boolean if a field has been set.

### GetLineItems

`func (o *PartnerCreateOrderRequest) GetLineItems() []PartnerCreateOrderRequestLineItemsInner`

GetLineItems returns the LineItems field if non-nil, zero value otherwise.

### GetLineItemsOk

`func (o *PartnerCreateOrderRequest) GetLineItemsOk() (*[]PartnerCreateOrderRequestLineItemsInner, bool)`

GetLineItemsOk returns a tuple with the LineItems field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLineItems

`func (o *PartnerCreateOrderRequest) SetLineItems(v []PartnerCreateOrderRequestLineItemsInner)`

SetLineItems sets LineItems field to given value.


### GetH3Cell

`func (o *PartnerCreateOrderRequest) GetH3Cell() string`

GetH3Cell returns the H3Cell field if non-nil, zero value otherwise.

### GetH3CellOk

`func (o *PartnerCreateOrderRequest) GetH3CellOk() (*string, bool)`

GetH3CellOk returns a tuple with the H3Cell field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetH3Cell

`func (o *PartnerCreateOrderRequest) SetH3Cell(v string)`

SetH3Cell sets H3Cell field to given value.

### HasH3Cell

`func (o *PartnerCreateOrderRequest) HasH3Cell() bool`

HasH3Cell returns a boolean if a field has been set.

### GetLat

`func (o *PartnerCreateOrderRequest) GetLat() float32`

GetLat returns the Lat field if non-nil, zero value otherwise.

### GetLatOk

`func (o *PartnerCreateOrderRequest) GetLatOk() (*float32, bool)`

GetLatOk returns a tuple with the Lat field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLat

`func (o *PartnerCreateOrderRequest) SetLat(v float32)`

SetLat sets Lat field to given value.


### GetLng

`func (o *PartnerCreateOrderRequest) GetLng() float32`

GetLng returns the Lng field if non-nil, zero value otherwise.

### GetLngOk

`func (o *PartnerCreateOrderRequest) GetLngOk() (*float32, bool)`

GetLngOk returns a tuple with the Lng field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLng

`func (o *PartnerCreateOrderRequest) SetLng(v float32)`

SetLng sets Lng field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


