# X-Panel MongoDB 支持设计方案

**日期**: 2026-03-25
**版本**: v2（修订版）
**仓库**: Sora39831/x-panel (GitHub)

## 概述

在现有 SQLite + GORM 数据库基础上，新增 MongoDB 作为可选数据库驱动。通过环境变量切换机制让用户在 SQLite 和 MongoDB 之间自由选择，默认保持 SQLite 以确保向后兼容。

### 架构选型：混合适配器模式

经过对代码库的全面分析，现有 service 层在 7 个文件中使用了约 50+ 个 GORM 操作调用（`database.GetDB()`），涵盖 Preload 关联加载、事务、JOIN 查询、raw SQL（含 SQLite JSON 函数）、gorm.Expr 表达式等高级用法。

**选定方案**：混合适配器模式（Hybrid Adapter）

- 保留现有 `*gorm.DB` 全局变量用于 SQLite 模式（完全不变）
- 新增 `DBProvider` 接口 + MongoDB 实现用于 MongoDB 模式
- Service 层通过 `database.GetProvider()` 获取 provider，按数据库类型走不同代码路径
- 优点：SQLite 模式零改动，向后兼容性最强；缺点：service 层需条件分支

**拒绝的方案**：
- GORM MongoDB 驱动：社区维护的 MongoDB GORM 驱动不稳定，且无法支持 Preload/Joins 等 SQL 概念
- 全量 Repository 重构：改动 7 个文件约 50+ 处调用点，风险过高

## 1. 数据库接口层架构

### 1.1 文件结构

```
database/
├── db.go          # 保留，新增 provider 全局变量和工厂函数
├── provider.go    # 新增：DBProvider 接口定义
├── sqlite.go      # 新增：SQLite 实现（封装现有 db.go 中的 GORM 操作）
├── mongodb.go     # 新增：MongoDB 实现（使用 mongo-driver）
├── history.go     # 保留，LinkHistory 操作通过 DBProvider 调用
├── model/         # 保留所有数据模型
└── migrate.go     # 新增：SQLite→MongoDB 数据迁移工具
```

### 1.2 DBProvider 接口

接口覆盖了代码库中所有实际使用的数据库操作（基于对 `user.go`、`setting.go`、`outbound.go`、`inbound.go`、`check_client_ip_job.go`、`subService.go`、`server.go` 的完整分析）：

```go
package database

import (
    "x-ui/database/model"
    "x-ui/xray"
    "gorm.io/gorm"
)

type DBProvider interface {
    // === 生命周期 ===
    Init(dbPath string) error
    Close() error

    // === 通用查询 ===
    IsNotFound(err error) bool

    // === 用户操作 ===
    GetFirstUser() (*model.User, error)
    GetUserByUsername(username string) (*model.User, error)
    CreateUser(user *model.User) error
    UpdateUserByID(id int, updates map[string]any) error
    SaveUser(user *model.User) error
    GetAllUsers() ([]*model.User, error)

    // === 入站操作（最复杂，约 30+ 调用点）===
    GetInboundsWithClientStats() ([]*model.Inbound, error)  // Preload("ClientStats")
    GetAllInboundsWithClientStats() ([]*model.Inbound, error)
    GetInboundByID(id int) (*model.Inbound, error)
    CreateInbound(inbound *model.Inbound) error
    SaveInbound(inbound *model.Inbound) error
    DeleteInboundByID(id int) error
    GetInboundTagByID(id int) (string, error)  // Select("tag").Where().First()
    GetInboundIDs() ([]int, error)              // Pluck("id", &ids)
    CountInboundsByPort(port int) (int64, error) // Where().Count()
    GetInboundEmails() ([]string, error)         // Raw SQL with JSON_EXTRACT

    // === 入站事务操作 ===
    BeginTransaction() (DBProvider, error)
    CommitTransaction() error
    RollbackTransaction() error

    // === 客户端流量（ClientTraffic）===
    CreateClientTraffic(traffic *xray.ClientTraffic) error
    SaveClientTraffic(traffic *xray.ClientTraffic) error
    GetClientTrafficsByEmails(emails []string) ([]*xray.ClientTraffic, error)
    GetClientTrafficsByIDs(ids []int) ([]*xray.ClientTraffic, error)
    DeleteClientTrafficByID(id int) error
    UpdateClientTrafficByEmail(email string, up, down int64) error   // gorm.Expr("up + ?", ...)
    UpdateClientTrafficsBatch(traffics []*xray.ClientTraffic) error  // Save(slice)
    SelectClientTrafficEnableByEmail(email string) (bool, error)

    // === 客户端 IP ===
    GetInboundClientIps() ([]*model.InboundClientIps, error)
    GetInboundClientIpsByInboundID(inboundID int) ([]*model.InboundClientIps, error)
    SaveInboundClientIps(ips *model.InboundClientIps) error
    DeleteClientIpsByInboundID(inboundID int) error
    UpdateClientIpsReset(inboundID int) error

    // === 出站流量 ===
    GetOutboundTraffics() ([]*model.OutboundTraffics, error)
    FirstOrCreateOutboundTraffic(tag string) (*model.OutboundTraffics, error)
    SaveOutboundTraffic(traffic *model.OutboundTraffics) error
    ResetOutboundTraffics(tag string, allTags bool) error

    // === 设置 ===
    GetAllSettings() ([]*model.Setting, error)
    GetSettingByKey(key string) (*model.Setting, error)
    CreateSetting(setting *model.Setting) error
    SaveSetting(setting *model.Setting) error
    DeleteAllSettings() error

    // === 链接历史 ===
    AddLinkHistory(record *LinkHistory) error
    GetLinkHistory() ([]*LinkHistory, error)

    // === 抽奖 ===
    HasUserWonToday(userID int64) (bool, error)
    RecordUserWin(userID int64, prize string) error

    // === Seeder 历史 ===
    GetSeederNames() ([]string, error)            // Pluck("seeder_name", ...)
    CreateSeederHistory(name string) error
    IsTableEmpty(tableName string) (bool, error)

    // === 入站高级操作 ===
    DisableInvalidInbounds(expiryTime int64) (int64, error)
    DisableInvalidClients(expiryTime int64) (int64, error)
    MigrationRemoveOrphanedTraffics() error

    // === SQLite 专属操作（MongoDB 实现为 no-op 或 panic）===
    Checkpoint() error
    IsSQLiteDB(file io.ReaderAt) (bool, error)
    GetGormDB() *gorm.DB  // 仅 SQLite 模式可用，MongoDB 模式返回 nil
}
```

### 1.3 全局 Provider 和工厂函数

`database/db.go` 新增全局 provider 变量：

```go
var provider DBProvider

func GetProvider() DBProvider {
    return provider
}

func NewProvider(dbType string) (DBProvider, error) {
    switch dbType {
    case "mongodb":
        return &MongoDBProvider{}, nil
    case "sqlite", "":
        return &SQLiteProvider{}, nil
    default:
        return nil, fmt.Errorf("unsupported database type: %s", dbType)
    }
}

func InitProvider(dbType, dbPath string) error {
    p, err := NewProvider(dbType)
    if err != nil {
        return err
    }
    provider = p
    return provider.Init(dbPath)
}
```

### 1.4 切换机制

完整的配置链路：

1. `x-ui.sh` 选项 27 将用户选择写入 `/etc/x-ui/db-type.conf`（内容如 `XUI_DB_TYPE=mongodb`）
2. Systemd service 文件通过 `EnvironmentFile=-/etc/x-ui/db-type.conf` 加载该文件为环境变量（`-` 前缀表示文件不存在时不报错）
3. Go 代码通过 `os.Getenv("XUI_DB_TYPE")` 读取
4. 未设置时默认 `sqlite`

`config/config.go` 新增：
```go
func GetDBType() string {
    dbType := os.Getenv("XUI_DB_TYPE")
    if dbType == "" {
        return "sqlite"
    }
    return dbType
}
```

`main.go` 中 `runWebServer()` 改为调用 `database.InitProvider(config.GetDBType(), config.GetDBPath())`。

`Init(dbPath)` 的 MongoDB 含义：SQLite 模式下 `dbPath` 是 `.db` 文件路径；MongoDB 模式下 `dbPath` 参数被忽略（MongoDB 连接信息从 `/etc/x-ui/mongodb.conf` 读取）。

### 1.5 Service 层适配策略

Service 层中需要区分数据库类型的地方，使用条件分支：

```go
// 示例：user.go GetFirstUser
func (s *UserService) GetFirstUser() (*model.User, error) {
    if config.GetDBType() == "mongodb" {
        return database.GetProvider().GetFirstUser()
    }
    // 原有 SQLite/GORM 代码保持不变
    db := database.GetDB()
    user := &model.User{}
    err := db.Model(model.User{}).First(user).Error
    if err != nil {
        return nil, err
    }
    return user, nil
}
```

需要适配的文件（共 7 个）：
1. `web/service/user.go` — 4 处 `database.GetDB()` 调用
2. `web/service/setting.go` — 4 处调用
3. `web/service/outbound.go` — 3 处调用
4. `web/service/inbound.go` — 30+ 处调用（最复杂）
5. `web/job/check_client_ip_job.go` — 6 处调用
6. `sub/subService.go` — 2 处调用（含 SQLite JSON 函数）
7. `web/service/server.go` — 使用 `Checkpoint()`、`IsSQLiteDB()`、`InitDB()`

## 2. MongoDB 实现细节

### 2.1 连接配置

配置文件 `/etc/x-ui/mongodb.conf`：

```
MONGO_HOST=localhost
MONGO_PORT=27017
MONGO_USER=
MONGO_PASS=
MONGO_DB=xui
```

由 `x-ui.sh` 选项 27 写入，Go 代码读取此文件拼接连接串。

### 2.2 Go 实现

- 依赖：`go.mongodb.org/mongo-driver`（官方 MongoDB Go 驱动，当前最新 v1.17.x）
- `config/config.go` 新增 `GetMongoURI() string`，读取 `mongodb.conf` 拼接连接串
- `mongodb.go` 使用 `mongo.Connect()` 建立连接池
- 连接池配置：`maxPoolSize=100`, `minPoolSize=5`, `maxConnIdleTime=30min`

### 2.3 模型映射

- GORM `gorm:"primaryKey;autoIncrement"` → MongoDB `bson:"_id,omitempty"` + counter collection 维护自增 ID
- `AutoMigrate()` → MongoDB 无需迁移（schemaless），首次使用确保集合存在即可
- GORM `Transaction` → MongoDB `session.StartTransaction()` + `session.CommitTransaction()`

### 2.4 ID 兼容性

使用 counter collection 方案维护自增整数 ID，确保对外 API 的 JSON 结构与 SQLite 模式完全一致：

```go
func (p *MongoDBProvider) getNextSequence(collection string) (int, error) {
    filter := bson.M{"_id": collection}
    update := bson.M{"$inc": bson.M{"seq": 1}}
    opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
    result := p.counters.FindOneAndUpdate(context.TODO(), filter, update, opts)
    var doc struct { Seq int `bson:"seq"` }
    result.Decode(&doc)
    return doc.Seq, nil
}
```

### 2.5 GORM 操作的 MongoDB 等价实现

以下是代码库中所有使用的 GORM 操作及其 MongoDB 等价方案：

| GORM 操作 | 使用位置 | MongoDB 等价 |
|-----------|---------|-------------|
| `db.Model().First()` | user.go, setting.go | `collection.FindOne(ctx, filter)` |
| `db.Model().Find()` | 多处 | `collection.Find(ctx, filter)` |
| `db.Model().Create()` | 多处 | `collection.InsertOne(ctx, doc)` |
| `db.Save()` | 多处 | `collection.ReplaceOne(ctx, filter, doc)` |
| `db.Delete(model{}, id)` | inbound.go | `collection.DeleteOne(ctx, filter)` |
| `db.Model().Where().Updates(map)` | 多处 | `collection.UpdateMany(ctx, filter, update)` |
| `db.Model().Count()` | inbound.go | `collection.CountDocuments(ctx, filter)` |
| `db.Model().Pluck("col", &slice)` | inbound.go, db.go | `collection.Distinct(ctx, "col", filter)` |
| `db.Begin()/Commit()/Rollback()` | inbound.go, outbound.go | `session.StartTransaction()` / `CommitTransaction()` |
| `db.Preload("ClientStats")` | inbound.go | 查询后手动填充关联数据：先查 Inbound，再批量查 ClientTraffic |
| `db.Model().Where().FirstOrCreate()` | outbound.go | `FindOne` + 条件 `InsertOne`（或 `UpdateOne` with upsert） |
| `gorm.Expr("up + ?", val)` | inbound.go | `$inc: {up: val}` |
| `db.Model().Where().Order().Limit()` | history.go | `Find(ctx, filter, options.Find().SetSort().SetLimit())` |
| `db.Raw("SELECT ... JSON_EXTRACT(...)")` | inbound.go | MongoDB 聚合管道 `$project` + `$match` |
| `db.Joins("JOIN ...")` | inbound.go | 聚合管道 `$lookup` 或应用层两步查询 |
| `db.Model().Where("settings LIKE ?", ...)` | check_client_ip_job.go | `bson.M{"settings": bson.M{"$regex": pattern}}` |
| `db.Model().Where("email IN (?)", emails)` | inbound.go | `bson.M{"email": bson.M{"$in": emails}}` |
| `db.Exec("PRAGMA wal_checkpoint")` | db.go/server.go | No-op（MongoDB 无需 checkpoint） |

### 2.6 SQLite JSON 函数的 MongoDB 等价

`subService.go` 中使用了 SQLite 特有的 JSON 函数：

```sql
-- subService.go 中的查询（简化）
WHERE JSON_TYPE(settings, '$.clients') = 'array'
  AND EXISTS (SELECT * FROM json_each(settings, '$.clients') WHERE json_extract(value, '$.email') = ?)
```

MongoDB 等价：MongoDB 原生支持 JSON 文档，`settings.clients` 本身就是数组字段：

```go
// MongoDB 查询
filter := bson.M{
    "settings.clients": bson.M{"$elemMatch": bson.M{"email": email}},
}
```

### 2.7 SQLite 专属操作处理

| 函数 | SQLite 行为 | MongoDB 行为 |
|------|------------|-------------|
| `Checkpoint()` | 执行 `PRAGMA wal_checkpoint` | `return nil`（no-op） |
| `IsSQLiteDB()` | 检查文件头签名 | `return false, nil` |
| `GetGormDB()` | 返回 `*gorm.DB` | 返回 `nil` |

## 3. x-ui.sh 菜单新增

### 3.1 选项 26: 安装 MongoDB 数据库

函数名：`install_mongodb`

流程：
1. 检测系统类型（debian/ubuntu/centos/rocky/alma 等）
2. 添加 MongoDB 官方 GPG key 和仓库（支持 MongoDB 7.0+）
   - Debian/Ubuntu：添加 `repo.mongodb.org` apt 源
   - CentOS/RHEL/Rocky：添加 `/etc/yum.repos.d/mongodb-org-7.0.repo`
3. 通过 `apt install -y mongodb-org` 或 `yum install -y mongodb-org` 安装
4. 启动并设置开机自启 `systemctl enable --now mongod`
5. 使用 `mongosh --eval "db.adminCommand('ping')"` 验证安装是否成功
6. 输出成功/失败信息

### 3.2 选项 27: 配置数据库切换

函数名：`config_database`

流程：
1. 读取当前数据库类型（检查 `/etc/x-ui/db-type.conf` 中的 `XUI_DB_TYPE`）
2. 显示当前数据库类型
3. 用户选择：`1. SQLite` / `2. MongoDB`
4. 若选择 MongoDB：
   a. 依次输入 Host（默认 localhost）、Port（默认 27017）、用户名（可为空）、密码（可为空）、数据库名（默认 xui）
   b. 写入 `/etc/x-ui/mongodb.conf`
   c. 写入 `/etc/x-ui/db-type.conf`（内容：`XUI_DB_TYPE=mongodb`）
   d. 通过 systemd EnvironmentFile 传递给面板进程
   e. 询问是否执行 SQLite→MongoDB 数据迁移
   f. 重启面板
5. 若选择 SQLite：
   a. 写入 `/etc/x-ui/db-type.conf`（内容：`XUI_DB_TYPE=sqlite`）
   b. 重启面板

### 3.3 Systemd 集成

在 x-ui 的 systemd service 文件中添加 EnvironmentFile：

```ini
[Service]
EnvironmentFile=-/etc/x-ui/db-type.conf
```

`-` 前缀表示文件不存在时不报错（SQLite 默认模式无需此文件）。

### 3.4 菜单显示

- 范围更新为 `[0-27]`
- show_menu() 中新增显示行：
  ```
  ——————————————————————
    ${green}26.${plain} 安装 MongoDB 数据库
    ${green}27.${plain} 配置数据库切换
  ——————————————————————
  ```
- show_menu() 的 `read -p` 提示更新为 `请输入选项 [0-27]`
- case 语句新增 `26` 和 `27` 分支

## 4. 构建与发布

**现有 `release.yml` 已经支持 linux-amd64 构建，无需新增 workflow 文件。**

当前 `.github/workflows/release.yml` 的 `build-linux` job 已经：
- 使用 matrix 策略构建 amd64、arm64、armv7、armv6、386、armv5、s390x
- 设置 `CGO_ENABLED=1`（line 52）
- 使用 Bootlin musl 交叉编译工具链
- 通过 `softprops/action-gh-release` 自动上传到 GitHub Release

添加 `go.mongodb.org/mongo-driver` 到 `go.mod` 后，构建流程无需任何修改——依赖会自动随 `go build` 编译进二进制文件。

### 4.1 唯一需要的变更

`go.mod` 新增依赖：

```
go.mongodb.org/mongo-driver v1.17.x
```

运行 `go mod tidy` 更新 `go.sum`。

## 5. 依赖变更

`go.mod` 新增：

```
go.mongodb.org/mongo-driver v1.17.x
```

`go.mod` 中 Go 版本保持 `go 1.26` 不变（mongo-driver v1.17.x 支持 Go 1.20+）。

## 6. 兼容性

- 默认行为不变：未配置 `XUI_DB_TYPE` 时自动使用 SQLite
- 现有 SQLite 用户无需任何操作
- MongoDB 模式下的 API 返回格式与 SQLite 模式完全一致（使用 counter collection 保持整数 ID）
- `GetGormDB()` 在 MongoDB 模式下返回 `nil`，调用方需检查 `config.GetDBType()` 后决定使用哪个接口

## 7. 实施顺序

1. `config/config.go` — 新增 `GetDBType()`
2. `database/provider.go` — 定义 `DBProvider` 接口
3. `database/db.go` — 新增 `provider` 全局变量、`GetProvider()`、`NewProvider()`、`InitProvider()`
4. `database/sqlite.go` — SQLite 实现（封装现有 GORM 代码）
5. `database/mongodb.go` — MongoDB 实现
6. `database/migrate.go` — SQLite→MongoDB 数据迁移
7. `main.go` — 修改启动流程使用 `InitProvider()`
8. Service 层适配（7 个文件）——按复杂度从低到高：
   a. `web/service/user.go`
   b. `web/service/setting.go`
   c. `web/service/outbound.go`
   d. `web/service/server.go`
   e. `web/job/check_client_ip_job.go`
   f. `sub/subService.go`
   g. `web/service/inbound.go`（最复杂，30+ 处调用）
9. `x-ui.sh` — 新增选项 26（安装 MongoDB）和 27（配置数据库切换）
10. `go.mod` / `go.sum` — 添加 mongo-driver 依赖
11. 测试与验证
