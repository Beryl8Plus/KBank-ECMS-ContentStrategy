// @title						KBank ECMS CMS Delivery API
// @version					1.0
// @description				Backend API for KBank ECMS CMS Delivery Runtime Service.
// @host						localhost:8082
// @BasePath					/
// @securityDefinitions.apikey	XUserIdAuth
// @in							header
// @name						X-User-Id
package main

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/fx"

	deliveryservice "kbank-ecms/cmd/server/service"
	"kbank-ecms/internal/domain/entity"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/infrastructure/logger"
	"kbank-ecms/internal/repository"
)

func main() {
	_ = godotenv.Load()
	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "CMS-DELIVERY", Level: "INFO", Message: "Starting cms-delivery pod",
	})

	app := fx.New(
		fx.Provide(
			ProvideConfig,
			ProvidePostgresDB,
			ProvideRedisRepository,

			// Repository interface bindings
			fx.Annotate(repository.NewScheduleOccurrencePostgresRepository,
				fx.As(new(domainrepo.ScheduleOccurrenceRepository))),
			fx.Annotate(repository.NewDecisionRulePostgresRepository,
				fx.As(new(domainrepo.DecisionRuleRepository))),
			fx.Annotate(repository.NewCLENLeadRepository,
				fx.As(new(domainrepo.LeadRepository))),
			fx.Annotate(repository.NewCLENCustomerProfileRepository,
				fx.As(new(domainrepo.CustomerProfileRepository))),
			fx.Annotate(repository.NewCLENSchemaRegistryPostgresRepository,
				fx.As(new(domainrepo.CLENSchemaRegistryRepository))),
			fx.Annotate(repository.NewAttributePostgresRepository,
				fx.As(new(domainrepo.AttributeRepository))),

			// RedisCacheRepository satisfied from the same *RedisRepository singleton
			func(r *repository.RedisRepository) domainrepo.RedisCacheRepository { return r },

			// CLEN clients + configs
			ProvideCLENLeadConfig,
			ProvideCLENLeadClient,
			ProvideCLENCustomerProfileConfig,
			ProvideCLENCustomerProfileClient,
			ProvideCustomerProfileEnrichConfig,

			// Core
			ProvideCacheMemory,
			fx.Annotate(ProvideRuntimeEvaluator,
				fx.As(new(deliveryservice.RuntimeEvaluator))),
			ProvideCMSDeliveryService,
			// DeliveryService interface satisfied from the same *CMSDeliveryService singleton
			func(s *deliveryservice.CMSDeliveryService) deliveryservice.DeliveryService { return s },
			ProvideRouter,
		),
		fx.Invoke(
			RegisterSwaggerHost,
			RegisterServiceLifecycle,
			RegisterHTTPServer,
		),
	)

	if err := app.Err(); err != nil {
		logger.LSystem(context.Background(), entity.SystemLog{
			Service: "CMS-DELIVERY", Level: "FATAL",
			Message: "Container failed to initialise: " + err.Error(),
		})
		os.Exit(1)
	}

	app.Run()
}
