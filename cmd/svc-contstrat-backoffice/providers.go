package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/gorm"

	"kbank-ecms/cmd/svc-contstrat-backoffice/handler"
	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/pubsub"
	"kbank-ecms/internal/repository"
	"kbank-ecms/internal/service"

	localservice "kbank-ecms/cmd/svc-contstrat-backoffice/service"
)

// Application holds the top-level components returned by Wire.
// Wire injectors may only return a single non-error value, so the
// HTTP engine and all background workers are bundled here.
type Application struct {
	Router              *gin.Engine
	OccurrenceWorker    *service.OccurrenceWorker
	AttributeSyncWorker *service.AttributeSyncWorker
}

// NewApplication assembles the Application from its wired components.
func NewApplication(r *gin.Engine, w *service.OccurrenceWorker, sw *service.AttributeSyncWorker) *Application {
	return &Application{Router: r, OccurrenceWorker: w, AttributeSyncWorker: sw}
}

// ProvideMatConfig returns the default MaterializationConfig (7d window, 30d retention).
func ProvideMatConfig() service.MaterializationConfig {
	return service.MaterializationConfig{}
}

// ProvideWorkerConfig returns the default OccurrenceWorkerConfig (1h materialize, cleanup at 00:00).
func ProvideWorkerConfig() service.OccurrenceWorkerConfig {
	return service.OccurrenceWorkerConfig{}
}

// ProvideAttributeSyncWorkerConfig returns the default AttributeSyncWorkerConfig.
// Attribute sync runs daily at 03:00 local time; integrity check runs every 5 minutes.
func ProvideAttributeSyncWorkerConfig() service.AttributeSyncWorkerConfig {
	return service.AttributeSyncWorkerConfig{
		SyncHour:   3,
		SyncMinute: 0,
	}
}

// ProvideExternalAttributeAPIClient returns the stub client until a real
// CLEN HTTP client is implemented.
func ProvideExternalAttributeAPIClient() service.ExternalAttributeAPIClient {
	return service.NewStubExternalAttributeAPIClient()
}

// ProvideRouter initializes the Gin engine with middleware and registers all routes.
func ProvideRouter(
	db *gorm.DB,
	rateLimit entity.RateLimit,
	ruleManagementHandler *handler.RuleManagementHandler,
	scheduleHandler *handler.ScheduleHandler,
	decisionRuleHandler *handler.DecisionRuleHandler,
	wizardHandler *handler.DecisionRuleWizardHandler,
	occurrenceHandler *handler.ScheduleOccurrenceHandler,
	attributeHandler *handler.AttributeHandler,
	channelHandler *handler.ChannelHandler,
	placementHandler *handler.PlacementHandler,
) *gin.Engine {
	r := deliveryhttp.InitNewRouter(db, rateLimit)
	handler.RegisterRoutes(r, ruleManagementHandler, scheduleHandler, decisionRuleHandler, wizardHandler, occurrenceHandler, attributeHandler, channelHandler, placementHandler)
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

	repository.NewDecisionRuleWizardPostgresRepository,
	wire.Bind(new(domainrepo.DecisionRuleWizardRepository), new(*repository.DecisionRuleWizardPostgresRepository)),

	repository.NewAttributePostgresRepository,
	wire.Bind(new(domainrepo.AttributeRepository), new(*repository.AttributePostgresRepository)),

	repository.NewChannelPostgresRepository,
	wire.Bind(new(domainrepo.ChannelRepository), new(*repository.ChannelPostgresRepository)),

	repository.NewPlacementPostgresRepository,
	wire.Bind(new(domainrepo.PlacementRepository), new(*repository.PlacementPostgresRepository)),

	// Pub/Sub publisher (Redis cache repo is supplied by main.go and may be nil
	// when Redis is unavailable; pubsub.Publisher tolerates a nil dependency).
	pubsub.NewPublisher,

	// Services
	service.NewScheduleService,
	service.NewScheduleOccurrenceService,
	service.NewDecisionRuleService,
	service.NewRuleManagementService,
	service.NewAttributeService,
	service.NewChannelService,
	service.NewPlacementService,
	service.NewDecisionRuleWizardService,

	// Schedule Occurrence materialization worker
	ProvideMatConfig,
	service.NewScheduleMaterializationService,
	ProvideWorkerConfig,
	service.NewOccurrenceWorker,

	// Attribute sync + integrity checker
	repository.NewAttributeSyncPostgresRepository,
	wire.Bind(new(domainrepo.AttributeSyncRepository), new(*repository.AttributeSyncPostgresRepository)),

	repository.NewIntegrityPostgresRepository,
	wire.Bind(new(domainrepo.IntegrityRepository), new(*repository.IntegrityPostgresRepository)),

	ProvideExternalAttributeAPIClient,
	service.NewAttributeValidatorService,
	service.NewAttributeSyncService,
	service.NewIntegrityCheckerService,
	ProvideAttributeSyncWorkerConfig,
	service.NewAttributeSyncWorker,

	// Activation service for decision rule wizard
	localservice.NewActivationService,

	// Handlers
	handler.NewScheduleHandler,
	handler.NewScheduleOccurrenceHandler,
	handler.NewDecisionRuleHandler,
	handler.NewDecisionRuleWizardHandler,
	handler.NewRuleManagementHandler,
	handler.NewAttributeHandler,
	handler.NewChannelHandler,
	handler.NewPlacementHandler,

	// Router
	ProvideRouter,

	// Top-level application bundle
	NewApplication,
)
