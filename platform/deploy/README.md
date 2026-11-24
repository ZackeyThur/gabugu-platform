# Gabugu 部署模板与校验工具

本目录提供 Gabugu AI 网关的通用部署模板、配置示例与只读源码同步校验工具。

> 生产环境的实际发布记录、主机清单（`production-source.json`、`source-manifests/`、`releases/`）与备份脚本独立保存在私有运维仓库，不混入本代码库。

## 产品与凭据边界

Gabugu 以上游 API 接入为主，并保留管理员按需配置的号池、OAuth/Cookie 与代理能力；两类资源分别管理。上游 API Key 由服务端加密保存；用户 API Key 使用 HMAC 摘要认证，可另存加密副本供本人查看。运行边界详见 `../docs/GABUGU_SCOPE.md`。

`linuxdo_connect`、`oidc_connect` 等 OAuth/SSO 配置（如启用）只用于平台用户登录，不用于访问上游 API，也不会替代上游线路的 API Key。它们在示例配置中默认关闭。

## 蓝绿端口约定

宿主机默认绑定回环地址（不直接对公网暴露应用容器端口，由 Nginx / Caddy 终止 TLS 并反代）：

| 槽位 | 宿主机端口 | 说明 |
| --- | --- | --- |
| Blue | `127.0.0.1:18081` | 蓝色应用槽位 |
| Green | `127.0.0.1:18082` | 绿色应用槽位 |

## 文件说明

| 文件 | 用途 |
| --- | --- |
| `.env.example` | 部署环境变量模板 |
| `config.example.yaml` | 人工运行时的配置模板 |
| `docker-compose.yml` / `docker-compose.*.yml` | Docker Compose 部署与本地开发模板 |
| `Dockerfile` / `build_image.sh` | 镜像构建模板 |
| `Caddyfile` | Caddy 反向代理参考配置 |
| `EDGE_SECURITY.md` | 反向代理、可信代理链、TLS 和入口安全说明 |
| `DOCKER.md` | Docker 运行约定 |
| `release_sync.py` / `release_live.py` | 源码字节清单与远端/线上状态只读校验工具（支持 `--ops-root` 指向外部私有运维仓库） |
| `release_sync_test.py` / `test_release_overlay.py` | 校验工具单元测试 |

## 前置条件

- Linux 主机、Docker Compose v2、curl、Nginx 或 Caddy；
- PostgreSQL 16 和 Redis 7+ 的持久化目录及备份空间；
- 已配置 HTTPS 证书和域名；
- 独立生成并安全保存以下密钥：
  - `POSTGRES_PASSWORD`
  - `JWT_SECRET`
  - `TOTP_ENCRYPTION_KEY`（启用 2FA 时）
  - `GABUGU_API_KEY_HMAC_KEY`
  - `GABUGU_UPSTREAM_ENCRYPTION_KEY`

不要把真实密码、API Key、OAuth secret、Cookie 或令牌写进仓库、日志、镜像标签或公开 Issue。`.env` 应限制为 `chmod 600`。

## 配置模板与预检

```bash
cd platform/deploy
cp .env.example .env
chmod 600 .env
# 编辑 .env 填入强密钥后做 Compose 预检（不启动容器）
docker compose --env-file .env -f docker-compose.yml config
```

如使用独立配置文件：

```bash
cp config.example.yaml config.yaml
chmod 600 config.yaml
```

## 源码同步只读校验（配合独立私有运维仓库）

当生产基线存放在独立的私有运维仓库（例如 `../gabugu-ops`）时，可运行：

```bash
python -B platform/deploy/release_sync.py verify \
  --ops-root ../gabugu-ops \
  --baseline production-source.json \
  --remote-ref refs/heads/master \
  --live-vps <ssh-alias>
```

单元测试：

```bash
cd platform/deploy
python -B -m unittest -v release_sync_test test_release_overlay
```

## 安全与备份参考

- 入口安全、可信代理链、TLS 和请求头处理见 [`EDGE_SECURITY.md`](EDGE_SECURITY.md)
- PostgreSQL 备份与恢复演练见 [`../docs/POSTGRES_BACKUP.md`](../docs/POSTGRES_BACKUP.md)
