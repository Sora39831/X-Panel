# 用户注册功能设计文档

## 概述

在 X-Panel 的登录页增加注册功能，允许用户自行注册。注册过程自动在所有 inbound 中创建代理客户端，模拟管理员手动添加客户端的流程。注册用户可以登录面板查看自己的流量和到期信息。

## 核心概念

- **注册 = 自动创建代理客户端**：用户注册时，系统基于 xray 的 UUID 生成机制创建客户端 ID，在每个 inbound 的 `Settings.clients` 数组中创建 `Client` 对象，同时在 `ClientTraffic` 表创建对应的流量记录。
- **统一身份标识**：用户的 email 同时作为面板登录用户名和代理客户端的 email 标识，在所有 inbound 中保持一致。
- **基于角色的访问控制**：`User` 模型新增 `Role` 字段（`"admin"` / `"user"`）。管理员保留完整面板权限；普通用户只能查看自己的流量信息。

## 数据模型变更

### User 模型 (`database/model/model.go`)

新增一个字段：

```go
type User struct {
    Id       int    `json:"id" gorm:"primaryKey;autoIncrement" bson:"_id,omitempty"`
    Username string `json:"username" bson:"username"`
    Password string `json:"password" bson:"password"`
    Role     string `json:"role" bson:"role"` // "admin" 或 "user"
}
```

- 默认管理员用户：`Role = "admin"`
- 注册用户：`Role = "user"`
- 数据迁移：已有的用户自动填充 `Role = "admin"`

### 无需新建表

所有数据存储在已有表中：`users`、`inbounds`（Settings JSON）、`client_traffics`。

## 注册流程

1. 用户访问登录页，点击"注册"选项卡
2. 输入 email（作为用户名）和密码（仅用于面板登录）
3. 前端校验输入，发送 POST 请求到 `/register`
4. 后端处理：
   - 校验 email 格式、长度（4-64 字符）、允许字符：字母、数字、`_`、`@`、`.`
   - 校验密码：最少 6 位，必须包含字母和数字
   - 检查 `User.Username` 是否已存在
   - 检查 `ClientTraffic.Email` 是否已存在（不允许重复的 email）
   - 使用 xray 的 UUID 生成机制创建客户端 ID
   - 遍历所有已有的 inbound：
     - 解析 `Inbound.Settings` JSON
     - 在 `clients` 数组中追加新的 `Client` 对象（`email=用户名, TotalGB=0, ExpiryTime=0, Enable=true`）
     - 保存更新后的 inbound
     - 创建 `ClientTraffic` 记录（`Email=用户名, Up=0, Down=0, Total=0, ExpiryTime=0, Enable=true`）
   - 创建 `User` 记录，`Role="user"`，密码使用 bcrypt 加密
5. 返回成功，前端跳转到登录页

## 安全设计

### 注册接口防护

| 威胁 | 防护措施 |
|------|----------|
| 暴力注册/爆破 | IP 限流：同一 IP 每分钟最多 5 次注册请求 |
| SQL/NoSQL 注入 | 参数化查询，输入消毒 |
| 弱密码 | 最少 6 位，必须包含字母和数字 |
| 重复注册 | 同时检查 `User.Username` 和 `ClientTraffic.Email` |
| XSS / 路径遍历 | 限制 email 为 `[a-zA-Z0-9_@.]`，长度 4-64 字符 |
| 验证码绕过 | 服务端生成简单算术验证码，提交时校验 |

### 登录接口加固

| 威胁 | 防护措施 |
|------|----------|
| 暴力破解 | IP 锁定：连续 5 次失败后锁定 15 分钟 |
| 用户枚举 | 统一返回"用户名或密码错误"，不区分用户是否存在 |

### 会话 / 访问控制

- 会话中存储 `Role`，与用户对象一起序列化
- `checkLogin` 中间件基于角色进行路由过滤：
  - 管理员 → 完整面板访问权限
  - 普通用户 → 仅允许访问 `/panel/userinfo` 和 `/panel/api/user/info`
- 所有管理 API 拒绝 `role="user"` 的请求

## 前端变更

### login.html

- 在现有登录表单旁增加"注册"选项卡
- 注册表单字段：email、密码、确认密码、验证码
- 表单校验：email 格式、密码匹配、密码强度
- 注册成功后：显示提示信息，自动切换到登录选项卡

### userinfo.html（新建页面）

- 简洁的卡片式布局，无侧边栏导航
- 显示内容：
  - Email（用户名）
  - 遍历包含该客户端的所有 inbound：
    - Inbound 备注 + 协议类型
    - 上传 / 下载流量（人性化字节显示）
    - 总流量限制（0 = 不限）
    - 到期时间（0 = 永久）
    - 启用状态

## 后端变更

### database/model/model.go

- User 结构体新增 `Role` 字段

### database/db.go

- 数据迁移：为已有用户填充 `Role="admin"`

### web/service/user.go

- `Register(email, password)` — 完整注册逻辑
- `GetClientTrafficByEmail(email)` — 返回匹配 email 的所有 ClientTraffic 记录
- `CheckEmailExists(email)` — 检查 User 表和 ClientTraffic 表

### web/controller/index.go

- 新增路由：`POST /register`
- 处理函数：校验输入 → 限流检查 → 验证码校验 → 调用注册服务

### web/controller/xui.go（或新建 controller）

- 新增路由：`GET /panel/userinfo` — 渲染 userinfo.html（带角色检查）
- 新增路由：`GET /panel/api/user/info` — 返回当前用户的 ClientTraffic 数据

### web/controller/base.go

- `checkLogin` 中间件：为 `role="user"` 增加基于路径的访问过滤

## 需要创建/修改的文件

| 文件 | 操作 |
|------|------|
| `database/model/model.go` | 修改：User 结构体新增 Role 字段 |
| `database/db.go` | 修改：为已有用户的数据迁移 |
| `web/service/user.go` | 修改：新增 Register、GetClientTrafficByEmail 方法 |
| `web/controller/index.go` | 修改：新增 /register 路由 |
| `web/controller/base.go` | 修改：基于角色的访问控制 |
| `web/controller/xui.go` | 修改：新增 userinfo 页面路由 |
| `web/html/login.html` | 修改：新增注册选项卡 |
| `web/html/userinfo.html` | 新建：用户信息页面 |
