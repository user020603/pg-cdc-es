# PG-CDC-ES

A PostgreSQL Change Data Capture (CDC) to Elasticsearch streaming service.

## Features

- **Real-time Synchronization**: Stream data from PostgreSQL to Elasticsearch
- **Configurable Batch Processing**: Adjust batch sizes for optimal performance
- **Parallel Processing**: Utilize multiple workers for high throughput
- **Graceful Shutdown**: Proper signal handling ensures clean termination
- **Containerized Deployment**: Docker-ready for easy deployment
- **Environment-based Configuration**: Simple setup through environment variables

## Architecture

The service consists of three main components:

1. **PostgreSQL Repository**: Connects to the PostgreSQL database and retrieves data
2. **Elasticsearch Repository**: Handles the connection and data insertion to Elasticsearch
3. **Sync Service**: Orchestrates the data flow between PostgreSQL and Elasticsearch

## Prerequisites

- Docker (for pulling Postgres & Elasticsearch image and packaging project)
- Go 1.24+

## Quick Start

```shell
# Clone the repository
git clone https://github.com/user020603/pg-cdc-es.git
cd pg-cdc-es

# docker compose
docker-compose up -d
```

## Configuration

The service is configured through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| PG_HOST | PostgreSQL host | localhost |
| PG_PORT | PostgreSQL port | 5432 |
| PG_USER | PostgreSQL username | postgres |
| PG_PASSWORD | PostgreSQL password | postgres |
| PG_DBNAME | PostgreSQL database name | postgres |
| ES_HOST | Elasticsearch host URL | http://localhost:9200 |
| ES_INDEX | Elasticsearch index name | pg_audit_logs |

## Project Structure

```
pg-cdc-es/
├── cmd/
│   └── main.go            # Application entry point
├── internal/
│   ├── repositories/      # Data access components
│   └── services/          # Business logic
├── pkg/
│   └── logger/            # Logging utilities
├── scripts/               # Init trigger & performance testing
├── Dockerfile             # Container definition
└── README.md              # Project documentation
```

## Testing performance

```shell
chmod -x ./scripts/performance_test.sh
./scripts/performance_test.sh
```

Output: 
![image](https://github.com/user-attachments/assets/a2613ce4-b429-4a6f-999a-6b455a379cf2)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Maintainer

- **GitHub**: [user020603](https://github.com/user020603)
- **Last Updated**: 2025-04-05 03:37:28 UTC

---

*This README was generated on 2025-04-05 03:37:28 UTC by user020603*
