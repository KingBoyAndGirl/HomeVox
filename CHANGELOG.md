# Changelog

## Unreleased

### Changed

- 完成 Issue #5 的 2D 墙体校正交互：2D SVG 与 3D room-bounds 预览改为独立分区，避免覆盖和事件争抢。
- 高分辨率图纸下，墙体、端点及透明命中层按 CSS 像素自适应，支持墙体选择、高亮、底图显隐、端点拖拽和 Undo/Redo。
- 加强解析响应运行时校验与请求竞态保护，旧文件请求不会覆盖新文件状态；门窗标签仅显示真实字段。
- 升级 Vitest 至 4.1.10，与 Vite 8 共用单一工具链。
- Go 服务改为在唯一的 `0.0.0.0:18088` 入口同时提供 `/api/*` 和 `frontend/dist`，支持前端 SPA fallback，并对未知 API 与缺失静态资源保持 404。
- 服务启动时校验 `frontend/dist/index.html`，避免后端存活但前端构建缺失的半可用状态。
- 在真实懒猫浏览器中验证同源 `/api/health`、AI 缺配置的 503 失败关闭，以及 fixture 下的墙体选择、共享端点拖拽、Undo/Redo、底图显隐和 2D/3D 分区；真实 AI 正向解析仍待提供运行时 AI 配置后验收。
