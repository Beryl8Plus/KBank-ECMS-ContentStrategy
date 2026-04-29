package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	cmshandler "kbank-ecms/cmd/server/handler"
	evaluator "kbank-ecms/cmd/server/internal/evaluator"
	deliveryservice "kbank-ecms/cmd/server/service"
	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/repository"
	"kbank-ecms/pkg/config"
)

// ProvideRouter initializes Gin and registers delivery routes.
func ProvideRouter(
	cfg config.AppConfig,
	db *gorm.DB,
	redisCache *repository.RedisRepository,
	svc deliveryservice.DeliveryService,
) *gin.Engine {
	var redisClient *redis.Client
	if redisCache != nil {
		redisClient = redisCache.Client()
	}
	r := deliveryhttp.InitNewRouter(cfg, db, redisClient)
	cmshandler.RegisterRoutes(r, svc)
	return r
}

// ProvideCMSDeliveryService constructs the service with cache TTL from config.
func ProvideCMSDeliveryService(
	cfg config.AppConfig,
	cacheRepo domainrepo.RedisCacheRepository,
	occurrenceRepo domainrepo.ScheduleOccurrenceRepository,
	decisionRuleRepo domainrepo.DecisionRuleRepository,
	evaluator deliveryservice.RuntimeEvaluator,
	cacheMemory *deliveryservice.MemoryCache,
) *deliveryservice.CMSDeliveryService {
	return deliveryservice.NewCMSDeliveryService(
		cacheRepo, occurrenceRepo, decisionRuleRepo,
		evaluator, cacheMemory, cfg.Cache.TTL, cfg.Cache.RefreshInterval,
	)
}

// ProvideCacheMemory provides the L1 cache.
func ProvideCacheMemory() (*deliveryservice.MemoryCache, func()) {
	schedules := cache.NewCacheMemory[[]*entity.Schedule]("cms-runtime", 0.60, 24*time.Hour)
	decisionRule := cache.NewCacheMemory[*entity.DecisionRule]("cms-runtime", 0.60, 24*time.Hour)
	versionHashes := cache.NewCacheMemory[string]("cms-runtime-versions", 0.60, 24*time.Hour)
	lastSync := cache.NewCacheMemory[time.Time]("cms-runtime-syncs", 0.60, 24*time.Hour)
	memoryCache := deliveryservice.MemoryCache{
		Schedules:     schedules,
		DecisionRule:  decisionRule,
		VersionHashes: versionHashes,
		LastSync:      lastSync,
	}
	return &memoryCache, func() {
		schedules.Stop()
		decisionRule.Stop()
		versionHashes.Stop()
		lastSync.Stop()
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
	wire.Bind(new(deliveryservice.DeliveryService), new(*deliveryservice.CMSDeliveryService)),

	// Providers
	ProvideCacheMemory,
	ProvideRuntimeEvaluator,
	ProvideCMSDeliveryService,
	ProvideRouter,
	ProvideApp,
)
