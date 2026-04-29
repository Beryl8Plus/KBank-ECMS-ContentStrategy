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
			decisionRules.GET("/schedule/:scheduleId",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				decisionRuleHandler.GetDecisionRuleBySchedule)

			// View routes — require VIEW_ALL scope
			decisionRules.GET("",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				wizardHandler.ListDecisionRules)
			decisionRules.GET("/:id/conditions",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				wizardHandler.GetConditions)
			decisionRules.GET("/:id/rule-sets",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				wizardHandler.GetRuleSets)
			decisionRules.GET("/:id/schedules",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				wizardHandler.GetSchedules)

			// Create — require CREATE scope
			decisionRules.POST("",
				middleware.RequireScope("decision_rule", "CREATE"),
				wizardHandler.CreateDecisionRule)
			decisionRules.POST("/:id/clone",
				middleware.RequireScope("decision_rule", "CREATE"),
				wizardHandler.CloneDecisionRule)

			// Edit — require EDIT or EDIT_ALL scope
			decisionRules.PUT("/:id",
				middleware.RequireScope("decision_rule", "EDIT", "EDIT_ALL"),
				wizardHandler.UpdateDecisionRule)
			decisionRules.PUT("/:id/rule-sets",
				middleware.RequireScope("decision_rule", "EDIT", "EDIT_ALL"),
				wizardHandler.SaveRuleSets)
			decisionRules.PUT("/:id/schedules",
				middleware.RequireScope("decision_rule", "EDIT", "EDIT_ALL"),
				wizardHandler.SaveSchedules)
			decisionRules.PUT("/:id/activate",
				middleware.RequireScope("decision_rule", "EDIT", "EDIT_ALL"),
				wizardHandler.ActivateDecisionRule)
			decisionRules.PUT("/:id/deactivate",
				middleware.RequireScope("decision_rule", "EDIT", "EDIT_ALL"),
				wizardHandler.DeactivateDecisionRule)

			// Delete — require DELETE or DELETE_ALL scope
			decisionRules.DELETE("/:id",
				middleware.RequireScope("decision_rule", "DELETE", "DELETE_ALL"),
				wizardHandler.DeleteDecisionRule)
		}

		// Schedule occurrences are read-only views over decision-rule schedules
		// — reuse the decision_rule:VIEW_ALL scope for read access.
		scheduleOccurrences := protected.Group("/schedule-occurrences")
		{
			scheduleOccurrences.GET("/active",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				occurrenceHandler.ListActiveOccurrences)
		}

		// Attributes / channels / placements are master-data resources used by
		// decision rules. They share the same permission semantics as decision_rule:
		//   GET    → VIEW_ALL
		//   POST   → CREATE
		//   PUT    → EDIT or EDIT_ALL
		//   DELETE → DELETE or DELETE_ALL
		attributes := protected.Group("/attributes")
		{
			attributes.POST("",
				middleware.RequireScope("decision_rule", "CREATE"),
				attributeHandler.CreateAttribute)
			attributes.GET("",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				attributeHandler.ListAttributes)
			attributes.GET("/:id",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				attributeHandler.GetAttribute)
			attributes.PUT("/:id",
				middleware.RequireScope("decision_rule", "EDIT", "EDIT_ALL"),
				attributeHandler.UpdateAttribute)
			attributes.DELETE("/:id",
				middleware.RequireScope("decision_rule", "DELETE", "DELETE_ALL"),
				attributeHandler.DeleteAttribute)
		}

		channels := protected.Group("/channels")
		{
			channels.POST("",
				middleware.RequireScope("decision_rule", "CREATE"),
				channelHandler.CreateChannel)
			channels.GET("",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				channelHandler.ListChannels)
			channels.GET("/:id",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				channelHandler.GetChannel)
			channels.PUT("/:id",
				middleware.RequireScope("decision_rule", "EDIT", "EDIT_ALL"),
				channelHandler.UpdateChannel)
			channels.DELETE("/:id",
				middleware.RequireScope("decision_rule", "DELETE", "DELETE_ALL"),
				channelHandler.DeleteChannel)
		}

		placements := protected.Group("/placements")
		{
			placements.POST("",
				middleware.RequireScope("decision_rule", "CREATE"),
				placementHandler.CreatePlacement)
			placements.GET("",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				placementHandler.ListPlacements)
			placements.GET("/:id",
				middleware.RequireScope("decision_rule", "VIEW_ALL"),
				placementHandler.GetPlacement)
			placements.PUT("/:id",
				middleware.RequireScope("decision_rule", "EDIT", "EDIT_ALL"),
				placementHandler.UpdatePlacement)
			placements.DELETE("/:id",
				middleware.RequireScope("decision_rule", "DELETE", "DELETE_ALL"),
				placementHandler.DeletePlacement)
		}
	}
}
