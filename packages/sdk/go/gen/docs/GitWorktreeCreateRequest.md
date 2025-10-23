# GitWorktreeCreateRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RepoPath** | **string** |  | 
**Path** | **string** |  | 
**Branch** | **string** |  | 
**ExistingBranch** | Pointer to **string** |  | [optional] 
**UseCOW** | Pointer to **bool** |  | [optional] 

## Methods

### NewGitWorktreeCreateRequest

`func NewGitWorktreeCreateRequest(repoPath string, path string, branch string, ) *GitWorktreeCreateRequest`

NewGitWorktreeCreateRequest instantiates a new GitWorktreeCreateRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitWorktreeCreateRequestWithDefaults

`func NewGitWorktreeCreateRequestWithDefaults() *GitWorktreeCreateRequest`

NewGitWorktreeCreateRequestWithDefaults instantiates a new GitWorktreeCreateRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRepoPath

`func (o *GitWorktreeCreateRequest) GetRepoPath() string`

GetRepoPath returns the RepoPath field if non-nil, zero value otherwise.

### GetRepoPathOk

`func (o *GitWorktreeCreateRequest) GetRepoPathOk() (*string, bool)`

GetRepoPathOk returns a tuple with the RepoPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoPath

`func (o *GitWorktreeCreateRequest) SetRepoPath(v string)`

SetRepoPath sets RepoPath field to given value.


### GetPath

`func (o *GitWorktreeCreateRequest) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *GitWorktreeCreateRequest) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *GitWorktreeCreateRequest) SetPath(v string)`

SetPath sets Path field to given value.


### GetBranch

`func (o *GitWorktreeCreateRequest) GetBranch() string`

GetBranch returns the Branch field if non-nil, zero value otherwise.

### GetBranchOk

`func (o *GitWorktreeCreateRequest) GetBranchOk() (*string, bool)`

GetBranchOk returns a tuple with the Branch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBranch

`func (o *GitWorktreeCreateRequest) SetBranch(v string)`

SetBranch sets Branch field to given value.


### GetExistingBranch

`func (o *GitWorktreeCreateRequest) GetExistingBranch() string`

GetExistingBranch returns the ExistingBranch field if non-nil, zero value otherwise.

### GetExistingBranchOk

`func (o *GitWorktreeCreateRequest) GetExistingBranchOk() (*string, bool)`

GetExistingBranchOk returns a tuple with the ExistingBranch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExistingBranch

`func (o *GitWorktreeCreateRequest) SetExistingBranch(v string)`

SetExistingBranch sets ExistingBranch field to given value.

### HasExistingBranch

`func (o *GitWorktreeCreateRequest) HasExistingBranch() bool`

HasExistingBranch returns a boolean if a field has been set.

### GetUseCOW

`func (o *GitWorktreeCreateRequest) GetUseCOW() bool`

GetUseCOW returns the UseCOW field if non-nil, zero value otherwise.

### GetUseCOWOk

`func (o *GitWorktreeCreateRequest) GetUseCOWOk() (*bool, bool)`

GetUseCOWOk returns a tuple with the UseCOW field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUseCOW

`func (o *GitWorktreeCreateRequest) SetUseCOW(v bool)`

SetUseCOW sets UseCOW field to given value.

### HasUseCOW

`func (o *GitWorktreeCreateRequest) HasUseCOW() bool`

HasUseCOW returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


