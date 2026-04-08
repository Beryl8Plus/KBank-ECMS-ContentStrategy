package http

import (
	"kbank-ecms/internal/delivery/http/handler"
	"kbank-ecms/internal/delivery/http/middleware"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/repository"
	"kbank-ecms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// NewRouter creates the Gin engine and wires all layers in order:
// service → handler → middleware → router
func NewRouter(db *gorm.DB, rateLimit entity.RateLimit) *gin.Engine {
	r := gin.New()

	// Middleware layer — applied globally before any handler
	middleware.Apply(r, db, rateLimit)

	// System routes (observability, docs) — no auth/permission guards
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Service → Handler wiring
	ruleManagementSvc := service.NewRuleManagementService()
	ruleManagementHandler := handler.NewRuleManagementHandler(ruleManagementSvc)

	scheduleRepo := repository.NewSchedulePostgresRepository(db)
	scheduleSvc := service.NewScheduleService(scheduleRepo)
	scheduleHandler := handler.NewScheduleHandler(scheduleSvc)

	// Route registration
	r.POST("/rule-management", ruleManagementHandler.IngressRuleManagement)

	schedules := r.Group("/schedules")
	{
		schedules.POST("", scheduleHandler.CreateSchedule)
		schedules.GET("", scheduleHandler.ListSchedules)
		schedules.GET("/:id", scheduleHandler.GetSchedule)
		schedules.PUT("/:id", scheduleHandler.UpdateSchedule)
		schedules.DELETE("/:id", scheduleHandler.DeleteSchedule)
	}

	return r
}
