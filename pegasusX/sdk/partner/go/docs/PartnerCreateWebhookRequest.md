# PartnerCreateWebhookRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Url** | **string** |  | 
**EventTypes** | Pointer to **[]string** |  | [optional] 

## Methods

### NewPartnerCreateWebhookRequest

`func NewPartnerCreateWebhookRequest(url string, ) *PartnerCreateWebhookRequest`

NewPartnerCreateWebhookRequest instantiates a new PartnerCreateWebhookRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerCreateWebhookRequestWithDefaults

`func NewPartnerCreateWebhookRequestWithDefaults() *PartnerCreateWebhookRequest`

NewPartnerCreateWebhookRequestWithDefaults instantiates a new PartnerCreateWebhookRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetUrl

`func (o *PartnerCreateWebhookRequest) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *PartnerCreateWebhookRequest) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *PartnerCreateWebhookRequest) SetUrl(v string)`

SetUrl sets Url field to given value.


### GetEventTypes

`func (o *PartnerCreateWebhookRequest) GetEventTypes() []string`

GetEventTypes returns the EventTypes field if non-nil, zero value otherwise.

### GetEventTypesOk

`func (o *PartnerCreateWebhookRequest) GetEventTypesOk() (*[]string, bool)`

GetEventTypesOk returns a tuple with the EventTypes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEventTypes

`func (o *PartnerCreateWebhookRequest) SetEventTypes(v []string)`

SetEventTypes sets EventTypes field to given value.

### HasEventTypes

`func (o *PartnerCreateWebhookRequest) HasEventTypes() bool`

HasEventTypes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


