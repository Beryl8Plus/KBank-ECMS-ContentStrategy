package httpclient

import (
	"context"
	"time"

	resty "resty.dev/v3"
)

// RestClient is a generic HTTP client wrapper utilizing resty v3.
type RestClient struct {
	Client *resty.Client
}

// Config defines the configuration for the RestClient.
type Config struct {
	BaseURL       string
	Timeout       time.Duration
	RetryCount    int
	RetryWaitTime time.Duration
	RetryMaxWait  time.Duration
}

// NewRestClient constructs a mapped instance of RestClient with defaults.
func NewRestClient(cfg Config) *RestClient {
	client := resty.New()

	if cfg.BaseURL != "" {
		client.SetBaseURL(cfg.BaseURL)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	client.SetTimeout(timeout)

	retryCount := cfg.RetryCount
	if retryCount == 0 {
		// Minimum 3 retries default
		retryCount = 3
	}
	client.SetRetryCount(retryCount)

	retryWaitTime := cfg.RetryWaitTime
	if retryWaitTime == 0 {
		// Provide a default fallback wait time
		retryWaitTime = 2 * time.Second
	}
	client.SetRetryWaitTime(retryWaitTime)

	if cfg.RetryMaxWait > 0 {
		client.SetRetryMaxWaitTime(cfg.RetryMaxWait)
	}

	// Retry condition: automatically retry on status >= 500
	client.AddRetryConditions(func(r *resty.Response, err error) bool {
		return r.StatusCode() >= 500
	})

	return &RestClient{
		Client: client,
	}
}

// Get executes an HTTP GET request with context, mapping to the given result interface.
func (c *RestClient) Get(ctx context.Context, path string, result interface{}) (*resty.Response, error) {
	req := c.Client.R().SetContext(ctx)
	if result != nil {
		req.SetResult(result)
	}
	return req.Get(path)
}

// Post executes an HTTP POST request with context.
func (c *RestClient) Post(ctx context.Context, path string, body interface{}, result interface{}) (*resty.Response, error) {
	req := c.Client.R().SetContext(ctx)
	if body != nil {
		req.SetBody(body)
	}
	if result != nil {
		req.SetResult(result)
	}
	return req.Post(path)
}

// Put executes an HTTP PUT request with context.
func (c *RestClient) Put(ctx context.Context, path string, body interface{}, result interface{}) (*resty.Response, error) {
	req := c.Client.R().SetContext(ctx)
	if body != nil {
		req.SetBody(body)
	}
	if result != nil {
		req.SetResult(result)
	}
	return req.Put(path)
}

// Delete executes an HTTP DELETE request with context.
func (c *RestClient) Delete(ctx context.Context, path string) (*resty.Response, error) {
	return c.Client.R().SetContext(ctx).Delete(path)
}
