package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gorm.io/gorm"

	cmshandler "kbank-ecms/cmd/cms-delivery/handler"
	deliveryservice "kbank-ecms/cmd/cms-delivery/service"
	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	domainservice "kbank-ecms/internal/domain/service"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/repository"
)

// ProvideRouter initializes Gin and registers delivery routes.
func ProvideRouter(
	db *gorm.DB,
	rateLimit entity.RateLimit,
	svc domainservice.DeliveryService,
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
	evaluator domainservice.RuntimeEvaluator,
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

	ProvideCacheMemory,
	ProvideCMSDeliveryService,
	wire.Bind(new(domainservice.DeliveryService), new(*deliveryservice.CMSDeliveryService)),

	ProvideRouter,
	ProvideApp,
)
