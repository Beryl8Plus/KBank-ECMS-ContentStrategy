package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"

	ecmsdocs "kbank-ecms/cmd/server/docs"
	cmshandler "kbank-ecms/cmd/server/handler"
	evaluator "kbank-ecms/cmd/server/internal/evaluator"
	deliveryservice "kbank-ecms/cmd/server/service"
	deliveryhttp "kbank-ecms/internal/delivery/http"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	httpclient "kbank-ecms/internal/http/client"
	"kbank-ecms/internal/infrastructure/cache"
	"kbank-ecms/internal/infrastructure/database"
	"kbank-ecms/internal/infrastructure/logger"
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
	leadRepo domainrepo.LeadRepository,
	customerProfileRepo domainrepo.CustomerProfileRepository,
	customerProfileEnrich deliveryservice.CustomerProfileEnrichConfig,
	schemaRegistryRepo domainrepo.CLENSchemaRegistryRepository,
	attributeRepo domainrepo.AttributeRepository,
) *deliveryservice.CMSDeliveryService {
	return deliveryservice.NewCMSDeliveryService(
		cacheRepo, occurrenceRepo, decisionRuleRepo,
		evaluator, cacheMemory, cfg.Cache.TTL, cfg.Cache.RefreshInterval,
		leadRepo,
		customerProfileRepo, customerProfileEnrich,
		schemaRegistryRepo,
		attributeRepo,
	)
}

// ProvideCLENLeadConfig reads CLEN Lead API settings from env.
// CLEN_LEAD_EXP_F defaults to "true" (only non-expired leads). Set to "false"
// for expired leads, or to the literal string "none" to omit the filter.
func ProvideCLENLeadConfig() httpclient.CLENLeadConfig {
	retry := 2
	if raw := os.Getenv("CLEN_LEAD_API_RETRIES"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			retry = v
		}
	}
	expFilter := os.Getenv("CLEN_LEAD_EXP_F")
	switch strings.ToLower(expFilter) {
	case "":
		expFilter = "true" // default: only non-expired leads
	case "none":
		expFilter = "" // explicit opt-out: CLEN returns both expired and active
	}
	return httpclient.CLENLeadConfig{
		BaseURL:       os.Getenv("CLEN_LEAD_API_BASE_URL"),
		Path:          os.Getenv("CLEN_LEAD_API_PATH"), // constructor fills default when empty
		APIKey:        os.Getenv("CLEN_LEAD_API_KEY"),
		AppIdentifier: os.Getenv("CLEN_LEAD_APP_ID"),
		Timeout:       parseDurationEnv("CLEN_LEAD_API_TIMEOUT", 5*time.Second),
		RetryCount:    retry,
		ExpireFilter:  expFilter,
	}
}

// ProvideCLENLeadClient returns a nil client when the config is empty so
// local dev boots without CLEN credentials. Callers tolerate nil.
func ProvideCLENLeadClient(cfg httpclient.CLENLeadConfig) *httpclient.CLENLeadClient {
	return httpclient.NewCLENLeadClient(cfg)
}

// ProvideCLENCustomerProfileConfig reads CLEN Customer Profile API settings
// from env. Leaving CLEN_CUSTOMER_PROFILE_API_BASE_URL or KEY blank disables
// the integration.
func ProvideCLENCustomerProfileConfig() httpclient.CLENCustomerProfileConfig {
	retry := 2
	if raw := os.Getenv("CLEN_CUSTOMER_PROFILE_API_RETRIES"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			retry = v
		}
	}
	return httpclient.CLENCustomerProfileConfig{
		BaseURL:       os.Getenv("CLEN_CUSTOMER_PROFILE_API_BASE_URL"),
		Path:          os.Getenv("CLEN_CUSTOMER_PROFILE_API_PATH"), // constructor fills default when empty
		APIKey:        os.Getenv("CLEN_CUSTOMER_PROFILE_API_KEY"),
		AppIdentifier: os.Getenv("CLEN_CUSTOMER_PROFILE_APP_ID"),
		Timeout:       parseDurationEnv("CLEN_CUSTOMER_PROFILE_API_TIMEOUT", 5*time.Second),
		RetryCount:    retry,
	}
}

// ProvideCLENCustomerProfileClient returns a nil client when the config is
// empty so local dev boots without CLEN credentials. Callers tolerate nil.
func ProvideCLENCustomerProfileClient(cfg httpclient.CLENCustomerProfileConfig) *httpclient.CLENCustomerProfileClient {
	return httpclient.NewCLENCustomerProfileClient(cfg)
}

// ProvideCustomerProfileEnrichConfig reads the cache TTL used when persisting
// CLEN customer-profile enrichment back to Redis.
func ProvideCustomerProfileEnrichConfig() deliveryservice.CustomerProfileEnrichConfig {
	return deliveryservice.CustomerProfileEnrichConfig{
		CacheTTL: parseDurationEnv("CLEN_CUSTOMER_PROFILE_CACHE_TTL", 5*time.Minute),
	}
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

// parseDurationEnv reads a time.Duration from the given env var, falling back
// to def when the var is unset or the value cannot be parsed. Used by the
// CLEN provider funcs to read per-integration timeouts and cache TTLs that
// are not yet modelled in the central AppConfig.
func parseDurationEnv(key string, def time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return d
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

// ProvideConfig loads AppConfig so fx can inject it throughout the container.
func ProvideConfig() (config.AppConfig, error) {
	return config.LoadConfig("configs/delivery.yaml")
}

// ProvidePostgresDB opens the Postgres connection and registers an OnStop hook to close it.
func ProvidePostgresDB(lc fx.Lifecycle, cfg config.AppConfig) (*gorm.DB, error) {
	db, err := database.NewPostgresDB(cfg.Postgres)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			sqlDB, _ := db.DB()
			return sqlDB.Close()
		},
	})
	return db, nil
}

// ProvideRedisRepository connects to Redis and registers an OnStop hook to close the client.
func ProvideRedisRepository(lc fx.Lifecycle, cfg config.AppConfig) (*repository.RedisRepository, error) {
	repo, err := repository.NewRedisRepository(context.Background(), cfg.Redis)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return repo.Client().Close()
		},
	})
	return repo, nil
}

// RegisterServiceLifecycle starts the background evaluation ticker on app start
// and stops it on shutdown.
func RegisterServiceLifecycle(lc fx.Lifecycle, svc *deliveryservice.CMSDeliveryService) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return svc.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return svc.Stop()
		},
	})
}

// RegisterHTTPServer starts the Gin HTTP server in a goroutine on app start
// and shuts it down gracefully on stop.
func RegisterHTTPServer(lc fx.Lifecycle, cfg config.AppConfig, router *gin.Engine) {
	srv := &http.Server{Addr: ":" + cfg.Server.Port, Handler: router}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.LStartup(context.Background(), entity.StartupLog{
					Service: "CMS-DELIVERY", Level: "INFO",
					Message: "Listening on :" + cfg.Server.Port,
				})
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.LSystem(context.Background(), entity.SystemLog{
						Service: "CMS-DELIVERY", Level: "FATAL",
						Message: "Server error: " + err.Error(),
					})
					os.Exit(1)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}

// RegisterSwaggerHost overrides the Swagger UI host when SWAGGER_HOST env var is set.
func RegisterSwaggerHost(cfg config.AppConfig) {
	if cfg.Swagger.Host != "" {
		ecmsdocs.SwaggerInfo.Host = cfg.Swagger.Host
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

	// CLEN Lead integration
	ProvideCLENLeadConfig,
	ProvideCLENLeadClient,
	repository.NewCLENLeadRepository,
	wire.Bind(new(domainrepo.LeadRepository), new(*repository.CLENLeadRepository)),

	// CLEN Customer Profile integration
	ProvideCLENCustomerProfileConfig,
	ProvideCLENCustomerProfileClient,
	repository.NewCLENCustomerProfileRepository,
	wire.Bind(new(domainrepo.CustomerProfileRepository), new(*repository.CLENCustomerProfileRepository)),
	ProvideCustomerProfileEnrichConfig,

	// CLEN Schema Registry (drives per-rule field discovery for delta fetch)
	repository.NewCLENSchemaRegistryPostgresRepository,
	wire.Bind(new(domainrepo.CLENSchemaRegistryRepository), new(*repository.CLENSchemaRegistryPostgresRepository)),

	// Attribute repository (drives field-name → UUID transform on resolveUserAttrs return)
	repository.NewAttributePostgresRepository,
	wire.Bind(new(domainrepo.AttributeRepository), new(*repository.AttributePostgresRepository)),

	// Providers
	ProvideCacheMemory,
	ProvideRuntimeEvaluator,
	ProvideCMSDeliveryService,
	ProvideRouter,
	ProvideApp,
)
