package handler

import (
	"github.com/gin-gonic/gin"

	"kbank-ecms/cmd/server/service"
	"kbank-ecms/pkg/util"
)

// RegisterRoutes attaches all routes to the provided gin.Engine.
// The caller is responsible for constructing the DeliveryService (and calling
// Start/Stop for the background ticker if applicable).
func RegisterRoutes(
	r *gin.Engine,
	svc service.DeliveryService,
) {
	// Initialize the handler with the service
	handler := NewHandler(svc)

	// Initialize API routes with the configured prefix
	prefix := util.GetEnvWithDefault("PREFIX_CONTENT_STRATEGY_API_V1", "/api/content-strategy/v1")
	content := r.Group(prefix)
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
