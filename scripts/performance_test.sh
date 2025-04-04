#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  Postgres CDC Performance Test      ${NC}"
echo -e "${BLUE}======================================${NC}\n"

# Configuration
PG_CONTAINER="postgres"       # PostgreSQL container name
ES_CONTAINER="elasticsearch"  # Elasticsearch container name
PG_USER="postgres"
PG_DB="postgres"
ES_INDEX="pg_audit_logs"
TEST_TABLE="performance_test"
NUM_RECORDS=5000
BATCH_SIZE=1000

# Check if containers are running
echo -e "${YELLOW}Checking if containers are running...${NC}"

# Check PostgreSQL container
if ! docker ps | grep -q $PG_CONTAINER; then
    echo -e "${RED}Error: PostgreSQL container '$PG_CONTAINER' is not running.${NC}"
    exit 1
fi
echo -e "${GREEN}PostgreSQL container is running.${NC}"

# Check Elasticsearch container
if ! docker ps | grep -q $ES_CONTAINER; then
    echo -e "${RED}Error: Elasticsearch container '$ES_CONTAINER' is not running.${NC}"
    exit 1
fi
echo -e "${GREEN}Elasticsearch container is running.${NC}"

# Check PostgreSQL connection
echo -e "\n${YELLOW}Checking PostgreSQL connection...${NC}"
if ! docker exec $PG_CONTAINER psql -U $PG_USER -d $PG_DB -c "SELECT 1" &> /dev/null; then
    echo -e "${RED}Error: Cannot connect to PostgreSQL.${NC}"
    exit 1
fi
echo -e "${GREEN}PostgreSQL connection successful.${NC}"

# Setup test table
echo -e "\n${YELLOW}Setting up test environment...${NC}"
docker exec $PG_CONTAINER psql -U $PG_USER -d $PG_DB -c "
    DROP TABLE IF EXISTS $TEST_TABLE;
    CREATE TABLE $TEST_TABLE (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100),
        value INTEGER,
        data JSONB,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    -- Create trigger for the test table
    CREATE TRIGGER audit_trigger_$TEST_TABLE
    AFTER INSERT OR UPDATE OR DELETE ON $TEST_TABLE
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
    
    -- Clear existing audit logs
    DELETE FROM audit_log WHERE table_name = '$TEST_TABLE';
" &> /dev/null

echo -e "${GREEN}Test table created and configured.${NC}"

# Generate test data
echo -e "\n${YELLOW}Generating $NUM_RECORDS test records...${NC}"

# Note the start time
start_time=$(date +%s.%N)

for ((i=1; i<=$NUM_RECORDS; i+=$BATCH_SIZE)); do
    end=$((i+$BATCH_SIZE-1))
    if [ $end -gt $NUM_RECORDS ]; then
        end=$NUM_RECORDS
    fi
    
    # Generate insert values
    values=""
    for ((j=i; j<=$end; j++)); do
        if [ ! -z "$values" ]; then
            values="$values,"
        fi
        random_value=$((RANDOM % 1000))
        values="$values ('Test Record $j', $random_value, '{\"key\": \"value\", \"number\": $j}')"
    done
    
    # Batch insert
    docker exec $PG_CONTAINER psql -U $PG_USER -d $PG_DB -c "
        INSERT INTO $TEST_TABLE (name, value, data) VALUES $values
    " &> /dev/null
    
    # Show progress
    progress=$((end*100/NUM_RECORDS))
    echo -ne "${BLUE}Insertion progress: $progress%\r${NC}"
done

# Note the time after insertion
insert_time=$(date +%s.%N)
insert_duration=$(echo "$insert_time - $start_time" | bc)

echo -e "\n${GREEN}Successfully generated $NUM_RECORDS test records in $insert_duration seconds.${NC}"

# Test sync service performance
echo -e "\n${YELLOW}Testing sync service performance...${NC}"
echo -e "${BLUE}Waiting for sync service to process logs...${NC}"

# Wait for sync service to process logs
timeout=120 # 2 minutes timeout
elapsed=0
while true; do
    # Check how many logs are still unprocessed
    unprocessed=$(docker exec $PG_CONTAINER psql -U $PG_USER -d $PG_DB -t -c "
        SELECT COUNT(*) FROM audit_log 
        WHERE table_name = '$TEST_TABLE' AND processed = FALSE
    " | xargs)
    
    if [ "$unprocessed" -eq 0 ]; then
        # All processed, exit loop
        break
    fi
    
    # Check timeout
    if [ "$elapsed" -ge "$timeout" ]; then
        echo -e "${RED}Timeout reached. Sync service could not process all logs in time.${NC}"
        break
    fi
    
    # Wait a bit and increment counter
    sleep 3
    elapsed=$((elapsed+3))
    
    # Show progress
    processed=$((NUM_RECORDS-unprocessed))
    progress=$((processed*100/NUM_RECORDS))
    echo -ne "${BLUE}Processing progress: $progress% ($processed/$NUM_RECORDS records processed)\r${NC}"
done

# Calculate processing time and speed
end_time=$(date +%s.%N)
processing_time=$(echo "$end_time - $insert_time" | bc)
records_per_second=$(echo "$NUM_RECORDS / $processing_time" | bc)

echo -e "\n\n${YELLOW}Performance Results:${NC}"
echo -e "Total records processed: $NUM_RECORDS"
echo -e "Processing time: $processing_time seconds"
echo -e "Processing speed: $records_per_second records/second"

# Verify data in Elasticsearch (optional)
echo -e "\n${YELLOW}Checking Elasticsearch for processed data...${NC}"
# Wait a moment for Elasticsearch to index documents
sleep 5

# Get current day's index name
current_date=$(date +%Y.%m.%d)
index_name="${ES_INDEX}-${current_date}"

# Check document count in Elasticsearch
es_count=$(docker exec $ES_CONTAINER curl -s "http://localhost:9200/${index_name}/_count" | grep -o '"count":[0-9]*' | cut -d':' -f2)

if [ -z "$es_count" ]; then
    echo -e "${RED}Could not get document count from Elasticsearch.${NC}"
else
    echo -e "Documents in Elasticsearch: $es_count"
fi

# Check if performance meets requirements
if (( $(echo "$records_per_second > 500" | bc -l) )); then
    echo -e "\n${GREEN}✓ PASS: Processing speed exceeds 500 records/second!${NC}"
else
    echo -e "\n${RED}✗ FAIL: Processing speed is below 500 records/second.${NC}"
fi

# Cleanup (optional - comment out if you want to keep the test data)
echo -e "\n${YELLOW}Cleaning up test data...${NC}"
docker exec $PG_CONTAINER psql -U $PG_USER -d $PG_DB -c "
    DROP TABLE IF EXISTS $TEST_TABLE;
    DELETE FROM audit_log WHERE table_name = '$TEST_TABLE';
" &> /dev/null
echo -e "${GREEN}Cleanup complete.${NC}"