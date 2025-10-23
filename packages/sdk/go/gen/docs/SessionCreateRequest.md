# SessionCreateRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Dir** | **string** |  | 
**BranchName** | **string** |  | 
**AgentName** | **string** |  | 

## Methods

### NewSessionCreateRequest

`func NewSessionCreateRequest(dir string, branchName string, agentName string, ) *SessionCreateRequest`

NewSessionCreateRequest instantiates a new SessionCreateRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionCreateRequestWithDefaults

`func NewSessionCreateRequestWithDefaults() *SessionCreateRequest`

NewSessionCreateRequestWithDefaults instantiates a new SessionCreateRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDir

`func (o *SessionCreateRequest) GetDir() string`

GetDir returns the Dir field if non-nil, zero value otherwise.

### GetDirOk

`func (o *SessionCreateRequest) GetDirOk() (*string, bool)`

GetDirOk returns a tuple with the Dir field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDir

`func (o *SessionCreateRequest) SetDir(v string)`

SetDir sets Dir field to given value.


### GetBranchName

`func (o *SessionCreateRequest) GetBranchName() string`

GetBranchName returns the BranchName field if non-nil, zero value otherwise.

### GetBranchNameOk

`func (o *SessionCreateRequest) GetBranchNameOk() (*string, bool)`

GetBranchNameOk returns a tuple with the BranchName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBranchName

`func (o *SessionCreateRequest) SetBranchName(v string)`

SetBranchName sets BranchName field to given value.


### GetAgentName

`func (o *SessionCreateRequest) GetAgentName() string`

GetAgentName returns the AgentName field if non-nil, zero value otherwise.

### GetAgentNameOk

`func (o *SessionCreateRequest) GetAgentNameOk() (*string, bool)`

GetAgentNameOk returns a tuple with the AgentName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentName

`func (o *SessionCreateRequest) SetAgentName(v string)`

SetAgentName sets AgentName field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


