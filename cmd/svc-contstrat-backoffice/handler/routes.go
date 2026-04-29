package handler

import (
	"kbank-ecms/internal/delivery/http/middleware"
	"kbank-ecms/pkg/auth"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.Engine,
	jwtService *auth.JWTService,
	tokenHandler *TokenHandler,
	ruleManagementHandler *RuleManagementHandler,
	scheduleHandler *ScheduleHandler,
	decisionRuleHandler *DecisionRuleHandler,
	wizardHandler *DecisionRuleWizardHandler,
	occurrenceHandler *ScheduleOccurrenceHandler,
	attributeHandler *AttributeHandler,
	channelHandler *ChannelHandler,
	placementHandler *PlacementHandler,
) {
	// Public routes (no authentication required)
	public := r.Group("/")
	{
		public.POST("/rule-management", ruleManagementHandler.IngressRuleManagement)
		// OAuth2 token endpoint for client credentials flow
		public.POST("/token", tokenHandler.HandleToken)
	}

	// Protected routes (JWT authentication required)
	// Following Gin Framework authentication standards
	protected := r.Group("/")
	protected.Use(middleware.JWTMiddleware(jwtService))
	{
		schedules := protected.Group("/schedules")
		{
			schedules.POST("", scheduleHandler.CreateSchedule)
			schedules.GET("", scheduleHandler.ListSchedules)
			schedules.GET("/:id", scheduleHandler.GetSchedule)
			schedules.PUT("/:id", scheduleHandler.UpdateSchedule)
			schedules.DELETE("/:id", scheduleHandler.DeleteSchedule)
			schedules.GET("/:id/occurrences", occurrenceHandler.ListOccurrencesBySchedule)
		}

		decisionRules := protected.Group("/decision-rules")
		{
			// Existing route (static prefix takes precedence over :id param)
			decisionRules.GET("/schedule/:scheduleId", decisionRuleHandler.GetDecisionRuleBySchedule)

			// Wizard routes
			decisionRules.POST("", wizardHandler.CreateDecisionRule)
			decisionRules.GET("", wizardHandler.ListDecisionRules)
			decisionRules.PUT("/:id", wizardHandler.UpdateDecisionRule)
			decisionRules.GET("/:id/conditions", wizardHandler.GetConditions)
			decisionRules.GET("/:id/rule-sets", wizardHandler.GetRuleSets)
			decisionRules.PUT("/:id/rule-sets", wizardHandler.SaveRuleSets)
			decisionRules.GET("/:id/schedules", wizardHandler.GetSchedules)
			decisionRules.PUT("/:id/schedules", wizardHandler.SaveSchedules)
			decisionRules.PUT("/:id/activate", wizardHandler.ActivateDecisionRule)
			decisionRules.POST("/:id/clone", wizardHandler.CloneDecisionRule)
			decisionRules.PUT("/:id/deactivate", wizardHandler.DeactivateDecisionRule)
			decisionRules.DELETE("/:id", wizardHandler.DeleteDecisionRule)
		}

		scheduleOccurrences := protected.Group("/schedule-occurrences")
		{
			scheduleOccurrences.GET("/active", occurrenceHandler.ListActiveOccurrences)
		}

		attributes := protected.Group("/attributes")
		{
			attributes.POST("", attributeHandler.CreateAttribute)
			attributes.GET("", attributeHandler.ListAttributes)
			attributes.GET("/:id", attributeHandler.GetAttribute)
			attributes.PUT("/:id", attributeHandler.UpdateAttribute)
			attributes.DELETE("/:id", attributeHandler.DeleteAttribute)
		}

		channels := protected.Group("/channels")
		{
			channels.POST("", channelHandler.CreateChannel)
			channels.GET("", channelHandler.ListChannels)
			channels.GET("/:id", channelHandler.GetChannel)
			channels.PUT("/:id", channelHandler.UpdateChannel)
			channels.DELETE("/:id", channelHandler.DeleteChannel)
		}

		placements := protected.Group("/placements")
		{
			placements.POST("", placementHandler.CreatePlacement)
			placements.GET("", placementHandler.ListPlacements)
			placements.GET("/:id", placementHandler.GetPlacement)
			placements.PUT("/:id", placementHandler.UpdatePlacement)
			placements.DELETE("/:id", placementHandler.DeletePlacement)
		}
	}
}
