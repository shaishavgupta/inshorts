package routes

import (
	"news-inshorts/src/controllers"
	"news-inshorts/src/infra"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// SetupRoutes configures all routes and middleware for the application
func SetupRoutes(app *fiber.App, infraInstance *infra.Infrastructure, cfg *infra.Config) {
	appLogger := infra.GetLogger()

	ctrls := controllers.NewControllers(cfg, infraInstance.DB, infraInstance.Redis)
	appLogger.Info("Controllers initialized", nil)

	// Register recover middleware (panic recovery)
	app.Use(recover.New())

	// Register CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Register logging middleware
	app.Use(func(c *fiber.Ctx) error {
		start := c.Context().Time()

		// Process request
		err := c.Next()

		// Log request details
		duration := c.Context().Time().Sub(start)
		appLogger.Info("HTTP request", map[string]interface{}{
			"method":   c.Method(),
			"path":     c.Path(),
			"status":   c.Response().StatusCode(),
			"duration": duration.String(),
			"ip":       c.IP(),
		})

		return err
	})

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "inshorts-api",
		})
	})

	// Define route groups for /api/v1/news and /api/v1/interactions
	apiV1 := app.Group("/api/")

	// News routes
	newsRoutes := apiV1.Group("v1/news")
	newsRoutes.Post("/", ctrls.News.CreateArticle)
	newsRoutes.Get("/query", ctrls.News.QueryNews)
	newsRoutes.Get("/trending", ctrls.News.GetTrending)
	newsRoutes.Get("/filter", ctrls.News.FilterArticles)
	newsRoutes.Post("/load", ctrls.News.LoadData)

	// User interaction routes
	interactionRoutes := apiV1.Group("v1/interactions")
	interactionRoutes.Post("/record", ctrls.UserInteraction.RecordInteraction)
}
