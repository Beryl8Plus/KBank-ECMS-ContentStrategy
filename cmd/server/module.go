package main

import (
	"go.uber.org/fx"

	deliveryservice "kbank-ecms/cmd/server/service"
	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/internal/repository"
)

// App composes every fx.Module that makes up the cms-delivery container.
// Modules are grouped by concern so fx logs scope OnStart/OnStop events to
// the originating module name, making startup and shutdown order legible.
func App() fx.Option {
	return fx.Options(
		configModule(),
		infraModule(),
		repositoryModule(),
		clenModule(),
		serviceModule(),
		httpModule(),
	)
}

func configModule() fx.Option {
	return fx.Module("config",
		fx.Provide(ProvideConfig),
	)
}

func infraModule() fx.Option {
	return fx.Module("infra",
		fx.Provide(
			ProvidePostgresDB,
			ProvideRedisRepository,
		),
	)
}

func repositoryModule() fx.Option {
	return fx.Module("repository",
		fx.Provide(
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
			// Same *RedisRepository singleton also satisfies RedisCacheRepository.
			func(r *repository.RedisRepository) domainrepo.RedisCacheRepository { return r },
		),
	)
}

func clenModule() fx.Option {
	return fx.Module("clen",
		fx.Provide(
			ProvideCLENLeadConfig,
			ProvideCLENLeadClient,
			ProvideCLENCustomerProfileConfig,
			ProvideCLENCustomerProfileClient,
			ProvideCustomerProfileEnrichConfig,
		),
	)
}

func serviceModule() fx.Option {
	return fx.Module("service",
		fx.Provide(
			ProvideCacheMemory,
			fx.Annotate(ProvideRuntimeEvaluator,
				fx.As(new(deliveryservice.RuntimeEvaluator))),
			ProvideCMSDeliveryService,
			// Same *CMSDeliveryService singleton also satisfies DeliveryService.
			func(s *deliveryservice.CMSDeliveryService) deliveryservice.DeliveryService { return s },
		),
		fx.Invoke(RegisterServiceLifecycle),
	)
}

func httpModule() fx.Option {
	return fx.Module("http",
		fx.Provide(ProvideRouter),
		fx.Invoke(
			RegisterSwaggerHost,
			RegisterHTTPServer,
		),
	)
}
