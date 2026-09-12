# PartnerAs2Config

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Configured** | Pointer to **bool** |  | [optional] 
**As2Enabled** | Pointer to **bool** |  | [optional] 
**OurAs2Id** | Pointer to **string** |  | [optional] 
**PartnerAs2Id** | Pointer to **string** |  | [optional] 
**PartnerUrl** | Pointer to **string** |  | [optional] 
**OurCertSecretRef** | Pointer to **string** |  | [optional] 
**OurKeySecretRef** | Pointer to **string** |  | [optional] 
**PartnerCertSecretRef** | Pointer to **string** |  | [optional] 
**SignRequired** | Pointer to **bool** |  | [optional] 
**EncryptRequired** | Pointer to **bool** |  | [optional] 

## Methods

### NewPartnerAs2Config

`func NewPartnerAs2Config() *PartnerAs2Config`

NewPartnerAs2Config instantiates a new PartnerAs2Config object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerAs2ConfigWithDefaults

`func NewPartnerAs2ConfigWithDefaults() *PartnerAs2Config`

NewPartnerAs2ConfigWithDefaults instantiates a new PartnerAs2Config object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConfigured

`func (o *PartnerAs2Config) GetConfigured() bool`

GetConfigured returns the Configured field if non-nil, zero value otherwise.

### GetConfiguredOk

`func (o *PartnerAs2Config) GetConfiguredOk() (*bool, bool)`

GetConfiguredOk returns a tuple with the Configured field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfigured

`func (o *PartnerAs2Config) SetConfigured(v bool)`

SetConfigured sets Configured field to given value.

### HasConfigured

`func (o *PartnerAs2Config) HasConfigured() bool`

HasConfigured returns a boolean if a field has been set.

### GetAs2Enabled

`func (o *PartnerAs2Config) GetAs2Enabled() bool`

GetAs2Enabled returns the As2Enabled field if non-nil, zero value otherwise.

### GetAs2EnabledOk

`func (o *PartnerAs2Config) GetAs2EnabledOk() (*bool, bool)`

GetAs2EnabledOk returns a tuple with the As2Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAs2Enabled

`func (o *PartnerAs2Config) SetAs2Enabled(v bool)`

SetAs2Enabled sets As2Enabled field to given value.

### HasAs2Enabled

`func (o *PartnerAs2Config) HasAs2Enabled() bool`

HasAs2Enabled returns a boolean if a field has been set.

### GetOurAs2Id

`func (o *PartnerAs2Config) GetOurAs2Id() string`

GetOurAs2Id returns the OurAs2Id field if non-nil, zero value otherwise.

### GetOurAs2IdOk

`func (o *PartnerAs2Config) GetOurAs2IdOk() (*string, bool)`

GetOurAs2IdOk returns a tuple with the OurAs2Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOurAs2Id

`func (o *PartnerAs2Config) SetOurAs2Id(v string)`

SetOurAs2Id sets OurAs2Id field to given value.

### HasOurAs2Id

`func (o *PartnerAs2Config) HasOurAs2Id() bool`

HasOurAs2Id returns a boolean if a field has been set.

### GetPartnerAs2Id

`func (o *PartnerAs2Config) GetPartnerAs2Id() string`

GetPartnerAs2Id returns the PartnerAs2Id field if non-nil, zero value otherwise.

### GetPartnerAs2IdOk

`func (o *PartnerAs2Config) GetPartnerAs2IdOk() (*string, bool)`

GetPartnerAs2IdOk returns a tuple with the PartnerAs2Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPartnerAs2Id

`func (o *PartnerAs2Config) SetPartnerAs2Id(v string)`

SetPartnerAs2Id sets PartnerAs2Id field to given value.

### HasPartnerAs2Id

`func (o *PartnerAs2Config) HasPartnerAs2Id() bool`

HasPartnerAs2Id returns a boolean if a field has been set.

### GetPartnerUrl

`func (o *PartnerAs2Config) GetPartnerUrl() string`

GetPartnerUrl returns the PartnerUrl field if non-nil, zero value otherwise.

### GetPartnerUrlOk

`func (o *PartnerAs2Config) GetPartnerUrlOk() (*string, bool)`

GetPartnerUrlOk returns a tuple with the PartnerUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPartnerUrl

`func (o *PartnerAs2Config) SetPartnerUrl(v string)`

SetPartnerUrl sets PartnerUrl field to given value.

### HasPartnerUrl

`func (o *PartnerAs2Config) HasPartnerUrl() bool`

HasPartnerUrl returns a boolean if a field has been set.

### GetOurCertSecretRef

`func (o *PartnerAs2Config) GetOurCertSecretRef() string`

GetOurCertSecretRef returns the OurCertSecretRef field if non-nil, zero value otherwise.

### GetOurCertSecretRefOk

`func (o *PartnerAs2Config) GetOurCertSecretRefOk() (*string, bool)`

GetOurCertSecretRefOk returns a tuple with the OurCertSecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOurCertSecretRef

`func (o *PartnerAs2Config) SetOurCertSecretRef(v string)`

SetOurCertSecretRef sets OurCertSecretRef field to given value.

### HasOurCertSecretRef

`func (o *PartnerAs2Config) HasOurCertSecretRef() bool`

HasOurCertSecretRef returns a boolean if a field has been set.

### GetOurKeySecretRef

`func (o *PartnerAs2Config) GetOurKeySecretRef() string`

GetOurKeySecretRef returns the OurKeySecretRef field if non-nil, zero value otherwise.

### GetOurKeySecretRefOk

`func (o *PartnerAs2Config) GetOurKeySecretRefOk() (*string, bool)`

GetOurKeySecretRefOk returns a tuple with the OurKeySecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOurKeySecretRef

`func (o *PartnerAs2Config) SetOurKeySecretRef(v string)`

SetOurKeySecretRef sets OurKeySecretRef field to given value.

### HasOurKeySecretRef

`func (o *PartnerAs2Config) HasOurKeySecretRef() bool`

HasOurKeySecretRef returns a boolean if a field has been set.

### GetPartnerCertSecretRef

`func (o *PartnerAs2Config) GetPartnerCertSecretRef() string`

GetPartnerCertSecretRef returns the PartnerCertSecretRef field if non-nil, zero value otherwise.

### GetPartnerCertSecretRefOk

`func (o *PartnerAs2Config) GetPartnerCertSecretRefOk() (*string, bool)`

GetPartnerCertSecretRefOk returns a tuple with the PartnerCertSecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPartnerCertSecretRef

`func (o *PartnerAs2Config) SetPartnerCertSecretRef(v string)`

SetPartnerCertSecretRef sets PartnerCertSecretRef field to given value.

### HasPartnerCertSecretRef

`func (o *PartnerAs2Config) HasPartnerCertSecretRef() bool`

HasPartnerCertSecretRef returns a boolean if a field has been set.

### GetSignRequired

`func (o *PartnerAs2Config) GetSignRequired() bool`

GetSignRequired returns the SignRequired field if non-nil, zero value otherwise.

### GetSignRequiredOk

`func (o *PartnerAs2Config) GetSignRequiredOk() (*bool, bool)`

GetSignRequiredOk returns a tuple with the SignRequired field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignRequired

`func (o *PartnerAs2Config) SetSignRequired(v bool)`

SetSignRequired sets SignRequired field to given value.

### HasSignRequired

`func (o *PartnerAs2Config) HasSignRequired() bool`

HasSignRequired returns a boolean if a field has been set.

### GetEncryptRequired

`func (o *PartnerAs2Config) GetEncryptRequired() bool`

GetEncryptRequired returns the EncryptRequired field if non-nil, zero value otherwise.

### GetEncryptRequiredOk

`func (o *PartnerAs2Config) GetEncryptRequiredOk() (*bool, bool)`

GetEncryptRequiredOk returns a tuple with the EncryptRequired field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEncryptRequired

`func (o *PartnerAs2Config) SetEncryptRequired(v bool)`

SetEncryptRequired sets EncryptRequired field to given value.

### HasEncryptRequired

`func (o *PartnerAs2Config) HasEncryptRequired() bool`

HasEncryptRequired returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


