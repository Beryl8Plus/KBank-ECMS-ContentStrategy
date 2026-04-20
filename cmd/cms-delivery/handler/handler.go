package handler

import (
	"encoding/json"
	"net/http"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity/enums"
	"kbank-ecms/internal/domain/service"
	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the cms-delivery module.
type Handler struct {
	svc service.DeliveryService
}

// NewHandler creates a new cms-delivery Handler.
func NewHandler(svc service.DeliveryService) *Handler {
	return &Handler{svc: svc}
}

// getContent handles GET /content?requestType=personalizedContent&placement=a&placement=b
//
// @Summary Get content by placements
// @Description Returns evaluated content results for one or more placement names.
// @Tags CmsDelivery
// @Accept json
// @Produce json
// @Param requestType query string true "The type of content request" Enums(personalizedContent,staticContent,articleCategory)
// @Param placement query []string true "One or more placement names" collectionFormat(multi) default(wsaHomeBanner, wsaPortBanner, wsaTransaction)
// @Param customerId query string false "Customer identifier value (required when customerIdType=CIS_ID)"
// @Param customerIdType query string false "Customer identifier scheme" Enums(CIS_ID)
// @Success 200 {object} dto.APIResponse{data=[]service.ContentResult}
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /content [get]
func (h *Handler) getContent(c *gin.Context) {
	requestType := enums.RequestType(c.Query("requestType"))
	if !requestType.IsValid() {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "requestType must be one of: personalizedContent, staticContent, articleCategory"})
		return
	}

	placements := c.QueryArray("placement")
	cisID := c.Query("customerId")
	userID, _ := ctxconsts.GetUserID(c.Request.Context())

	// Override cisID when the caller supplies an explicit CIS_ID customer identifier.
	customerIdType := dto.CustomerIdType(c.Query("customerIdType"))
	if customerIdType == dto.CustomerIdTypeCISID || customerIdType == "" {
		if cisID == "" || (customerIdType == "" && cisID != "") {
			c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "customerId is required when customerIdType is CIS_ID or unspecified"})
			return
		}
	}

	var results []service.ContentResult
	var err error
	if cisID == "" {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Error: "customerId is required for personalized content"})
		return
	}

	userIDStr := cisID
	if userID != nil {
		userIDStr = userID.String()
	}
	userAttrs := make(map[string]json.RawMessage) // Service resolves cis_id:{customerId} from Redis when available.
	results, err = h.svc.GetPersonalizedContent(c.Request.Context(), cisID, userIDStr, placements, userAttrs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: err.Error()})
		return
	}
	responseData := make([]service.ContentResult, len(results))
	for i, result := range results {
		responseData[i] = result.ToResponse()
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: responseData})
}

// getCacheStatus handles GET /purge_requests
//
// @Summary Get cache status
// @Description Returns list purge request status.
// @Tags CmsDelivery
// @Accept json
// @Produce json
// @Success 200 {object} dto.APIResponse{data=map[string]string}
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /purge_requests [get]
func (h *Handler) getStatus(c *gin.Context) {
	// TODO: Implement actual status retrieval logic. For now, return a placeholder response.

	c.JSON(http.StatusOK, dto.APIResponse{})
}

// flushCache handles POST /purge_requests
//
// @Summary Flush content cache
// @Description Flushes the cache for specified placements. An empty or missing body flushes all placements.
// @Tags CmsDelivery
// @Accept json
// @Produce json
// @Param body body dto.FlushRequest false "Placements to flush; omit to flush all"
// @Success 200 {object} dto.APIResponse{data=dto.FlushResponse}
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /purge_requests [post]
func (h *Handler) flushCache(c *gin.Context) {
	var req dto.FlushRequest
	// Ignore bind errors — missing/empty body means flush all (req.Placements stays nil).
	_ = c.ShouldBindJSON(&req)

	if err := h.svc.FlushCache(c.Request.Context(), req.Placements, req.IsEvaluate); err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.FlushResponse{Message: "flushed"}})
}
