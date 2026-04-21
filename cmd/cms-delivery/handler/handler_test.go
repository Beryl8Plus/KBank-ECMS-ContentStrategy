package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ service.DeliveryService = (*mockDeliveryService)(nil)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- mock ----------------------------------------------------------------

type mockDeliveryService struct {
	getPersonalizedFn func(ctx context.Context, cisID, userID string, placements []string, userAttrs map[string]json.RawMessage) ([]dto.ContentResult, error)
	flushFn           func(ctx context.Context, placements []string, isEvaluate bool) error
}

func (m *mockDeliveryService) GetPersonalizedContent(ctx context.Context, cisID, userID string, placements []string, userAttrs map[string]json.RawMessage) ([]dto.ContentResult, error) {
	if m.getPersonalizedFn != nil {
		return m.getPersonalizedFn(ctx, cisID, userID, placements, userAttrs)
	}
	return []dto.ContentResult{}, nil
}

func (m *mockDeliveryService) FlushCache(ctx context.Context, placements []string, isEvaluate bool) error {
	if m.flushFn != nil {
		return m.flushFn(ctx, placements, isEvaluate)
	}
	return nil
}

// ---- helpers -------------------------------------------------------------

func setupRouter(svc service.DeliveryService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.GET("/content", h.getContent)
	r.POST("/flush", h.flushCache)
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
		{ContentPath: "/a", Score: 0.9, RuleSetType: "Mass"},
	}
	r := setupRouter(&mockDeliveryService{
		getPersonalizedFn: func(_ context.Context, cisID, _ string, placements []string, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
			assert.Equal(t, "cis-123", cisID)
			assert.Equal(t, []string{"hero"}, placements)
			return expected, nil
		},
	})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=staticContent&placement=hero&customerIdType=CIS_ID&customerId=cis-123", "")

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
		getPersonalizedFn: func(_ context.Context, _, _ string, _ []string, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
			return nil, errors.New("redis down")
		},
	})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=staticContent&placement=hero&customerIdType=CIS_ID&customerId=cis-123", "")

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
		url := "/content?placement=hero"
		if rt != "" {
			url = "/content?requestType=" + rt + "&placement=hero"
		}
		w := doRequest(t, r, http.MethodGet, url, "")
		assert.Equal(t, http.StatusBadRequest, w.Code, "requestType=%q should be rejected", rt)
	}
}

// TestHandler_GetContent_ArticleCategory verifies requestType=articleCategory routes to GetPersonalizedContent.
func TestHandler_GetContent_ArticleCategory(t *testing.T) {
	t.Parallel()

	called := false
	r := setupRouter(&mockDeliveryService{
		getPersonalizedFn: func(_ context.Context, cisID, _ string, placements []string, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
			called = true
			assert.Equal(t, "cis-123", cisID)
			assert.Equal(t, []string{"hero"}, placements)
			return []dto.ContentResult{}, nil
		},
	})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=articleCategory&placement=hero&customerIdType=CIS_ID&customerId=cis-123", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "GetPersonalizedContent should be called for articleCategory")
}

// TestHandler_GetContent_UnspecifiedCustomerIdTypeRejectsCustomerId verifies bare customerId is rejected.
func TestHandler_GetContent_UnspecifiedCustomerIdTypeRejectsCustomerId(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=staticContent&placement=hero&customerId=cis-123", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	assert.Contains(t, apiResp.Error, "customerId")
}

// TestHandler_GetContent_MissingCustomerId verifies any valid requestType without customerId returns 400.
func TestHandler_GetContent_MissingCustomerId(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{})

	for _, requestType := range []string{"personalizedContent", "staticContent", "articleCategory"} {
		w := doRequest(t, r, http.MethodGet, "/content?requestType="+requestType+"&placement=hero", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	}
}

// TestHandler_GetContent_CISID_MissingCustomerId verifies customerIdType=CIS_ID without customerId returns 400.
func TestHandler_GetContent_CISID_MissingCustomerId(t *testing.T) {
	t.Parallel()

	r := setupRouter(&mockDeliveryService{})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=personalizedContent&placement=hero&customerIdType=CIS_ID", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var apiResp dto.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&apiResp))
	assert.Contains(t, apiResp.Error, "customerId")
}

// TestHandler_GetContent_CISID_UsesCustomerId verifies customerIdType=CIS_ID with customerId passes the value as cisID.
func TestHandler_GetContent_CISID_UsesCustomerId(t *testing.T) {
	t.Parallel()

	var capturedCISID string
	r := setupRouter(&mockDeliveryService{
		getPersonalizedFn: func(_ context.Context, cisID, _ string, _ []string, _ map[string]json.RawMessage) ([]dto.ContentResult, error) {
			capturedCISID = cisID
			return []dto.ContentResult{}, nil
		},
	})

	w := doRequest(t, r, http.MethodGet, "/content?requestType=personalizedContent&placement=hero&customerIdType=CIS_ID&customerId=12345", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "12345", capturedCISID)
}
