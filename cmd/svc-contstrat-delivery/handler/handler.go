package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	deliveryservice "kbank-ecms/cmd/svc-contstrat-delivery/service"
	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/pkg/ctxconsts"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Handler handles HTTP requests for the cms-delivery module.
type Handler struct {
	svc deliveryservice.DeliveryService
}

// NewHandler creates a new cms-delivery Handler.
func NewHandler(svc deliveryservice.DeliveryService) *Handler {
	return &Handler{svc: svc}
}

// getContent handles GET /content?requestType=personalizedContent&placement=a&placement=b
//
// @Summary Get content by placements
// @Description Returns evaluated content results for one or more placement names.
// @Tags svc-contstrat-delivery
// @Accept json
// @Produce json
// @Param requestType query string true "The type of content request" Enums(personalizedContent,staticContent,articleCategory)
// @Param placement query []string true "One or more placement names" collectionFormat(multi) default(wsaHomeBanner, wsaPortBanner, wsaTransaction)
// @Param customerId query string false "Customer identifier value (required when customerIdType=CIS_ID)"
// @Param customerIdType query string false "Customer identifier scheme" Enums(CIS_ID, IP_ID, KPLUS_MOBILE_NUMBER, LINE_UUID) default(CIS_ID)
// @Success 200 {object} dto.APIResponse{data=[]dto.ContentResult}
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /content [get]
func (h *Handler) getContent(c *gin.Context) {
	// Validate requestType query parameter
	var req dto.ContentRequestQueryParams
	if err := c.ShouldBindQuery(&req); err != nil {
		var messageTranslator = func(fe validator.FieldError) string {
			switch fe.Tag() {
			case "required":
				return "This field is required"
			case "oneof":
				return fmt.Sprintf("This field must be one of: %s", fe.Param())
			case "required_if":
				return "This field is required when the specified condition is met"
			case "min":
				return fmt.Sprintf("At least %s item,length(s) are required", fe.Param())
			case "max":
				return fmt.Sprintf("At most %s item,length(s) are allowed", fe.Param())
			case "numeric":
				return "This field must be a numeric string"
			case "len":
				return fmt.Sprintf("This field must be exactly %s characters long", fe.Param())
			}
			return fe.Error() // default error
		}
		if ve, ok := errors.AsType[validator.ValidationErrors](err); ok {
			out := make([]dto.ValidationError, len(ve))
			for i, fe := range ve {
				out[i] = dto.ValidationError{
					Field:   fe.Field(),
					Message: messageTranslator(fe),
				}
			}
			c.JSON(http.StatusBadRequest, dto.APIResponse{
				Code:  enums.ErrorCodeBadRequest.String(),
				Error: "Invalid query parameters",
				Data:  out,
			})
		}
		return
	}

	var results []dto.ContentResult
	var err error
	userID, _ := ctxconsts.GetUserID(c.Request.Context())
	userIDStr := req.CustomerID
	if userID != nil {
		userIDStr = userID.String()
	}
	// TODO: call api CLEN req.CustomerID {cisId}, req.Channel {WAMP}
	// get user attributes for this user from Redis (if available) to pass to the service for logic evaluation.
	// This allows the service to evaluate placement logic that depends on user attributes without having to call Redis itself,
	// keeping the service layer simpler and more focused on its core responsibility of content delivery.
	userAttrs := make(map[string]json.RawMessage) // Service resolves cis_id:{customerId} from Redis when available.
	results, err = h.svc.GetPersonalizedContent(
		c.Request.Context(),
		req.CustomerID,
		userIDStr,
		req.Placements,
		userAttrs,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Code:  enums.ErrorCodeInternalError.String(),
			Error: err.Error(),
		})
		return
	}
	responseData := dto.ToContentResultResponses(results)
	if req.PageSize > 0 && len(responseData) > req.PageSize {
		responseData = responseData[:req.PageSize]
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: responseData})
}

// getCacheStatus handles GET /purge_requests
//
// @Summary Get cache status
// @Description Returns in-memory cache keys, heap pressure flag, and heap usage ratio.
// @Tags svc-contstrat-delivery
// @Accept json
// @Produce json
// @Success 200 {object} dto.APIResponse{data=dto.CacheStatusResponse}
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /purge_requests [get]
func (h *Handler) getStatus(c *gin.Context) {
	ctx := c.Request.Context()

	isMemPressure, memUsagePct, err := h.svc.GetCacheStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Code:  enums.ErrorCodeInternalError.String(),
			Error: err.Error(),
		})
		return
	}

	keys, err := h.svc.GetCacheKeys(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Code:  enums.ErrorCodeInternalError.String(),
			Error: err.Error(),
		})
		return
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service: "SVS-CONTSTRAT-DELIVERY",
		Message: fmt.Sprintf("cache status: pressure=%v usage=%.2f%% keys=%d", isMemPressure, memUsagePct*100, len(keys)),
	})

	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.CacheStatusResponse{
		IsMemPressure:  isMemPressure,
		MemoryUsagePct: memUsagePct,
		CacheKeys:      keys,
	}})
}

// getCacheValue handles GET /purge_requests/value?key={key}
//
// @Summary Get cache value
// @Description Returns the cached value for a given key. Used for monitoring and debugging.
// @Tags svc-contstrat-delivery
// @Accept json
// @Produce json
// @Param key query string true "The cache key to retrieve"
// @Success 200 {object} dto.APIResponse{data=json.RawMessage}
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Security XUserIdAuth
// @Router /purge_requests/value [get]
func (h *Handler) getCacheValue(c *gin.Context) {
	var req struct {
		Key string `form:"key" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Code:  enums.ErrorCodeBadRequest.String(),
			Error: "Missing required 'key' query parameter"})
		return
	}
	value, err := h.svc.GetCacheValue(c.Request.Context(), req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Code:  enums.ErrorCodeInternalError.String(),
			Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: value})
}

// flushCache handles POST /purge_requests
//
// @Summary Flush content cache
// @Description Flushes the cache for specified placements. An empty or missing body flushes all placements.
// @Tags svc-contstrat-delivery
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
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Code:  enums.ErrorCodeInternalError.String(),
			Error: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.APIResponse{Data: dto.FlushResponse{Message: "flushed"}})
}
