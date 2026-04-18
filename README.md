# go-multi-db-migrator

Go 数据库迁移 CLI，支持 MySQL / MariaDB、PostgreSQL、达梦（DM）。项目目标是用独立 Job 在业务服务发布前执行迁移，并按数据库实例隔离迁移历史。

## 核心能力

- 多数据库统一迁移流程。
- 按数据库实例组织迁移文件：`migrations/{db_instance_name}/{dialect}/`。
- 使用 `schema_migrations` 记录已执行版本和 checksum，防止已应用 migration 被篡改。
- 使用数据库锁防止同一实例并发迁移。
- 支持灰度 migration：`up` 可迁移到指定目标版本。
- 支持受控回滚：`down` 按已应用版本倒序回滚到指定目标版本。

## 目录结构

```text
migrations/
  test/
    postgres/
      000001_init_schema.up.sql
      000001_init_schema.down.sql
    mysql/
      000001_init_schema.up.sql
      000001_init_schema.down.sql
    dm/
      000001_init_schema.up.sql
      000001_init_schema.down.sql
```

文件名格式：

```text
{6位版本号}_{snake_case}.{up|down}.sql
```

示例：

```text
000002_add_user_email.up.sql
000002_add_user_email.down.sql
```

已执行的 `.up.sql` 不要修改。工具会在跳过和回滚前校验 `.up.sql` checksum。

## 创建 migration

```bash
go run ./cmd/mk_migration --db-instance test --name add_user_email --all
```

生成：

```text
migrations/test/mysql/000002_add_user_email.up.sql
migrations/test/mysql/000002_add_user_email.down.sql
migrations/test/postgres/000002_add_user_email.up.sql
migrations/test/postgres/000002_add_user_email.down.sql
migrations/test/dm/000002_add_user_email.up.sql
migrations/test/dm/000002_add_user_email.down.sql
```

## 运行

```bash
DB_DIALECT=postgres \
DB_DSN='postgres://postgres:postgres@127.0.0.1:5432/test?sslmode=disable' \
DB_INSTANCE_NAME=test \
MIGRATIONS_DIR=./migrations \
go run ./cmd/migrator
```

环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DB_DIALECT` | 无 | 必填。`mysql`、`mariadb`、`postgres`、`postgresql`、`dm` |
| `DB_DSN` | 无 | 必填。数据库连接串 |
| `DB_INSTANCE_NAME` | 从 DSN 解析 | 数据库实例名，生产建议显式指定 |
| `MIGRATIONS_DIR` | `/app/migrations` | migration 根目录 |
| `MIGRATION_DIRECTION` | `up` | `up` 或 `down` |
| `MIGRATION_TARGET_VERSION` | 空 | 目标版本。`up` 时可选，`down` 时必填 |
| `LOG_SQL` | `false` | 是否打印 SQL |

## 灰度 migration

灰度发布时，先让 migration 向前兼容业务旧版本，再用目标版本分批推进：

```bash
MIGRATION_DIRECTION=up \
MIGRATION_TARGET_VERSION=000002 \
DB_DIALECT=postgres \
DB_DSN='postgres://postgres:postgres@127.0.0.1:5432/test?sslmode=disable' \
DB_INSTANCE_NAME=test \
MIGRATIONS_DIR=./migrations \
go run ./cmd/migrator
```

`up` 行为：

- 未设置 `MIGRATION_TARGET_VERSION`：执行所有未应用的 `.up.sql`。
- 设置 `MIGRATION_TARGET_VERSION`：只执行版本 `<= target` 的 `.up.sql`。
- target 必须是本地存在的 migration 版本，避免拼错后静默执行错误范围。

建议采用 expand-contract：

1. 先加兼容字段、表、索引，不删除旧字段。
2. 发布兼容新旧 schema 的业务代码。
3. 验证后再提交清理旧 schema 的后续 migration。

## 回滚策略

回滚使用 `.down.sql`，按已应用版本倒序执行，直到数据库停在目标版本：

```bash
MIGRATION_DIRECTION=down \
MIGRATION_TARGET_VERSION=000001 \
DB_DIALECT=postgres \
DB_DSN='postgres://postgres:postgres@127.0.0.1:5432/test?sslmode=disable' \
DB_INSTANCE_NAME=test \
MIGRATIONS_DIR=./migrations \
go run ./cmd/migrator
```

假设当前已应用 `000001`、`000002`、`000003`，目标版本是 `000001`，工具会执行：

```text
000003_*.down.sql
000002_*.down.sql
```

`000001` 会保留。若要回滚全部已应用版本，使用 `MIGRATION_TARGET_VERSION=000000`。

回滚约束：

- `down` 必须显式设置 `MIGRATION_TARGET_VERSION`。
- 回滚前会校验对应 `.up.sql` checksum。
- 每个版本的 `.down.sql` 执行成功后，才删除 `schema_migrations` 中对应记录。
- 工具不会在 `up` 失败时自动回滚。DDL 是否可逆取决于数据库和 SQL 内容，失败后应人工确认现场，再执行显式 `down`。

## Makefile

```bash
make build
make docker
make run-postgres
make run-mysql
make run-dm
make mk-migration db=test name=add_user_email
```

## Kubernetes Job

推荐用独立 Job 执行 migration，成功后再发布业务服务。

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrator-test-postgres
spec:
  backoffLimit: 1
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: migrator
          image: your-registry/go-multi-db-migrator:latest
          env:
            - name: DB_DIALECT
              value: postgres
            - name: DB_INSTANCE_NAME
              value: test
            - name: MIGRATIONS_DIR
              value: /app/migrations
            - name: MIGRATION_DIRECTION
              value: up
            - name: DB_DSN
              valueFrom:
                secretKeyRef:
                  name: db-migrator-test-secret
                  key: DB_DSN
```

## 开发

```bash
go test ./...
go build ./...
```

实践约束：

- 一个 migration 只做一件事。
- schema 变更和数据修复分开。
- 对生产回滚有要求的 migration 必须认真编写并测试 `.down.sql`。
- 不要在每个业务 Pod 启动时执行 migration，也不要用多副本并发执行 migration。
