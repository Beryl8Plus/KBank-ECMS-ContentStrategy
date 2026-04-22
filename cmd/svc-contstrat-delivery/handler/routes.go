package handler

import (
	domainservice "kbank-ecms/internal/domain/service"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes attaches all routes to the provided gin.Engine.
// The caller is responsible for constructing the DeliveryService (and calling
// Start/Stop for the background ticker if applicable).
func RegisterRoutes(
	r *gin.Engine,
	svc domainservice.DeliveryService,
) {
	handler := NewHandler(svc)

	r.GET("/content", handler.getContent)

	purges := r.Group("/purge_requests")
	{
		purges.GET("", handler.getStatus)
		purges.POST("", handler.flushCache)
	}
}
