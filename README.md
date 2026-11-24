# Gabugu Platform

Gabugu AI API 网关与管理平台源码仓库，基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 固定版本分叉定制（上游归属与许可详见 [platform/UPSTREAM.md](platform/UPSTREAM.md) 与 [platform/LICENSE](platform/LICENSE)）。

本仓库仅存放纯净的应用源码、通用部署模板与开发文档；生产环境的部署清单、源站基线、历史发版记录与运维配置独立保存在私有运维仓库中，不混入本代码库。

## 目录结构

- `platform/backend/`：Go + Gin + Ent + PostgreSQL 16 + Redis 7+ 网关与计费后端
- `platform/frontend/`：Vue 3 + TypeScript + Vite + Tailwind CSS 前端应用（固定使用 `pnpm 9.15.9`）
- `platform/deploy/`：通用 Docker / Compose / Nginx / Caddy 部署模板与只读发布校验工具
- `platform/docs/`：产品架构、支付集成、备份与管理后台使用文档
- `docs/`：按领域拆分的开发协作手册（前端、后端、运营后台、发布运维、Agent 协作）

## 快速本地开发与验证

### 后端单元测试（`platform/backend`）
```bash
go test -count=1 -tags=unit ./...
```

### 前端门禁与构建（`platform/frontend`）
```bash
pnpm install --frozen-lockfile
pnpm check:i18n
pnpm typecheck
pnpm lint:check
pnpm test:run
pnpm build
```

### 部署校验脚本单测（`platform/deploy`）
```bash
python -B -m unittest -v release_sync_test test_release_overlay
```

> 详细开发规范与注意事项请查阅 [AGENTS.md](AGENTS.md) 与 `docs/` 下对应领域的入口手册。
