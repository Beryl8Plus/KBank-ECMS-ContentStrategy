package handler

import (
	"context"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	"kbank-ecms/internal/service"
)

// scheduleServicer is the interface the handler depends on, enabling test doubles.
type scheduleServicer interface {
	CreateSchedule(ctx context.Context, schedule *entity.Schedule) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*entity.Schedule, error)
	ListSchedules(ctx context.Context) ([]*entity.Schedule, error)
	ListSchedulesPaginated(ctx context.Context, page, limit int) ([]*entity.Schedule, int64, error)
	UpdateSchedule(ctx context.Context, schedule *entity.Schedule) error
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
}

// ScheduleHandler handles HTTP requests for schedule management.
type ScheduleHandler struct {
	service scheduleServicer
}

// NewScheduleHandler creates a new ScheduleHandler with the injected service.
func NewScheduleHandler(svc *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{service: svc}
}

// CreateSchedule handles POST /schedules.
//
//	@Summary		Create a schedule
//	@Description	Create a new schedule linking a decision rule to a placement with recurrence configuration.
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			X-User-Id	header		string						true	"User ID (UUID)"
//	@Param			body		body		dto.CreateScheduleRequest	true	"Create schedule request body"
//	@Success		201			{object}	dto.APIResponse
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		422			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/schedules [post]
func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	var req dto.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	if req.EffectiveFrom.After(req.EffectiveUntil) || req.EffectiveFrom.Equal(req.EffectiveUntil) {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "effectiveFrom must be before effectiveUntil"})
		return
	}

	if !req.RecurrenceType.IsValid() {
		req.RecurrenceType = enums.RecurrenceTypeOnce // Default to ONCE if invalid
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "Asia/Bangkok"
	}

	schedule := &entity.Schedule{
		DecisionRuleID: req.DecisionRuleID,
		PlacementID:    req.PlacementID,
		CalendarID:     req.CalendarID,
		RecurrenceType: req.RecurrenceType,
		RecurrenceRule: req.RecurrenceRule,
		EffectiveFrom:  req.EffectiveFrom,
		EffectiveUntil: req.EffectiveUntil,
		TimeOfDayStart: req.TimeOfDayStart,
		TimeOfDayEnd:   req.TimeOfDayEnd,
		AllDay:         req.AllDay,
		Timezone:       timezone,
		IsActive:       req.IsActive,
	}

	if err := h.service.CreateSchedule(c.Request.Context(), schedule); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.APIResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{Data: dto.ToScheduleResponse(schedule)})
}

// ListSchedules handles GET /schedules.
//
//	@Summary		List all schedules
//	@Description	Returns schedules with cursor-style offset pagination. Defaults: page=1, limit=20 (max 100).
//	@Tags			Schedules
//	@Produce		json
//	@Param			X-User-Id	header		string	true	"User ID (UUID)"
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			limit		query		int		false	"Page size (default: 20, max: 100)"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/schedules [get]
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	page, limit, ok := parsePaginationParams(c)
	if !ok {
		return
	}

	schedules, total, err := h.service.ListSchedulesPaginated(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve schedules"})
		return
	}

	responses := make([]dto.ScheduleResponse, 0, len(schedules))
	for _, s := range schedules {
		responses = append(responses, dto.ToScheduleResponse(s))
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

// parsePaginationParams reads page and limit from query string, validates them,
// and writes a 400 response + returns false on invalid input.
func parsePaginationParams(c *gin.Context) (page, limit int, ok bool) {
	const defaultPage = 1
	const defaultLimit = 20
	const maxLimit = 100

	page = defaultPage
	limit = defaultLimit

	if raw := c.Query("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "page must be a positive integer"})
			return 0, 0, false
		}
		page = v
	}

	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "limit must be a positive integer"})
			return 0, 0, false
		}
		if v > maxLimit {
			v = maxLimit
		}
		limit = v
	}

	return page, limit, true
}

// GetSchedule handles GET /schedules/:id.
//
//	@Summary		Get a schedule by ID
//	@Description	Returns a single schedule by its UUID.
//	@Tags			Schedules
//	@Produce		json
//	@Param			X-User-Id	header		string	true	"User ID (UUID)"
//	@Param			id			path		string	true	"Schedule ID (UUID)"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		404			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/schedules/{id} [get]
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid schedule ID"})
		return
	}

	schedule, err := h.service.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve schedule"})
		return
	}
	if schedule == nil {
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "schedule not found"})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToScheduleResponse(schedule)})
}

// UpdateSchedule handles PUT /schedules/:id.
//
//	@Summary		Update a schedule
//	@Description	Updates an existing schedule. DecisionRuleID and PlacementID are immutable.
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			X-User-Id	header		string						true	"User ID (UUID)"
//	@Param			id			path		string						true	"Schedule ID (UUID)"
//	@Param			body		body		dto.UpdateScheduleRequest	true	"Update schedule request body"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		404			{object}	dto.APIResponse
//	@Failure		422			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/schedules/{id} [put]
func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid schedule ID"})
		return
	}

	existing, err := h.service.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve schedule"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "schedule not found"})
		return
	}

	var req dto.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	if !req.RecurrenceType.IsValid() {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid recurrenceType: must be ONCE, RRULE, or CALENDAR"})
		return
	}

	if req.EffectiveFrom.After(req.EffectiveUntil) || req.EffectiveFrom.Equal(req.EffectiveUntil) {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "effectiveFrom must be before effectiveUntil"})
		return
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "Asia/Bangkok"
	}

	// Apply update fields; preserve immutable DecisionRuleID and PlacementID.
	existing.CalendarID = req.CalendarID
	existing.RecurrenceType = req.RecurrenceType
	existing.RecurrenceRule = req.RecurrenceRule
	existing.EffectiveFrom = req.EffectiveFrom
	existing.EffectiveUntil = req.EffectiveUntil
	existing.TimeOfDayStart = req.TimeOfDayStart
	existing.TimeOfDayEnd = req.TimeOfDayEnd
	existing.AllDay = req.AllDay
	existing.Timezone = timezone
	existing.IsActive = req.IsActive

	if err := h.service.UpdateSchedule(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.APIResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToScheduleResponse(existing)})
}

// DeleteSchedule handles DELETE /schedules/:id.
//
//	@Summary		Delete a schedule
//	@Description	Soft-deletes a schedule by its UUID.
//	@Tags			Schedules
//	@Produce		json
//	@Param			X-User-Id	header	string	true	"User ID (UUID)"
//	@Param			id			path	string	true	"Schedule ID (UUID)"
//	@Success		204			"No Content"
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/schedules/{id} [delete]
func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid schedule ID"})
		return
	}

	if err := h.service.DeleteSchedule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to delete schedule"})
		return
	}

	c.Status(http.StatusNoContent)
}
