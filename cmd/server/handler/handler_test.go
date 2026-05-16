package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	deliveryservice "kbank-ecms/cmd/server/service"
	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/pkg/ctxconsts"
)

var _ deliveryservice.DeliveryService = (*mockDeliveryService)(nil)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- mock ----------------------------------------------------------------

type mockDeliveryService struct {
	getPersonalizedFn func(ctx context.Context, customerInfo *dto.CustomerRequest, channel string, placements []string) ([]dto.ContentResult, error)
	flushFn           func(ctx context.Context, placements []string, isEvaluate bool) error
	keysFn            func(ctx context.Context) ([]string, error)
	getValueFn        func(ctx context.Context, key string) (json.RawMessage, error)
	statusFn          func(ctx context.Context) (isMemPressure bool, memoryUsagePct float64, err error)
}

func (m *mockDeliveryService) GetPersonalizedContent(ctx context.Context, customerInfo *dto.CustomerRequest, channel string, placements []string) ([]dto.ContentResult, error) {
	if m.getPersonalizedFn != nil {
		return m.getPersonalizedFn(ctx, customerInfo, channel, placements)
	}
	return []dto.ContentResult{}, nil
}

func (m *mockDeliveryService) FlushCache(ctx context.Context, placements []string, isEvaluate bool) error {
	if m.flushFn != nil {
		return m.flushFn(ctx, placements, isEvaluate)
	}
	return nil
}

func (m *mockDeliveryService) GetCacheKeys(ctx context.Context) ([]string, error) {
	if m.keysFn != nil {
		return m.keysFn(ctx)
	}
	return []string{}, nil
}

func (m *mockDeliveryService) GetCacheStatus(ctx context.Context) (bool, float64, error) {
	if m.statusFn != nil {
		return m.statusFn(ctx)
	}
	return false, 0.0, nil
}

func (m *mockDeliveryService) GetCacheValue(ctx context.Context, key string) (json.RawMessage, error) {
	if m.getValueFn != nil {
		return m.getValueFn(ctx, key)
	}
	return nil, nil
}

// ---- helpers -------------------------------------------------------------

func setupRouter(svc deliveryservice.DeliveryService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.GET("/content", h.getContent)
	r.POST("/flush", h.flushCache)
	r.GET("/purge_requests", h.getStatus)
	r.GET("/purge_requests/value", h.getCacheValue)
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	var req *http.Request
	if body != "" {
		req, _ = http.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	r.ServeHTTP(w, req)
	return w
}

// ---- tests ---------------------------------------------------------------

// TestHandler_GetContent_OK verifies GET /content returns 200 with the slice from the personalized service path.
func TestHandler_GetContent_OK(t *testing.T) {
	t.Parallel()

	expected := []dto.ContentResult{
		{ContentPath: "/a", Score: 0.9, DecisionRuleType: "Mass"},
	}
	r := setupRouter(&mockDeliveryService{
		getPersonalizedFn: func(_ context.Context, customerInfo *dto.CustomerRequest, _ string, placements []string) ([]dto.ContentResult, error) {
			assert.Equal(t, "1234567890", customerInfo.CIS_ID)
			assert.Equal(t, []string{"hero"}, placements)
			return expected, nil
		},
	})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=staticContent&mode=knownContent&channel=web&placement=hero&customerIdType=CIS_ID&customerId=1234567890", "")

	assert.Equal(t, http.StatusOK, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	dataBytes, err := json.Marshal(apiResp.Data)
	require.NoError(t, err)
	var body []dto.ContentResult
	require.NoError(t, json.Unmarshal(dataBytes, &body))
	assert.Equal(t, "/a", body[0].ContentPath)
}

// TestHandler_GetContent_ServiceError verifies GET /content returns 500 on personalized service error.
func TestHandler_GetContent_ServiceError(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{
		getPersonalizedFn: func(_ context.Context, _ *dto.CustomerRequest, _ string, _ []string) ([]dto.ContentResult, error) {
			return nil, errors.New("redis down")
		},
	})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=staticContent&mode=knownContent&channel=web&placement=hero&customerIdType=CIS_ID&customerId=1234567890", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestHandler_FlushCache_OK verifies POST /flush returns 200 and calls the service.
func TestHandler_FlushCache_OK(t *testing.T) {
	t.Parallel()

	var flushedPlacements []string
	r := setupRouter(&mockDeliveryService{
		flushFn: func(_ context.Context, placements []string, isEvaluate bool) error {
			flushedPlacements = placements
			return nil
		},
	})

	w := doRequest(t, r, http.MethodPost, "/flush", `{"placements":["hero","sidebar"]}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"hero", "sidebar"}, flushedPlacements)
}

// TestHandler_FlushCache_ServiceError verifies POST /flush returns 500 on service error.
func TestHandler_FlushCache_ServiceError(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{
		flushFn: func(_ context.Context, _ []string, _ bool) error {
			return errors.New("flush failed")
		},
	})

	w := doRequest(t, r, http.MethodPost, "/flush", `{"placements":["hero"]}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestHandler_GetContent_InvalidRequestType verifies a missing or unknown requestType returns 400.
func TestHandler_GetContent_InvalidRequestType(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{})

	for _, rt := range []string{"", "unknown", "PERSONALIZED"} {
		url := "/content?placement=hero&channel=web"
		if rt != "" {
			url = "/content?requestType=" + rt + "&mode=knownContent&placement=hero&channel=web"
		}
		w := doRequest(t, r, http.MethodGet, url, "")
		assert.Equal(t, http.StatusBadRequest, w.Code, "requestType=%q should be rejected", rt)
	}
}

// TestHandler_GetContent_ArticleCategory verifies requestType=articleCategory routes to GetPersonalizedContent.
func TestHandler_GetContent_ArticleCategory(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"articleCategory", "knownContent"} {
		called := false
		r := setupRouter(&mockDeliveryService{
			getPersonalizedFn: func(_ context.Context, customerInfo *dto.CustomerRequest, _ string, placements []string) ([]dto.ContentResult, error) {
				called = true
				assert.Equal(t, "1234567890", customerInfo.CIS_ID)
				assert.Equal(t, []string{"hero"}, placements)
				return []dto.ContentResult{}, nil
			},
		})

		w := doRequest(t, r, http.MethodGet, "/content?requestType=articleCategory&mode="+mode+"&channel=web&placement=hero&customerIdType=CIS_ID&customerId=1234567890", "")

		assert.Equal(t, http.StatusOK, w.Code, "mode=%s", mode)
		assert.True(t, called, "GetPersonalizedContent should be called for articleCategory (mode=%s)", mode)
	}
}

// TestHandler_GetContent_UnspecifiedCustomerIdTypeRejectsCustomerId verifies bare customerId is rejected.
func TestHandler_GetContent_UnspecifiedCustomerIdTypeRejectsCustomerId(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=staticContent&mode=knownContent&channel=web&placement=hero&customerId=cis-123", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	assert.Equal(t, "Invalid query parameters", apiResp.Error)
}

// TestHandler_GetContent_MissingCustomerId verifies any valid requestType without customerId returns 400.
func TestHandler_GetContent_MissingCustomerId(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{})

	for _, requestType := range []string{"personalizedContent", "staticContent", "articleCategory"} {
		w := doRequest(t, r, http.MethodGet, "/content?requestType="+requestType+"&mode=knownContent&placement=hero&channel=web", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	}
}

// TestHandler_GetContent_CISID_MissingCustomerId verifies customerIdType=CIS_ID without customerId returns 400.
func TestHandler_GetContent_CISID_MissingCustomerId(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=personalizedContent&mode=knownContent&channel=web&placement=hero&customerIdType=CIS_ID", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	assert.Equal(t, "Invalid query parameters", apiResp.Error)
}

// TestHandler_GetContent_CISID_UsesCustomerId verifies customerIdType=CIS_ID with customerId passes the value as cisID.
func TestHandler_GetContent_CISID_UsesCustomerId(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"logicalBased", "knownContent"} {
		var capturedCISID string
		r := setupRouter(&mockDeliveryService{
			getPersonalizedFn: func(_ context.Context, customerInfo *dto.CustomerRequest, _ string, _ []string) ([]dto.ContentResult, error) {
				capturedCISID = customerInfo.CIS_ID
				return []dto.ContentResult{}, nil
			},
		})

		w := doRequest(t, r, http.MethodGet, "/content?requestType=personalizedContent&mode="+mode+"&channel=web&placement=hero&customerIdType=CIS_ID&customerId=1234567890", "")

		assert.Equal(t, http.StatusOK, w.Code, "mode=%s", mode)
		assert.Equal(t, "1234567890", capturedCISID, "mode=%s", mode)
	}
}

// TestHandler_GetContent_WithUserIDInContext verifies that when ctxconsts.UserIDKey is present, the handler still
// correctly passes CustomerRequest to the service.
func TestHandler_GetContent_WithUserIDInContext(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	for _, mode := range []string{"logicalBased", "knownContent"} {
		var capturedCISID string

		r := gin.New()
		r.Use(func(c *gin.Context) {
			ctx := context.WithValue(c.Request.Context(), ctxconsts.UserIDKey, userID)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		})
		h := NewHandler(&mockDeliveryService{
			getPersonalizedFn: func(_ context.Context, customerInfo *dto.CustomerRequest, _ string, _ []string) ([]dto.ContentResult, error) {
				capturedCISID = customerInfo.CIS_ID
				return []dto.ContentResult{}, nil
			},
		})
		r.GET("/content", h.getContent)

		w := doRequest(t, r, http.MethodGet, "/content?requestType=personalizedContent&mode="+mode+"&channel=web&placement=hero&customerIdType=CIS_ID&customerId=1234567890", "")

		assert.Equal(t, http.StatusOK, w.Code, "mode=%s", mode)
		assert.Equal(t, "1234567890", capturedCISID, "mode=%s", mode)
	}
}

// TestHandler_GetStatus_OK verifies GET /purge_requests returns 200 with pressure flag, usage, and keys.
func TestHandler_GetStatus_OK(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{
		statusFn: func(_ context.Context) (bool, float64, error) {
			return true, 0.75, nil
		},
		keysFn: func(_ context.Context) ([]string, error) {
			return []string{"cms:placement:hero:schedules", "rule:abc"}, nil
		},
	})

	w := doRequest(t, r, http.MethodGet, "/purge_requests", "")

	require.Equal(t, http.StatusOK, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	dataBytes, err := json.Marshal(apiResp.Data)
	require.NoError(t, err)
	var body dto.CacheStatusResponse
	require.NoError(t, json.Unmarshal(dataBytes, &body))
	assert.True(t, body.IsMemPressure)
	assert.InDelta(t, 0.75, body.MemoryUsagePct, 0.0001)
	assert.Equal(t, []string{"cms:placement:hero:schedules", "rule:abc"}, body.CacheKeys)
}

// TestHandler_GetStatus_CacheStatusError verifies GET /purge_requests returns 500 when GetCacheStatus fails.
func TestHandler_GetStatus_CacheStatusError(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{
		statusFn: func(_ context.Context) (bool, float64, error) {
			return false, 0, errors.New("status unavailable")
		},
	})

	w := doRequest(t, r, http.MethodGet, "/purge_requests", "")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	assert.Contains(t, apiResp.Error, "status unavailable")
}

// TestHandler_GetStatus_CacheKeysError verifies GET /purge_requests returns 500 when GetCacheKeys fails.
func TestHandler_GetStatus_CacheKeysError(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{
		statusFn: func(_ context.Context) (bool, float64, error) {
			return false, 0.3, nil
		},
		keysFn: func(_ context.Context) ([]string, error) {
			return nil, errors.New("keys unavailable")
		},
	})

	w := doRequest(t, r, http.MethodGet, "/purge_requests", "")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	assert.Contains(t, apiResp.Error, "keys unavailable")
}

// TestHandler_GetCacheValue_MissingKey verifies GET /purge_requests/value without key param returns 400.
func TestHandler_GetCacheValue_MissingKey(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{})
	w := doRequest(t, r, http.MethodGet, "/purge_requests/value", "")

	require.Equal(t, http.StatusBadRequest, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	assert.Contains(t, apiResp.Error, "key")
}

// TestHandler_GetCacheValue_OK verifies GET /purge_requests/value?key=... returns 200 with the value from the service.
func TestHandler_GetCacheValue_OK(t *testing.T) {
	t.Parallel()

	var capturedKey string
	r := setupRouter(&mockDeliveryService{
		getValueFn: func(_ context.Context, key string) (json.RawMessage, error) {
			capturedKey = key
			return json.RawMessage(`{"id":"abc"}`), nil
		},
	})
	w := doRequest(t, r, http.MethodGet, "/purge_requests/value?key=rule:abc", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "rule:abc", capturedKey)
}

// TestHandler_GetCacheValue_ServiceError verifies GET /purge_requests/value returns 500 when the service errors.
func TestHandler_GetCacheValue_ServiceError(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{
		getValueFn: func(_ context.Context, _ string) (json.RawMessage, error) {
			return nil, errors.New("key not found in cache")
		},
	})
	w := doRequest(t, r, http.MethodGet, "/purge_requests/value?key=rule:missing", "")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	assert.Contains(t, apiResp.Error, "key not found in cache")
}
