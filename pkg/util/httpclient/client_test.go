package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockResult struct {
	Message string `json:"message"`
}

type MockBody struct {
	Data string `json:"data"`
}

func setupMockServer(t *testing.T, method string, path string, statusCode int, responseBody interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, method, r.Method)
		if r.URL.Path != path {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if responseBody != nil {
			_ = json.NewEncoder(w).Encode(responseBody)
		}
	}))
}

func TestNewRestClient(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		client := NewRestClient(Config{})
		assert.NotNil(t, client)
		assert.NotNil(t, client.Client)
	})

	t.Run("Custom configuration", func(t *testing.T) {
		cfg := Config{
			BaseURL:       "http://localhost:8080",
			Timeout:       5 * time.Second,
			RetryCount:    5,
			RetryWaitTime: 1 * time.Second,
			RetryMaxWait:  3 * time.Second,
		}
		client := NewRestClient(cfg)
		assert.NotNil(t, client)
	})
}

func TestRestClient_Get(t *testing.T) {
	expectedResult := MockResult{Message: "Success"}
	srv := setupMockServer(t, http.MethodGet, "/test", http.StatusOK, expectedResult)
	defer srv.Close()

	client := NewRestClient(Config{BaseURL: srv.URL})

	var result MockResult
	resp, err := client.Get(context.Background(), "/test", &result)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, expectedResult.Message, result.Message)
}

func TestRestClient_Post(t *testing.T) {
	expectedResult := MockResult{Message: "Created"}
	srv := setupMockServer(t, http.MethodPost, "/test", http.StatusCreated, expectedResult)
	defer srv.Close()

	client := NewRestClient(Config{BaseURL: srv.URL})
	body := MockBody{Data: "test data"}

	var result MockResult
	resp, err := client.Post(context.Background(), "/test", body, &result)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode())
	assert.Equal(t, expectedResult.Message, result.Message)
}

func TestRestClient_Put(t *testing.T) {
	expectedResult := MockResult{Message: "Updated"}
	srv := setupMockServer(t, http.MethodPut, "/test", http.StatusOK, expectedResult)
	defer srv.Close()

	client := NewRestClient(Config{BaseURL: srv.URL})
	body := MockBody{Data: "update data"}

	var result MockResult
	resp, err := client.Put(context.Background(), "/test", body, &result)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, expectedResult.Message, result.Message)
}

func TestRestClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewRestClient(Config{BaseURL: srv.URL})

	resp, err := client.Delete(context.Background(), "/test")

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode())
}

func TestRestClient_RetryCondition(t *testing.T) {
	var callCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Fail first 2 times
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(MockResult{Message: "Recovered"})
	}))
	defer srv.Close()

	client := NewRestClient(Config{
		BaseURL:       srv.URL,
		RetryCount:    3,
		RetryWaitTime: 10 * time.Millisecond,
	})

	var result MockResult
	resp, err := client.Get(context.Background(), "/test", &result)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, "Recovered", result.Message)
	assert.Equal(t, 3, callCount)
}
