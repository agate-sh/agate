# GitStageRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RepoPath** | **string** |  | 
**File** | **string** |  | 

## Methods

### NewGitStageRequest

`func NewGitStageRequest(repoPath string, file string, ) *GitStageRequest`

NewGitStageRequest instantiates a new GitStageRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitStageRequestWithDefaults

`func NewGitStageRequestWithDefaults() *GitStageRequest`

NewGitStageRequestWithDefaults instantiates a new GitStageRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRepoPath

`func (o *GitStageRequest) GetRepoPath() string`

GetRepoPath returns the RepoPath field if non-nil, zero value otherwise.

### GetRepoPathOk

`func (o *GitStageRequest) GetRepoPathOk() (*string, bool)`

GetRepoPathOk returns a tuple with the RepoPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoPath

`func (o *GitStageRequest) SetRepoPath(v string)`

SetRepoPath sets RepoPath field to given value.


### GetFile

`func (o *GitStageRequest) GetFile() string`

GetFile returns the File field if non-nil, zero value otherwise.

### GetFileOk

`func (o *GitStageRequest) GetFileOk() (*string, bool)`

GetFileOk returns a tuple with the File field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFile

`func (o *GitStageRequest) SetFile(v string)`

SetFile sets File field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


