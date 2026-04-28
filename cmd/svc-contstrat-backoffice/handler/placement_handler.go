package handler

import (
	"context"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/service"
)

// placementServicer is the interface the handler depends on, enabling test doubles.
type placementServicer interface {
	CreatePlacement(ctx context.Context, placement *entity.Placement) error
	GetPlacementByID(ctx context.Context, id uuid.UUID) (*entity.Placement, error)
	ListPlacementsPaginated(ctx context.Context, page, limit int) ([]*entity.Placement, int64, error)
	UpdatePlacement(ctx context.Context, placement *entity.Placement) error
	DeletePlacement(ctx context.Context, id uuid.UUID) error
}

// PlacementHandler handles HTTP requests for placement management.
type PlacementHandler struct {
	service placementServicer
}

// NewPlacementHandler creates a new PlacementHandler with the injected service.
func NewPlacementHandler(svc *service.PlacementService) *PlacementHandler {
	return &PlacementHandler{service: svc}
}

// CreatePlacement handles POST /placements.
//
//	@Summary		Create a placement
//	@Description	Create a new content placement slot within a channel.
//	@Tags			Placements
//	@Accept			json
//	@Produce		json
//	@Param			X-User-Id	header		string						true	"User ID (UUID)"
//	@Param			body		body		dto.CreatePlacementRequest	true	"Create placement request body"
//	@Success		201			{object}	dto.APIResponse{data=dto.PlacementResponse}
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/placements [post]
func (h *PlacementHandler) CreatePlacement(c *gin.Context) {
	var req dto.CreatePlacementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	placement := &entity.Placement{
		PlacementName: req.PlacementName,
		ChannelID:     req.ChannelID,
	}

	if err := h.service.CreatePlacement(c.Request.Context(), placement); err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to create placement"})
		return
	}

	// Fetch with Channel preloaded so the response is complete.
	created, err := h.service.GetPlacementByID(c.Request.Context(), placement.ID)
	if err != nil || created == nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve created placement"})
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{Data: dto.ToPlacementResponse(created)})
}

// ListPlacements handles GET /placements.
//
//	@Summary		List all placements
//	@Description	Returns placements with offset pagination. Defaults: page=1, limit=20 (max 100). Channel is embedded in each item.
//	@Tags			Placements
//	@Produce		json
//	@Param			X-User-Id	header		string	true	"User ID (UUID)"
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			limit		query		int		false	"Page size (default: 20, max: 100)"
//	@Success		200			{object}	dto.APIResponse{data=[]dto.PlacementResponse}
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/placements [get]
func (h *PlacementHandler) ListPlacements(c *gin.Context) {
	page, limit, ok := parsePaginationParams(c)
	if !ok {
		return
	}

	placements, total, err := h.service.ListPlacementsPaginated(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve placements"})
		return
	}

	responses := make([]dto.PlacementResponse, 0, len(placements))
	for _, p := range placements {
		responses = append(responses, dto.ToPlacementResponse(p))
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

// GetPlacement handles GET /placements/:id.
//
//	@Summary		Get a placement by ID
//	@Description	Returns a single placement by its UUID with Channel embedded.
//	@Tags			Placements
//	@Produce		json
//	@Param			X-User-Id	header		string	true	"User ID (UUID)"
//	@Param			id			path		string	true	"Placement ID (UUID)"
//	@Success		200			{object}	dto.APIResponse{data=dto.PlacementResponse}
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		404			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/placements/{id} [get]
func (h *PlacementHandler) GetPlacement(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid placement ID"})
		return
	}

	placement, err := h.service.GetPlacementByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve placement"})
		return
	}
	if placement == nil {
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "placement not found"})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToPlacementResponse(placement)})
}

// UpdatePlacement handles PUT /placements/:id.
//
//	@Summary		Update a placement
//	@Description	Updates an existing placement. ChannelID can be reassigned.
//	@Tags			Placements
//	@Accept			json
//	@Produce		json
//	@Param			X-User-Id	header		string						true	"User ID (UUID)"
//	@Param			id			path		string						true	"Placement ID (UUID)"
//	@Param			body		body		dto.UpdatePlacementRequest	true	"Update placement request body"
//	@Success		200			{object}	dto.APIResponse{data=dto.PlacementResponse}
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		404			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/placements/{id} [put]
func (h *PlacementHandler) UpdatePlacement(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid placement ID"})
		return
	}

	existing, err := h.service.GetPlacementByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve placement"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "placement not found"})
		return
	}

	var req dto.UpdatePlacementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	existing.PlacementName = req.PlacementName
	existing.ChannelID = req.ChannelID
	existing.Channel = nil // cleared so GORM does not attempt to upsert the association

	if err := h.service.UpdatePlacement(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to update placement"})
		return
	}

	// Re-fetch to get the updated Channel embedded in the response.
	updated, err := h.service.GetPlacementByID(c.Request.Context(), id)
	if err != nil || updated == nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve updated placement"})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToPlacementResponse(updated)})
}

// DeletePlacement handles DELETE /placements/:id.
//
//	@Summary		Delete a placement
//	@Description	Soft-deletes a placement by its UUID.
//	@Tags			Placements
//	@Produce		json
//	@Param			X-User-Id	header	string	true	"User ID (UUID)"
//	@Param			id			path	string	true	"Placement ID (UUID)"
//	@Success		204			"No Content"
//	@Failure		400			{object}	dto.APIResponse
//	@Failure		500			{object}	dto.APIResponse
//	@Security		XUserIdAuth
//	@Router			/placements/{id} [delete]
func (h *PlacementHandler) DeletePlacement(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid placement ID"})
		return
	}

	if err := h.service.DeletePlacement(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to delete placement"})
		return
	}

	c.Status(http.StatusNoContent)
}
