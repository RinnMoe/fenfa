<p align="center">
  <strong>RinnMoe Fenfa</strong> &nbsp;·&nbsp; 自托管应用分发平台
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/openprx/fenfa"><img src="https://img.shields.io/badge/fork-openprx%2Ffenfa-8b5cf6" alt="Fork of openprx/fenfa"></a>
</p>

<p align="center">
  <a href="README.md">English</a>
</p>

---

RinnMoe Fenfa 是基于 [openprx/fenfa](https://github.com/openprx/fenfa) 独立维护的 Fork，用于自托管软件版本管理、发布和分发。保留上游 Product -> Variant -> Release 数据模型，并新增强制更新、产品公告、面向软件客户端的公开 API 和反馈邮件通知。

## 特性

- **多平台** — iOS (IPA)、Android (APK)、macOS、Windows、Linux
- **智能上传** — 自动从安装包解析 Bundle ID、版本号、图标
- **iOS UDID 绑定** — Ad-Hoc 分发的设备注册流程
- **产品页面** — 公开下载页，带二维码和智能平台检测
- **管理后台** — 产品、变体、版本、公告、设备、系统设置一站式管理
- **强制更新** — 可在后台或 API 中将某个版本标记为强制更新
- **产品公告** — 创建、定时、启用和停用产品级公告
- **公开客户端 API** — 面向软件客户端的 latest release 与 announcements 接口
- **S3/R2 存储** — Cloudflare R2、AWS S3 及其他 S3 兼容对象存储
- **反馈邮件提醒** — 可在后台配置 SMTP，在收到用户反馈时发送提醒
- **单文件部署** — Go 二进制内嵌前端
- **SQLite** — 零外部依赖，数据存储在单个文件
- **多语言** — 中英文界面

## 快速开始

### Docker

```bash
docker pull ghcr.io/rinnmoe/fenfa:v0.4.0
docker run -d --name fenfa -p 127.0.0.1:8000:8000 \
  -e FENFA_ADMIN_TOKEN=你的管理密钥 \
  -e FENFA_UPLOAD_TOKEN=你的上传密钥 \
  -e FENFA_PRIMARY_DOMAIN=https://dist.example.com \
  -v ./data:/data \
  -v ./uploads:/app/uploads \
  ghcr.io/rinnmoe/fenfa:v0.4.0
```

访问 `http://localhost:8000/admin`。

1Panel 生产部署请参考 [docs/DEPLOY_1PANEL.md](docs/DEPLOY_1PANEL.md)。

### 从源码构建

需要：Go 1.25+，Node.js 20+

```bash
make build   # 构建前端 + 后端
make run     # 启动服务
```

或手动：

```bash
cd web/front && npm ci && npm run build && cd ../..
cd web/admin && npm ci && npm run build && cd ../..
go build -o fenfa ./cmd/server
./fenfa
```

## 配置

完整配置参考 [docs/config.example.json](docs/config.example.json)。

| 环境变量 | 说明 |
|----------|------|
| `FENFA_PORT` | HTTP 端口 |
| `FENFA_DATA_DIR` | 数据库目录 |
| `FENFA_PRIMARY_DOMAIN` | 主域名 |
| `FENFA_ADMIN_PATH` | 管理后台路径后缀（默认 `admin`） |
| `FENFA_TRUSTED_PROXIES` | 允许提供客户端 IP 请求头的代理 IP/CIDR，逗号分隔；留空表示禁用转发 IP 请求头 |
| `FENFA_ADMIN_TOKEN` | 管理员 Token |
| `FENFA_UPLOAD_TOKEN` | 上传 Token |

如果部署在 Nginx 或其他反向代理之后，请将 `FENFA_TRUSTED_PROXIES` 设置为代理的实际 IP 或网段，不要使用 `0.0.0.0/0` 或 `::/0`。

邮件提醒配置位于后台「系统设置 > 邮件通知」，支持 STARTTLS、SSL/TLS 和无加密 SMTP。密码不会回显；保存后可使用测试邮件按钮验证配置。

### 存储

文件默认存储在本地。可在后台 系统设置 > 存储 中配置 S3 兼容对象存储（Cloudflare R2、AWS S3、MinIO）。使用 Cloudflare R2 时，S3 endpoint 应填写账号级 endpoint，不要追加 Bucket 名称。

## 公开客户端 API

以下接口为只读，不需要 Admin 或 Upload Token。

| 方法 | 接口 | 说明 |
|------|------|------|
| GET | `/api/v1/products/:slug/latest?variant=<variantID>` | 获取指定产品变体的最新版本 |
| GET | `/api/v1/products/:slug/announcements` | 获取当前生效的公告 |

## 架构

```
请求 → Gin 路由 → 认证中间件 → Handler → GORM → SQLite
```

```
cmd/server/          程序入口
internal/server/     HTTP 处理器和路由
internal/email/      SMTP 发送与反馈通知
internal/store/      数据模型 (GORM + SQLite)
internal/web/        内嵌前端资源 (go:embed)
web/front/           公开下载页 (Vue 3 + Vite)
web/admin/           管理后台 (Vue 3 + Vite)
```

## Upstream 与维护

- 初始上游基线：`e0fbb666d779274329091677e4d6218b15c6d81c`
- 本仓库为独立长期维护版本，不以上游 PR 为目标。
- `upstream` remote 指向 [openprx/fenfa](https://github.com/openprx/fenfa)，仅用于选择性吸收安全修复和 bugfix。

## 文档

- [架构](docs/architecture.md)
- [API 规范](docs/api-spec.md)
- [自定义功能说明](docs/customization.md)
- [1Panel 部署](docs/DEPLOY_1PANEL.md)
- [备份与升级](docs/BACKUP.md)
- [已知问题](docs/KNOWN_ISSUES.md)

## 开源协议

[MIT](LICENSE)。上游版权归 Fenfa 原作者所有。
