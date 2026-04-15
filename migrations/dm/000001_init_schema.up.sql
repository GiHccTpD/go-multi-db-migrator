-- write your UP migration here
CREATE TABLE IF NOT EXISTS schema_migrations (
    version        VARCHAR(64) PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    checksum       VARCHAR(128) NOT NULL,
    applied_at     TIMESTAMP NOT NULL,
    success        INTEGER NOT NULL,
    execution_ms   BIGINT NOT NULL
);

