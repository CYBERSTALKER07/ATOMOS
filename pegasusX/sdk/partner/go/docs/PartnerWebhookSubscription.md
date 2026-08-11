# PartnerWebhookSubscription

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SubscriptionId** | Pointer to **string** |  | [optional] 
**Url** | Pointer to **string** |  | [optional] 
**EventTypes** | Pointer to **[]string** |  | [optional] 
**IsActive** | Pointer to **bool** |  | [optional] 
**SecretPrefix** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewPartnerWebhookSubscription

`func NewPartnerWebhookSubscription() *PartnerWebhookSubscription`

NewPartnerWebhookSubscription instantiates a new PartnerWebhookSubscription object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerWebhookSubscriptionWithDefaults

`func NewPartnerWebhookSubscriptionWithDefaults() *PartnerWebhookSubscription`

NewPartnerWebhookSubscriptionWithDefaults instantiates a new PartnerWebhookSubscription object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSubscriptionId

`func (o *PartnerWebhookSubscription) GetSubscriptionId() string`

GetSubscriptionId returns the SubscriptionId field if non-nil, zero value otherwise.

### GetSubscriptionIdOk

`func (o *PartnerWebhookSubscription) GetSubscriptionIdOk() (*string, bool)`

GetSubscriptionIdOk returns a tuple with the SubscriptionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubscriptionId

`func (o *PartnerWebhookSubscription) SetSubscriptionId(v string)`

SetSubscriptionId sets SubscriptionId field to given value.

### HasSubscriptionId

`func (o *PartnerWebhookSubscription) HasSubscriptionId() bool`

HasSubscriptionId returns a boolean if a field has been set.

### GetUrl

`func (o *PartnerWebhookSubscription) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *PartnerWebhookSubscription) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *PartnerWebhookSubscription) SetUrl(v string)`

SetUrl sets Url field to given value.

### HasUrl

`func (o *PartnerWebhookSubscription) HasUrl() bool`

HasUrl returns a boolean if a field has been set.

### GetEventTypes

`func (o *PartnerWebhookSubscription) GetEventTypes() []string`

GetEventTypes returns the EventTypes field if non-nil, zero value otherwise.

### GetEventTypesOk

`func (o *PartnerWebhookSubscription) GetEventTypesOk() (*[]string, bool)`

GetEventTypesOk returns a tuple with the EventTypes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEventTypes

`func (o *PartnerWebhookSubscription) SetEventTypes(v []string)`

SetEventTypes sets EventTypes field to given value.

### HasEventTypes

`func (o *PartnerWebhookSubscription) HasEventTypes() bool`

HasEventTypes returns a boolean if a field has been set.

### GetIsActive

`func (o *PartnerWebhookSubscription) GetIsActive() bool`

GetIsActive returns the IsActive field if non-nil, zero value otherwise.

### GetIsActiveOk

`func (o *PartnerWebhookSubscription) GetIsActiveOk() (*bool, bool)`

GetIsActiveOk returns a tuple with the IsActive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsActive

`func (o *PartnerWebhookSubscription) SetIsActive(v bool)`

SetIsActive sets IsActive field to given value.

### HasIsActive

`func (o *PartnerWebhookSubscription) HasIsActive() bool`

HasIsActive returns a boolean if a field has been set.

### GetSecretPrefix

`func (o *PartnerWebhookSubscription) GetSecretPrefix() string`

GetSecretPrefix returns the SecretPrefix field if non-nil, zero value otherwise.

### GetSecretPrefixOk

`func (o *PartnerWebhookSubscription) GetSecretPrefixOk() (*string, bool)`

GetSecretPrefixOk returns a tuple with the SecretPrefix field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretPrefix

`func (o *PartnerWebhookSubscription) SetSecretPrefix(v string)`

SetSecretPrefix sets SecretPrefix field to given value.

### HasSecretPrefix

`func (o *PartnerWebhookSubscription) HasSecretPrefix() bool`

HasSecretPrefix returns a boolean if a field has been set.

### GetCreatedAt

`func (o *PartnerWebhookSubscription) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *PartnerWebhookSubscription) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *PartnerWebhookSubscription) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *PartnerWebhookSubscription) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


