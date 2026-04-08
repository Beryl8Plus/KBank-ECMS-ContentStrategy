package handler

import (
	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RuleManagementHandler handles HTTP requests for rule management.
type RuleManagementHandler struct {
	service *service.RuleManagementService
}

// NewRuleManagementHandler creates a new handler with injected service.
func NewRuleManagementHandler(svc *service.RuleManagementService) *RuleManagementHandler {
	return &RuleManagementHandler{service: svc}
}

// IngressRuleManagement is the endpoint handler for the Rule Management API.
// @Summary Inbound Rule Management
// @Description Inbound rule management for ECMS.
// @Tags RuleManagement
// @Accept json
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param body body dto.RuleManagementRequest true "Rule management request body"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /rule-management [post]
func (h *RuleManagementHandler) IngressRuleManagement(c *gin.Context) {
	var req dto.RuleManagementRequest
	_ = c.ShouldBindJSON(&req)

	c.Header("Content-Type", "application/json; charset=UTF-8")
	c.Header("Request-ID", c.GetHeader("requestID"))
	c.Header("Request-Time", time.Now().Format("2006-01-02T15:04:05.000"))
	c.Header("Status-Code", "200")
	c.Header("Status-Msg", "OK")
	c.Header("Access-Control-Expose-Headers", "Request-ID, Request-Time, Status-Code, Status-Msg")

	result := h.service.ProcessRuleManagement()

	c.JSON(http.StatusOK, dto.APIResponse{Code: "200", Data: result})
}
