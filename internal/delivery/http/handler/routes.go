package handler

import (
	"kbank-ecms/internal/repository"
	"kbank-ecms/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	ruleManagementSvc := service.NewRuleManagementService()
	ruleManagementHandler := NewRuleManagementHandler(ruleManagementSvc)

	scheduleRepo := repository.NewSchedulePostgresRepository(db)
	scheduleSvc := service.NewScheduleService(scheduleRepo)
	scheduleHandler := NewScheduleHandler(scheduleSvc)

	r.POST("/rule-management", ruleManagementHandler.IngressRuleManagement)

	schedules := r.Group("/schedules")
	{
		schedules.POST("", scheduleHandler.CreateSchedule)
		schedules.GET("", scheduleHandler.ListSchedules)
		schedules.GET("/:id", scheduleHandler.GetSchedule)
		schedules.PUT("/:id", scheduleHandler.UpdateSchedule)
		schedules.DELETE("/:id", scheduleHandler.DeleteSchedule)
	}
}
