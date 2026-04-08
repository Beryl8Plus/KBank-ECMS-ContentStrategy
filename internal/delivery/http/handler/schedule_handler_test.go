package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- mock implementing scheduleServicer ---

type mockScheduleService struct {
	createFn        func(ctx context.Context, schedule *entity.Schedule) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*entity.Schedule, error)
	listFn          func(ctx context.Context) ([]*entity.Schedule, error)
	listPaginatedFn func(ctx context.Context, page, limit int) ([]*entity.Schedule, int64, error)
	updateFn        func(ctx context.Context, schedule *entity.Schedule) error
	deleteFn        func(ctx context.Context, id uuid.UUID) error
}

func (m *mockScheduleService) CreateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	if m.createFn != nil {
		return m.createFn(ctx, schedule)
	}
	return nil
}

func (m *mockScheduleService) GetScheduleByID(ctx context.Context, id uuid.UUID) (*entity.Schedule, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	s := &entity.Schedule{}
	s.ID = id
	return s, nil
}

func (m *mockScheduleService) ListSchedules(ctx context.Context) ([]*entity.Schedule, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []*entity.Schedule{}, nil
}

func (m *mockScheduleService) ListSchedulesPaginated(ctx context.Context, page, limit int) ([]*entity.Schedule, int64, error) {
	if m.listPaginatedFn != nil {
		return m.listPaginatedFn(ctx, page, limit)
	}
	return []*entity.Schedule{}, 0, nil
}

func (m *mockScheduleService) UpdateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, schedule)
	}
	return nil
}

func (m *mockScheduleService) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// --- helpers ---

func newTestRouter(h *ScheduleHandler) *gin.Engine {
	r := gin.New()
	r.POST("/schedules", h.CreateSchedule)
	r.GET("/schedules", h.ListSchedules)
	r.GET("/schedules/:id", h.GetSchedule)
	r.PUT("/schedules/:id", h.UpdateSchedule)
	r.DELETE("/schedules/:id", h.DeleteSchedule)
	return r
}

func performRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
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

func validCreateRequest() dto.CreateScheduleRequest {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	return dto.CreateScheduleRequest{
		DecisionRuleID: uuid.New(),
		PlacementID:    uuid.New(),
		RecurrenceType: enums.RecurrenceTypeOnce,
		EffectiveFrom:  now,
		EffectiveUntil: later,
		IsActive:       true,
	}
}

func validUpdateRequest() dto.UpdateScheduleRequest {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	return dto.UpdateScheduleRequest{
		RecurrenceType: enums.RecurrenceTypeOnce,
		EffectiveFrom:  now,
		EffectiveUntil: later,
		IsActive:       true,
	}
}

// --- CreateSchedule ---

func TestScheduleHandler_CreateSchedule_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	w := performRequest(t, r, http.MethodPost, "/schedules", `{"allDay": true}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScheduleHandler_CreateSchedule_InvalidRecurrenceType(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	req := validCreateRequest()
	req.RecurrenceType = "INVALID"
	body, _ := json.Marshal(req)
	w := performRequest(t, r, http.MethodPost, "/schedules", string(body))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScheduleHandler_CreateSchedule_EffectiveFromAfterUntil(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	req := validCreateRequest()
	req.EffectiveFrom = req.EffectiveUntil.Add(time.Hour)
	body, _ := json.Marshal(req)
	w := performRequest(t, r, http.MethodPost, "/schedules", string(body))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScheduleHandler_CreateSchedule_ServiceError(t *testing.T) {
	t.Parallel()
	mock := &mockScheduleService{
		createFn: func(_ context.Context, _ *entity.Schedule) error { return assert.AnError },
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	body, _ := json.Marshal(validCreateRequest())
	w := performRequest(t, r, http.MethodPost, "/schedules", string(body))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestScheduleHandler_CreateSchedule_Success(t *testing.T) {
	t.Parallel()
	req := validCreateRequest()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	body, _ := json.Marshal(req)
	w := performRequest(t, r, http.MethodPost, "/schedules", string(body))
	assert.Equal(t, http.StatusCreated, w.Code)

	var apiResp struct {
		Code string               `json:"code"`
		Data dto.ScheduleResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apiResp))
	assert.Equal(t, "201", apiResp.Code)
	assert.Equal(t, req.PlacementID, apiResp.Data.PlacementID)
}

// --- ListSchedules ---

func TestScheduleHandler_ListSchedules_DefaultPagination(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	w := performRequest(t, r, http.MethodGet, "/schedules", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var apiResp struct {
		Code       string                 `json:"code"`
		Data       []dto.ScheduleResponse `json:"data"`
		Pagination *dto.Pagination        `json:"pagination"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apiResp))
	assert.Equal(t, "200", apiResp.Code)
	assert.Empty(t, apiResp.Data)
	require.NotNil(t, apiResp.Pagination)
	assert.Equal(t, 1, apiResp.Pagination.Page)
	assert.Equal(t, 20, apiResp.Pagination.Limit)
	assert.Equal(t, int64(0), apiResp.Pagination.TotalItems)
}

func TestScheduleHandler_ListSchedules_ReturnsList(t *testing.T) {
	t.Parallel()
	s1 := &entity.Schedule{}
	s1.ID = uuid.New()
	mock := &mockScheduleService{
		listPaginatedFn: func(_ context.Context, _, _ int) ([]*entity.Schedule, int64, error) {
			return []*entity.Schedule{s1}, 1, nil
		},
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	w := performRequest(t, r, http.MethodGet, "/schedules?page=1&limit=10", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var apiResp struct {
		Code       string                 `json:"code"`
		Data       []dto.ScheduleResponse `json:"data"`
		Pagination *dto.Pagination        `json:"pagination"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apiResp))
	assert.Equal(t, "200", apiResp.Code)
	assert.Len(t, apiResp.Data, 1)
	assert.Equal(t, s1.ID, apiResp.Data[0].ID)
	require.NotNil(t, apiResp.Pagination)
	assert.Equal(t, int64(1), apiResp.Pagination.TotalItems)
	assert.Equal(t, 1, apiResp.Pagination.TotalPages)
}

func TestScheduleHandler_ListSchedules_InvalidPage(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	w := performRequest(t, r, http.MethodGet, "/schedules?page=0", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScheduleHandler_ListSchedules_InvalidLimit(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	w := performRequest(t, r, http.MethodGet, "/schedules?limit=abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScheduleHandler_ListSchedules_LimitCappedAt100(t *testing.T) {
	t.Parallel()
	var capturedLimit int
	mock := &mockScheduleService{
		listPaginatedFn: func(_ context.Context, _, limit int) ([]*entity.Schedule, int64, error) {
			capturedLimit = limit
			return []*entity.Schedule{}, 0, nil
		},
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	w := performRequest(t, r, http.MethodGet, "/schedules?limit=999", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 100, capturedLimit)
}

func TestScheduleHandler_ListSchedules_ServiceError(t *testing.T) {
	t.Parallel()
	mock := &mockScheduleService{
		listPaginatedFn: func(_ context.Context, _, _ int) ([]*entity.Schedule, int64, error) {
			return nil, 0, assert.AnError
		},
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	w := performRequest(t, r, http.MethodGet, "/schedules", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- GetSchedule ---

func TestScheduleHandler_GetSchedule_InvalidID(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	w := performRequest(t, r, http.MethodGet, "/schedules/not-a-uuid", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScheduleHandler_GetSchedule_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockScheduleService{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*entity.Schedule, error) { return nil, nil },
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	w := performRequest(t, r, http.MethodGet, "/schedules/"+uuid.New().String(), "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestScheduleHandler_GetSchedule_Found(t *testing.T) {
	t.Parallel()
	found := &entity.Schedule{}
	found.ID = uuid.New()
	mock := &mockScheduleService{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*entity.Schedule, error) { return found, nil },
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	w := performRequest(t, r, http.MethodGet, "/schedules/"+found.ID.String(), "")
	assert.Equal(t, http.StatusOK, w.Code)

	var apiResp struct {
		Code string               `json:"code"`
		Data dto.ScheduleResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apiResp))
	assert.Equal(t, "200", apiResp.Code)
	assert.Equal(t, found.ID, apiResp.Data.ID)
}

// --- UpdateSchedule ---

func TestScheduleHandler_UpdateSchedule_InvalidID(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	body, _ := json.Marshal(validUpdateRequest())
	w := performRequest(t, r, http.MethodPut, "/schedules/bad-id", string(body))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScheduleHandler_UpdateSchedule_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockScheduleService{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*entity.Schedule, error) { return nil, nil },
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	body, _ := json.Marshal(validUpdateRequest())
	w := performRequest(t, r, http.MethodPut, "/schedules/"+uuid.New().String(), string(body))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestScheduleHandler_UpdateSchedule_ServiceError(t *testing.T) {
	t.Parallel()
	existing := &entity.Schedule{}
	existing.ID = uuid.New()
	mock := &mockScheduleService{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*entity.Schedule, error) { return existing, nil },
		updateFn:  func(_ context.Context, _ *entity.Schedule) error { return assert.AnError },
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	body, _ := json.Marshal(validUpdateRequest())
	w := performRequest(t, r, http.MethodPut, "/schedules/"+existing.ID.String(), string(body))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestScheduleHandler_UpdateSchedule_Success(t *testing.T) {
	t.Parallel()
	existing := &entity.Schedule{}
	existing.ID = uuid.New()
	mock := &mockScheduleService{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*entity.Schedule, error) { return existing, nil },
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	body, _ := json.Marshal(validUpdateRequest())
	w := performRequest(t, r, http.MethodPut, "/schedules/"+existing.ID.String(), string(body))
	assert.Equal(t, http.StatusOK, w.Code)

	var apiResp struct {
		Code string               `json:"code"`
		Data dto.ScheduleResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apiResp))
	assert.Equal(t, "200", apiResp.Code)
	assert.Equal(t, existing.ID, apiResp.Data.ID)
}

// --- DeleteSchedule ---

func TestScheduleHandler_DeleteSchedule_InvalidID(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	w := performRequest(t, r, http.MethodDelete, "/schedules/bad-id", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScheduleHandler_DeleteSchedule_ServiceError(t *testing.T) {
	t.Parallel()
	mock := &mockScheduleService{
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return assert.AnError },
	}
	r := newTestRouter(&ScheduleHandler{service: mock})
	w := performRequest(t, r, http.MethodDelete, "/schedules/"+uuid.New().String(), "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestScheduleHandler_DeleteSchedule_Success(t *testing.T) {
	t.Parallel()
	r := newTestRouter(&ScheduleHandler{service: &mockScheduleService{}})
	w := performRequest(t, r, http.MethodDelete, "/schedules/"+uuid.New().String(), "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}
