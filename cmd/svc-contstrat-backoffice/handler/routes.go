package handler

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.Engine,
	ruleManagementHandler *RuleManagementHandler,
	scheduleHandler *ScheduleHandler,
	decisionRuleHandler *DecisionRuleHandler,
	wizardHandler *DecisionRuleWizardHandler,
	occurrenceHandler *ScheduleOccurrenceHandler,
	attributeHandler *AttributeHandler,
	channelHandler *ChannelHandler,
	placementHandler *PlacementHandler,
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
		// Existing route (static prefix takes precedence over :id param)
		decisionRules.GET("/schedule/:scheduleId", decisionRuleHandler.GetDecisionRuleBySchedule)

		// Wizard routes
		decisionRules.POST("", wizardHandler.CreateDecisionRule)
		decisionRules.GET("", wizardHandler.ListDecisionRules)
		decisionRules.GET("/:id/conditions", wizardHandler.GetConditions)
		decisionRules.GET("/:id/rule-sets", wizardHandler.GetRuleSets)
		decisionRules.PUT("/:id/rule-sets", wizardHandler.SaveRuleSets)
		decisionRules.GET("/:id/schedules", wizardHandler.GetSchedules)
		decisionRules.PUT("/:id/schedules", wizardHandler.SaveSchedules)
		decisionRules.PUT("/:id/activate", wizardHandler.ActivateDecisionRule)
	}

	scheduleOccurrences := r.Group("/schedule-occurrences")
	{
		scheduleOccurrences.GET("/active", occurrenceHandler.ListActiveOccurrences)
	}

	attributes := r.Group("/attributes")
	{
		attributes.POST("", attributeHandler.CreateAttribute)
		attributes.GET("", attributeHandler.ListAttributes)
		attributes.GET("/:id", attributeHandler.GetAttribute)
		attributes.PUT("/:id", attributeHandler.UpdateAttribute)
		attributes.DELETE("/:id", attributeHandler.DeleteAttribute)
	}

	channels := r.Group("/channels")
	{
		channels.POST("", channelHandler.CreateChannel)
		channels.GET("", channelHandler.ListChannels)
		channels.GET("/:id", channelHandler.GetChannel)
		channels.PUT("/:id", channelHandler.UpdateChannel)
		channels.DELETE("/:id", channelHandler.DeleteChannel)
	}

	placements := r.Group("/placements")
	{
		placements.POST("", placementHandler.CreatePlacement)
		placements.GET("", placementHandler.ListPlacements)
		placements.GET("/:id", placementHandler.GetPlacement)
		placements.PUT("/:id", placementHandler.UpdatePlacement)
		placements.DELETE("/:id", placementHandler.DeletePlacement)
	}
}
