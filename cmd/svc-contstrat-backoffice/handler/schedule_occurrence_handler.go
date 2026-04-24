package handler

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/service"
)

// scheduleOccurrenceServicer is the interface the handler depends on.
type scheduleOccurrenceServicer interface {
	ListByScheduleID(ctx context.Context, scheduleID uuid.UUID, page, limit int) ([]*entity.ScheduleOccurrence, int64, error)
	ListActiveAt(ctx context.Context, at time.Time) ([]*entity.ScheduleOccurrence, error)
}

// ScheduleOccurrenceHandler handles HTTP requests for schedule occurrence queries.
type ScheduleOccurrenceHandler struct {
	service scheduleOccurrenceServicer
}

// NewScheduleOccurrenceHandler creates a new ScheduleOccurrenceHandler.
func NewScheduleOccurrenceHandler(svc *service.ScheduleOccurrenceService) *ScheduleOccurrenceHandler {
	return &ScheduleOccurrenceHandler{service: svc}
}

// ListOccurrencesBySchedule handles GET /schedules/:id/occurrences.
//
// @Summary List occurrences for a schedule
// @Description Returns a paginated list of pre-computed occurrences for a given schedule, ordered by occurrence start time ascending.
// @Tags ScheduleOccurrences
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Schedule ID (UUID)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Page size (default: 20, max: 100)"
// @Success 200 {object} dto.APIResponse{data=[]dto.ScheduleOccurrenceResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /schedules/{id}/occurrences [get]
func (h *ScheduleOccurrenceHandler) ListOccurrencesBySchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid schedule ID"})
		return
	}

	page, limit, ok := parsePaginationParams(c)
	if !ok {
		return
	}

	occurrences, total, err := h.service.ListByScheduleID(c.Request.Context(), id, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve occurrences"})
		return
	}

	responses := make([]dto.ScheduleOccurrenceResponse, 0, len(occurrences))
	for _, o := range occurrences {
		responses = append(responses, dto.ToScheduleOccurrenceResponse(o))
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, dto.APIResponse{
		Data: responses,
		Pagination: &dto.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	})
}

// ListActiveOccurrences handles GET /schedule-occurrences/active.
//
// @Summary List active occurrences at a given time
// @Description Returns all occurrences that are ACTIVE and whose window contains the requested time (defaults to now).
// @Tags ScheduleOccurrences
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param at query string false "RFC3339 timestamp to query active occurrences at (default: current server time)"
// @Success 200 {object} dto.APIResponse{data=[]dto.ScheduleOccurrenceResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /schedule-occurrences/active [get]
func (h *ScheduleOccurrenceHandler) ListActiveOccurrences(c *gin.Context) {
	at := time.Now()
	if raw := c.Query("at"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "at must be a valid RFC3339 timestamp (e.g. 2026-04-21T10:00:00Z)"})
			return
		}
		at = parsed
	}

	occurrences, err := h.service.ListActiveAt(c.Request.Context(), at)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve active occurrences"})
		return
	}

	responses := make([]dto.ScheduleOccurrenceResponse, 0, len(occurrences))
	for _, o := range occurrences {
		responses = append(responses, dto.ToScheduleOccurrenceResponse(o))
	}

	c.JSON(http.StatusOK, dto.APIResponse{Data: responses})
}
