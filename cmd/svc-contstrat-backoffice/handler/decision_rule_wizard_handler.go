package handler

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kbank-ecms/internal/delivery/http/dto"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/service"

	localservice "kbank-ecms/cmd/svc-contstrat-backoffice/service"
)

// decisionRuleWizardServicer is the interface the handler depends on.
type decisionRuleWizardServicer interface {
	SaveStep1(ctx context.Context, req dto.WizardStep1Request) (*dto.WizardStep1Response, error)
	GetConditions(ctx context.Context, id uuid.UUID) (*dto.WizardConditionsResponse, error)
	GetRuleSets(ctx context.Context, id uuid.UUID) (*dto.WizardRuleSetsResponse, error)
	SaveStep2(ctx context.Context, id uuid.UUID, req dto.WizardStep2Request) (*dto.WizardStep2Response, error)
	GetSchedules(ctx context.Context, id uuid.UUID) (*dto.WizardSchedulesResponse, error)
	SaveStep3(ctx context.Context, id uuid.UUID, req dto.WizardStep3Request) (*dto.WizardStep3Response, error)
	ActivateStep4(ctx context.Context, id uuid.UUID) (*dto.WizardStep4Response, error)
	ListDecisionRules(ctx context.Context, f domainrepo.DecisionRuleListFilter) ([]*dto.DecisionRuleListItemResponse, int64, error)
}

// DecisionRuleWizardHandler handles HTTP requests for the wizard API.
type DecisionRuleWizardHandler struct {
	service         decisionRuleWizardServicer
	activateService *localservice.ActivationService
}

// NewDecisionRuleWizardHandler creates a new DecisionRuleWizardHandler.
func NewDecisionRuleWizardHandler(svc *service.DecisionRuleWizardService, activateSvc *localservice.ActivationService) *DecisionRuleWizardHandler {
	return &DecisionRuleWizardHandler{service: svc, activateService: activateSvc}
}

// CreateDecisionRule handles POST /decision-rules.
//
// @Summary Create a draft decision rule (Wizard Step 1)
// @Description Creates a DecisionRule in DRAFT status with its condition tree.
// @Tags DecisionRuleWizard
// @Accept json
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param body body dto.WizardStep1Request true "Step 1 request body"
// @Success 201 {object} dto.APIResponse{data=dto.WizardStep1Response}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 422 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules [post]
func (h *DecisionRuleWizardHandler) CreateDecisionRule(c *gin.Context) {
	var req dto.WizardStep1Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	resp, err := h.service.SaveStep1(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.APIResponse{Data: resp})
}

// GetConditions handles GET /decision-rules/:id/conditions.
//
// @Summary Get condition tree for edit (Wizard Step 1 read)
// @Tags DecisionRuleWizard
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Decision Rule ID (UUID)"
// @Success 200 {object} dto.APIResponse{data=dto.WizardConditionsResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules/{id}/conditions [get]
func (h *DecisionRuleWizardHandler) GetConditions(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	resp, err := h.service.GetConditions(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: resp})
}

// GetRuleSets handles GET /decision-rules/:id/rule-sets.
//
// @Summary Get rule set columns and rows (Wizard Step 2 read)
// @Tags DecisionRuleWizard
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Decision Rule ID (UUID)"
// @Success 200 {object} dto.APIResponse{data=dto.WizardRuleSetsResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules/{id}/rule-sets [get]
func (h *DecisionRuleWizardHandler) GetRuleSets(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	resp, err := h.service.GetRuleSets(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: resp})
}

// SaveRuleSets handles PUT /decision-rules/:id/rule-sets.
//
// @Summary Save rule sets (Wizard Step 2 write)
// @Tags DecisionRuleWizard
// @Accept json
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Decision Rule ID (UUID)"
// @Param body body dto.WizardStep2Request true "Step 2 request body"
// @Success 200 {object} dto.APIResponse{data=dto.WizardStep2Response}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 422 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules/{id}/rule-sets [put]
func (h *DecisionRuleWizardHandler) SaveRuleSets(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var req dto.WizardStep2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}
	resp, err := h.service.SaveStep2(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: resp})
}

// GetSchedules handles GET /decision-rules/:id/schedules.
//
// @Summary Get schedules for a decision rule (Wizard Step 3 read)
// @Tags DecisionRuleWizard
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Decision Rule ID (UUID)"
// @Success 200 {object} dto.APIResponse{data=dto.WizardSchedulesResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules/{id}/schedules [get]
func (h *DecisionRuleWizardHandler) GetSchedules(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	resp, err := h.service.GetSchedules(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: resp})
}

// ActivateDecisionRule handles PUT /decision-rules/:id/activate.
//
// @Summary Activate a decision rule (Wizard Step 4)
// @Tags DecisionRuleWizard
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Decision Rule ID (UUID)"
// @Success 200 {object} dto.APIResponse{data=dto.WizardStep4Response}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 422 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules/{id}/activate [put]
func (h *DecisionRuleWizardHandler) ActivateDecisionRule(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	ctx := c.Request.Context()

	resp, err := h.service.ActivateStep4(ctx, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Publish cache-invalidation pings for all placements touched by this decision rule's schedules.
	h.activateService.ActivatePublish(ctx, resp.ID, resp.Schedules)

	c.JSON(http.StatusOK, dto.APIResponse{Data: resp})
}

// SaveSchedules handles PUT /decision-rules/:id/schedules.
//
// @Summary Save schedules for a decision rule (Wizard Step 3)
// @Tags DecisionRuleWizard
// @Accept json
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Decision Rule ID (UUID)"
// @Param body body dto.WizardStep3Request true "Step 3 request body"
// @Success 200 {object} dto.APIResponse{data=dto.WizardStep3Response}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 409 {object} dto.APIResponse
// @Failure 422 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules/{id}/schedules [put]
func (h *DecisionRuleWizardHandler) SaveSchedules(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var req dto.WizardStep3Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}
	resp, err := h.service.SaveStep3(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: resp})
}

// ListDecisionRules handles GET /decision-rules.
//
// @Summary List decision rules
// @Tags DecisionRuleWizard
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param type query string false "Filter by type (MASS, AUDIENCE, SALES_TARGET, NON_SALES)"
// @Param evaluateType query string false "Filter by evaluateType (SCORING, SEGMENT, ELIGIBLE)"
// @Param status query string false "Filter by status (DRAFT, ACTIVE, INACTIVE)"
// @Param keyword query string false "Keyword search on name and decisionRuleId"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Page size (default: 20, max: 100)"
// @Success 200 {object} dto.APIResponse{data=[]dto.DecisionRuleListItemResponse}
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules [get]
func (h *DecisionRuleWizardHandler) ListDecisionRules(c *gin.Context) {
	const maxLimit = 100
	page := 1
	limit := 0 // 0 = no limit, return all

	if rawPage := c.Query("page"); rawPage != "" {
		v, err := strconv.Atoi(rawPage)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "page must be a positive integer"})
			return
		}
		page = v
	}

	if rawLimit := c.Query("limit"); rawLimit != "" {
		v, err := strconv.Atoi(rawLimit)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "limit must be a positive integer"})
			return
		}
		if v > maxLimit {
			v = maxLimit
		}
		limit = v
	}

	f := domainrepo.DecisionRuleListFilter{
		Type:         c.Query("type"),
		EvaluateType: c.Query("evaluateType"),
		Status:       c.Query("status"),
		Keyword:      c.Query("keyword"),
		Page:         page,
		Limit:        limit,
	}
	items, total, err := h.service.ListDecisionRules(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to list decision rules"})
		return
	}
	if items == nil {
		items = []*dto.DecisionRuleListItemResponse{}
	}

	var pagination *dto.Pagination
	if limit > 0 {
		pagination = &dto.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: int(math.Ceil(float64(total) / float64(limit))),
		}
	} else {
		pagination = &dto.Pagination{
			Page:       1,
			Limit:      int(total),
			TotalItems: total,
			TotalPages: 1,
		}
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: items, Pagination: pagination})
}

// handleError maps service sentinel errors to HTTP status codes.
func (h *DecisionRuleWizardHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrWizardNotFound):
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: err.Error()})
	case errors.Is(err, service.ErrWizardValidation):
		c.JSON(http.StatusUnprocessableEntity, dto.APIResponse{Error: err.Error()})
	case errors.Is(err, service.ErrWizardConflict):
		c.JSON(http.StatusConflict, dto.APIResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "internal server error"})
	}
}

// parseUUID parses a UUID path parameter and writes 400 on failure.
func parseUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid " + param + " format"})
		return uuid.Nil, false
	}
	return id, true
}
