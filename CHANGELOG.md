# Changelog

## v0.3.3 - 2026-08-08

### Changed
- 将原始事件流水精简为平台版本粒度的访问量和下载量统计。
- 管理后台统计页只展示访问量、下载量两个总数。
- 公开 `POST /events` 仅用于增加平台版本访问量，不再保存事件明细。

### Removed
- 移除 `events` 表、事件明细页面、事件查询接口和事件导出接口。
- 移除前端点击事件上报和完整 UA 解析上报。

### Migration
- 启动时自动把旧 `visit` 事件按 `variant_id` 回填到 `variants.visit_count`，随后删除 `events` 表。
- 升级前请先备份数据库。
