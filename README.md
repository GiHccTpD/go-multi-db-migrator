# go-multi-db-migrations

支持多数据库迁移

## 开始

```sql
CREATE TABLE schema_migrations (
    version        VARCHAR(64) PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    checksum       VARCHAR(128) NOT NULL,
    applied_at     TIMESTAMP NOT NULL,
    success        INTEGER NOT NULL,
    execution_ms   BIGINT NOT NULL
);
```

## 推荐执行模型

### 最佳实践

一条 migration 文件只做一件事。

例如：

000001_init_schema.up.sql
000002_create_user_table.up.sql
000003_add_user_email_idx.up.sql

不要一份文件里同时做：

改 5 张表
回填历史数据
顺手删字段
再建索引

越拆开，失败越容易定位。