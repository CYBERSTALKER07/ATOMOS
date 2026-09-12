# PartnerExportJob

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**JobId** | Pointer to **string** |  | [optional] 
**TenantType** | Pointer to **string** |  | [optional] 
**TenantId** | Pointer to **string** |  | [optional] 
**Resource** | Pointer to **string** |  | [optional] 
**Format** | Pointer to **string** |  | [optional] 
**Status** | Pointer to **string** |  | [optional] 
**RowCount** | Pointer to **int32** |  | [optional] 
**SftpStatus** | Pointer to **string** |  | [optional] 
**Error** | Pointer to **string** |  | [optional] 
**DownloadUrl** | Pointer to **string** |  | [optional] 
**ObjectPath** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**FinishedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewPartnerExportJob

`func NewPartnerExportJob() *PartnerExportJob`

NewPartnerExportJob instantiates a new PartnerExportJob object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerExportJobWithDefaults

`func NewPartnerExportJobWithDefaults() *PartnerExportJob`

NewPartnerExportJobWithDefaults instantiates a new PartnerExportJob object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetJobId

`func (o *PartnerExportJob) GetJobId() string`

GetJobId returns the JobId field if non-nil, zero value otherwise.

### GetJobIdOk

`func (o *PartnerExportJob) GetJobIdOk() (*string, bool)`

GetJobIdOk returns a tuple with the JobId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetJobId

`func (o *PartnerExportJob) SetJobId(v string)`

SetJobId sets JobId field to given value.

### HasJobId

`func (o *PartnerExportJob) HasJobId() bool`

HasJobId returns a boolean if a field has been set.

### GetTenantType

`func (o *PartnerExportJob) GetTenantType() string`

GetTenantType returns the TenantType field if non-nil, zero value otherwise.

### GetTenantTypeOk

`func (o *PartnerExportJob) GetTenantTypeOk() (*string, bool)`

GetTenantTypeOk returns a tuple with the TenantType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTenantType

`func (o *PartnerExportJob) SetTenantType(v string)`

SetTenantType sets TenantType field to given value.

### HasTenantType

`func (o *PartnerExportJob) HasTenantType() bool`

HasTenantType returns a boolean if a field has been set.

### GetTenantId

`func (o *PartnerExportJob) GetTenantId() string`

GetTenantId returns the TenantId field if non-nil, zero value otherwise.

### GetTenantIdOk

`func (o *PartnerExportJob) GetTenantIdOk() (*string, bool)`

GetTenantIdOk returns a tuple with the TenantId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTenantId

`func (o *PartnerExportJob) SetTenantId(v string)`

SetTenantId sets TenantId field to given value.

### HasTenantId

`func (o *PartnerExportJob) HasTenantId() bool`

HasTenantId returns a boolean if a field has been set.

### GetResource

`func (o *PartnerExportJob) GetResource() string`

GetResource returns the Resource field if non-nil, zero value otherwise.

### GetResourceOk

`func (o *PartnerExportJob) GetResourceOk() (*string, bool)`

GetResourceOk returns a tuple with the Resource field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResource

`func (o *PartnerExportJob) SetResource(v string)`

SetResource sets Resource field to given value.

### HasResource

`func (o *PartnerExportJob) HasResource() bool`

HasResource returns a boolean if a field has been set.

### GetFormat

`func (o *PartnerExportJob) GetFormat() string`

GetFormat returns the Format field if non-nil, zero value otherwise.

### GetFormatOk

`func (o *PartnerExportJob) GetFormatOk() (*string, bool)`

GetFormatOk returns a tuple with the Format field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFormat

`func (o *PartnerExportJob) SetFormat(v string)`

SetFormat sets Format field to given value.

### HasFormat

`func (o *PartnerExportJob) HasFormat() bool`

HasFormat returns a boolean if a field has been set.

### GetStatus

`func (o *PartnerExportJob) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *PartnerExportJob) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *PartnerExportJob) SetStatus(v string)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *PartnerExportJob) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### GetRowCount

`func (o *PartnerExportJob) GetRowCount() int32`

GetRowCount returns the RowCount field if non-nil, zero value otherwise.

### GetRowCountOk

`func (o *PartnerExportJob) GetRowCountOk() (*int32, bool)`

GetRowCountOk returns a tuple with the RowCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRowCount

`func (o *PartnerExportJob) SetRowCount(v int32)`

SetRowCount sets RowCount field to given value.

### HasRowCount

`func (o *PartnerExportJob) HasRowCount() bool`

HasRowCount returns a boolean if a field has been set.

### GetSftpStatus

`func (o *PartnerExportJob) GetSftpStatus() string`

GetSftpStatus returns the SftpStatus field if non-nil, zero value otherwise.

### GetSftpStatusOk

`func (o *PartnerExportJob) GetSftpStatusOk() (*string, bool)`

GetSftpStatusOk returns a tuple with the SftpStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSftpStatus

`func (o *PartnerExportJob) SetSftpStatus(v string)`

SetSftpStatus sets SftpStatus field to given value.

### HasSftpStatus

`func (o *PartnerExportJob) HasSftpStatus() bool`

HasSftpStatus returns a boolean if a field has been set.

### GetError

`func (o *PartnerExportJob) GetError() string`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *PartnerExportJob) GetErrorOk() (*string, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *PartnerExportJob) SetError(v string)`

SetError sets Error field to given value.

### HasError

`func (o *PartnerExportJob) HasError() bool`

HasError returns a boolean if a field has been set.

### GetDownloadUrl

`func (o *PartnerExportJob) GetDownloadUrl() string`

GetDownloadUrl returns the DownloadUrl field if non-nil, zero value otherwise.

### GetDownloadUrlOk

`func (o *PartnerExportJob) GetDownloadUrlOk() (*string, bool)`

GetDownloadUrlOk returns a tuple with the DownloadUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDownloadUrl

`func (o *PartnerExportJob) SetDownloadUrl(v string)`

SetDownloadUrl sets DownloadUrl field to given value.

### HasDownloadUrl

`func (o *PartnerExportJob) HasDownloadUrl() bool`

HasDownloadUrl returns a boolean if a field has been set.

### GetObjectPath

`func (o *PartnerExportJob) GetObjectPath() string`

GetObjectPath returns the ObjectPath field if non-nil, zero value otherwise.

### GetObjectPathOk

`func (o *PartnerExportJob) GetObjectPathOk() (*string, bool)`

GetObjectPathOk returns a tuple with the ObjectPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObjectPath

`func (o *PartnerExportJob) SetObjectPath(v string)`

SetObjectPath sets ObjectPath field to given value.

### HasObjectPath

`func (o *PartnerExportJob) HasObjectPath() bool`

HasObjectPath returns a boolean if a field has been set.

### GetCreatedAt

`func (o *PartnerExportJob) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *PartnerExportJob) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *PartnerExportJob) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *PartnerExportJob) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetFinishedAt

`func (o *PartnerExportJob) GetFinishedAt() time.Time`

GetFinishedAt returns the FinishedAt field if non-nil, zero value otherwise.

### GetFinishedAtOk

`func (o *PartnerExportJob) GetFinishedAtOk() (*time.Time, bool)`

GetFinishedAtOk returns a tuple with the FinishedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinishedAt

`func (o *PartnerExportJob) SetFinishedAt(v time.Time)`

SetFinishedAt sets FinishedAt field to given value.

### HasFinishedAt

`func (o *PartnerExportJob) HasFinishedAt() bool`

HasFinishedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


