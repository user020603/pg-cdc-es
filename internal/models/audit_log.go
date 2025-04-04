package models

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID        int64           `json:"id" db:"id"`
	TableName string          `json:"table_name" db:"table_name"`
	Operation string          `json:"operation" db:"operation"`
	OldData   json.RawMessage `json:"old_data" db:"old_data"`
	NewData   json.RawMessage `json:"new_data" db:"new_data"`
	UserID    string          `json:"user_id" db:"user_id"`
	CreatedAt string          `json:"created_at" db:"created_at"`
	Processed bool            `json:"processed" db:"processed"`
}

type ElasticAuditLog struct {
	TableName string          `json:"table_name"`
	Operation string          `json:"operation"`
	OldData   json.RawMessage `json:"old_data"`
	NewData   json.RawMessage `json:"new_data"`
	UserID    string          `json:"user_id"`
	Timestamp time.Time       `json:"@timestamp"`
}
