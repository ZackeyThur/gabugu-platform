# 前端领域手册（FRONTEND）

> **何时阅读**：仅在修改网页界面、Vue 组件、样式配色、前端路由、全站文案，或执行前端测试与浏览器验收时阅读。纯后端或纯运维任务无需阅读。

---

## 一、技术栈与目录结构

- **核心栈**：Vue 3 + TypeScript + Vite + Tailwind CSS，源码位于 `platform/frontend/`。
- **包管理铁律**：固定使用 **`pnpm 9.15.9`**（Node.js 24），**严禁使用 `npm`**。
  - 修改 `package.json` 后必须运行 `pnpm install` 并**同步提交 `pnpm-lock.yaml`**，否则 CI 的 `pnpm install --frozen-lockfile` 必挂。
  - 若曾误用 `npm` 导致 `pnpm install` 报 `EPERM`，先彻底删除 `node_modules` 再重跑 `pnpm install`。
- **关键目录**：
  - `src/views/`（页面视图：落地页、认证、用户工作台、管理端 `/admin/*`、模型广场 `/model-plaza`、画室 `/studio`、榜单 `/arena`、新手引导 `/guide`、按 Key 查用量 `/key-usage`）
  - `src/components/`（共享组件；品牌字标 `brand/GabuguWordmark.vue`；动效组件 `motion/`）
  - `src/i18n/`（多语言词条，改文案后必跑 `pnpm check:i18n`）
  - `src/style.css` 与 `src/styles/brand.css`（设计令牌、深浅主题变量与品牌补充类）

---

## 二、当前线上视觉与品牌规范

### 1. 双舞台主题体系
- **品牌黑色舞台（固定深色黑银蓝）**：
  - **首页（`/home` 八屏整页）**：实时三维地球（Three.js WebGL 2，随滚轮/键盘一次翻一屏走完昼夜日出；GSAP 驱动入场动效）+ 左侧信息栏。在无 WebGL 2、软件渲染、开启“减少动态效果”或省流量模式下，自动降级为轨道日出静态生成图与常规滚动，严禁出现裁字或按钮被上移渐变遮挡。
  - **认证页（`/login`、`/register`、找回密码、邮箱验证）**：`AuthLayout.vue` 统一委托 `LoginLayout.vue`，桌面端采用左侧 `sticky` 折光黑曜石玻璃丝带主视觉（`src/assets/brand/auth-backdrop.webp`，旧 `public/brand/horizon.webp` 保留勿删以兼容缓存）+ 右侧表单双栏舞台，移动端收为单栏。
- **工作台与业务页（深浅双主题切换）**：
  - 工作台、模型广场、用量查询、画室、榜单、法律页、404 默认跟随系统主题并记忆用户偏好。
  - **浅色主题（暖米色体系）**：画布 `#F2E8DB`（`gray-50`）、卡片/输入/浮层 `#FBF7F0`（`bg-white`）、抬升/表头 `#E8DBC8`（`gray-100`）、边框 `#DACBB4`/`#C6B59B`（`gray-200`/`300`）、正文 `#221D17`、主色黛蓝 `#2D5890`（`primary-*`）、点缀赤陶 `#A5532F`（优惠胶囊等）。状态色压在 100 档浅底上的小字须用 `700` 档以保证对比度 ≥ 5.8。
  - **深色主题**：Starlight Indigo（`#0B0F1A`）/ Slate 层级，`.dark` 自动还原 Tailwind 标准灰阶与状态色变量。

### 2. 样式与组件防踩坑规则
- **禁止硬编码灰阶色值**：新代码统一使用 `bg-white`、`text-gray-*`、`border-gray-*`、`primary-*`；在 `<style>` 或图表配置中写 `rgb(var(--gray-200))` 等 CSS 变量，不要写死 `#e5e7eb`。
- **深底局部容器与二维码保护**：浅色页内的固定深底块（`.g-code`、`bg-gray-900` 等）会自动还原浅灰字可读性；支付二维码容器必须保留 `payment-qr-surface` 类（强制纯白静区，勿删）。
- **Logo 与品牌字标**：浅底用 `public/brand/gabugu-logo-light.png`（深蓝），深底用 `public/brand/gabugu-logo-dark.png`（银蓝）；字标统一复用矢量组件 `src/components/brand/GabuguWordmark.vue`（自定义站名场景保留纯文本原文）；彻底杜绝公开页残留 `Sub2API` 默认品牌词或上游 GitHub 外链。
- **工作台版式**：204px 精简侧栏 + Soft Elevation 无框软浮层 + 数字统一 `tabular-nums`；正文自带 `<h1>` 的路由（Dashboard / Usage / Guide / ChannelStatus / AdminChannelMonitor）桌面端顶栏不重复渲染双页头；公告弹窗与通知中心统一采用顶部 3px 细强调线与单层列表/详情切换排版。
- **页面命名口径**：登录后的工作台入口（`/dashboard`）对外中文统一称**「主页」**（不再称「仪表盘」）；公开落地页（`/home`）称「首页」。
- **字号与无障碍**：全站使用无衬线黑体（禁用宋体等衬线体）；移动端输入框字号至少 `16px`（防止手机浏览器聚焦自动放大），带货币符号或图标前缀的输入框预留左右间距；认证页保持中文独立字阶（标题 36/32px，输入与主按钮 17px，辅助提示 15px）。

### 3. 字体子集与第三方许可
- **改中文文案后更新字体子集**（在 `platform/frontend` 下执行）：
  - 改了 Landing 首页中文：`node scripts/update-landing-font.mjs`（自动生成带内容哈希的新 woff2 并改写 `landing.css`，**不要删除旧子集文件**以免线上旧缓存 404）。
  - 改了工作台中文：`node scripts/update-workspace-font.mjs`。
- **许可红线**：Three.js（MIT）；NASA 地球底图与 Solar System Scope 云图（页尾必须保留 CC BY 4.0 作者、来源、许可链接及“有改动”署名）；GSAP（商业网站标准免费许可，构建产物保留版权声明）；React Bits 动效组件（`src/components/motion/REACT_BITS_LICENSE.md`，仅限本应用内部使用，不单独分发）。`xlsx` 保持独立按需分包 `vendor-xlsx`，Three/GSAP 保持首页按需分块。

---

## 三、本地验证与 Playwright 验收流程

### 1. 代码门禁命令（在 `platform/frontend` 执行）
```bash
pnpm install --frozen-lockfile
pnpm check:i18n
pnpm typecheck
pnpm lint:check
pnpm test:run
pnpm build
```
- **注意构建产物写入位置**：默认 `pnpm build` 会把产物写进 `platform/backend/internal/web/dist`（该目录除 `.keep` 占位符外已被 `.gitignore` 忽略，不随源码提交）；多个 Agent 并行验证时须隔离输出或指定单人构建。

### 2. 本地 Playwright 页面实测（必做）
- 凡涉及页面视觉或交互改动，必须使用本地 Playwright（禁用 Computer Use）挂载构建产物或本地服务，实测桌面（如 `1440×900`）与手机（`390×844`）视口，覆盖浅色与深色主题。
- 逐张看图确认：无运行报错（Console 0 error）、无全页横向溢出、文案口径符合 `¥1 = $1` 约定；记录测试 URL、视口尺寸与截图保存路径。
- 登录态测试严禁把真实密码写入命令行参数、环境变量转储或 Playwright snapshot。

---

## 四、按需深读参考

- 首页八屏实现细节：`platform/frontend/src/components/landing/README.md`
- 品牌视觉规范概览：[BRAND_FRONTEND.md](../platform/docs/BRAND_FRONTEND.md)
