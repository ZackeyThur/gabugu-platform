# Gabugu PostgreSQL 备份与恢复演练

本文说明 Gabugu 当前蓝绿部署结构下的 PostgreSQL 日常逻辑备份。方案只处理
Gabugu 的 PostgreSQL，不读取旧系统 `127.0.0.1:8080` 的数据目录、配置或凭据。

## 方案边界

- `pg_backup.sh backup` 使用 PostgreSQL custom format (`pg_dump --format=custom`)
  生成一个可由 `pg_restore` 读取的备份文件。
- 备份默认写入 `backend/backups/`（可用 `GABUGU_BACKUP_DIR` 改到独立磁盘），
  每次成功备份后执行按天数的本地保留清理。
- 定时任务由 systemd service + timer 调度；安装前可以只打印计划，避免误改主机。
- 失败时只调用管理员显式配置的本地告警命令，不默认发起网络请求。
- `pg_restore_drill.sh` 在 Docker `none` 网络的一次性 PostgreSQL 容器中恢复并校验
  备份，演练目标与生产数据库隔离，完成后删除临时容器。
- 这是每日逻辑备份，不提供 WAL/PITR。重要部署应把备份复制到独立存储并定期执行
  恢复演练。

生产入口仍是 `backend/docker-compose.yml` 与 `backend/scripts/deploy.sh`。Compose
中的 `postgres` 服务和蓝绿应用（宿主机 `127.0.0.1:18081`、`127.0.0.1:18082`）
共享数据库；`deploy.sh deploy` 迁移前的备份与本方案的日常备份相互独立。

## 脚本入口

从 `platform/` 根目录运行：

```bash
bash backend/scripts/pg_backup.sh help
bash backend/scripts/pg_backup.sh backup
bash backend/scripts/pg_backup.sh retention
bash backend/scripts/pg_backup.sh install-schedule
bash backend/scripts/pg_restore_drill.sh /path/to/gabugu-postgres-20260927T033000Z.dump
```

`deploy.sh` 提供同一部署目录下的转发入口，便于沿用 `GABUGU_ENV_FILE`、
`GABUGU_BACKUP_DIR` 等备份环境变量；日常备份不依赖蓝绿部署的 `.deploy-state`：

```bash
bash backend/scripts/deploy.sh backup
bash backend/scripts/deploy.sh backup-schedule --dry-run
bash backend/scripts/deploy.sh restore-drill /path/to/backup.dump --dry-run
```

定时安装需要 root 或能写入 systemd 单元目录的部署权限；`--dry-run` 只打印将要
创建的 service/timer 和命令，不写文件、不 reload systemd、不启动 timer。

## 环境变量与凭据

除非特别说明，环境变量由服务用户的受限环境文件或 systemd `EnvironmentFile`
注入。不要把真实值写入仓库、命令行历史、备份文件名或日志。

| 变量 | 默认值 | 作用 |
| --- | --- | --- |
| `GABUGU_BACKUP_DIR` | `backend/backups` | 本地备份目录；建议使用独立磁盘挂载点。 |
| `GABUGU_BACKUP_PREFIX` | `gabugu-postgres` | 备份文件名前缀；保留清理只匹配该前缀生成的 `.dump` 文件。 |
| `GABUGU_BACKUP_RETENTION_DAYS` | `14` | 成功备份后保留最近天数；必须是非负整数，`0` 表示不保留超过 0 天的旧文件。 |
| `GABUGU_BACKUP_SCHEDULE` | 每日 `03:30` | systemd timer 的默认执行时间，格式为 `HH:MM`。 |
| `GABUGU_BACKUP_ON_CALENDAR` | 空 | 直接指定 systemd `OnCalendar` 表达式，设置后覆盖 `GABUGU_BACKUP_SCHEDULE`。 |
| `GABUGU_BACKUP_ALERT_COMMAND` | 空 | 备份失败时执行的管理员命令；为空则只返回失败并记录日志。 |
| `GABUGU_BACKUP_EXECUTOR` | `compose` | `compose` 使用当前 Compose 的 `postgres` 服务；`direct` 使用本机 PostgreSQL 客户端。 |
| `GABUGU_BACKUP_DRY_RUN` | `0` | 设为 `1` 时只打印计划，不执行 `pg_dump` 或删除文件。 |
| `GABUGU_ENV_FILE` | `deploy/.env` | Compose 及部署脚本读取的受限环境文件。 |
| `GABUGU_BACKUP_SERVICE_ENV_FILE` | `/etc/gabugu/backup.env` | systemd service 的可选 `EnvironmentFile`，用于告警命令、执行器等运行时变量。 |
| `GABUGU_BACKUP_SYSTEM_USER` | `root` | systemd service 的 `User=`；改为专用用户前必须配置 Docker socket、备份目录和环境文件权限。 |

Compose 执行器读取 `POSTGRES_USER`、`POSTGRES_DB` 和 `POSTGRES_PASSWORD`，连接
Compose 内部的 `postgres` 服务；不要把密码作为 `pg_dump` 参数。direct 执行器应
使用 `GABUGU_PG_HOST`、`GABUGU_PG_PORT`、`GABUGU_PG_USER`、
`GABUGU_PG_DATABASE`（也可使用标准 `PGHOST`、`PGPORT`、`PGUSER`、
`PGDATABASE`）以及 `.pgpass`（权限 `0600`）或进程环境注入密码，并确认目标是
演练或已批准的 Gabugu 数据库。

`GABUGU_ENV_FILE` 只作为 Compose 的 `--env-file` 和服务环境变量传入，备份脚本
不会把它当作 shell 脚本执行。定时服务的 `GABUGU_BACKUP_ALERT_COMMAND`、
`GABUGU_BACKUP_EXECUTOR`、`GABUGU_BACKUP_PREFIX` 等运行时变量应写入
`GABUGU_BACKUP_SERVICE_ENV_FILE` 指向的受限文件。只修改这个文件的内容无需
`systemctl daemon-reload`，下次 service 启动会重新读取；修改 EnvironmentFile 路径
或 unit 内容时才需要重新安装或执行 `systemctl daemon-reload`。

建议准备环境文件：

```bash
cp deploy/.env.example /etc/gabugu/gabugu-backup.env
chmod 600 /etc/gabugu/gabugu-backup.env
# 由部署管理员填写必要变量；不要把该文件提交到 Git。
export GABUGU_ENV_FILE=/etc/gabugu/gabugu-backup.env
```

为定时 service 单独创建只含备份运行时变量的环境文件；Compose 模式不要把数据库
密码或其他应用密钥复制进去。若选择 `direct` 执行器，再按需加入
`GABUGU_PG_*`/`PG*` 连接变量，密码优先放在运行用户的 `.pgpass`（`0600`）中：

```bash
sudo install -d -m 700 /etc/gabugu
sudo install -m 600 /dev/null /etc/gabugu/backup.env
sudoedit /etc/gabugu/backup.env
# 示例（按需填写；不要把真实值提交到仓库）：
# GABUGU_BACKUP_EXECUTOR=compose
# GABUGU_BACKUP_ALERT_COMMAND=/usr/local/sbin/gabugu-backup-alert
```

告警命令应是 root 可控的本地脚本（例如写入受限日志、调用已审查的监控客户端）。
脚本会通过 `GABUGU_BACKUP_ERROR`、`GABUGU_BACKUP_PATH`、
`GABUGU_BACKUP_TIMESTAMP`、`GABUGU_BACKUP_EXIT_CODE` 传递失败上下文；告警脚本
不得把错误变量原样拼接进 shell 代码，也不得输出密码。

## 手动备份与保留

先用 dry-run 检查目录、执行器和计划：

```bash
GABUGU_ENV_FILE=/etc/gabugu/gabugu-backup.env \
GABUGU_BACKUP_DRY_RUN=1 \
GABUGU_BACKUP_EXECUTOR=compose \
  bash backend/scripts/pg_backup.sh backup
```

确认输出只指向预期的 Gabugu PostgreSQL 后，再执行真实备份：

```bash
GABUGU_ENV_FILE=/etc/gabugu/gabugu-backup.env \
  bash backend/scripts/pg_backup.sh backup
```

成功条件包括：`pg_dump` 返回 0、文件非空、文件权限受限且 `pg_restore --list`
可以读取。失败时脚本返回非零；如果配置了 `GABUGU_BACKUP_ALERT_COMMAND`，会在
失败路径调用告警命令。确认失败原因和告警状态后再重试，不要删除最后一个可用备份。

保留清理只匹配脚本生成的备份命名，按 `GABUGU_BACKUP_RETENTION_DAYS` 删除过期
文件；不会清理目录中的其他文件，并会保留最新的匹配备份作为最低恢复点（即使它
已经超过保留天数）。可以单独预览清理：

```bash
GABUGU_BACKUP_DRY_RUN=1 bash backend/scripts/pg_backup.sh retention
```

保留天数只解决本机空间，不代表灾备副本。至少保留一份不在应用主机上的加密副本，
并按组织的 RPO/RTO 要求设置容量和生命周期。

## 安装每日定时任务

先审查生成内容：

```bash
GABUGU_ENV_FILE=/etc/gabugu/gabugu-backup.env \
GABUGU_BACKUP_SERVICE_ENV_FILE=/etc/gabugu/backup.env \
GABUGU_BACKUP_SCHEDULE='03:30' \
  bash backend/scripts/pg_backup.sh install-schedule --dry-run
```

确认 service 的用户、工作目录、环境文件、备份目录和执行器均符合主机基线后，
再安装并启用 timer：

```bash
sudo GABUGU_ENV_FILE=/etc/gabugu/gabugu-backup.env \
  GABUGU_BACKUP_SERVICE_ENV_FILE=/etc/gabugu/backup.env \
  GABUGU_BACKUP_SCHEDULE='03:30' \
  bash backend/scripts/pg_backup.sh install-schedule
systemctl list-timers --all | grep -i gabugu
systemctl status gabugu-postgres-backup.timer
```

生成的 unit 会写入 `User=$GABUGU_BACKUP_SYSTEM_USER`，默认以 root 运行，以便访问
常见 Docker socket。若主机要求专用非 root 用户，应先将
`GABUGU_BACKUP_SYSTEM_USER` 设为已存在的本地用户，为该用户配置 Docker socket、
备份目录和两个环境文件的读取权限，并审查生成的 unit，再启用 timer。

安装后立即检查一次 journal 和 timer 的下次触发时间。不要在同一主机为旧系统
8080 再注册一个指向 Gabugu 数据目录的定时任务；两个系统必须保持数据、凭据和
端口隔离。

## 失败告警处置

建议告警至少包含：备份命令退出码、UTC 时间、目标数据库标识（不含密码）、备份
目录、最近一次成功备份时间和错误摘要。收到告警后的最小处置顺序：

1. 查看 `journalctl -u gabugu-postgres-backup.service`，确认是连接、磁盘、权限、
   `pg_dump` 或校验失败。
2. 检查备份目录剩余空间、属主和 `0700/0600` 权限；确认 `.env`/`.pgpass` 没有
   被提交或被普通用户读取。
3. 修复后手动运行一次 `backup`，再确认 `pg_restore --list` 成功。
4. 若当日备份缺失，记录实际 RPO 风险并暂停会改变 schema 的部署，直到有可读备份。

## 可执行恢复演练

恢复演练脚本不会连接生产数据库。它会读取指定 custom-format 文件，在 Docker
`none` 网络的临时 PostgreSQL 容器中创建目标、运行 `pg_restore`，并在退出时删除
临时资源：

```bash
bash backend/scripts/pg_restore_drill.sh \
  /srv/gabugu/backups/gabugu-postgres-20260927T033000Z.dump
```

先只打印计划：

```bash
bash backend/scripts/pg_restore_drill.sh \
  /srv/gabugu/backups/gabugu-postgres-20260927T033000Z.dump --dry-run
```

演练前确认 Docker 可用、备份文件来自非生产副本或已批准的脱敏数据。成功标准是
容器启动、`pg_restore` 返回 0、目标可连接且脚本清理完成；容器没有宿主端口和外部
网络访问。
记录演练时间、备份文件校验值、使用的 PostgreSQL 镜像版本、耗时和恢复后的行数/应用
健康检查结果；失败时保留错误日志但删除临时凭据和容器。

生产恢复不是 `restore-drill` 的别名。真正恢复前应停止 Gabugu 两个应用槽位的写入，
核对备份时间和 schema 兼容性，取得变更批准，恢复到隔离数据库后再按部署回滚/切流
流程验证。`deploy.sh rollback` 只切回旧应用槽位，不会自动覆盖数据库。

## 生产启用前检查清单

- [ ] `GABUGU_ENV_FILE` 位于主机受限目录，权限为 `0600`，不在 Git 工作区。
- [ ] `GABUGU_BACKUP_DIR` 是有容量告警的独立磁盘或已配置异地复制的目录。
- [ ] 运行 `install-schedule --dry-run`，确认 service/timer 使用预期用户、路径、
      `GABUGU_BACKUP_SYSTEM_USER` 和 `GABUGU_BACKUP_SCHEDULE`。
- [ ] 运行一次成功的手动 `backup`，用 `pg_restore --list` 校验并记录文件大小/校验值。
- [ ] 设置 `GABUGU_BACKUP_RETENTION_DAYS`，确认至少覆盖一次恢复窗口。
- [ ] 配置并故意在测试环境触发一次失败告警，确认没有泄漏凭据。
- [ ] 用 `pg_restore_drill.sh --dry-run` 后执行一次真实演练，记录 RTO 和清理结果。
- [ ] 确认蓝绿端口只使用 `127.0.0.1:18081/18082`，旧 `127.0.0.1:8080` 未被访问。
- [ ] 变更/迁移前另外核对 `deploy.sh deploy` 的迁移前备份、健康检查和回滚条件。

## 离线最小验证

仓库中的部署测试使用命令替身，不连接 Docker、Nginx、PostgreSQL 或生产：

```bash
bash backend/scripts/deploy_test.sh
```

脚本 dry-run 也可在没有数据库凭据的环境中检查计划和路径：

```bash
GABUGU_BACKUP_DRY_RUN=1 bash backend/scripts/pg_backup.sh backup
GABUGU_BACKUP_DRY_RUN=1 bash backend/scripts/pg_backup.sh retention
bash backend/scripts/pg_backup.sh install-schedule --dry-run
bash backend/scripts/pg_restore_drill.sh /tmp/example.dump --dry-run
```

这些命令只能证明参数和计划可审查，不能代替使用真实但非生产的 PostgreSQL 实例
完成一次备份/恢复演练。
