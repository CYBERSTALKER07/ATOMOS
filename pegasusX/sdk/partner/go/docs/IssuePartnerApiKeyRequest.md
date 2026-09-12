# IssuePartnerApiKeyRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**TenantType** | Pointer to **string** |  | [optional] 
**TenantId** | Pointer to **string** |  | [optional] 
**Scopes** | Pointer to **[]string** |  | [optional] 

## Methods

### NewIssuePartnerApiKeyRequest

`func NewIssuePartnerApiKeyRequest() *IssuePartnerApiKeyRequest`

NewIssuePartnerApiKeyRequest instantiates a new IssuePartnerApiKeyRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIssuePartnerApiKeyRequestWithDefaults

`func NewIssuePartnerApiKeyRequestWithDefaults() *IssuePartnerApiKeyRequest`

NewIssuePartnerApiKeyRequestWithDefaults instantiates a new IssuePartnerApiKeyRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetTenantType

`func (o *IssuePartnerApiKeyRequest) GetTenantType() string`

GetTenantType returns the TenantType field if non-nil, zero value otherwise.

### GetTenantTypeOk

`func (o *IssuePartnerApiKeyRequest) GetTenantTypeOk() (*string, bool)`

GetTenantTypeOk returns a tuple with the TenantType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTenantType

`func (o *IssuePartnerApiKeyRequest) SetTenantType(v string)`

SetTenantType sets TenantType field to given value.

### HasTenantType

`func (o *IssuePartnerApiKeyRequest) HasTenantType() bool`

HasTenantType returns a boolean if a field has been set.

### GetTenantId

`func (o *IssuePartnerApiKeyRequest) GetTenantId() string`

GetTenantId returns the TenantId field if non-nil, zero value otherwise.

### GetTenantIdOk

`func (o *IssuePartnerApiKeyRequest) GetTenantIdOk() (*string, bool)`

GetTenantIdOk returns a tuple with the TenantId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTenantId

`func (o *IssuePartnerApiKeyRequest) SetTenantId(v string)`

SetTenantId sets TenantId field to given value.

### HasTenantId

`func (o *IssuePartnerApiKeyRequest) HasTenantId() bool`

HasTenantId returns a boolean if a field has been set.

### GetScopes

`func (o *IssuePartnerApiKeyRequest) GetScopes() []string`

GetScopes returns the Scopes field if non-nil, zero value otherwise.

### GetScopesOk

`func (o *IssuePartnerApiKeyRequest) GetScopesOk() (*[]string, bool)`

GetScopesOk returns a tuple with the Scopes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScopes

`func (o *IssuePartnerApiKeyRequest) SetScopes(v []string)`

SetScopes sets Scopes field to given value.

### HasScopes

`func (o *IssuePartnerApiKeyRequest) HasScopes() bool`

HasScopes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


