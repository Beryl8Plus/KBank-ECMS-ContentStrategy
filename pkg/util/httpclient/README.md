# HTTP Client Utility

A robust, generic HTTP client wrapper utilizing [Resty v3](https://resty.dev/).

## Features

- **Context Support**: All methods accept `context.Context` for cancellation and timeouts.
- **Automatic Retries**: Configured to retry on status codes >= 500 by default.
- **Timeout Management**: Easy configuration for request timeouts.
- **Generic Support**: Helper methods handle JSON marshaling/unmarshaling automatically.
- **Standardized Defaults**: Sensible defaults (15s timeout, 3 retries) for consistency.

## Configuration

The client is configured via the `Config` struct:

```go
type Config struct {
	BaseURL       string
	Timeout       time.Duration
	RetryCount    int
	RetryWaitTime time.Duration
	RetryMaxWait  time.Duration
}
```

## Usage

### 1. Initialization

```go
import "kbank-ecms/pkg/util/httpclient"

client := httpclient.NewRestClient(httpclient.Config{
    BaseURL:    "https://api.example.com",
    Timeout:    10 * time.Second,
    RetryCount: 3,
})
```

### 2. GET Request

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

var user User
resp, err := client.Get(ctx, "/users/1", &user)
if err != nil {
    return err
}

fmt.Printf("User: %+v\n", user)
```

### 3. POST Request

```go
type CreateUserRequest struct {
    Name string `json:"name"`
}

reqBody := CreateUserRequest{Name: "John Doe"}
var result User

resp, err := client.Post(ctx, "/users", reqBody, &result)
if err != nil {
    return err
}

fmt.Printf("Created User ID: %d\n", result.ID)
```

### 4. DELETE Request

```go
resp, err := client.Delete(ctx, "/users/1")
if err != nil {
    return err
}

if resp.StatusCode() == http.StatusNoContent {
    fmt.Println("Deleted successfully")
}
```

## Retry Logic

By default, the client is configured to retry if:
1. The response status code is >= 500.
2. The `RetryCount` is greater than 0.

You can customize the retry behavior in the `Config` passed to `NewRestClient`.

## Testing

The utility include comprehensive tests using `httptest.NewServer`. When adding new features, ensure tests are updated in `client_test.go`.

```bash
go test ./pkg/util/httpclient/...
```
