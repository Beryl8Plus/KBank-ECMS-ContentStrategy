// @title			KBank ECMS CMS Delivery API
// @version		1.0
// @description	Backend API for KBank ECMS CMS Delivery Runtime Service.
// @host			localhost:8082
// @BasePath		/api/content-strategy/v1
// @in				header
package main

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/fx"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

func main() {
	_ = godotenv.Load()
	logger.LStartup(context.Background(), entity.StartupLog{
		Service: "CMS-DELIVERY", Level: "INFO", Message: "Starting cms-delivery pod",
	})

	app := fx.New(App())

	// app.Err() catches DI graph construction errors (from fx.New) before app.Run() is called.
	if err := app.Err(); err != nil {
		logger.LSystem(context.Background(), entity.SystemLog{
			Service: "CMS-DELIVERY", Level: "FATAL",
			Message: "Container failed to initialise: " + err.Error(),
		})
		os.Exit(1)
	}

	app.Run()
}
