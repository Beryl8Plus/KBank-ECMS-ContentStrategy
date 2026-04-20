package handler

import (
	"context"
	"net/http"
	"time"

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

// setDecisionRuleResponseHeaders sets the standard custom response headers used across all endpoints.
func setDecisionRuleResponseHeaders(c *gin.Context, statusCode string, statusMsg string) {
	c.Header("Content-Type", "application/json; charset=UTF-8")
	c.Header("Request-ID", c.GetHeader("requestID"))
	c.Header("Request-Time", time.Now().Format("2006-01-02T15:04:05.000"))
	c.Header("Status-Code", statusCode)
	c.Header("Status-Msg", statusMsg)
	c.Header("Access-Control-Expose-Headers", "Request-ID, Request-Time, Status-Code, Status-Msg")
}

// GetDecisionRuleBySchedule handles GET /decision-rules/schedule/{scheduleId}.
//
// @Summary Get a decision rule by schedule ID
// @Description Returns the decision rule associated with the given schedule ID, including all rule conditions and attributes.
// @Tags DecisionRules
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id query string true "Schedule ID (UUID)"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /decision-rules/schedule/{scheduleId} [get]
func (h *DecisionRuleHandler) GetDecisionRuleBySchedule(c *gin.Context) {
	scheduleIdStr := c.Param("scheduleId")
	if scheduleIdStr == "" {
		setDecisionRuleResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "missing scheduleId parameter"})
		return
	}

	scheduleID, err := uuid.Parse(scheduleIdStr)
	if err != nil {
		setDecisionRuleResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid schedule ID format"})
		return
	}

	decisionRule, err := h.service.GetDecisionRuleByScheduleID(c.Request.Context(), scheduleID)
	if err != nil {
		setDecisionRuleResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve decision rule"})
		return
	}

	if decisionRule == nil {
		setDecisionRuleResponseHeaders(c, "404", "Not Found")
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "decision rule not found for the given schedule"})
		return
	}

	setDecisionRuleResponseHeaders(c, "200", "OK")
	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToDecisionRuleResponse(decisionRule)})
}
