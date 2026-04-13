package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	infrastructurepostgres "example.com/taskservice/internal/infrastructure/postgres"
	postgresrepo "example.com/taskservice/internal/repository/postgres"
	"example.com/taskservice/internal/scheduler"
	transporthttp "example.com/taskservice/internal/transport/http"
	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
	"example.com/taskservice/internal/usecase/task"
	"example.com/taskservice/internal/usecase/tasktemplate"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := infrastructurepostgres.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("open postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	taskRepo := postgresrepo.New(pool)
	txMgr := infrastructurepostgres.NewTransactionManager(pool)
	templateRepo := postgresrepo.NewTemplateRepository(pool, txMgr, postgresrepo.NewTemplateMapCache())

	taskUsecase := task.NewService(taskRepo)
	templateUsecase := tasktemplate.NewService(templateRepo, logger)

	taskHandler := httphandlers.NewTaskHandler(taskUsecase, templateUsecase)
	templateHandler := httphandlers.NewTemplateHandler(templateUsecase)
	docsHandler := swaggerdocs.NewHandler()

	router := transporthttp.NewRouter(taskHandler, templateHandler, docsHandler)

	bg := scheduler.New(templateUsecase, cfg.GenerationHorizonDays, logger)
	go bg.Run(ctx)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown http server", "error", err)
		}
	}()

	logger.Info("http server started", "addr", cfg.HTTPAddr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("listen and serve", "error", err)
		os.Exit(1)
	}
}

type config struct {
	HTTPAddr              string
	DatabaseDSN           string
	GenerationHorizonDays int
}

func loadConfig() config {
	cfg := config{
		HTTPAddr:              envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseDSN:           envOrDefault("DATABASE_DSN", "postgres://postgres:postgres@localhost:5432/taskservice?sslmode=disable"),
		GenerationHorizonDays: envOrDefaultInt("GENERATION_HORIZON_DAYS", 30),
	}

	if cfg.DatabaseDSN == "" {
		panic(fmt.Errorf("DATABASE_DSN is required"))
	}

	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return val
}
