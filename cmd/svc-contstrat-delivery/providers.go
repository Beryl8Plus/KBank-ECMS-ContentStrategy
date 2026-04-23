package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/gorm"

	cmshandler "kbank-ecms/cmd/svc-contstrat-delivery/handler"
	evaluator "kbank-ecms/cmd/svc-contstrat-delivery/internal/evaluator"
	deliveryservice "kbank-ecms/cmd/svc-contstrat-delivery/service"
	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/repository"
)

// ProvideRouter initializes Gin and registers delivery routes.
func ProvideRouter(
	db *gorm.DB,
	rateLimit entity.RateLimit,
	svc deliveryservice.DeliveryService,
) *gin.Engine {
	r := deliveryhttp.InitNewRouter(db, rateLimit)
	cmshandler.RegisterRoutes(r, svc)
	return r
}

// ProvideCMSDeliveryService constructs the service with env-based configs.
func ProvideCMSDeliveryService(
	cacheRepo domainrepo.RedisCacheRepository,
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository,
	decisionRuleRepo domainrepo.DecisionRuleRepository,
	evaluator deliveryservice.RuntimeEvaluator,
	cacheMemory *cache.CacheMemory[any],
) *deliveryservice.CMSDeliveryService {
	resultTTL := parseDurationEnv("CMS_RUNTIME_TTL", 15*time.Minute)
	tickInterval := parseDurationEnv("CMS_RUNTIME_INTERVAL", 5*time.Minute)

	return deliveryservice.NewCMSDeliveryService(
		cacheRepo, occurrenceRepo, decisionRuleRepo,
		evaluator, cacheMemory, resultTTL, tickInterval,
	)
}

// ProvideCacheMemory provides the L1 cache.
func ProvideCacheMemory() (*cache.CacheMemory[any], func()) {
	c := cache.NewCacheMemory[any]("cms_rule", 0.60)
	return c, func() {
		c.Stop()
	}
}

// ProvideRuntimeEvaluator constructs the LocalEvaluator as the RuntimeEvaluator implementation.
func ProvideRuntimeEvaluator() *evaluator.LocalEvaluator {
	return evaluator.NewLocalEvaluator()
}

// App groups the dependencies needed by main.
type App struct {
	Router  *gin.Engine
	Service *deliveryservice.CMSDeliveryService
}

// ProvideApp creates the App struct.
func ProvideApp(r *gin.Engine, svc *deliveryservice.CMSDeliveryService) *App {
	return &App{
		Router:  r,
		Service: svc,
	}
}

// ProviderSet definition
var ProviderSet = wire.NewSet(
	repository.NewScheduleOccurrencePostgresRepository,
	wire.Bind(new(domainrepo.ScheduleOccurrenceRepository), new(*repository.ScheduleOccurrencePostgresRepository)),

	repository.NewDecisionRulePostgresRepository,
	wire.Bind(new(domainrepo.DecisionRuleRepository), new(*repository.DecisionRulePostgresRepository)),
	wire.Bind(new(deliveryservice.RuntimeEvaluator), new(*evaluator.LocalEvaluator)),

	// Providers
	ProvideCacheMemory,
	ProvideRuntimeEvaluator,
	ProvideCMSDeliveryService,
	ProvideRouter,
	ProvideApp,
)
