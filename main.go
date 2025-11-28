package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"news-inshorts/src/infra"
	"news-inshorts/src/middleware"
	"news-inshorts/src/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg, err := infra.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	infraInstance, err := infra.NewInfrastructure(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize infrastructure: %v", err)
	}

	defer infraInstance.Close()

	app := fiber.New(fiber.Config{
		ErrorHandler:          middleware.ErrorHandler,
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		DisableStartupMessage: false,
		AppName:               "Inshorts API v1.0",
	})

	routes.SetupRoutes(app, infraInstance, cfg)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Server.Port)
		infraInstance.Logger.Info("Starting HTTP server", map[string]interface{}{
			"address": addr,
		})

		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	infraInstance.Logger.Info("Shutting down server...", nil)

	if err := app.Shutdown(); err != nil {
		infraInstance.Logger.Error("Server forced to shutdown", err, nil)
	}

	infraInstance.Logger.Info("Server stopped", nil)
}
