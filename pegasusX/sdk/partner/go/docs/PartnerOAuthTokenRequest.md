# PartnerOAuthTokenRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**GrantType** | **string** |  | 
**ClientId** | Pointer to **string** | PartnerApiKeys.KeyId or KeyPrefix | [optional] 
**ClientSecret** | **string** | Full pxk_… plaintext | 
**Scope** | Pointer to **string** | Space-separated subset of key scopes | [optional] 

## Methods

### NewPartnerOAuthTokenRequest

`func NewPartnerOAuthTokenRequest(grantType string, clientSecret string, ) *PartnerOAuthTokenRequest`

NewPartnerOAuthTokenRequest instantiates a new PartnerOAuthTokenRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerOAuthTokenRequestWithDefaults

`func NewPartnerOAuthTokenRequestWithDefaults() *PartnerOAuthTokenRequest`

NewPartnerOAuthTokenRequestWithDefaults instantiates a new PartnerOAuthTokenRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetGrantType

`func (o *PartnerOAuthTokenRequest) GetGrantType() string`

GetGrantType returns the GrantType field if non-nil, zero value otherwise.

### GetGrantTypeOk

`func (o *PartnerOAuthTokenRequest) GetGrantTypeOk() (*string, bool)`

GetGrantTypeOk returns a tuple with the GrantType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGrantType

`func (o *PartnerOAuthTokenRequest) SetGrantType(v string)`

SetGrantType sets GrantType field to given value.


### GetClientId

`func (o *PartnerOAuthTokenRequest) GetClientId() string`

GetClientId returns the ClientId field if non-nil, zero value otherwise.

### GetClientIdOk

`func (o *PartnerOAuthTokenRequest) GetClientIdOk() (*string, bool)`

GetClientIdOk returns a tuple with the ClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientId

`func (o *PartnerOAuthTokenRequest) SetClientId(v string)`

SetClientId sets ClientId field to given value.

### HasClientId

`func (o *PartnerOAuthTokenRequest) HasClientId() bool`

HasClientId returns a boolean if a field has been set.

### GetClientSecret

`func (o *PartnerOAuthTokenRequest) GetClientSecret() string`

GetClientSecret returns the ClientSecret field if non-nil, zero value otherwise.

### GetClientSecretOk

`func (o *PartnerOAuthTokenRequest) GetClientSecretOk() (*string, bool)`

GetClientSecretOk returns a tuple with the ClientSecret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientSecret

`func (o *PartnerOAuthTokenRequest) SetClientSecret(v string)`

SetClientSecret sets ClientSecret field to given value.


### GetScope

`func (o *PartnerOAuthTokenRequest) GetScope() string`

GetScope returns the Scope field if non-nil, zero value otherwise.

### GetScopeOk

`func (o *PartnerOAuthTokenRequest) GetScopeOk() (*string, bool)`

GetScopeOk returns a tuple with the Scope field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScope

`func (o *PartnerOAuthTokenRequest) SetScope(v string)`

SetScope sets Scope field to given value.

### HasScope

`func (o *PartnerOAuthTokenRequest) HasScope() bool`

HasScope returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


