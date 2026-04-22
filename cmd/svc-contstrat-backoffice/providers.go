package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/gorm"

	"kbank-ecms/cmd/svc-contstrat-backoffice/handler"
	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/repository"
	"kbank-ecms/internal/service"
)

// Application holds the two top-level components returned by Wire.
// Wire injectors may only return a single non-error value, so both the
// HTTP engine and the background worker are bundled here.
type Application struct {
	Router           *gin.Engine
	OccurrenceWorker *service.OccurrenceWorker
}

// NewApplication assembles the Application from its wired components.
func NewApplication(r *gin.Engine, w *service.OccurrenceWorker) *Application {
	return &Application{Router: r, OccurrenceWorker: w}
}

// ProvideMatConfig returns the default MaterializationConfig (7d window, 30d retention).
func ProvideMatConfig() service.MaterializationConfig {
	return service.MaterializationConfig{}
}

// ProvideWorkerConfig returns the default OccurrenceWorkerConfig (1h materialize, 24h cleanup).
func ProvideWorkerConfig() service.OccurrenceWorkerConfig {
	return service.OccurrenceWorkerConfig{}
}

// ProvideRouter initializes the Gin engine with middleware and registers all routes.
func ProvideRouter(
	db *gorm.DB,
	rateLimit entity.RateLimit,
	ruleManagementHandler *handler.RuleManagementHandler,
	scheduleHandler *handler.ScheduleHandler,
	decisionRuleHandler *handler.DecisionRuleHandler,
	occurrenceHandler *handler.ScheduleOccurrenceHandler,
	attributeHandler *handler.AttributeHandler,
) *gin.Engine {
	r := deliveryhttp.InitNewRouter(db, rateLimit)
	handler.RegisterRoutes(r, ruleManagementHandler, scheduleHandler, decisionRuleHandler, occurrenceHandler, attributeHandler)
	return r
}

// ProviderSet connects all dependencies for the server.
var ProviderSet = wire.NewSet(
	// Repositories
	repository.NewSchedulePostgresRepository,
	wire.Bind(new(domainrepo.ScheduleRepository), new(*repository.SchedulePostgresRepository)),

	repository.NewScheduleOccurrencePostgresRepository,
	wire.Bind(new(domainrepo.ScheduleOccurrenceRepository), new(*repository.ScheduleOccurrencePostgresRepository)),

	repository.NewDecisionRulePostgresRepository,
	wire.Bind(new(domainrepo.DecisionRuleRepository), new(*repository.DecisionRulePostgresRepository)),

	repository.NewAttributePostgresRepository,
	wire.Bind(new(domainrepo.AttributeRepository), new(*repository.AttributePostgresRepository)),

	// Services
	service.NewScheduleService,
	service.NewScheduleOccurrenceService,
	service.NewDecisionRuleService,
	service.NewRuleManagementService,
	service.NewAttributeService,

	// Schedule Occurrence materialization worker
	ProvideMatConfig,
	service.NewScheduleMaterializationService,
	ProvideWorkerConfig,
	service.NewOccurrenceWorker,

	// Handlers
	handler.NewScheduleHandler,
	handler.NewScheduleOccurrenceHandler,
	handler.NewDecisionRuleHandler,
	handler.NewRuleManagementHandler,
	handler.NewAttributeHandler,

	// Router
	ProvideRouter,

	// Top-level application bundle
	NewApplication,
)
