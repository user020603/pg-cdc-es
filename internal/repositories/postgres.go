package repositories

import (
	"context"
	"database/sql"
	"time"
	"user020603/pg-cdc-es/internal/models"
	"user020603/pg-cdc-es/pkg/logger"

	"github.com/jmoiron/sqlx"
)

type PostgresRepository struct {
	db     *sqlx.DB
	logger *logger.Logger
}

func NewPostgresRepository(connStr string, logger *logger.Logger) (*PostgresRepository, error) {
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
		return nil, err
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Minute * 5)
	return &PostgresRepository{
		db:     db,
		logger: logger,
	}, nil
}

func (r *PostgresRepository) GetUnprocessedLogs(ctx context.Context, limit int) ([]models.AuditLog, error) {
	logs := []models.AuditLog{}

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		UPDATE audit_log 
		SET processed = TRUE
		WHERE id IN (
			SELECT id FROM audit_log
			WHERE processed = FALSE 
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, table_name, operation, old_data, new_data, user_id, created_at,  processed
	`

	err = tx.SelectContext(ctx, &logs, query, limit)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *PostgresRepository) ResetFailedLogs(ctx context.Context, timeout time.Duration) error {
	query := `
		UPDATE audit_log
		SET processed = FALSE
		WHERE processed = TRUE
		AND created <= $1
	`
	timeThreshold := time.Now().Add(-timeout)
	_, err := r.db.ExecContext(ctx, query, timeThreshold)
	return err
}
