# GitCommitRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RepoPath** | **string** |  | 
**Message** | **string** |  | 

## Methods

### NewGitCommitRequest

`func NewGitCommitRequest(repoPath string, message string, ) *GitCommitRequest`

NewGitCommitRequest instantiates a new GitCommitRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGitCommitRequestWithDefaults

`func NewGitCommitRequestWithDefaults() *GitCommitRequest`

NewGitCommitRequestWithDefaults instantiates a new GitCommitRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRepoPath

`func (o *GitCommitRequest) GetRepoPath() string`

GetRepoPath returns the RepoPath field if non-nil, zero value otherwise.

### GetRepoPathOk

`func (o *GitCommitRequest) GetRepoPathOk() (*string, bool)`

GetRepoPathOk returns a tuple with the RepoPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoPath

`func (o *GitCommitRequest) SetRepoPath(v string)`

SetRepoPath sets RepoPath field to given value.


### GetMessage

`func (o *GitCommitRequest) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *GitCommitRequest) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *GitCommitRequest) SetMessage(v string)`

SetMessage sets Message field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


