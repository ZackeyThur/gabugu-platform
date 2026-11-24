# GABUGU 品牌字标

用户于 2026-11-13 批准第一版「弧线切角」字标及 v2 深浅色组合；2026-11-14 要求应用至网站所有 Logo＋品牌名展示。

`wordmarkPaths.ts` 从批准的第一版 PNG 主字标提取四个字形轮廓，以 Potrace 生成贝塞尔路径。两个 G 和两个 U 分别复用一个轮廓，避免生成样张中的细微重复差异。`GabuguWordmark.vue` 统一排列为六字母，不依赖用户安装的字体。

配色：浅色背景使用深蓝图标 `public/brand/gabugu-logo-light.png` 与 #101723 字标；深色背景使用银蓝图标 `public/brand/gabugu-logo-dark.png` 与白字。固定深色的首页和认证页面显式传入 `theme="dark"`，其他页面跟随 html.dark。

`name` 仅在 GABUGU / Gabugu AI 时使用批准字形；自定义站点名称继续显示原文。Logo 沿用原始图标及现有自定义来源处理。

批准样张与深浅底组合见本仓库 `design/gabugu-wordmark/`。这是专用品牌字标矢量组件，不是通用 TTF/OTF 字库；正文无需替换为此组件。
