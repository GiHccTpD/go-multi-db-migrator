-- write your UP migration here
CREATE TABLE IF NOT EXISTS schema_migrations (
    version        VARCHAR(64) PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    checksum       VARCHAR(128) NOT NULL,
    applied_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    success        SMALLINT NOT NULL,
    execution_ms   BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at
ON schema_migrations (applied_at DESC);
