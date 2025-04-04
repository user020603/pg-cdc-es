package repositories

import (
	"bytes"
	"context"
	"fmt"
	"time"
	"user020603/pg-cdc-es/internal/models"
	"user020603/pg-cdc-es/pkg/logger"

	"encoding/json"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type ElasticsearchRepository struct {
	client *elasticsearch.Client
	index  string
	logger *logger.Logger
}

func NewElasticsearchRepository(addresses []string, index string, logger *logger.Logger) (*ElasticsearchRepository, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		logger.Fatal("Failed to create Elasticsearch client: %v", err)
		return nil, err
	}

	return &ElasticsearchRepository{
		client: client,
		index:  index,
		logger: logger,
	}, nil
}

func (r *ElasticsearchRepository) BulkIndexLogs(ctx context.Context, logs []models.ElasticAuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	var buf bytes.Buffer

	for _, log := range logs {
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": fmt.Sprintf("%s-%s", r.index, time.Now().Format("2006.01.02")),
			},
		}

		if err := json.NewEncoder(&buf).Encode(meta); err != nil {
			r.logger.Error("Failed to encode meta: %v", err)
			return err
		}

		if err := json.NewEncoder(&buf).Encode(log); err != nil {
			r.logger.Error("Failed to encode log: %v", err)
			return err
		}
	}

	req := esapi.BulkRequest{
		Body:    bytes.NewReader(buf.Bytes()),
		Refresh: "false",
		Timeout: 30 * time.Second,
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		var raw map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
			r.logger.Error("Failed to decode response: %v", err)
			return err
		}
		return fmt.Errorf("elasticsearch status: %d, error: %v", res.StatusCode, raw)
	}
	
	r.logger.Info("Bulk index response: %s", res.String())
	if res.StatusCode != 200 {
		return fmt.Errorf("elasticsearch status: %d", res.StatusCode)
	}

	r.logger.Info("Successfully transferred %d logs to Elasticsearch", len(logs))
    for i, log := range logs {
        logData, _ := json.Marshal(log)
        r.logger.Debug("Log %d successfully indexed: %s", i+1, string(logData))
    }
    
    r.logger.Info("Bulk index successful")
	return nil
}
