# PartnerCreateExportRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Resource** | **string** |  | 
**Format** | Pointer to **string** |  | [optional] [default to "csv"]
**From** | Pointer to **string** | RFC3339 or YYYY-MM-DD | [optional] 
**To** | Pointer to **string** | RFC3339 or YYYY-MM-DD | [optional] 

## Methods

### NewPartnerCreateExportRequest

`func NewPartnerCreateExportRequest(resource string, ) *PartnerCreateExportRequest`

NewPartnerCreateExportRequest instantiates a new PartnerCreateExportRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPartnerCreateExportRequestWithDefaults

`func NewPartnerCreateExportRequestWithDefaults() *PartnerCreateExportRequest`

NewPartnerCreateExportRequestWithDefaults instantiates a new PartnerCreateExportRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetResource

`func (o *PartnerCreateExportRequest) GetResource() string`

GetResource returns the Resource field if non-nil, zero value otherwise.

### GetResourceOk

`func (o *PartnerCreateExportRequest) GetResourceOk() (*string, bool)`

GetResourceOk returns a tuple with the Resource field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResource

`func (o *PartnerCreateExportRequest) SetResource(v string)`

SetResource sets Resource field to given value.


### GetFormat

`func (o *PartnerCreateExportRequest) GetFormat() string`

GetFormat returns the Format field if non-nil, zero value otherwise.

### GetFormatOk

`func (o *PartnerCreateExportRequest) GetFormatOk() (*string, bool)`

GetFormatOk returns a tuple with the Format field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFormat

`func (o *PartnerCreateExportRequest) SetFormat(v string)`

SetFormat sets Format field to given value.

### HasFormat

`func (o *PartnerCreateExportRequest) HasFormat() bool`

HasFormat returns a boolean if a field has been set.

### GetFrom

`func (o *PartnerCreateExportRequest) GetFrom() string`

GetFrom returns the From field if non-nil, zero value otherwise.

### GetFromOk

`func (o *PartnerCreateExportRequest) GetFromOk() (*string, bool)`

GetFromOk returns a tuple with the From field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFrom

`func (o *PartnerCreateExportRequest) SetFrom(v string)`

SetFrom sets From field to given value.

### HasFrom

`func (o *PartnerCreateExportRequest) HasFrom() bool`

HasFrom returns a boolean if a field has been set.

### GetTo

`func (o *PartnerCreateExportRequest) GetTo() string`

GetTo returns the To field if non-nil, zero value otherwise.

### GetToOk

`func (o *PartnerCreateExportRequest) GetToOk() (*string, bool)`

GetToOk returns a tuple with the To field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTo

`func (o *PartnerCreateExportRequest) SetTo(v string)`

SetTo sets To field to given value.

### HasTo

`func (o *PartnerCreateExportRequest) HasTo() bool`

HasTo returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


