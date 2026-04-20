package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/gorm"

	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/delivery/http/handler"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/repository"
	"kbank-ecms/internal/service"
)

// ProvideRouter initializes the Gin engine with middleware and registers all routes.
func ProvideRouter(
	db *gorm.DB,
	rateLimit entity.RateLimit,
	ruleManagementHandler *handler.RuleManagementHandler,
	scheduleHandler *handler.ScheduleHandler,
	decisionRuleHandler *handler.DecisionRuleHandler,
) *gin.Engine {
	r := deliveryhttp.InitNewRouter(db, rateLimit)
	handler.RegisterRoutes(r, ruleManagementHandler, scheduleHandler, decisionRuleHandler)
	return r
}

// ProviderSet connects all dependencies for the server.
var ProviderSet = wire.NewSet(
	// Repositories
	repository.NewSchedulePostgresRepository,
	wire.Bind(new(domainrepo.ScheduleRepository), new(*repository.SchedulePostgresRepository)),

	repository.NewDecisionRulePostgresRepository,
	wire.Bind(new(domainrepo.DecisionRuleRepository), new(*repository.DecisionRulePostgresRepository)),

	// Services
	service.NewScheduleService,
	service.NewDecisionRuleService,
	service.NewRuleManagementService,

	// Handlers
	handler.NewScheduleHandler,
	handler.NewDecisionRuleHandler,
	handler.NewRuleManagementHandler,

	// Router
	ProvideRouter,
)
