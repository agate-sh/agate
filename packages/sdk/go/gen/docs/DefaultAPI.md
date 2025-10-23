# \DefaultAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GitCommit**](DefaultAPI.md#GitCommit) | **Post** /git/commit | 
[**GitDiscard**](DefaultAPI.md#GitDiscard) | **Post** /git/discard | 
[**GitHead**](DefaultAPI.md#GitHead) | **Get** /git/head | 
[**GitRepoInfo**](DefaultAPI.md#GitRepoInfo) | **Get** /git/repo-info | 
[**GitStage**](DefaultAPI.md#GitStage) | **Post** /git/stage | 
[**GitStageAll**](DefaultAPI.md#GitStageAll) | **Post** /git/stage-all | 
[**GitStatus**](DefaultAPI.md#GitStatus) | **Get** /git/status | 
[**GitUnstage**](DefaultAPI.md#GitUnstage) | **Post** /git/unstage | 
[**GitWorktreeCreate**](DefaultAPI.md#GitWorktreeCreate) | **Post** /git/worktree | 
[**GitWorktreeDelete**](DefaultAPI.md#GitWorktreeDelete) | **Delete** /git/worktree | 
[**GitWorktreesList**](DefaultAPI.md#GitWorktreesList) | **Get** /git/worktrees | 
[**HealthCheck**](DefaultAPI.md#HealthCheck) | **Get** /health | 
[**SessionCreate**](DefaultAPI.md#SessionCreate) | **Post** /sessions | 
[**SessionDelete**](DefaultAPI.md#SessionDelete) | **Delete** /sessions/{id} | 
[**SessionGet**](DefaultAPI.md#SessionGet) | **Get** /sessions/{id} | 
[**SessionList**](DefaultAPI.md#SessionList) | **Get** /sessions | 
[**SessionResize**](DefaultAPI.md#SessionResize) | **Post** /sessions/{id}/resize | 



## GitCommit

> GitCommit200Response GitCommit(ctx).GitCommitRequest(gitCommitRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	gitCommitRequest := *openapiclient.NewGitCommitRequest("RepoPath_example", "Message_example") // GitCommitRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitCommit(context.Background()).GitCommitRequest(gitCommitRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitCommit``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitCommit`: GitCommit200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitCommit`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitCommitRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gitCommitRequest** | [**GitCommitRequest**](GitCommitRequest.md) |  | 

### Return type

[**GitCommit200Response**](GitCommit200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitDiscard

> GitWorktreeDelete200Response GitDiscard(ctx).GitStageRequest(gitStageRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	gitStageRequest := *openapiclient.NewGitStageRequest("RepoPath_example", "File_example") // GitStageRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitDiscard(context.Background()).GitStageRequest(gitStageRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitDiscard``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitDiscard`: GitWorktreeDelete200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitDiscard`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitDiscardRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gitStageRequest** | [**GitStageRequest**](GitStageRequest.md) |  | 

### Return type

[**GitWorktreeDelete200Response**](GitWorktreeDelete200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitHead

> GitHead200Response GitHead(ctx).RepoPath(repoPath).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	repoPath := "repoPath_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitHead(context.Background()).RepoPath(repoPath).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitHead``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitHead`: GitHead200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitHead`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitHeadRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **repoPath** | **string** |  | 

### Return type

[**GitHead200Response**](GitHead200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitRepoInfo

> GitRepoInfo200Response GitRepoInfo(ctx).Dir(dir).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	dir := "dir_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitRepoInfo(context.Background()).Dir(dir).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitRepoInfo``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitRepoInfo`: GitRepoInfo200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitRepoInfo`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitRepoInfoRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **dir** | **string** |  | 

### Return type

[**GitRepoInfo200Response**](GitRepoInfo200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitStage

> GitWorktreeDelete200Response GitStage(ctx).GitStageRequest(gitStageRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	gitStageRequest := *openapiclient.NewGitStageRequest("RepoPath_example", "File_example") // GitStageRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitStage(context.Background()).GitStageRequest(gitStageRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitStage``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitStage`: GitWorktreeDelete200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitStage`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitStageRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gitStageRequest** | [**GitStageRequest**](GitStageRequest.md) |  | 

### Return type

[**GitWorktreeDelete200Response**](GitWorktreeDelete200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitStageAll

> GitWorktreeDelete200Response GitStageAll(ctx).GitStageAllRequest(gitStageAllRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	gitStageAllRequest := *openapiclient.NewGitStageAllRequest("RepoPath_example") // GitStageAllRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitStageAll(context.Background()).GitStageAllRequest(gitStageAllRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitStageAll``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitStageAll`: GitWorktreeDelete200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitStageAll`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitStageAllRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gitStageAllRequest** | [**GitStageAllRequest**](GitStageAllRequest.md) |  | 

### Return type

[**GitWorktreeDelete200Response**](GitWorktreeDelete200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitStatus

> GitStatus200Response GitStatus(ctx).RepoPath(repoPath).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	repoPath := "repoPath_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitStatus(context.Background()).RepoPath(repoPath).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitStatus``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitStatus`: GitStatus200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitStatus`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitStatusRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **repoPath** | **string** |  | 

### Return type

[**GitStatus200Response**](GitStatus200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitUnstage

> GitWorktreeDelete200Response GitUnstage(ctx).GitStageRequest(gitStageRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	gitStageRequest := *openapiclient.NewGitStageRequest("RepoPath_example", "File_example") // GitStageRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitUnstage(context.Background()).GitStageRequest(gitStageRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitUnstage``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitUnstage`: GitWorktreeDelete200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitUnstage`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitUnstageRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gitStageRequest** | [**GitStageRequest**](GitStageRequest.md) |  | 

### Return type

[**GitWorktreeDelete200Response**](GitWorktreeDelete200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitWorktreeCreate

> GitWorktreeCreate200Response GitWorktreeCreate(ctx).GitWorktreeCreateRequest(gitWorktreeCreateRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	gitWorktreeCreateRequest := *openapiclient.NewGitWorktreeCreateRequest("RepoPath_example", "Path_example", "Branch_example") // GitWorktreeCreateRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitWorktreeCreate(context.Background()).GitWorktreeCreateRequest(gitWorktreeCreateRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitWorktreeCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitWorktreeCreate`: GitWorktreeCreate200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitWorktreeCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitWorktreeCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gitWorktreeCreateRequest** | [**GitWorktreeCreateRequest**](GitWorktreeCreateRequest.md) |  | 

### Return type

[**GitWorktreeCreate200Response**](GitWorktreeCreate200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitWorktreeDelete

> GitWorktreeDelete200Response GitWorktreeDelete(ctx).GitWorktreeDeleteRequest(gitWorktreeDeleteRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	gitWorktreeDeleteRequest := *openapiclient.NewGitWorktreeDeleteRequest("RepoPath_example", "Path_example") // GitWorktreeDeleteRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitWorktreeDelete(context.Background()).GitWorktreeDeleteRequest(gitWorktreeDeleteRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitWorktreeDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitWorktreeDelete`: GitWorktreeDelete200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitWorktreeDelete`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitWorktreeDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **gitWorktreeDeleteRequest** | [**GitWorktreeDeleteRequest**](GitWorktreeDeleteRequest.md) |  | 

### Return type

[**GitWorktreeDelete200Response**](GitWorktreeDelete200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GitWorktreesList

> GitWorktreesList200Response GitWorktreesList(ctx).RepoPath(repoPath).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	repoPath := "repoPath_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.GitWorktreesList(context.Background()).RepoPath(repoPath).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.GitWorktreesList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GitWorktreesList`: GitWorktreesList200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.GitWorktreesList`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGitWorktreesListRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **repoPath** | **string** |  | 

### Return type

[**GitWorktreesList200Response**](GitWorktreesList200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HealthCheck

> HealthCheck200Response HealthCheck(ctx).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.HealthCheck(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.HealthCheck``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `HealthCheck`: HealthCheck200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.HealthCheck`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiHealthCheckRequest struct via the builder pattern


### Return type

[**HealthCheck200Response**](HealthCheck200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SessionCreate

> SessionCreate201Response SessionCreate(ctx).SessionCreateRequest(sessionCreateRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	sessionCreateRequest := *openapiclient.NewSessionCreateRequest("Dir_example", "BranchName_example", "AgentName_example") // SessionCreateRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.SessionCreate(context.Background()).SessionCreateRequest(sessionCreateRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.SessionCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SessionCreate`: SessionCreate201Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.SessionCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSessionCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **sessionCreateRequest** | [**SessionCreateRequest**](SessionCreateRequest.md) |  | 

### Return type

[**SessionCreate201Response**](SessionCreate201Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SessionDelete

> GitWorktreeDelete200Response SessionDelete(ctx, id).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	id := "id_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.SessionDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.SessionDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SessionDelete`: GitWorktreeDelete200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.SessionDelete`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSessionDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**GitWorktreeDelete200Response**](GitWorktreeDelete200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SessionGet

> SessionGet200Response SessionGet(ctx, id).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	id := "id_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.SessionGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.SessionGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SessionGet`: SessionGet200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.SessionGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSessionGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**SessionGet200Response**](SessionGet200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SessionList

> SessionList200Response SessionList(ctx).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.SessionList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.SessionList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SessionList`: SessionList200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.SessionList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiSessionListRequest struct via the builder pattern


### Return type

[**SessionList200Response**](SessionList200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SessionResize

> GitWorktreeDelete200Response SessionResize(ctx, id).SessionResizeRequest(sessionResizeRequest).Execute()





### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/agateclient"
)

func main() {
	id := "id_example" // string | 
	sessionResizeRequest := *openapiclient.NewSessionResizeRequest(float32(123), float32(123)) // SessionResizeRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.SessionResize(context.Background(), id).SessionResizeRequest(sessionResizeRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.SessionResize``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SessionResize`: GitWorktreeDelete200Response
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.SessionResize`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSessionResizeRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **sessionResizeRequest** | [**SessionResizeRequest**](SessionResizeRequest.md) |  | 

### Return type

[**GitWorktreeDelete200Response**](GitWorktreeDelete200Response.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

