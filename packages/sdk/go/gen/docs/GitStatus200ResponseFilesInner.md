# GitStatus200ResponseFilesInner

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Path** | **string** |  | 
**Index** | **string** |  | 
**WorkingDir** | **string** |  | 
**Additions** | Pointer to **float32** |  | [optional] 
**Deletions** | Pointer to **float32** |  | [optional] 

## Methods

### NewGitStatus200ResponseFilesInner

`func NewGitStatus200ResponseFilesInner(path string, index string, workingDir string, ) *GitStatus200ResponseFilesInner`

NewGitStatus200ResponseFilesInner instantiates a new GitStatus200ResponseFilesInner object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitStatus200ResponseFilesInnerWithDefaults

`func NewGitStatus200ResponseFilesInnerWithDefaults() *GitStatus200ResponseFilesInner`

NewGitStatus200ResponseFilesInnerWithDefaults instantiates a new GitStatus200ResponseFilesInner object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPath

`func (o *GitStatus200ResponseFilesInner) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *GitStatus200ResponseFilesInner) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *GitStatus200ResponseFilesInner) SetPath(v string)`

SetPath sets Path field to given value.


### GetIndex

`func (o *GitStatus200ResponseFilesInner) GetIndex() string`

GetIndex returns the Index field if non-nil, zero value otherwise.

### GetIndexOk

`func (o *GitStatus200ResponseFilesInner) GetIndexOk() (*string, bool)`

GetIndexOk returns a tuple with the Index field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndex

`func (o *GitStatus200ResponseFilesInner) SetIndex(v string)`

SetIndex sets Index field to given value.


### GetWorkingDir

`func (o *GitStatus200ResponseFilesInner) GetWorkingDir() string`

GetWorkingDir returns the WorkingDir field if non-nil, zero value otherwise.

### GetWorkingDirOk

`func (o *GitStatus200ResponseFilesInner) GetWorkingDirOk() (*string, bool)`

GetWorkingDirOk returns a tuple with the WorkingDir field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkingDir

`func (o *GitStatus200ResponseFilesInner) SetWorkingDir(v string)`

SetWorkingDir sets WorkingDir field to given value.


### GetAdditions

`func (o *GitStatus200ResponseFilesInner) GetAdditions() float32`

GetAdditions returns the Additions field if non-nil, zero value otherwise.

### GetAdditionsOk

`func (o *GitStatus200ResponseFilesInner) GetAdditionsOk() (*float32, bool)`

GetAdditionsOk returns a tuple with the Additions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditions

`func (o *GitStatus200ResponseFilesInner) SetAdditions(v float32)`

SetAdditions sets Additions field to given value.

### HasAdditions

`func (o *GitStatus200ResponseFilesInner) HasAdditions() bool`

HasAdditions returns a boolean if a field has been set.

### GetDeletions

`func (o *GitStatus200ResponseFilesInner) GetDeletions() float32`

GetDeletions returns the Deletions field if non-nil, zero value otherwise.

### GetDeletionsOk

`func (o *GitStatus200ResponseFilesInner) GetDeletionsOk() (*float32, bool)`

GetDeletionsOk returns a tuple with the Deletions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeletions

`func (o *GitStatus200ResponseFilesInner) SetDeletions(v float32)`

SetDeletions sets Deletions field to given value.

### HasDeletions

`func (o *GitStatus200ResponseFilesInner) HasDeletions() bool`

HasDeletions returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


