# Changelog

## Unreleased

### Added
- 新增反馈邮件提醒：后台「系统设置 > 邮件通知」支持 SMTP 配置（STARTTLS、SSL/TLS、无加密）、测试邮件，并在收到新反馈时异步发送通知。
- 新增 `FENFA_TRUSTED_PROXIES` 配置，仅允许受信任反向代理设置客户端 IP 请求头。

### Changed
- 反馈限流区分成功提交（5 次/10 分钟）和无效/超大请求（30 次/10 分钟），均按客户端 IP 计算。

### Security & Privacy
- SMTP 密码不会从管理 API 返回，仅通过 `smtp_password_set` 标记是否已设置。

## v0.4.0 - 2026-08-09

### Added
- 新增产品下载页匿名“用户反馈”入口，支持问题、建议和其他三类反馈。
- 新增 `POST /api/v1/feedback` 公开提交 API，保存产品、平台和版本快照。
- 新增管理后台反馈收件箱，支持状态、分类和产品筛选、分页、处理/重新打开及删除。

### Security & Privacy
- 增加请求体限制、联系方式和内容长度校验、蜜罐字段及单进程 IP 限流。
- 不保存提交者的原始 IP、User-Agent 或页面 URL。

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
