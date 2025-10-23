# SessionGet200Response

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | 
**Name** | **string** |  | 
**Agent** | **string** |  | 
**Cwd** | **string** |  | 
**Cols** | **float32** |  | 
**Rows** | **float32** |  | 
**Pid** | Pointer to **float32** |  | [optional] 
**IsAlive** | **bool** |  | 

## Methods

### NewSessionGet200Response

`func NewSessionGet200Response(id string, name string, agent string, cwd string, cols float32, rows float32, isAlive bool, ) *SessionGet200Response`

NewSessionGet200Response instantiates a new SessionGet200Response object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionGet200ResponseWithDefaults

`func NewSessionGet200ResponseWithDefaults() *SessionGet200Response`

NewSessionGet200ResponseWithDefaults instantiates a new SessionGet200Response object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *SessionGet200Response) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *SessionGet200Response) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *SessionGet200Response) SetId(v string)`

SetId sets Id field to given value.


### GetName

`func (o *SessionGet200Response) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *SessionGet200Response) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *SessionGet200Response) SetName(v string)`

SetName sets Name field to given value.


### GetAgent

`func (o *SessionGet200Response) GetAgent() string`

GetAgent returns the Agent field if non-nil, zero value otherwise.

### GetAgentOk

`func (o *SessionGet200Response) GetAgentOk() (*string, bool)`

GetAgentOk returns a tuple with the Agent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgent

`func (o *SessionGet200Response) SetAgent(v string)`

SetAgent sets Agent field to given value.


### GetCwd

`func (o *SessionGet200Response) GetCwd() string`

GetCwd returns the Cwd field if non-nil, zero value otherwise.

### GetCwdOk

`func (o *SessionGet200Response) GetCwdOk() (*string, bool)`

GetCwdOk returns a tuple with the Cwd field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCwd

`func (o *SessionGet200Response) SetCwd(v string)`

SetCwd sets Cwd field to given value.


### GetCols

`func (o *SessionGet200Response) GetCols() float32`

GetCols returns the Cols field if non-nil, zero value otherwise.

### GetColsOk

`func (o *SessionGet200Response) GetColsOk() (*float32, bool)`

GetColsOk returns a tuple with the Cols field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCols

`func (o *SessionGet200Response) SetCols(v float32)`

SetCols sets Cols field to given value.


### GetRows

`func (o *SessionGet200Response) GetRows() float32`

GetRows returns the Rows field if non-nil, zero value otherwise.

### GetRowsOk

`func (o *SessionGet200Response) GetRowsOk() (*float32, bool)`

GetRowsOk returns a tuple with the Rows field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRows

`func (o *SessionGet200Response) SetRows(v float32)`

SetRows sets Rows field to given value.


### GetPid

`func (o *SessionGet200Response) GetPid() float32`

GetPid returns the Pid field if non-nil, zero value otherwise.

### GetPidOk

`func (o *SessionGet200Response) GetPidOk() (*float32, bool)`

GetPidOk returns a tuple with the Pid field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPid

`func (o *SessionGet200Response) SetPid(v float32)`

SetPid sets Pid field to given value.

### HasPid

`func (o *SessionGet200Response) HasPid() bool`

HasPid returns a boolean if a field has been set.

### GetIsAlive

`func (o *SessionGet200Response) GetIsAlive() bool`

GetIsAlive returns the IsAlive field if non-nil, zero value otherwise.

### GetIsAliveOk

`func (o *SessionGet200Response) GetIsAliveOk() (*bool, bool)`

GetIsAliveOk returns a tuple with the IsAlive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsAlive

`func (o *SessionGet200Response) SetIsAlive(v bool)`

SetIsAlive sets IsAlive field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


