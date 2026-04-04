CREATE TABLE IF NOT EXISTS logs
(
    timestamp DateTime64(3) CODEC(Delta, ZSTD),
    service   LowCardinality(String) CODEC(ZSTD),
    stream    LowCardinality(String) CODEC(ZSTD),
    level     LowCardinality(String) CODEC(ZSTD),
    message   String CODEC(ZSTD),
    trace_id  String CODEC(ZSTD),
    span_id   String CODEC(ZSTD)
)
ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (service, timestamp)
TTL toDate(timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS llm_usage
(
    timestamp         DateTime64(3) CODEC(Delta, ZSTD),
    request_id        String CODEC(ZSTD),
    service           LowCardinality(String) CODEC(ZSTD),
    model             LowCardinality(String) CODEC(ZSTD),
    prompt_tokens     UInt32 CODEC(ZSTD),
    completion_tokens UInt32 CODEC(ZSTD),
    total_tokens      UInt32 CODEC(ZSTD),
    duration_ms       UInt32 CODEC(ZSTD),
    status            LowCardinality(String) CODEC(ZSTD),
    error             String DEFAULT '' CODEC(ZSTD)
)
ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (service, model, timestamp)
TTL toDate(timestamp) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
