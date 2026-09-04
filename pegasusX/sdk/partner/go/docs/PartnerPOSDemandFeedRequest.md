# PartnerPOSDemandFeedRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**StoreId** | Pointer to **string** |  | [optional] 
**ObservedAt** | Pointer to **time.Time** |  | [optional] 
**Lines** | [**[]PartnerPOSDemandFeedRequestLinesInner**](PartnerPOSDemandFeedRequestLinesInner.md) |  | 

## Methods

### NewPartnerPOSDemandFeedRequest

`func NewPartnerPOSDemandFeedRequest(lines []PartnerPOSDemandFeedRequestLinesInner, ) *PartnerPOSDemandFeedRequest`

NewPartnerPOSDemandFeedRequest instantiates a new PartnerPOSDemandFeedRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerPOSDemandFeedRequestWithDefaults

`func NewPartnerPOSDemandFeedRequestWithDefaults() *PartnerPOSDemandFeedRequest`

NewPartnerPOSDemandFeedRequestWithDefaults instantiates a new PartnerPOSDemandFeedRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetStoreId

`func (o *PartnerPOSDemandFeedRequest) GetStoreId() string`

GetStoreId returns the StoreId field if non-nil, zero value otherwise.

### GetStoreIdOk

`func (o *PartnerPOSDemandFeedRequest) GetStoreIdOk() (*string, bool)`

GetStoreIdOk returns a tuple with the StoreId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStoreId

`func (o *PartnerPOSDemandFeedRequest) SetStoreId(v string)`

SetStoreId sets StoreId field to given value.

### HasStoreId

`func (o *PartnerPOSDemandFeedRequest) HasStoreId() bool`

HasStoreId returns a boolean if a field has been set.

### GetObservedAt

`func (o *PartnerPOSDemandFeedRequest) GetObservedAt() time.Time`

GetObservedAt returns the ObservedAt field if non-nil, zero value otherwise.

### GetObservedAtOk

`func (o *PartnerPOSDemandFeedRequest) GetObservedAtOk() (*time.Time, bool)`

GetObservedAtOk returns a tuple with the ObservedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedAt

`func (o *PartnerPOSDemandFeedRequest) SetObservedAt(v time.Time)`

SetObservedAt sets ObservedAt field to given value.

### HasObservedAt

`func (o *PartnerPOSDemandFeedRequest) HasObservedAt() bool`

HasObservedAt returns a boolean if a field has been set.

### GetLines

`func (o *PartnerPOSDemandFeedRequest) GetLines() []PartnerPOSDemandFeedRequestLinesInner`

GetLines returns the Lines field if non-nil, zero value otherwise.

### GetLinesOk

`func (o *PartnerPOSDemandFeedRequest) GetLinesOk() (*[]PartnerPOSDemandFeedRequestLinesInner, bool)`

GetLinesOk returns a tuple with the Lines field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLines

`func (o *PartnerPOSDemandFeedRequest) SetLines(v []PartnerPOSDemandFeedRequestLinesInner)`

SetLines sets Lines field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


