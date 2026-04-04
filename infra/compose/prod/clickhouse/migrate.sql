ALTER TABLE logs ADD COLUMN IF NOT EXISTS trace_id String CODEC(ZSTD);
ALTER TABLE logs ADD COLUMN IF NOT EXISTS span_id  String CODEC(ZSTD);

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
