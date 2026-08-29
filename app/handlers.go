package app

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "whatsapp-notifier/docs" // This is required to serve swagger docs, it registers the swagger files with the httpSwagger handler
	"whatsapp-notifier/http/handlers"
	"whatsapp-notifier/http/middleware"
)

// initHandlers initializes all HTTP handlers and registers them with the server
func (app *App) initHandlers() error {

	var err error
	// Initialize metrics handler
	metricsHandlerConf := handlers.MetricsHandlerConfig{
		Log:           app.log,
		Observability: app.observability,
		Cfg:           app.config,
	}

	app.handlers = &Handlers{}
	app.handlers.metricsHandler = metricsHandlerConf.NewMetricsHandler()

	// Initialize version handler
	versionHandlerConf := handlers.VersionHandlerConfig{
		Log:           app.log,
		Observability: app.observability,
		Cfg:           app.config,
		AppBuildInfo:  app.buildInfo,
		BasePath:      app.basePath,
	}
	app.handlers.versionHandler, err = versionHandlerConf.NewVersionHandler()
	if err != nil {
		app.log.Error("Failed to initialize version handler", "error", err)
		return err
	}

	// Initialize swagger handler
	swaggerHandlerConf := handlers.SwaggerHandlerConfig{
		Log:           app.log,
		Observability: app.observability,
		Cfg:           app.config,
	}
	swaggerHandler := swaggerHandlerConf.NewSwaggerHandler()
	app.handlers.swaggerHandler = swaggerHandler

	// Initialize health handler
	healthHandlerConf := handlers.HealthHandlerConfig{
		Log:           app.log,
		Observability: app.observability,
		Cfg:           app.config,
		BasePath:      app.basePath,
	}
	app.handlers.healthHandler, err = healthHandlerConf.NewHealthHandler()
	if err != nil {
		app.log.Error("Failed to initialize health handler", "error", err)
		return err
	}

	notificationHandlerConf := handlers.NotificationHandlerConfig{
		Log:      app.log,
		Sender:   app.notificationSender,
		BasePath: app.basePath,
	}
	app.handlers.notificationHandler, err = notificationHandlerConf.NewNotificationHandler()
	if err != nil {
		return fmt.Errorf("initialize notification handler: %w", err)
	}

	return nil
}

// registerRoutes registers all HTTP routes to the server's mux with appropriate middleware
func (app *App) registerRoutes() {
	// Define middleware chains for different endpoints
	loggingMiddleware := middleware.LoggingMiddleware(app.log)
	metricsMiddleware, err := middleware.MetricsMiddleware(app.observability.MeterProvider)
	if err != nil {
		app.log.Error("Failed to create metrics middleware", "error", err)
		return
	}

	// Register metrics handler (no middleware)
	if app.config.O11.PrometheusPath != "" {
		app.log.Info("Registering metrics handler", "path", app.config.O11.PrometheusPath)
		app.handlers.metricsHandler.RegisterRoutes(app.server, promhttp.Handler())
	}

	// Register health handler with metrics middleware
	app.handlers.healthHandler.RegisterRoutes(app.server, metricsMiddleware)

	// Register version handler with metrics and logging middleware
	app.handlers.versionHandler.RegisterRoutes(app.server, metricsMiddleware, loggingMiddleware)

	// Register notification handler with metrics and logging middleware.
	app.handlers.notificationHandler.RegisterRoutes(app.server, metricsMiddleware, loggingMiddleware)

	// Register swagger handler with metrics middleware
	app.handlers.swaggerHandler.RegisterRoutes(app.server, httpSwagger.WrapHandler, metricsMiddleware)
}
