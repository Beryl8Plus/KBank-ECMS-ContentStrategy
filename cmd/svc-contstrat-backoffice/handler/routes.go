package handler

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.Engine,
	ruleManagementHandler *RuleManagementHandler,
	scheduleHandler *ScheduleHandler,
	decisionRuleHandler *DecisionRuleHandler,
	occurrenceHandler *ScheduleOccurrenceHandler,
) {
	r.POST("/rule-management", ruleManagementHandler.IngressRuleManagement)

	schedules := r.Group("/schedules")
	{
		schedules.POST("", scheduleHandler.CreateSchedule)
		schedules.GET("", scheduleHandler.ListSchedules)
		schedules.GET("/:id", scheduleHandler.GetSchedule)
		schedules.PUT("/:id", scheduleHandler.UpdateSchedule)
		schedules.DELETE("/:id", scheduleHandler.DeleteSchedule)
		schedules.GET("/:id/occurrences", occurrenceHandler.ListOccurrencesBySchedule)
	}

	decisionRules := r.Group("/decision-rules")
	{
		decisionRules.GET("/schedule/:scheduleId", decisionRuleHandler.GetDecisionRuleBySchedule)
	}

	scheduleOccurrences := r.Group("/schedule-occurrences")
	{
		scheduleOccurrences.GET("/active", occurrenceHandler.ListActiveOccurrences)
	}
}
