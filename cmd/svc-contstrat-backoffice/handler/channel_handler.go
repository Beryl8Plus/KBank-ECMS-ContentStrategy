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

// channelServicer is the interface the handler depends on, enabling test doubles.
type channelServicer interface {
	CreateChannel(ctx context.Context, channel *entity.Channel) error
	GetChannelByID(ctx context.Context, id uuid.UUID) (*entity.Channel, error)
	ListChannelsPaginated(ctx context.Context, page, limit int) ([]*entity.Channel, int64, error)
	UpdateChannel(ctx context.Context, channel *entity.Channel) error
	DeleteChannel(ctx context.Context, id uuid.UUID) error
}

// ChannelHandler handles HTTP requests for channel management.
type ChannelHandler struct {
	service channelServicer
}

// NewChannelHandler creates a new ChannelHandler with the injected service.
func NewChannelHandler(svc *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{service: svc}
}

func setChannelResponseHeaders(c *gin.Context, statusCode string, statusMsg string) {
	c.Header("Content-Type", "application/json; charset=UTF-8")
	c.Header("Request-ID", c.GetHeader("requestID"))
	c.Header("Request-Time", time.Now().Format("2006-01-02T15:04:05.000"))
	c.Header("Status-Code", statusCode)
	c.Header("Status-Msg", statusMsg)
	c.Header("Access-Control-Expose-Headers", "Request-ID, Request-Time, Status-Code, Status-Msg")
}

// CreateChannel handles POST /channels.
//
// @Summary Create a channel
// @Description Create a new delivery channel.
// @Tags Channels
// @Accept json
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param body body dto.CreateChannelRequest true "Create channel request body"
// @Success 201 {object} dto.APIResponse{data=dto.ChannelResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /channels [post]
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req dto.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		setChannelResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	channel := &entity.Channel{
		ChannelName: req.ChannelName,
	}

	if err := h.service.CreateChannel(c.Request.Context(), channel); err != nil {
		setChannelResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to create channel"})
		return
	}

	setChannelResponseHeaders(c, "201", "Created")
	c.JSON(http.StatusCreated, dto.APIResponse{Data: dto.ToChannelResponse(channel)})
}

// ListChannels handles GET /channels.
//
// @Summary List all channels
// @Description Returns channels with offset pagination. Defaults: page=1, limit=20 (max 100).
// @Tags Channels
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Page size (default: 20, max: 100)"
// @Success 200 {object} dto.APIResponse{data=[]dto.ChannelResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /channels [get]
func (h *ChannelHandler) ListChannels(c *gin.Context) {
	page, limit, ok := parsePaginationParams(c)
	if !ok {
		return
	}

	channels, total, err := h.service.ListChannelsPaginated(c.Request.Context(), page, limit)
	if err != nil {
		setChannelResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve channels"})
		return
	}

	responses := make([]dto.ChannelResponse, 0, len(channels))
	for _, ch := range channels {
		responses = append(responses, dto.ToChannelResponse(ch))
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	setChannelResponseHeaders(c, "200", "OK")
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

// GetChannel handles GET /channels/:id.
//
// @Summary Get a channel by ID
// @Description Returns a single channel by its UUID.
// @Tags Channels
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Channel ID (UUID)"
// @Success 200 {object} dto.APIResponse{data=dto.ChannelResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /channels/{id} [get]
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		setChannelResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid channel ID"})
		return
	}

	channel, err := h.service.GetChannelByID(c.Request.Context(), id)
	if err != nil {
		setChannelResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve channel"})
		return
	}
	if channel == nil {
		setChannelResponseHeaders(c, "404", "Not Found")
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "channel not found"})
		return
	}

	setChannelResponseHeaders(c, "200", "OK")
	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToChannelResponse(channel)})
}

// UpdateChannel handles PUT /channels/:id.
//
// @Summary Update a channel
// @Description Updates an existing channel by its UUID.
// @Tags Channels
// @Accept json
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Channel ID (UUID)"
// @Param body body dto.UpdateChannelRequest true "Update channel request body"
// @Success 200 {object} dto.APIResponse{data=dto.ChannelResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /channels/{id} [put]
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		setChannelResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid channel ID"})
		return
	}

	existing, err := h.service.GetChannelByID(c.Request.Context(), id)
	if err != nil {
		setChannelResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to retrieve channel"})
		return
	}
	if existing == nil {
		setChannelResponseHeaders(c, "404", "Not Found")
		c.JSON(http.StatusNotFound, dto.APIResponse{Error: "channel not found"})
		return
	}

	var req dto.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		setChannelResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: err.Error()})
		return
	}

	existing.ChannelName = req.ChannelName

	if err := h.service.UpdateChannel(c.Request.Context(), existing); err != nil {
		setChannelResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to update channel"})
		return
	}

	setChannelResponseHeaders(c, "200", "OK")
	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.ToChannelResponse(existing)})
}

// DeleteChannel handles DELETE /channels/:id.
//
// @Summary Delete a channel
// @Description Soft-deletes a channel by its UUID.
// @Tags Channels
// @Produce json
// @Param X-User-Id header string true "User ID (UUID)"
// @Param id path string true "Channel ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /channels/{id} [delete]
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		setChannelResponseHeaders(c, "400", "Bad Request")
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "invalid channel ID"})
		return
	}

	if err := h.service.DeleteChannel(c.Request.Context(), id); err != nil {
		setChannelResponseHeaders(c, "500", "Internal Server Error")
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: "failed to delete channel"})
		return
	}

	c.Status(http.StatusNoContent)
}
