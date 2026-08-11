# PartnerSftpConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Configured** | Pointer to **bool** |  | [optional] 
**Host** | Pointer to **string** |  | [optional] 
**Port** | Pointer to **int32** |  | [optional] 
**Username** | Pointer to **string** |  | [optional] 
**SecretRef** | Pointer to **string** | GSM secret name (not plaintext) | [optional] 
**RemoteDir** | Pointer to **string** |  | [optional] 
**IsActive** | Pointer to **bool** |  | [optional] 
**InboundDir** | Pointer to **string** |  | [optional] 
**OutboundDir** | Pointer to **string** |  | [optional] 
**ArchiveDir** | Pointer to **string** |  | [optional] 
**EdiEnabled** | Pointer to **bool** |  | [optional] 
**HostKeySha256** | Pointer to **string** | Base64 SHA-256 of SSH host key (pin); required when PARTNER_SFTP_STRICT_HOSTKEY&#x3D;true | [optional] 

## Methods

### NewPartnerSftpConfig

`func NewPartnerSftpConfig() *PartnerSftpConfig`

NewPartnerSftpConfig instantiates a new PartnerSftpConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerSftpConfigWithDefaults

`func NewPartnerSftpConfigWithDefaults() *PartnerSftpConfig`

NewPartnerSftpConfigWithDefaults instantiates a new PartnerSftpConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConfigured

`func (o *PartnerSftpConfig) GetConfigured() bool`

GetConfigured returns the Configured field if non-nil, zero value otherwise.

### GetConfiguredOk

`func (o *PartnerSftpConfig) GetConfiguredOk() (*bool, bool)`

GetConfiguredOk returns a tuple with the Configured field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfigured

`func (o *PartnerSftpConfig) SetConfigured(v bool)`

SetConfigured sets Configured field to given value.

### HasConfigured

`func (o *PartnerSftpConfig) HasConfigured() bool`

HasConfigured returns a boolean if a field has been set.

### GetHost

`func (o *PartnerSftpConfig) GetHost() string`

GetHost returns the Host field if non-nil, zero value otherwise.

### GetHostOk

`func (o *PartnerSftpConfig) GetHostOk() (*string, bool)`

GetHostOk returns a tuple with the Host field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHost

`func (o *PartnerSftpConfig) SetHost(v string)`

SetHost sets Host field to given value.

### HasHost

`func (o *PartnerSftpConfig) HasHost() bool`

HasHost returns a boolean if a field has been set.

### GetPort

`func (o *PartnerSftpConfig) GetPort() int32`

GetPort returns the Port field if non-nil, zero value otherwise.

### GetPortOk

`func (o *PartnerSftpConfig) GetPortOk() (*int32, bool)`

GetPortOk returns a tuple with the Port field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPort

`func (o *PartnerSftpConfig) SetPort(v int32)`

SetPort sets Port field to given value.

### HasPort

`func (o *PartnerSftpConfig) HasPort() bool`

HasPort returns a boolean if a field has been set.

### GetUsername

`func (o *PartnerSftpConfig) GetUsername() string`

GetUsername returns the Username field if non-nil, zero value otherwise.

### GetUsernameOk

`func (o *PartnerSftpConfig) GetUsernameOk() (*string, bool)`

GetUsernameOk returns a tuple with the Username field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUsername

`func (o *PartnerSftpConfig) SetUsername(v string)`

SetUsername sets Username field to given value.

### HasUsername

`func (o *PartnerSftpConfig) HasUsername() bool`

HasUsername returns a boolean if a field has been set.

### GetSecretRef

`func (o *PartnerSftpConfig) GetSecretRef() string`

GetSecretRef returns the SecretRef field if non-nil, zero value otherwise.

### GetSecretRefOk

`func (o *PartnerSftpConfig) GetSecretRefOk() (*string, bool)`

GetSecretRefOk returns a tuple with the SecretRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRef

`func (o *PartnerSftpConfig) SetSecretRef(v string)`

SetSecretRef sets SecretRef field to given value.

### HasSecretRef

`func (o *PartnerSftpConfig) HasSecretRef() bool`

HasSecretRef returns a boolean if a field has been set.

### GetRemoteDir

`func (o *PartnerSftpConfig) GetRemoteDir() string`

GetRemoteDir returns the RemoteDir field if non-nil, zero value otherwise.

### GetRemoteDirOk

`func (o *PartnerSftpConfig) GetRemoteDirOk() (*string, bool)`

GetRemoteDirOk returns a tuple with the RemoteDir field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRemoteDir

`func (o *PartnerSftpConfig) SetRemoteDir(v string)`

SetRemoteDir sets RemoteDir field to given value.

### HasRemoteDir

`func (o *PartnerSftpConfig) HasRemoteDir() bool`

HasRemoteDir returns a boolean if a field has been set.

### GetIsActive

`func (o *PartnerSftpConfig) GetIsActive() bool`

GetIsActive returns the IsActive field if non-nil, zero value otherwise.

### GetIsActiveOk

`func (o *PartnerSftpConfig) GetIsActiveOk() (*bool, bool)`

GetIsActiveOk returns a tuple with the IsActive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsActive

`func (o *PartnerSftpConfig) SetIsActive(v bool)`

SetIsActive sets IsActive field to given value.

### HasIsActive

`func (o *PartnerSftpConfig) HasIsActive() bool`

HasIsActive returns a boolean if a field has been set.

### GetInboundDir

`func (o *PartnerSftpConfig) GetInboundDir() string`

GetInboundDir returns the InboundDir field if non-nil, zero value otherwise.

### GetInboundDirOk

`func (o *PartnerSftpConfig) GetInboundDirOk() (*string, bool)`

GetInboundDirOk returns a tuple with the InboundDir field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInboundDir

`func (o *PartnerSftpConfig) SetInboundDir(v string)`

SetInboundDir sets InboundDir field to given value.

### HasInboundDir

`func (o *PartnerSftpConfig) HasInboundDir() bool`

HasInboundDir returns a boolean if a field has been set.

### GetOutboundDir

`func (o *PartnerSftpConfig) GetOutboundDir() string`

GetOutboundDir returns the OutboundDir field if non-nil, zero value otherwise.

### GetOutboundDirOk

`func (o *PartnerSftpConfig) GetOutboundDirOk() (*string, bool)`

GetOutboundDirOk returns a tuple with the OutboundDir field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutboundDir

`func (o *PartnerSftpConfig) SetOutboundDir(v string)`

SetOutboundDir sets OutboundDir field to given value.

### HasOutboundDir

`func (o *PartnerSftpConfig) HasOutboundDir() bool`

HasOutboundDir returns a boolean if a field has been set.

### GetArchiveDir

`func (o *PartnerSftpConfig) GetArchiveDir() string`

GetArchiveDir returns the ArchiveDir field if non-nil, zero value otherwise.

### GetArchiveDirOk

`func (o *PartnerSftpConfig) GetArchiveDirOk() (*string, bool)`

GetArchiveDirOk returns a tuple with the ArchiveDir field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArchiveDir

`func (o *PartnerSftpConfig) SetArchiveDir(v string)`

SetArchiveDir sets ArchiveDir field to given value.

### HasArchiveDir

`func (o *PartnerSftpConfig) HasArchiveDir() bool`

HasArchiveDir returns a boolean if a field has been set.

### GetEdiEnabled

`func (o *PartnerSftpConfig) GetEdiEnabled() bool`

GetEdiEnabled returns the EdiEnabled field if non-nil, zero value otherwise.

### GetEdiEnabledOk

`func (o *PartnerSftpConfig) GetEdiEnabledOk() (*bool, bool)`

GetEdiEnabledOk returns a tuple with the EdiEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEdiEnabled

`func (o *PartnerSftpConfig) SetEdiEnabled(v bool)`

SetEdiEnabled sets EdiEnabled field to given value.

### HasEdiEnabled

`func (o *PartnerSftpConfig) HasEdiEnabled() bool`

HasEdiEnabled returns a boolean if a field has been set.

### GetHostKeySha256

`func (o *PartnerSftpConfig) GetHostKeySha256() string`

GetHostKeySha256 returns the HostKeySha256 field if non-nil, zero value otherwise.

### GetHostKeySha256Ok

`func (o *PartnerSftpConfig) GetHostKeySha256Ok() (*string, bool)`

GetHostKeySha256Ok returns a tuple with the HostKeySha256 field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostKeySha256

`func (o *PartnerSftpConfig) SetHostKeySha256(v string)`

SetHostKeySha256 sets HostKeySha256 field to given value.

### HasHostKeySha256

`func (o *PartnerSftpConfig) HasHostKeySha256() bool`

HasHostKeySha256 returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


