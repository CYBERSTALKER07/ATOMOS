# PartnerAs2ConfigUpdate

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
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

### NewPartnerAs2ConfigUpdate

`func NewPartnerAs2ConfigUpdate() *PartnerAs2ConfigUpdate`

NewPartnerAs2ConfigUpdate instantiates a new PartnerAs2ConfigUpdate object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerAs2ConfigUpdateWithDefaults

`func NewPartnerAs2ConfigUpdateWithDefaults() *PartnerAs2ConfigUpdate`

NewPartnerAs2ConfigUpdateWithDefaults instantiates a new PartnerAs2ConfigUpdate object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAs2Enabled

`func (o *PartnerAs2ConfigUpdate) GetAs2Enabled() bool`

GetAs2Enabled returns the As2Enabled field if non-nil, zero value otherwise.

### GetAs2EnabledOk

`func (o *PartnerAs2ConfigUpdate) GetAs2EnabledOk() (*bool, bool)`

GetAs2EnabledOk returns a tuple with the As2Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAs2Enabled

`func (o *PartnerAs2ConfigUpdate) SetAs2Enabled(v bool)`

SetAs2Enabled sets As2Enabled field to given value.

### HasAs2Enabled

`func (o *PartnerAs2ConfigUpdate) HasAs2Enabled() bool`

HasAs2Enabled returns a boolean if a field has been set.

### GetOurAs2Id

`func (o *PartnerAs2ConfigUpdate) GetOurAs2Id() string`

GetOurAs2Id returns the OurAs2Id field if non-nil, zero value otherwise.

### GetOurAs2IdOk

`func (o *PartnerAs2ConfigUpdate) GetOurAs2IdOk() (*string, bool)`

GetOurAs2IdOk returns a tuple with the OurAs2Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOurAs2Id

`func (o *PartnerAs2ConfigUpdate) SetOurAs2Id(v string)`

SetOurAs2Id sets OurAs2Id field to given value.

### HasOurAs2Id

`func (o *PartnerAs2ConfigUpdate) HasOurAs2Id() bool`

HasOurAs2Id returns a boolean if a field has been set.

### GetPartnerAs2Id

`func (o *PartnerAs2ConfigUpdate) GetPartnerAs2Id() string`

GetPartnerAs2Id returns the PartnerAs2Id field if non-nil, zero value otherwise.

### GetPartnerAs2IdOk

`func (o *PartnerAs2ConfigUpdate) GetPartnerAs2IdOk() (*string, bool)`

GetPartnerAs2IdOk returns a tuple with the PartnerAs2Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPartnerAs2Id

`func (o *PartnerAs2ConfigUpdate) SetPartnerAs2Id(v string)`

SetPartnerAs2Id sets PartnerAs2Id field to given value.

### HasPartnerAs2Id

`func (o *PartnerAs2ConfigUpdate) HasPartnerAs2Id() bool`

HasPartnerAs2Id returns a boolean if a field has been set.

### GetPartnerUrl

`func (o *PartnerAs2ConfigUpdate) GetPartnerUrl() string`

GetPartnerUrl returns the PartnerUrl field if non-nil, zero value otherwise.

### GetPartnerUrlOk

`func (o *PartnerAs2ConfigUpdate) GetPartnerUrlOk() (*string, bool)`

GetPartnerUrlOk returns a tuple with the PartnerUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPartnerUrl

`func (o *PartnerAs2ConfigUpdate) SetPartnerUrl(v string)`

SetPartnerUrl sets PartnerUrl field to given value.

### HasPartnerUrl

`func (o *PartnerAs2ConfigUpdate) HasPartnerUrl() bool`

HasPartnerUrl returns a boolean if a field has been set.

### GetOurCertSecretRef

`func (o *PartnerAs2ConfigUpdate) GetOurCertSecretRef() string`

GetOurCertSecretRef returns the OurCertSecretRef field if non-nil, zero value otherwise.

### GetOurCertSecretRefOk

`func (o *PartnerAs2ConfigUpdate) GetOurCertSecretRefOk() (*string, bool)`

GetOurCertSecretRefOk returns a tuple with the OurCertSecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOurCertSecretRef

`func (o *PartnerAs2ConfigUpdate) SetOurCertSecretRef(v string)`

SetOurCertSecretRef sets OurCertSecretRef field to given value.

### HasOurCertSecretRef

`func (o *PartnerAs2ConfigUpdate) HasOurCertSecretRef() bool`

HasOurCertSecretRef returns a boolean if a field has been set.

### GetOurKeySecretRef

`func (o *PartnerAs2ConfigUpdate) GetOurKeySecretRef() string`

GetOurKeySecretRef returns the OurKeySecretRef field if non-nil, zero value otherwise.

### GetOurKeySecretRefOk

`func (o *PartnerAs2ConfigUpdate) GetOurKeySecretRefOk() (*string, bool)`

GetOurKeySecretRefOk returns a tuple with the OurKeySecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOurKeySecretRef

`func (o *PartnerAs2ConfigUpdate) SetOurKeySecretRef(v string)`

SetOurKeySecretRef sets OurKeySecretRef field to given value.

### HasOurKeySecretRef

`func (o *PartnerAs2ConfigUpdate) HasOurKeySecretRef() bool`

HasOurKeySecretRef returns a boolean if a field has been set.

### GetPartnerCertSecretRef

`func (o *PartnerAs2ConfigUpdate) GetPartnerCertSecretRef() string`

GetPartnerCertSecretRef returns the PartnerCertSecretRef field if non-nil, zero value otherwise.

### GetPartnerCertSecretRefOk

`func (o *PartnerAs2ConfigUpdate) GetPartnerCertSecretRefOk() (*string, bool)`

GetPartnerCertSecretRefOk returns a tuple with the PartnerCertSecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPartnerCertSecretRef

`func (o *PartnerAs2ConfigUpdate) SetPartnerCertSecretRef(v string)`

SetPartnerCertSecretRef sets PartnerCertSecretRef field to given value.

### HasPartnerCertSecretRef

`func (o *PartnerAs2ConfigUpdate) HasPartnerCertSecretRef() bool`

HasPartnerCertSecretRef returns a boolean if a field has been set.

### GetSignRequired

`func (o *PartnerAs2ConfigUpdate) GetSignRequired() bool`

GetSignRequired returns the SignRequired field if non-nil, zero value otherwise.

### GetSignRequiredOk

`func (o *PartnerAs2ConfigUpdate) GetSignRequiredOk() (*bool, bool)`

GetSignRequiredOk returns a tuple with the SignRequired field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignRequired

`func (o *PartnerAs2ConfigUpdate) SetSignRequired(v bool)`

SetSignRequired sets SignRequired field to given value.

### HasSignRequired

`func (o *PartnerAs2ConfigUpdate) HasSignRequired() bool`

HasSignRequired returns a boolean if a field has been set.

### GetEncryptRequired

`func (o *PartnerAs2ConfigUpdate) GetEncryptRequired() bool`

GetEncryptRequired returns the EncryptRequired field if non-nil, zero value otherwise.

### GetEncryptRequiredOk

`func (o *PartnerAs2ConfigUpdate) GetEncryptRequiredOk() (*bool, bool)`

GetEncryptRequiredOk returns a tuple with the EncryptRequired field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEncryptRequired

`func (o *PartnerAs2ConfigUpdate) SetEncryptRequired(v bool)`

SetEncryptRequired sets EncryptRequired field to given value.

### HasEncryptRequired

`func (o *PartnerAs2ConfigUpdate) HasEncryptRequired() bool`

HasEncryptRequired returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


