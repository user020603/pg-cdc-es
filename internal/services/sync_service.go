package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	"user020603/pg-cdc-es/internal/models"
	"user020603/pg-cdc-es/internal/repositories"
	"user020603/pg-cdc-es/pkg/logger"
)

type SyncService struct {
	pgRepo      *repositories.PostgresRepository
	esRepo      *repositories.ElasticsearchRepository
	batchSize   int
	numWorkers  int
	logger      *logger.Logger
	pollTimeout time.Duration
}

func NewSyncService(
	pgRepo *repositories.PostgresRepository,
	esRepo *repositories.ElasticsearchRepository,
	batchSize int,
	numWorkers int,
	logger *logger.Logger,
) *SyncService {
	return &SyncService{
		pgRepo:     pgRepo,
		esRepo:     esRepo,
		batchSize:  batchSize,
		numWorkers: numWorkers,
		logger:     logger,
	}
}

func (s *SyncService) Start(ctx context.Context) error {
	// worker pool
	jobs := make(chan []models.AuditLog, s.numWorkers)
	var wg sync.WaitGroup

	for i := 0; i < s.numWorkers; i++ {
		wg.Add(1)
		go s.worker(ctx, jobs, &wg)
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.pgRepo.ResetFailedLogs(ctx, 10*time.Minute); err != nil {
					s.logger.Error("Failed to reset failed logs: %v", err)
				}
			}
		}
	}()

	// Main polling loop
	for {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		default:
			logs, err := s.pgRepo.GetUnprocessedLogs(ctx, s.batchSize)
			if err != nil {
				s.logger.Error("Failed to get unprocessed logs: %v", err)
				time.Sleep(s.pollTimeout)
				continue
			}

			if len(logs) == 0 {
				time.Sleep(s.pollTimeout)
				continue
			}

			s.logger.Info("Found %d unprocessed logs", len(logs))
			jobs <- logs
		}
	}
}

func (s *SyncService) worker(ctx context.Context, jobs <-chan []models.AuditLog, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Worker stopped")
			return
		case logs, ok := <-jobs:
			if !ok {
				s.logger.Info("No more jobs to process")
				return
			}
			if err := s.processLogs(ctx, logs); err != nil {
				s.logger.Error("Failed to process logs: %v", err)
			}
		}
	}
}

func (s *SyncService) processLogs(ctx context.Context, logs []models.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	esLogs := make([]models.ElasticAuditLog, len(logs))
	for i, log := range logs {
		esLog := models.ElasticAuditLog{
			TableName: log.TableName,
			Operation: log.Operation,
			UserID:    log.UserID,
			Timestamp: log.CreatedAt,
		}

		if log.OldData.Valid && log.OldData.String != "" {
			esLog.OldData = json.RawMessage(log.OldData.String)
		}

		// Only set NewData if it's valid
		if log.NewData.Valid && log.NewData.String != "" {
			esLog.NewData = json.RawMessage(log.NewData.String)
		}

		esLogs[i] = esLog
	}

	if err := s.esRepo.BulkIndexLogs(ctx, esLogs); err != nil {
		s.logger.Error("Failed to bulk index logs: %v", err)
		return err
	}

	return nil
}
