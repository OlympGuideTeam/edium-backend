CREATE TABLE IF NOT EXISTS logs
(
    timestamp DateTime64(3) CODEC(Delta, ZSTD),
    service   LowCardinality(String) CODEC(ZSTD),
    stream    LowCardinality(String) CODEC(ZSTD),
    level     LowCardinality(String) CODEC(ZSTD),
    message   String CODEC(ZSTD)
)
ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (service, timestamp)
TTL toDate(timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;
