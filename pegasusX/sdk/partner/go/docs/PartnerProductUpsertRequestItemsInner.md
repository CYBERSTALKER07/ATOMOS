# PartnerProductUpsertRequestItemsInner

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ExternalId** | **string** |  | 
**Name** | **string** |  | 
**CategoryId** | Pointer to **string** |  | [optional] 
**PriceMinor** | Pointer to **int32** |  | [optional] 
**Currency** | Pointer to **string** |  | [optional] 
**Unit** | Pointer to **string** |  | [optional] 
**Barcode** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**ImageUrl** | Pointer to **string** |  | [optional] 
**IsActive** | Pointer to **bool** |  | [optional] 
**HandlingClass** | Pointer to **string** |  | [optional] 

## Methods

### NewPartnerProductUpsertRequestItemsInner

`func NewPartnerProductUpsertRequestItemsInner(externalId string, name string, ) *PartnerProductUpsertRequestItemsInner`

NewPartnerProductUpsertRequestItemsInner instantiates a new PartnerProductUpsertRequestItemsInner object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerProductUpsertRequestItemsInnerWithDefaults

`func NewPartnerProductUpsertRequestItemsInnerWithDefaults() *PartnerProductUpsertRequestItemsInner`

NewPartnerProductUpsertRequestItemsInnerWithDefaults instantiates a new PartnerProductUpsertRequestItemsInner object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetExternalId

`func (o *PartnerProductUpsertRequestItemsInner) GetExternalId() string`

GetExternalId returns the ExternalId field if non-nil, zero value otherwise.

### GetExternalIdOk

`func (o *PartnerProductUpsertRequestItemsInner) GetExternalIdOk() (*string, bool)`

GetExternalIdOk returns a tuple with the ExternalId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExternalId

`func (o *PartnerProductUpsertRequestItemsInner) SetExternalId(v string)`

SetExternalId sets ExternalId field to given value.


### GetName

`func (o *PartnerProductUpsertRequestItemsInner) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *PartnerProductUpsertRequestItemsInner) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *PartnerProductUpsertRequestItemsInner) SetName(v string)`

SetName sets Name field to given value.


### GetCategoryId

`func (o *PartnerProductUpsertRequestItemsInner) GetCategoryId() string`

GetCategoryId returns the CategoryId field if non-nil, zero value otherwise.

### GetCategoryIdOk

`func (o *PartnerProductUpsertRequestItemsInner) GetCategoryIdOk() (*string, bool)`

GetCategoryIdOk returns a tuple with the CategoryId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCategoryId

`func (o *PartnerProductUpsertRequestItemsInner) SetCategoryId(v string)`

SetCategoryId sets CategoryId field to given value.

### HasCategoryId

`func (o *PartnerProductUpsertRequestItemsInner) HasCategoryId() bool`

HasCategoryId returns a boolean if a field has been set.

### GetPriceMinor

`func (o *PartnerProductUpsertRequestItemsInner) GetPriceMinor() int32`

GetPriceMinor returns the PriceMinor field if non-nil, zero value otherwise.

### GetPriceMinorOk

`func (o *PartnerProductUpsertRequestItemsInner) GetPriceMinorOk() (*int32, bool)`

GetPriceMinorOk returns a tuple with the PriceMinor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriceMinor

`func (o *PartnerProductUpsertRequestItemsInner) SetPriceMinor(v int32)`

SetPriceMinor sets PriceMinor field to given value.

### HasPriceMinor

`func (o *PartnerProductUpsertRequestItemsInner) HasPriceMinor() bool`

HasPriceMinor returns a boolean if a field has been set.

### GetCurrency

`func (o *PartnerProductUpsertRequestItemsInner) GetCurrency() string`

GetCurrency returns the Currency field if non-nil, zero value otherwise.

### GetCurrencyOk

`func (o *PartnerProductUpsertRequestItemsInner) GetCurrencyOk() (*string, bool)`

GetCurrencyOk returns a tuple with the Currency field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrency

`func (o *PartnerProductUpsertRequestItemsInner) SetCurrency(v string)`

SetCurrency sets Currency field to given value.

### HasCurrency

`func (o *PartnerProductUpsertRequestItemsInner) HasCurrency() bool`

HasCurrency returns a boolean if a field has been set.

### GetUnit

`func (o *PartnerProductUpsertRequestItemsInner) GetUnit() string`

GetUnit returns the Unit field if non-nil, zero value otherwise.

### GetUnitOk

`func (o *PartnerProductUpsertRequestItemsInner) GetUnitOk() (*string, bool)`

GetUnitOk returns a tuple with the Unit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUnit

`func (o *PartnerProductUpsertRequestItemsInner) SetUnit(v string)`

SetUnit sets Unit field to given value.

### HasUnit

`func (o *PartnerProductUpsertRequestItemsInner) HasUnit() bool`

HasUnit returns a boolean if a field has been set.

### GetBarcode

`func (o *PartnerProductUpsertRequestItemsInner) GetBarcode() string`

GetBarcode returns the Barcode field if non-nil, zero value otherwise.

### GetBarcodeOk

`func (o *PartnerProductUpsertRequestItemsInner) GetBarcodeOk() (*string, bool)`

GetBarcodeOk returns a tuple with the Barcode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBarcode

`func (o *PartnerProductUpsertRequestItemsInner) SetBarcode(v string)`

SetBarcode sets Barcode field to given value.

### HasBarcode

`func (o *PartnerProductUpsertRequestItemsInner) HasBarcode() bool`

HasBarcode returns a boolean if a field has been set.

### GetDescription

`func (o *PartnerProductUpsertRequestItemsInner) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *PartnerProductUpsertRequestItemsInner) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *PartnerProductUpsertRequestItemsInner) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *PartnerProductUpsertRequestItemsInner) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetImageUrl

`func (o *PartnerProductUpsertRequestItemsInner) GetImageUrl() string`

GetImageUrl returns the ImageUrl field if non-nil, zero value otherwise.

### GetImageUrlOk

`func (o *PartnerProductUpsertRequestItemsInner) GetImageUrlOk() (*string, bool)`

GetImageUrlOk returns a tuple with the ImageUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImageUrl

`func (o *PartnerProductUpsertRequestItemsInner) SetImageUrl(v string)`

SetImageUrl sets ImageUrl field to given value.

### HasImageUrl

`func (o *PartnerProductUpsertRequestItemsInner) HasImageUrl() bool`

HasImageUrl returns a boolean if a field has been set.

### GetIsActive

`func (o *PartnerProductUpsertRequestItemsInner) GetIsActive() bool`

GetIsActive returns the IsActive field if non-nil, zero value otherwise.

### GetIsActiveOk

`func (o *PartnerProductUpsertRequestItemsInner) GetIsActiveOk() (*bool, bool)`

GetIsActiveOk returns a tuple with the IsActive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsActive

`func (o *PartnerProductUpsertRequestItemsInner) SetIsActive(v bool)`

SetIsActive sets IsActive field to given value.

### HasIsActive

`func (o *PartnerProductUpsertRequestItemsInner) HasIsActive() bool`

HasIsActive returns a boolean if a field has been set.

### GetHandlingClass

`func (o *PartnerProductUpsertRequestItemsInner) GetHandlingClass() string`

GetHandlingClass returns the HandlingClass field if non-nil, zero value otherwise.

### GetHandlingClassOk

`func (o *PartnerProductUpsertRequestItemsInner) GetHandlingClassOk() (*string, bool)`

GetHandlingClassOk returns a tuple with the HandlingClass field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHandlingClass

`func (o *PartnerProductUpsertRequestItemsInner) SetHandlingClass(v string)`

SetHandlingClass sets HandlingClass field to given value.

### HasHandlingClass

`func (o *PartnerProductUpsertRequestItemsInner) HasHandlingClass() bool`

HasHandlingClass returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


