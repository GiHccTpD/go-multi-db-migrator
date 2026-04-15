# 📦 go-multi-db-migrator

一个轻量级、可扩展的 Go 数据库迁移工具，支持多数据库（MySQL / PostgreSQL / 达梦），专为 **k3s / Kubernetes Job 场景设计**。

---

# ✨ 特性

* ✅ 支持多数据库：MySQL / MariaDB / PostgreSQL / 达梦（DM）
* ✅ 按 **数据库实例维度** 管理迁移（非服务维度）
* ✅ 多数据库统一迁移逻辑
* ✅ migration 校验（checksum 防篡改）
* ✅ 内置 migration 历史表
* ✅ 内置 **分布式锁（锁表方案）**
* ✅ 支持一次性 Job 执行（推荐）
* ✅ 支持多实例并行迁移（不同 DB）

---

# 🧱 目录结构设计

## 核心原则

👉 **按数据库实例名划分，而不是按服务划分**

---

## 目录结构

```text
migrations/{db_instance_name}/{dialect}/
```

示例：

```text
migrations/
  test/
    postgres/
      000001_init_base_schema.up.sql
      000001_init_base_schema.down.sql

  uc_chat_prod/
    mysql/
      000001_init_chat_schema.up.sql

  im_meta/
    dm/
      000001_init_meta_schema.up.sql
```

---

## 为什么按数据库实例划分？

多个服务可能共用一个数据库，例如：

* uc_chat
* uc_account

如果按服务划分：

```text
❌ migrations/uc_chat/postgres/
❌ migrations/uc_account/postgres/
```

会导致：

* migration 冲突
* 表结构不一致
* 难以维护

---

👉 正确做法：

```text
✅ migrations/test/postgres/
```

---

# 🔧 环境变量说明

## 必填

| 变量名          | 说明                           |
| ------------ | ---------------------------- |
| `DB_DIALECT` | 数据库类型（mysql / postgres / dm） |
| `DB_DSN`     | 数据库连接串                       |

---

## 推荐必填（生产环境）

| 变量名                | 说明             |
| ------------------ | -------------- |
| `DB_INSTANCE_NAME` | 数据库实例名（建议显式指定） |

---

## 可选

| 变量名              | 默认值               | 说明            |
| ---------------- | ----------------- | ------------- |
| `MIGRATIONS_DIR` | `/app/migrations` | migration 根目录 |
| `LOG_SQL`        | `false`           | 是否打印 SQL      |

---

## 示例

### PostgreSQL

```bash
DB_DIALECT=postgres
DB_DSN='postgres://postgres:postgres@127.0.0.1:5432/test?sslmode=disable'
DB_INSTANCE_NAME=test
```

---

### MySQL

```bash
DB_DIALECT=mysql
DB_DSN='root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&multiStatements=true'
DB_INSTANCE_NAME=test
```

---

### 达梦（DM）

```bash
DB_DIALECT=dm
DB_DSN='dm://SYSDBA:SYSDBA@127.0.0.1:5236/test?socketTimeout=30&autoCommit=true'
DB_INSTANCE_NAME=test
```

---

# 🧾 Migration 命名规范

```text
{6位版本号}_{snake_case}.{up|down}.sql
```

---

## 示例

```text
000001_init_base_schema.up.sql
000001_init_base_schema.down.sql

000002_add_user_email_index.up.sql
```

---

## 规则

* 版本号必须递增
* 必须 6 位（000001）
* 文件不可修改（已执行后）
* 一个 migration 只做一件事

---

# 🛠 创建 migration

使用生成器：

```bash
go run ./cmd/mk_migration --db-instance test --name init_base_schema --all
```

生成：

```text
migrations/test/mysql/000001_init_base_schema.up.sql
migrations/test/postgres/000001_init_base_schema.up.sql
migrations/test/dm/000001_init_base_schema.up.sql
```

---

# 🚀 运行 migrator

## 本地运行

```bash
DB_DIALECT=postgres \
DB_DSN='postgres://postgres:postgres@127.0.0.1:5432/test?sslmode=disable' \
DB_INSTANCE_NAME=test \
MIGRATIONS_DIR=./migrations \
go run ./cmd/migrator
```

---

# ⚙️ 执行模型（非常重要）

## 推荐方式

👉 **独立 Job 执行 migration**

流程：

```text
1. 构建 migrator 镜像
2. 执行 migration Job
3. Job 成功
4. 部署业务服务
```

---

## ❌ 不推荐方式

不要：

```text
❌ 每个 Pod 启动时执行 migration
❌ initContainer 执行 migration
```

原因：

* 多副本并发执行
* 容易冲突
* 难以排查问题

---

# 🔒 分布式锁设计

## 锁 key 规则

```text
db-migrator:{dialect}:{db_instance_name}
```

示例：

```text
db-migrator:postgres:test
db-migrator:mysql:uc_chat_prod
db-migrator:dm:im_meta
```

---

# ✅ Makefile 使用

编译：

```bash
make build
```

构建镜像：

```bash
make docker
```

本地跑 PostgreSQL：

```bash
make run-postgres
```

生成 migration：

```bash
make mk-migration db=test name=init_base_schema
```

---

## 锁实现方式（统一三库）

👉 使用 **锁表**

```sql
CREATE TABLE IF NOT EXISTS schema_migration_lock (
    lock_key    VARCHAR(128) PRIMARY KEY,
    holder      VARCHAR(128) NOT NULL,
    acquired_at TIMESTAMP NOT NULL
);
```

---

## 加锁逻辑

```sql
INSERT INTO schema_migration_lock(lock_key, holder, acquired_at)
VALUES (?, ?, ?)
```

* 成功：拿到锁
* 失败：锁已存在（其他实例在执行）

---

## 解锁逻辑

```sql
DELETE FROM schema_migration_lock
WHERE lock_key = ? AND holder = ?
```

---

## holder 字段

来源：

```text
POD_NAME（优先）
hostname（兜底）
```

---

## 为什么不用 PG / MySQL 原生锁？

因为：

* 三库行为不一致
* 连接断开语义不同
* 难排查
* 达梦支持不统一

👉 锁表方案最稳定

---

# 📊 migration 历史表

## PostgreSQL 示例

```sql
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
```

---

# ☸️ k3s / Kubernetes Job 示例

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrator-test
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
              value: "postgres"
            - name: DB_DSN
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: dsn
            - name: DB_INSTANCE_NAME
              value: "test"
            - name: MIGRATIONS_DIR
              value: "/app/migrations"
```

---

# 📌 最佳实践

* ✅ migration 按数据库实例划分
* ✅ 每个数据库独立 migration 目录
* ✅ 使用独立 Job 执行
* ✅ 必须开启分布式锁
* ✅ 不要修改已执行 migration
* ✅ schema 变更和数据修复分开
* ✅ migration 保持原子性（小而清晰）

---

# ⚠️ 常见问题

## Q：多个 Job 同时执行怎么办？

👉 分布式锁保证只有一个成功

---

## Q：锁没释放怎么办？

可能原因：

* Pod 崩溃
* Job 被 kill

解决：

```sql
DELETE FROM schema_migration_lock WHERE lock_key = 'xxx';
```

---

## Q：可以多数据库同时跑吗？

可以：

```text
test
uc_chat_prod
im_meta
```

互不影响

---

# 📦 项目定位

这是一个：

👉 **适用于多数据库、多实例、云原生场景的通用迁移引擎**

适合：

* IM 系统
* 微服务架构
* 多租户系统
* k3s / Kubernetes

---

# 🚀 后续建议（强烈推荐）

可以继续增强：

* 灰度 migration（向前兼容）
* migration 回滚策略
* DDL 审计
* 自动 CI 执行
* 多环境管理（dev / test / prod）
