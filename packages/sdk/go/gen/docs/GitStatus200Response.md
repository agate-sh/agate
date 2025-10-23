# GitStatus200Response

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Files** | [**[]GitStatus200ResponseFilesInner**](GitStatus200ResponseFilesInner.md) |  | 
**Staged** | **[]string** |  | 
**Modified** | **[]string** |  | 
**NotAdded** | **[]string** |  | 
**Conflicted** | **[]string** |  | 
**Current** | **NullableString** |  | 
**Tracking** | **NullableString** |  | 
**Ahead** | **float32** |  | 
**Behind** | **float32** |  | 

## Methods

### NewGitStatus200Response

`func NewGitStatus200Response(files []GitStatus200ResponseFilesInner, staged []string, modified []string, notAdded []string, conflicted []string, current NullableString, tracking NullableString, ahead float32, behind float32, ) *GitStatus200Response`

NewGitStatus200Response instantiates a new GitStatus200Response object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitStatus200ResponseWithDefaults

`func NewGitStatus200ResponseWithDefaults() *GitStatus200Response`

NewGitStatus200ResponseWithDefaults instantiates a new GitStatus200Response object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFiles

`func (o *GitStatus200Response) GetFiles() []GitStatus200ResponseFilesInner`

GetFiles returns the Files field if non-nil, zero value otherwise.

### GetFilesOk

`func (o *GitStatus200Response) GetFilesOk() (*[]GitStatus200ResponseFilesInner, bool)`

GetFilesOk returns a tuple with the Files field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFiles

`func (o *GitStatus200Response) SetFiles(v []GitStatus200ResponseFilesInner)`

SetFiles sets Files field to given value.


### GetStaged

`func (o *GitStatus200Response) GetStaged() []string`

GetStaged returns the Staged field if non-nil, zero value otherwise.

### GetStagedOk

`func (o *GitStatus200Response) GetStagedOk() (*[]string, bool)`

GetStagedOk returns a tuple with the Staged field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStaged

`func (o *GitStatus200Response) SetStaged(v []string)`

SetStaged sets Staged field to given value.


### GetModified

`func (o *GitStatus200Response) GetModified() []string`

GetModified returns the Modified field if non-nil, zero value otherwise.

### GetModifiedOk

`func (o *GitStatus200Response) GetModifiedOk() (*[]string, bool)`

GetModifiedOk returns a tuple with the Modified field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetModified

`func (o *GitStatus200Response) SetModified(v []string)`

SetModified sets Modified field to given value.


### GetNotAdded

`func (o *GitStatus200Response) GetNotAdded() []string`

GetNotAdded returns the NotAdded field if non-nil, zero value otherwise.

### GetNotAddedOk

`func (o *GitStatus200Response) GetNotAddedOk() (*[]string, bool)`

GetNotAddedOk returns a tuple with the NotAdded field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNotAdded

`func (o *GitStatus200Response) SetNotAdded(v []string)`

SetNotAdded sets NotAdded field to given value.


### GetConflicted

`func (o *GitStatus200Response) GetConflicted() []string`

GetConflicted returns the Conflicted field if non-nil, zero value otherwise.

### GetConflictedOk

`func (o *GitStatus200Response) GetConflictedOk() (*[]string, bool)`

GetConflictedOk returns a tuple with the Conflicted field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConflicted

`func (o *GitStatus200Response) SetConflicted(v []string)`

SetConflicted sets Conflicted field to given value.


### GetCurrent

`func (o *GitStatus200Response) GetCurrent() string`

GetCurrent returns the Current field if non-nil, zero value otherwise.

### GetCurrentOk

`func (o *GitStatus200Response) GetCurrentOk() (*string, bool)`

GetCurrentOk returns a tuple with the Current field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrent

`func (o *GitStatus200Response) SetCurrent(v string)`

SetCurrent sets Current field to given value.


### SetCurrentNil

`func (o *GitStatus200Response) SetCurrentNil(b bool)`

 SetCurrentNil sets the value for Current to be an explicit nil

### UnsetCurrent
`func (o *GitStatus200Response) UnsetCurrent()`

UnsetCurrent ensures that no value is present for Current, not even an explicit nil
### GetTracking

`func (o *GitStatus200Response) GetTracking() string`

GetTracking returns the Tracking field if non-nil, zero value otherwise.

### GetTrackingOk

`func (o *GitStatus200Response) GetTrackingOk() (*string, bool)`

GetTrackingOk returns a tuple with the Tracking field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTracking

`func (o *GitStatus200Response) SetTracking(v string)`

SetTracking sets Tracking field to given value.


### SetTrackingNil

`func (o *GitStatus200Response) SetTrackingNil(b bool)`

 SetTrackingNil sets the value for Tracking to be an explicit nil

### UnsetTracking
`func (o *GitStatus200Response) UnsetTracking()`

UnsetTracking ensures that no value is present for Tracking, not even an explicit nil
### GetAhead

`func (o *GitStatus200Response) GetAhead() float32`

GetAhead returns the Ahead field if non-nil, zero value otherwise.

### GetAheadOk

`func (o *GitStatus200Response) GetAheadOk() (*float32, bool)`

GetAheadOk returns a tuple with the Ahead field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAhead

`func (o *GitStatus200Response) SetAhead(v float32)`

SetAhead sets Ahead field to given value.


### GetBehind

`func (o *GitStatus200Response) GetBehind() float32`

GetBehind returns the Behind field if non-nil, zero value otherwise.

### GetBehindOk

`func (o *GitStatus200Response) GetBehindOk() (*float32, bool)`

GetBehindOk returns a tuple with the Behind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBehind

`func (o *GitStatus200Response) SetBehind(v float32)`

SetBehind sets Behind field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


