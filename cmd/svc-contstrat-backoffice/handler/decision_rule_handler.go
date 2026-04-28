package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/service"
)

// decisionRuleServicer is the interface the handler depends on, enabling test doubles.
type decisionRuleServicer interface {
	GetDecisionRuleByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*entity.DecisionRule, error)
}

// DecisionRuleHandler handles HTTP requests for decision rule management.
type DecisionRuleHandler struct {
	service decisionRuleServicer
}

// NewDecisionRuleHandler creates a new DecisionRuleHandler with the injected service.
func NewDecisionRuleHandler(svc *service.DecisionRuleService) *DecisionRuleHandler {
	return &DecisionRuleHandler{service: svc}
}

// GetDecisionRuleBySchedule handles GET /decision-rules/schedule/{scheduleId}.
//
//	@Summary		Get a decision rule by schedule ID
//	@Description	Returns the decision rule associated with the given schedule ID, including all rule conditions and attributes.
//	@Tags			DecisionRules
//	@Produce		json
//	@Param			X-User-Id	header		string	true	"User ID (UUID)"
//	@Param			id			query		string	true	"Schedule ID (UUID)"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		404			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/decision-rules/schedule/{scheduleId} [get]
func (h *DecisionRuleHandler) GetDecisionRuleBySchedule(c *gin.Context) {
	scheduleIdStr := c.Param("scheduleId")
	if scheduleIdStr == "" {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "missing scheduleId parameter"})
		return
	}

	scheduleID, err := uuid.Parse(scheduleIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid schedule ID format"})
		return
	}

	decisionRule, err := h.service.GetDecisionRuleByScheduleID(c.Request.Context(), scheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve decision rule"})
		return
	}

	if decisionRule == nil {
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "decision rule not found for the given schedule"})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToDecisionRuleResponse(decisionRule)})
}
