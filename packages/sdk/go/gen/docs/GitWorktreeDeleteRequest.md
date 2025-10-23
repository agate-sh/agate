# GitWorktreeDeleteRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RepoPath** | **string** |  | 
**Path** | **string** |  | 
**Force** | Pointer to **bool** |  | [optional] 

## Methods

### NewGitWorktreeDeleteRequest

`func NewGitWorktreeDeleteRequest(repoPath string, path string, ) *GitWorktreeDeleteRequest`

NewGitWorktreeDeleteRequest instantiates a new GitWorktreeDeleteRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitWorktreeDeleteRequestWithDefaults

`func NewGitWorktreeDeleteRequestWithDefaults() *GitWorktreeDeleteRequest`

NewGitWorktreeDeleteRequestWithDefaults instantiates a new GitWorktreeDeleteRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRepoPath

`func (o *GitWorktreeDeleteRequest) GetRepoPath() string`

GetRepoPath returns the RepoPath field if non-nil, zero value otherwise.

### GetRepoPathOk

`func (o *GitWorktreeDeleteRequest) GetRepoPathOk() (*string, bool)`

GetRepoPathOk returns a tuple with the RepoPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoPath

`func (o *GitWorktreeDeleteRequest) SetRepoPath(v string)`

SetRepoPath sets RepoPath field to given value.


### GetPath

`func (o *GitWorktreeDeleteRequest) GetPath() string`

GetPath returns the Path field if non-nil, zero value otherwise.

### GetPathOk

`func (o *GitWorktreeDeleteRequest) GetPathOk() (*string, bool)`

GetPathOk returns a tuple with the Path field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPath

`func (o *GitWorktreeDeleteRequest) SetPath(v string)`

SetPath sets Path field to given value.


### GetForce

`func (o *GitWorktreeDeleteRequest) GetForce() bool`

GetForce returns the Force field if non-nil, zero value otherwise.

### GetForceOk

`func (o *GitWorktreeDeleteRequest) GetForceOk() (*bool, bool)`

GetForceOk returns a tuple with the Force field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetForce

`func (o *GitWorktreeDeleteRequest) SetForce(v bool)`

SetForce sets Force field to given value.

### HasForce

`func (o *GitWorktreeDeleteRequest) HasForce() bool`

HasForce returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


