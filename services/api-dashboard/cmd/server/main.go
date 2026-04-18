package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/config"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/handler"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/router"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Database
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// OpenTelemetry
	tp, err := initTracer(ctx, cfg.OTLPEndpoint)
	if err != nil {
		slog.Error("failed to init tracer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer tp.Shutdown(ctx)
	otel.SetTracerProvider(tp)

	// Auth middleware (JWKS)
	auth, err := middleware.NewAuth(cfg.BetterAuthJWKSURL)
	if err != nil {
		slog.Error("failed to initialize JWKS auth", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// DI — organizations
	orgRepo := repository.NewOrganizationRepository(pool)
	orgService := service.NewOrganizationService(orgRepo, tp)
	orgHandler := handler.NewOrganizationHandler(orgService)

	// DI — users
	userRepo := repository.NewUserRepository(pool)
	userService := service.NewUserService(userRepo, tp)
	userHandler := handler.NewUserHandler(userService)

	// DI — org invites
	orgInviteRepo := repository.NewOrgInviteRepository(pool)
	orgInviteService := service.NewOrgInviteService(orgInviteRepo, tp)
	orgInviteHandler := handler.NewOrgInviteHandler(orgInviteService)

	// DI — publisher apps
	publisherAppRepo := repository.NewPublisherAppRepository(pool)
	publisherAppService := service.NewPublisherAppService(publisherAppRepo, tp)
	publisherAppHandler := handler.NewPublisherAppHandler(publisherAppService)

	// DI — api keys
	apiKeyRepo := repository.NewAPIKeyRepository(pool)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, tp)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)

	// DI — publisher rules
	publisherRuleRepo := repository.NewPublisherRuleRepository(pool)
	publisherRuleService := service.NewPublisherRuleService(publisherRuleRepo, tp)
	publisherRuleHandler := handler.NewPublisherRuleHandler(publisherRuleService)

	healthHandler := handler.NewHealthHandler()

	mux := router.NewRouter(&router.Handlers{
		Health:        healthHandler,
		Organization:  orgHandler,
		User:          userHandler,
		OrgInvite:     orgInviteHandler,
		PublisherApp:  publisherAppHandler,
		APIKey:        apiKeyHandler,
		PublisherRule: publisherRuleHandler,
	}, auth)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	slog.Info("starting server", slog.String("port", cfg.Port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func initTracer(ctx context.Context, endpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("api-dashboard"),
		)),
	)
	return tp, nil
}
