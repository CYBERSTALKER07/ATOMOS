# PartnerPriceUpsertRequestItemsInner

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ExternalId** | **string** |  | 
**PriceMinor** | **int32** |  | 
**Currency** | Pointer to **string** |  | [optional] 
**RetailerId** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewPartnerPriceUpsertRequestItemsInner

`func NewPartnerPriceUpsertRequestItemsInner(externalId string, priceMinor int32, ) *PartnerPriceUpsertRequestItemsInner`

NewPartnerPriceUpsertRequestItemsInner instantiates a new PartnerPriceUpsertRequestItemsInner object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerPriceUpsertRequestItemsInnerWithDefaults

`func NewPartnerPriceUpsertRequestItemsInnerWithDefaults() *PartnerPriceUpsertRequestItemsInner`

NewPartnerPriceUpsertRequestItemsInnerWithDefaults instantiates a new PartnerPriceUpsertRequestItemsInner object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetExternalId

`func (o *PartnerPriceUpsertRequestItemsInner) GetExternalId() string`

GetExternalId returns the ExternalId field if non-nil, zero value otherwise.

### GetExternalIdOk

`func (o *PartnerPriceUpsertRequestItemsInner) GetExternalIdOk() (*string, bool)`

GetExternalIdOk returns a tuple with the ExternalId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExternalId

`func (o *PartnerPriceUpsertRequestItemsInner) SetExternalId(v string)`

SetExternalId sets ExternalId field to given value.


### GetPriceMinor

`func (o *PartnerPriceUpsertRequestItemsInner) GetPriceMinor() int32`

GetPriceMinor returns the PriceMinor field if non-nil, zero value otherwise.

### GetPriceMinorOk

`func (o *PartnerPriceUpsertRequestItemsInner) GetPriceMinorOk() (*int32, bool)`

GetPriceMinorOk returns a tuple with the PriceMinor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriceMinor

`func (o *PartnerPriceUpsertRequestItemsInner) SetPriceMinor(v int32)`

SetPriceMinor sets PriceMinor field to given value.


### GetCurrency

`func (o *PartnerPriceUpsertRequestItemsInner) GetCurrency() string`

GetCurrency returns the Currency field if non-nil, zero value otherwise.

### GetCurrencyOk

`func (o *PartnerPriceUpsertRequestItemsInner) GetCurrencyOk() (*string, bool)`

GetCurrencyOk returns a tuple with the Currency field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrency

`func (o *PartnerPriceUpsertRequestItemsInner) SetCurrency(v string)`

SetCurrency sets Currency field to given value.

### HasCurrency

`func (o *PartnerPriceUpsertRequestItemsInner) HasCurrency() bool`

HasCurrency returns a boolean if a field has been set.

### GetRetailerId

`func (o *PartnerPriceUpsertRequestItemsInner) GetRetailerId() string`

GetRetailerId returns the RetailerId field if non-nil, zero value otherwise.

### GetRetailerIdOk

`func (o *PartnerPriceUpsertRequestItemsInner) GetRetailerIdOk() (*string, bool)`

GetRetailerIdOk returns a tuple with the RetailerId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRetailerId

`func (o *PartnerPriceUpsertRequestItemsInner) SetRetailerId(v string)`

SetRetailerId sets RetailerId field to given value.

### HasRetailerId

`func (o *PartnerPriceUpsertRequestItemsInner) HasRetailerId() bool`

HasRetailerId returns a boolean if a field has been set.

### SetRetailerIdNil

`func (o *PartnerPriceUpsertRequestItemsInner) SetRetailerIdNil(b bool)`

 SetRetailerIdNil sets the value for RetailerId to be an explicit nil

### UnsetRetailerId
`func (o *PartnerPriceUpsertRequestItemsInner) UnsetRetailerId()`

UnsetRetailerId ensures that no value is present for RetailerId, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


