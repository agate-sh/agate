# SessionList200Response

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Sessions** | [**[]SessionList200ResponseSessionsInner**](SessionList200ResponseSessionsInner.md) |  | 
**ActiveSession** | **string** |  | 
**DefaultAgent** | **string** |  | 

## Methods

### NewSessionList200Response

`func NewSessionList200Response(sessions []SessionList200ResponseSessionsInner, activeSession string, defaultAgent string, ) *SessionList200Response`

NewSessionList200Response instantiates a new SessionList200Response object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionList200ResponseWithDefaults

`func NewSessionList200ResponseWithDefaults() *SessionList200Response`

NewSessionList200ResponseWithDefaults instantiates a new SessionList200Response object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSessions

`func (o *SessionList200Response) GetSessions() []SessionList200ResponseSessionsInner`

GetSessions returns the Sessions field if non-nil, zero value otherwise.

### GetSessionsOk

`func (o *SessionList200Response) GetSessionsOk() (*[]SessionList200ResponseSessionsInner, bool)`

GetSessionsOk returns a tuple with the Sessions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessions

`func (o *SessionList200Response) SetSessions(v []SessionList200ResponseSessionsInner)`

SetSessions sets Sessions field to given value.


### GetActiveSession

`func (o *SessionList200Response) GetActiveSession() string`

GetActiveSession returns the ActiveSession field if non-nil, zero value otherwise.

### GetActiveSessionOk

`func (o *SessionList200Response) GetActiveSessionOk() (*string, bool)`

GetActiveSessionOk returns a tuple with the ActiveSession field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActiveSession

`func (o *SessionList200Response) SetActiveSession(v string)`

SetActiveSession sets ActiveSession field to given value.


### GetDefaultAgent

`func (o *SessionList200Response) GetDefaultAgent() string`

GetDefaultAgent returns the DefaultAgent field if non-nil, zero value otherwise.

### GetDefaultAgentOk

`func (o *SessionList200Response) GetDefaultAgentOk() (*string, bool)`

GetDefaultAgentOk returns a tuple with the DefaultAgent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDefaultAgent

`func (o *SessionList200Response) SetDefaultAgent(v string)`

SetDefaultAgent sets DefaultAgent field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


