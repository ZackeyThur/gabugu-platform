# 发布、同步与运维领域手册（OPS & RELEASE）

> **何时阅读**：**仅在获得明确授权准备部署上线、执行回滚、核对线上运行版本、开展例行巡检或操作数据库备份时阅读。** 平时在本地编写代码、修复缺陷和验证阶段无需提前阅读。

---

## 一、上线第一步：先判定发布类型（只做必需步骤）

动手前先读取线上真实配置与状态，按最小必要范围选择发布方式，严禁为纯数据/配置改动准备整套后端镜像构建与切流：

| 改动内容 | 标准发布方式 | 核心要求 |
| --- | --- | --- |
| **仅改后台配置或数据库数据** | 只执行配置更新或经验证的数据变更 | 不重编程序、不重启主网关；SQL 必须先在本地隔离库用真实数据形状验证影响行数与最终值 |
| **仅改前端静态资源** | 独立版本目录原子切换 Nginx 静态根 | 新旧哈希资源共存防已打开页面报错，后端 0 重启 |
| **仅改后端程序（无 SQL 迁移）** | 双实例蓝绿无感交接 | 新实例待命且暂不启共享后台任务 → 切流 → 旧实例连接与在途结算排空退出 → 新实例接管后台任务 |
| **带向后兼容 SQL 迁移** | 本地隔离库验证迁移 + 双实例交接 | 确认旧程序在新表结构可运行；含 `CREATE INDEX CONCURRENTLY` 的 `_notx.sql` 禁放进事务，用可续跑迁移控制器 |
| **不兼容迁移或资金写入语义变更** | 优先拆分为多次兼容发布 | 确实无法拆分时，先备齐候选、验证与恢复方案，申请明确的维护窗口授权 |

---

## 二、生产操作安全门槛

1. **构建一律在本地，生产主机只收成品**：本地从干净检出构建并留下构建证明（记录源码提交、二进制 SHA、Go 版本与构建参数）；上传前核对主机磁盘余量；所有线上发布操作必须在发布锁下顺序执行。
2. **代码与运维档案分离**：具体生产主机地址、发布历史记录、`production-source.json`、逐文件 `source-manifests/*.sha256` 清单及上游成本表，统一存放在**独立的私有运维仓库**中，严禁提交进本代码仓库。
3. **安全回退原则**：回退使用当次发布控制器的 `rollback` 原子切回已核验的上一版程序并排空连接；**绝不可用旧数据库备份覆盖已产生新交易的生产库**。
4. **原始字节与行尾保护**：仓库 `.gitattributes` 为 `* -text`（不做自动换行转换），同步校验按原始字节算 SHA256。修改 `production-source.json` 保持 `CRLF`，修改 `source-manifests/*.sha256` 保持 `LF`。

---

## 三、发布或回滚收尾：外部私有运维基线三端校验

每次上线或回滚后，通过 `release_sync.py` 的 `--ops-root` 参数，将**本地干净代码仓库、远端代码仓库 `master`、私有运维仓库基线与线上实际运行态**完整对齐校验：

```bash
python -B platform/deploy/release_sync.py verify \
  --ops-root /path/to/gabugu-ops \
  --baseline production-source.json \
  --remote-ref refs/heads/master \
  --live-vps <ssh-alias>
```

完整校验通过才算“同步完成”；若有任一未对上项，严禁伪造指针，须在私有运维仓库的发布记录中如实写明剩余差异。

---

## 四、通用部署专题参考

| 具体专题 | 查阅文档 |
| --- | --- |
| Docker / Compose / 蓝绿部署模板与环境变量说明 | [platform/deploy/README.md](../platform/deploy/README.md)、[platform/deploy/DOCKER.md](../platform/deploy/DOCKER.md) |
| PostgreSQL 在线备份与本地隔离恢复演练 | [POSTGRES_BACKUP.md](../platform/docs/POSTGRES_BACKUP.md) |
| 边缘安全、Nginx 与网关超时/防护策略 | [EDGE_SECURITY.md](../platform/deploy/EDGE_SECURITY.md) |
