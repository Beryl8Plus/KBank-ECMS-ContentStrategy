package handler

import (
	deliveryservice "kbank-ecms/cmd/svc-contstrat-delivery/service"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes attaches all routes to the provided gin.Engine.
// The caller is responsible for constructing the DeliveryService (and calling
// Start/Stop for the background ticker if applicable).
func RegisterRoutes(
	r *gin.Engine,
	svc deliveryservice.DeliveryService,
) {
	handler := NewHandler(svc)

	content := r.Group("/api/content-strategy/v1/")
	{
		// Get content by placements
		content.GET("/personalized-content", handler.getContent)

		// Purge endpoints for cache management
		purges := content.Group("/purge_requests")
		{
			purges.GET("", handler.getStatus)
			purges.GET("/value", handler.getCacheValue)
			purges.POST("", handler.flushCache)
		}
	}
}
