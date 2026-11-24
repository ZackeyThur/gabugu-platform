# 后端与网关领域手册（BACKEND）

> **何时阅读**：仅在修改 Go 后端代码、API 网关协议转发、账号/号池调度、数据库 Schema 与迁移、计费定价逻辑，或合并上游 `sub2api` 新版本时阅读。纯前端或纯文案任务无需阅读。

---

## 一、技术栈与目录结构

- **核心栈**：Go（版本以 `platform/backend/go.mod` 为准）+ Gin + Ent ORM + PostgreSQL 16 + Redis 7+，源码位于 `platform/backend/`。
- **关键目录**：
  - `cmd/server/`：服务主程序入口
  - `ent/schema/` 与 `ent/`：数据库 Schema 定义及 Ent 自动生成代码
  - `internal/handler/`（HTTP/SSE/WS 处理器）、`internal/service/`（核心调度与计费业务）、`internal/repository/`（数据访问层）、`internal/server/`（路由与服务装配）
  - `migrations/`：启动时自动执行的内嵌 SQL 迁移脚本

---

## 二、网关架构与产品边界

- **原生主线 + 双源并存**：基于固定上游 `Wei-Shaw/sub2api` 原生架构（当前 v0.2.23 基线），同时支持 **API 上游线路**（上游站点 + 加密 API Key）与 **自养号池**（订阅账号、OAuth/Cookie/Setup Token、凭据自动刷新、独立账号出口代理）并存，由管理端按分组和路由策略调度，**绝不以“纯上游”为由拆除号池能力**。
- **对终端用户透明隔离**：终端用户仅持绑定单一分组的平台 API Key，按 OpenAI（`/v1`）、Anthropic（根路径）、Gemini（根路径 + `v1beta`）原生协议调用，绝不向用户暴露真实上游站点、线路名、号池凭据或内部路由规则。
- **账务、定价与凭据安全不变量**：
  - **1:1 汇率与精度**：站内余额与单价以 USD 计量，与人民币严格 `1:1` 等值（`¥1 = $1`），入账精度四舍五入至 8 位小数 `NUMERIC(20,8)`；生产可按需开启 `GABUGU_LEGACY_WALLET_COMPAT=true` 兼容历史本金/赠送/退款/冻结账务语义。
  - **转发前定价完整性拦截**：主力文本网关、Gemini/Antigravity handler 与 OpenAI WS 入口在转发前校验有效价格；未知模型绝不套用测试默认零价，缺价直接返回本地 `pricing_unavailable`（HTTP 503 或流内 error），不记为上游账号故障，也不触发付费重试；显式配置的 `0` 仍作为合法免费档生效。
  - **凭据加密**：用户 API Key 采用 HMAC 哈希、上游 Key 采用 AES-GCM 加密、兑换码采用哈希存储，禁止新增明文可逆凭据列。

---

## 三、本地开发与防踩坑清单（必看）

1. **启动即写库（无独立 migrate 子命令）**：
   - `server` 初始化和正常启动都会**自动执行内嵌 SQL 迁移写库**。绝不可连生产数据库“启动看看”！
   - 本地开发必须配置独立的 PostgreSQL 库、Redis 命名空间、`DATA_DIR` 和端口（`SERVER_HOST=127.0.0.1`，`SERVER_PORT=18081`，双实例时用 `18082`；配置参考 `platform/deploy/config.example.yaml`）。
2. **Windows 本地开发特例**：
   - 打包归档涉及盘符路径时，务必显式调用 `C:\Windows\System32\tar.exe`（Git 自带 GNU tar 会把盘符冒号误判为远程主机报错）。
   - Windows 本地若无 `make`，直接跑下方的 `go test` / `go build` 原始命令。
3. **Ent 与 Go Interface 同步铁律**：
   - 修改 `ent/schema/*.go` 后，必须运行 `go generate ./ent` 并将生成的 `ent/` 文件一并提交。
   - 给任何 Go `interface` 新增方法后，必须全局检索 `rg "type.*(Stub|Mock).*struct" internal/`，补全所有测试桩实现，否则单测编译必败。
4. **PostgreSQL 与 SQL 迁移防坑**：
   - Windows 本地连接一律用 `127.0.0.1` 代替 `localhost`（避免 IPv6 `::1` 握手报错）。
   - `psql -f` 不支持含中文的路径（可先拷到纯英文临时路径执行）；PowerShell 双引号会把 bcrypt hash 里的 `$2a$10...` 当变量吞掉，含 `$` 的 SQL 一律写进文件通过 `psql -f` 执行。
   - **迁移脚本不可篡改**：已应用的 `migrations/*.sql` 严禁事后修改；应用按 `strings.TrimSpace` 后的内容计算 SHA256 校验和；包含 `CREATE INDEX CONCURRENTLY` 的迁移须使用 `_notx.sql` 非事务命名并在隔离库先行验证。

---

## 四、测试与构建命令

在 `platform/backend` 下执行（仓库 CI 配置位于 `.github/workflows/gabugu-ci.yml`）：

```bash
# 单元测试
go test -count=1 -tags=unit ./...

# 集成测试（需先指向本地独立测试 PostgreSQL / Redis）
go test -count=1 -tags=integration ./...

# 代码检查与嵌入前端构建
golangci-lint run ./...
go build -tags=embed ./cmd/server
```

> 生产环境的逐文件 SHA256 清单与发布基线由独立私有运维仓库维护，上线后按 [OPS_RELEASE.md](OPS_RELEASE.md) 执行同步校验。

---

## 五、后端子模块按需深读（仅改到对应模块时再打开）

| 涉及子模块 | 查阅文档 |
| --- | --- |
| 支付渠道配置、满赠折扣、订阅套餐、退款与管理端支付 API | [PAYMENT_CN.md](../platform/docs/PAYMENT_CN.md)、[ADMIN_PAYMENT_INTEGRATION_API.md](../platform/docs/ADMIN_PAYMENT_INTEGRATION_API.md) |
| 生图网关（Codex 非流式保 HTTP 状态码、反向代理 900s 超时预算、异步生图对象存储） | [OPENAI_IMAGES_CODEX.md](../platform/docs/OPENAI_IMAGES_CODEX.md)、[ASYNC_IMAGE_TASKS.md](../platform/docs/ASYNC_IMAGE_TASKS.md) |
| 合并上游 `Wei-Shaw/sub2api` 新版本（固定 tag 三路 `git apply --3way`，禁直接 merge 浮动分支） | [UPSTREAM.md](../platform/UPSTREAM.md) |
