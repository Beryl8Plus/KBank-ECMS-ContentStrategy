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

// attributeServicer is the interface the handler depends on, enabling test doubles.
type attributeServicer interface {
	CreateAttribute(ctx context.Context, attribute *entity.Attribute) error
	GetAttributeByID(ctx context.Context, id uuid.UUID) (*entity.Attribute, error)
	ListAttributesPaginated(ctx context.Context, page, limit int) ([]*entity.Attribute, int64, error)
	UpdateAttribute(ctx context.Context, attribute *entity.Attribute) error
	DeleteAttribute(ctx context.Context, id uuid.UUID) error
}

// AttributeHandler handles HTTP requests for attribute management.
type AttributeHandler struct {
	service attributeServicer
}

// NewAttributeHandler creates a new AttributeHandler with the injected service.
func NewAttributeHandler(svc *service.AttributeService) *AttributeHandler {
	return &AttributeHandler{service: svc}
}

func setAttributeResponseHeaders(c *gin.Context, statusCode string, statusMsg string) {
	c.Header("Content-Type", "application/json; charset=UTF-8")
	c.Header("Request-ID", c.GetHeader("requestID"))
	c.Header("Request-Time", time.Now().Format("2006-01-02T15:04:05.000"))
	c.Header("Status-Code", statusCode)
	c.Header("Status-Msg", statusMsg)
	c.Header("Access-Control-Expose-Headers", "Request-ID, Request-Time, Status-Code, Status-Msg")
}

// CreateAttribute handles POST /attributes.
//
// @Summary Create an attribute
// @Description Create a new attribute for use in rule conditions.
// @Tags Attributes
// @Accept json
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param body body dto.CreateAttributeRequest true "Create attribute request body"
// @Success 201 {object} dto.APIResponse
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /attributes [post]
func (h *AttributeHandler) CreateAttribute(c *gin.Context) {
	var req dto.CreateAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		setAttributeResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	if !req.DataType.IsValid() {
		setAttributeResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid dataType: must be Text, Date, Number, or Boolean"})
		return
	}

	attribute := &entity.Attribute{
		ClenSchemaRegistryID: req.ClenSchemaRegistryID,
		FieldName:            req.FieldName,
		DisplayName:          req.DisplayName,
		DataType:             req.DataType,
		Value:                req.Value,
		Description:          req.Description,
		SourceSystem:         req.SourceSystem,
		IsActive:             req.IsActive,
	}

	if err := h.service.CreateAttribute(c.Request.Context(), attribute); err != nil {
		setAttributeResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to create attribute"})
		return
	}

	setAttributeResponseHeaders(c, "201", "Created")
	c.JSON(http.StatusCreated, dto.APIResponse{Data: dto.ToAttributeResponse(attribute)})
}

// ListAttributes handles GET /attributes.
//
// @Summary List all attributes
// @Description Returns attributes with cursor-style offset pagination. Defaults: page=1, limit=20 (max 100).
// @Tags Attributes
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Page size (default: 20, max: 100)"
// @Success 200 {object} dto.APIResponse{data=[]dto.AttributeResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /attributes [get]
func (h *AttributeHandler) ListAttributes(c *gin.Context) {
	page, limit, ok := parsePaginationParams(c)
	if !ok {
		return
	}

	attributes, total, err := h.service.ListAttributesPaginated(c.Request.Context(), page, limit)
	if err != nil {
		setAttributeResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve attributes"})
		return
	}

	responses := make([]dto.AttributeResponse, 0, len(attributes))
	for _, a := range attributes {
		responses = append(responses, dto.ToAttributeResponse(a))
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	setAttributeResponseHeaders(c, "200", "OK")
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

// GetAttribute handles GET /attributes/:id.
//
// @Summary Get an attribute by ID
// @Description Returns a single attribute by its UUID.
// @Tags Attributes
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Attribute ID (UUID)"
// @Success 200 {object} dto.APIResponse{data=[]dto.AttributeResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /attributes/{id} [get]
func (h *AttributeHandler) GetAttribute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		setAttributeResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid attribute ID"})
		return
	}

	attribute, err := h.service.GetAttributeByID(c.Request.Context(), id)
	if err != nil {
		setAttributeResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve attribute"})
		return
	}
	if attribute == nil {
		setAttributeResponseHeaders(c, "404", "Not Found")
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "attribute not found"})
		return
	}

	setAttributeResponseHeaders(c, "200", "OK")
	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToAttributeResponse(attribute)})
}

// UpdateAttribute handles PUT /attributes/:id.
//
// @Summary Update an attribute
// @Description Updates an existing attribute. ClenSchemaRegistryID is immutable after creation.
// @Tags Attributes
// @Accept json
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Attribute ID (UUID)"
// @Param body body dto.UpdateAttributeRequest true "Update attribute request body"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /attributes/{id} [put]
func (h *AttributeHandler) UpdateAttribute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		setAttributeResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid attribute ID"})
		return
	}

	existing, err := h.service.GetAttributeByID(c.Request.Context(), id)
	if err != nil {
		setAttributeResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve attribute"})
		return
	}
	if existing == nil {
		setAttributeResponseHeaders(c, "404", "Not Found")
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "attribute not found"})
		return
	}

	var req dto.UpdateAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		setAttributeResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	if !req.DataType.IsValid() {
		setAttributeResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid dataType: must be Text, Date, Number, or Boolean"})
		return
	}

	// Apply update fields; preserve immutable ClenSchemaRegistryID.
	existing.FieldName = req.FieldName
	existing.DisplayName = req.DisplayName
	existing.DataType = req.DataType
	existing.Value = req.Value
	existing.Description = req.Description
	existing.SourceSystem = req.SourceSystem
	existing.IsActive = req.IsActive

	if err := h.service.UpdateAttribute(c.Request.Context(), existing); err != nil {
		setAttributeResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to update attribute"})
		return
	}

	setAttributeResponseHeaders(c, "200", "OK")
	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToAttributeResponse(existing)})
}

// DeleteAttribute handles DELETE /attributes/:id.
//
// @Summary Delete an attribute
// @Description Soft-deletes an attribute by its UUID.
// @Tags Attributes
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Attribute ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /attributes/{id} [delete]
func (h *AttributeHandler) DeleteAttribute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		setAttributeResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid attribute ID"})
		return
	}

	if err := h.service.DeleteAttribute(c.Request.Context(), id); err != nil {
		setAttributeResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to delete attribute"})
		return
	}

	c.Status(http.StatusNoContent)
}
