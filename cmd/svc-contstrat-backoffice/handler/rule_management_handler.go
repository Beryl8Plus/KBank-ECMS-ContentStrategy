package handler

import (
	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/service"
	"net/http"

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
//
//	@Summary		Inbound Rule Management
//	@Description	Inbound rule management for ECMS.
//	@Tags			RuleManagement
//	@Accept			json
//	@Produce		json
//	@Param			X-User-Id	header		string						true	"User ID (UUID)"
//	@Param			body		body		dto.RuleManagementRequest	true	"Rule management request body"
//	@Success		200			{object}	dto.APIResponse
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/rule-management [post]
func (h *RuleManagementHandler) IngressRuleManagement(c *gin.Context) {
	var req dto.RuleManagementRequest
	_ = c.ShouldBindJSON(&req)

	result := h.service.ProcessRuleManagement()

	c.JSON(http.StatusOK, dto.APIResponse{Data: result})
}
