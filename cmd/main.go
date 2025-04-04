package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"user020603/pg-cdc-es/internal/repositories"
	"user020603/pg-cdc-es/internal/services"
	"user020603/pg-cdc-es/pkg/logger"
)

func main() {
	logger := logger.NewLogger("info")

	pgConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("PG_HOST", "localhost"),
		getEnv("PG_PORT", "5432"),
		getEnv("PG_USER", "postgres"),
		getEnv("PG_PASSWORD", "postgres"),
		getEnv("PG_DBNAME", "postgres"),
	)

	pgRepo, err := repositories.NewPostgresRepository(pgConnStr, logger)
	if err != nil {
		logger.Fatal("Failed to create Postgres repository: %v", err)
	}

	esHost := getEnv("ES_HOST", "http://localhost:9200")
	esIndex := getEnv("ES_INDEX", "pg_audit_logs")

	esRepo, err := repositories.NewElasticsearchRepository([]string{esHost}, esIndex, logger)
	if err != nil {
		logger.Fatal("Failed to create Elasticsearch repository: %v", err)
	}

	batchSize := 1000
	numWorkers := 5

	syncServer := services.NewSyncService(pgRepo, esRepo, batchSize, numWorkers, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("Received signal: %s", sig)
		cancel()
	}()

	logger.Info("Starting sync service...")
	if err := syncServer.Start(ctx); err != nil {
		logger.Fatal("Failed to start sync service: %v", err)
	}
	logger.Info("Sync service stopped.")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}