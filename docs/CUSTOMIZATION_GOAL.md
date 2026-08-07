你现在需要基于开源项目 Fenfa 进行一次“小范围、低侵入式二次开发”。

上游仓库：

https://github.com/openprx/fenfa

项目最终用途：

这是一个自托管的软件版本管理、发布和分发系统，用于管理我的多款 Android / Windows / 其他软件。

安装包仍然使用 Fenfa 现有 Storage abstraction，并最终使用 Cloudflare R2/S3-compatible storage。

本次开发的首要原则：

==================================================
一、最高优先级原则
==================================================

1. 基于 Fenfa 现有架构进行扩展。

2. 不进行无必要重构。

3. 不重新设计 Product / Variant / Release 数据模型。

4. 不修改现有 Storage abstraction。

5. 不重写现有上传、下载、R2/S3、认证、统计等功能。

6. 不更换 SQLite。

7. 不引入 PostgreSQL、Redis、消息队列或其他新的基础设施。

8. 不增加新的复杂“发布渠道”“Deployment”“Rollout”“Notification”等抽象层。

9. 优先复用项目已经存在的：
   - model
   - repository/database
   - handler
   - router
   - admin API
   - frontend component
   - form
   - table
   - validation
   - i18n
   - migration
   - error handling

10. 修改范围必须尽可能小。

11. 如果发现我下面描述的某项需求已经由 Fenfa 原生支持：
    不重复实现。
    直接复用现有实现。

12. 不因为个人代码风格偏好而重构无关代码。

13. 不在第一阶段修改项目名称、Logo、版权信息或品牌。

14. 不修改 R2 / S3 的工作方式。

15. 不修改现有 /d/:releaseID 下载机制。

16. 所有数据库结构变化必须兼容已有数据库升级。
    不允许要求用户删除 fenfa.db 重新初始化。

17. 保持上游项目未来仍较容易 merge/rebase。

==================================================
二、本次唯一允许增加的功能
==================================================

本次只增加以下内容：

A. Release 增加强制更新字段

在现有 Release 模型中增加：

force_update

类型：

boolean

默认值：

false

用途：

客户端查询最新版本时，知道当前 Release 是否被标记为强制更新版本。

不要增加额外 Release Policy 模型。

暂时不要增加：

minimum_version
minimum_build
rollout_percentage
灰度发布
设备分组
地区过滤
用户过滤
复杂渠道策略

只增加 force_update。


B. 新增 Announcement 公告

新增一个非常简单的 Announcement 数据模型。

最低需要字段：

id

product_id

title

content

enabled

start_at

end_at

created_at

updated_at

要求：

product_id 必须关联 Fenfa 现有 Product。

Announcement 表示某一个 Product 对应的软件公告。

不设计：

公告用户组
公告设备组
地区过滤
版本过滤
公告灰度
服务器端“已读状态”
复杂优先级系统
推送系统

客户端是否弹窗、以什么 UI 展示，由客户端自己决定。

start_at 和 end_at 可为空。

enabled=false 的公告不应出现在公开客户端 API 中。

如果 start_at 不为空：
只有当前时间 >= start_at 时才返回。

如果 end_at 不为空：
只有当前时间 <= end_at 时才返回。


C. Admin 管理后台增加公告管理能力

在现有 Admin UI 中，以最符合项目当前 UI/路由结构的方式增加 Announcement 管理。

需要支持：

查看某个 Product 的公告列表

创建公告

编辑公告

删除公告

启用 / 停用公告

编辑：

title
content
enabled
start_at
end_at

不要重新设计整个 Product 页面。

不要重新设计导航系统。

先研究现有 Admin UI 的组织方式，再选择侵入最小的位置添加入口。


D. Release Admin UI 增加 force_update

在现有 Release 创建 / 编辑流程中增加：

“强制更新”

boolean checkbox / switch

要求：

默认关闭。

必须直接使用现有 Release API / DTO / Model 体系扩展。

不要另建一个“Update Policy”页面。


E. 增加客户端公开 API

增加两个面向软件客户端的公开读取 API。

注意：

这些接口不是 Admin API。

客户端不应该携带 FENFA_ADMIN_TOKEN。

接口应该是 read-only。

具体 URL 在完成现有 router 调研之后决定，但优先目标形式为：

GET /api/v1/products/:slug/latest

GET /api/v1/products/:slug/announcements


latest API：

根据 Product slug 查询对应产品的最新 Release。

由于 Fenfa 一个 Product 可以有多个 Variant，因此必须先研究 Fenfa 当前 Variant / Release 获取逻辑。

不得自己凭空规定“最新版本算法”。

优先复用 Fenfa 当前产品下载页选择 latest release 的逻辑。

如果必须由客户端指定 Variant，应使用项目现有 Variant identifier / ID / platform 机制，而不是创造新的 channel 系统。

返回内容至少包含：

product

variant

release id

version

build number

changelog

force_update

download URL

published/created time

具体 JSON 字段命名应首先遵循 Fenfa 当前 API DTO 风格。

不要为了与上面的字段名完全一致而破坏项目已有 API 命名规范。


announcements API：

GET /api/v1/products/:slug/announcements

只返回：

属于该 Product

enabled = true

并且当前处于 start_at / end_at 生效时间范围内

的公告。

返回字段至少：

id
title
content
start_at
end_at

不要返回没有必要的内部字段。


==================================================
三、第一阶段：获取项目
==================================================

首先检查当前 workspace。

如果当前目录已经存在 Fenfa 仓库：

不要重复 clone。

先执行：

git status
git remote -v
git branch --show-current
git log -5 --oneline

确认当前代码来源和状态。

如果当前 workspace 尚不存在 Fenfa：

执行：

git clone https://github.com/openprx/fenfa.git

进入：

fenfa

然后记录：

git remote -v
git status
git branch --show-current
git rev-parse HEAD

不要修改 upstream remote。


创建开发分支：

feat/minimal-release-announcement

即：

git checkout -b feat/minimal-release-announcement

如果分支已经存在，则直接切换。


在开始修改之前：

记录当前 upstream commit hash。

后续所有二开必须基于这个 commit。


==================================================
四、第二阶段：基线验证
==================================================

在任何代码修改之前，必须先验证原版 Fenfa 可以正常运行。

首先阅读：

README
go.mod
package.json
Dockerfile
docker-compose / compose 文件
Makefile（如果存在）
项目开发文档

然后识别：

Go 版本
Node.js / package manager
前端目录
后端入口
数据库初始化位置
GORM model 位置
migration / AutoMigrate 位置
router 定义
Admin handler
Release handler
Product handler
Variant handler
前端 Admin 入口
API client
Release 页面
Product 页面
i18n 文件
测试目录

不要依赖 README 的目录结构猜测。
必须实际查看代码。


然后完成：

1. Go dependency install

2. frontend dependency install

3. backend build

4. frontend build

5. 项目测试

6. 如果官方提供 Docker build：
   完成一次 Docker build

优先使用项目官方提供的命令。

例如如果存在：

make build
make test

则优先使用官方命令。

不要自己创造完全不同的 build workflow。


记录所有基线测试结果。

如果 upstream 原版就存在失败测试：

明确记录：

“baseline existing failure”

不要把上游已有失败误认为本次修改造成。


==================================================
五、第三阶段：代码架构调研
==================================================

在修改任何业务代码前，先完成代码调研。

重点回答以下问题：

1. Product model 在哪里？

2. Variant model 在哪里？

3. Release model 在哪里？

4. Release 是如何关联 Variant 的？

5. Variant 是如何关联 Product 的？

6. Release 的 version/build/changelog/file 信息字段分别叫什么？

7. Release 的创建、修改和删除 Handler 在哪里？

8. Release Admin API 在哪里？

9. Product public page 如何获取 latest release？

10. 当前项目如何决定一个 Variant 的最新 Release？

11. Download URL 是如何生成的？

12. /d/:releaseID 是在哪里处理的？

13. GORM AutoMigrate / migration 是如何工作的？

14. Admin UI 的 API 请求层在哪里？

15. Admin Release 编辑 UI 在哪里？

16. Product 页面结构在哪里？

17. 项目是否使用 TypeScript？

18. 项目 i18n 怎么组织？

19. 后端测试模式是什么？

20. 前端测试模式是什么？

21. API response 是否有统一 envelope？

例如：

{
  "data": ...
}

还是直接返回对象？

22. API error 是否有统一格式？

23. 时间字段使用什么格式？

24. 删除操作是 hard delete 还是 soft delete？

25. 项目现有 model 是否使用 gorm.Model？

完成调研后：

先输出一份：

IMPLEMENTATION_PLAN.md

或者在 Agent 工作日志中形成等价计划。

计划必须包含：

需要修改的实际文件

需要新增的实际文件

数据库迁移方式

后端 API 变化

前端 UI 变化

测试方案

禁止在这一阶段开始大规模编码。


==================================================
六、第四阶段：Release.force_update
==================================================

完成架构调研后，首先实现最简单的 Release 修改。

在现有 Release model 增加：

ForceUpdate bool

GORM / JSON 命名遵循项目现有风格。

数据库字段：

force_update

要求默认：

false

确认：

旧数据库升级后，
历史 Release 自动视为 false。

不得破坏已有 Release。


然后修改必要：

DTO
request
response
handler
serializer

前端：

在 Release 创建/编辑界面增加“强制更新”。

UI 风格必须复用现有 component。

如果项目已有：

Switch
Checkbox
FormItem

优先复用。


同时增加：

中文 i18n
英文 i18n

如果 Fenfa 当前支持更多语言：

不要为了这个功能一次性翻译所有语言。

研究当前 fallback 策略。

至少保证已有默认语言和中文/英文不出现 raw key。


测试：

创建 Release 默认 force_update=false

创建 Release force_update=true

编辑 false -> true

编辑 true -> false

读取 Release 能正确返回值

旧 Release 不受影响。


完成这一阶段后：

运行 backend test/build

运行 frontend build

不要等到最后一起测。


==================================================
七、第五阶段：Announcement 后端
==================================================

新增 Announcement model。

必须保持简单。

建议关系：

Product
  has many
Announcement

字段：

ID
ProductID
Title
Content
Enabled
StartAt
EndAt
CreatedAt
UpdatedAt

实际 Go 类型按照项目现有 Model 风格实现。

Title：

非空。

Content：

非空。

ProductID：

必须对应存在的 Product。

Enabled：

default true。

StartAt：

nullable。

EndAt：

nullable。


验证：

如果 StartAt 和 EndAt 都存在：

StartAt 不应晚于 EndAt。

如果项目当前没有复杂 validation framework：
使用现有 handler validation 风格即可。

不要引入新的 validation dependency。


删除 Product 时 Announcement 如何处理：

先研究现有 Product -> Variant -> Release 删除方式。

按照项目现有关系处理原则实现。

不要自己擅自设计新的级联策略。


数据库升级：

必须加入项目现有 migration / AutoMigrate。

不能要求清空 SQLite。


==================================================
八、第六阶段：Announcement Admin API
==================================================

按照 Fenfa 当前 Admin API 风格实现 Announcement CRUD。

必须使用现有 Admin auth middleware。

不要创建新的认证体系。

需要支持：

list announcements by product

get announcement

create announcement

update announcement

delete announcement


API URL 必须优先遵循 Fenfa 当前 API conventions。

不要因为 Prompt 示例而强行制造不一致的 URL。


必须校验：

Product 是否存在

Announcement 是否属于指定 Product

title/content 是否为空

时间范围是否合法


权限：

Announcement 所有写接口必须是 Admin 权限。

Upload token 不应获得 Announcement 管理权限，除非 Fenfa 当前 token scope 机制本来就这样设计并且无法简单区分。

优先保持现有认证体系语义。


==================================================
九、第七阶段：Announcement Admin UI
==================================================

研究 Product 管理页面结构。

在侵入最小的位置增加：

Announcements / 公告

入口。

不要重新布局整个后台。


公告列表至少展示：

Title
Enabled
Start At
End At
Created At
Actions


Actions：

Edit
Enable/Disable
Delete


创建/编辑表单：

Title

Content

Enabled

Start At

End At


Content 第一版使用：

textarea

不要引入富文本编辑器。

不要引入 Markdown 编辑器。

不要增加图片上传。

不要增加附件。

不要增加公告预览系统。


删除动作遵循项目现有确认 Modal。


UI 必须支持：

中文
英文

并保持现有 UI Design System。


==================================================
十、第八阶段：客户端 latest API
==================================================

这是本次最需要谨慎的部分。

不要自己重新写“查询最新版”的业务逻辑。

首先定位：

Fenfa Product public page
或者现有 Release API

当前是如何获得：

某 Variant 的 latest Release

的。

提取 / 复用现有逻辑。

避免产生：

Public Page 认为 A 是最新版本

Client API 认为 B 是最新版本

这种双套逻辑。


接口目标：

GET /api/v1/products/:slug/latest


但是：

如果一个 Product 同时存在：

Android
iOS
Windows

显然不能简单返回一个 Release。

因此：

必须根据 Fenfa 当前 Variant 机制设计。

优先方案：

客户端明确提供已有 Variant identifier / variant id。

例如：

GET .../latest?variant=...

但参数名称和值必须服从 Fenfa 已有模型。

不要新增：

stable
beta
channel

除非这些概念 Fenfa 原本就存在。


找不到：

Product
Variant
Release

时：

使用现有 API 错误格式。


response：

遵循现有 DTO 风格。

至少能获得：

product metadata

variant metadata

release id

version

build

changelog

force_update

download URL

release time


download URL：

复用现有 Fenfa 下载 URL 构造逻辑。

不要自行拼接 R2 URL。

不要绕过 /d/:releaseID。

本阶段不修改 Download handler。


==================================================
十一、第九阶段：公开 Announcement API
==================================================

实现：

GET /api/v1/products/:slug/announcements


该接口公开、read-only。

不需要 Admin Token。


查询条件：

product_id == 当前 Product

enabled == true

并且：

(start_at IS NULL OR start_at <= now)

AND

(end_at IS NULL OR end_at >= now)


排序：

优先使用：

created_at DESC

如果项目现有公告/事件排序有统一习惯，则遵循项目习惯。


不要暴露：

内部关联对象
没有必要的数据库字段
管理员信息


404 / empty：

Product 不存在：
404。

Product 存在但没有公告：

HTTP 200

返回空数组

不要返回 404。


==================================================
十二、API 安全要求
==================================================

公开 API 仅允许读取。

不得暴露：

Admin token
Upload token
S3 secret
R2 secret
系统设置 secret
Apple credentials
内部文件路径


所有数据库查询使用 GORM 正常参数绑定。

禁止字符串拼接 SQL。


检查：

slug
variant
ID

输入。


Announcement content 最终会由前端展示。

后端不能因为保存文本而引入 HTML 注入逻辑。

第一版 Announcement content 视为 plain text。


Admin UI 输出公告时：

必须使用 Vue 默认安全文本绑定。

禁止直接使用：

v-html

除非项目本身明确有可信 HTML sanitization，
本次需求并不需要 HTML。


==================================================
十三、测试要求
==================================================

至少覆盖以下后端测试：

Release：

1.
旧 Release force_update 默认 false

2.
新 Release force_update=false

3.
新 Release force_update=true

4.
更新 Release force_update


Announcement：

5.
创建公告

6.
修改公告

7.
删除公告

8.
disabled 公告不出现在 public API

9.
未来 start_at 公告不返回

10.
已经过 end_at 公告不返回

11.
当前有效公告正常返回

12.
不存在 Product 返回 404

13.
存在 Product 无公告返回 []


Latest API：

14.
合法 Product + Variant 返回 latest release

15.
force_update 正确返回

16.
download URL 正确

17.
不存在 Product

18.
不存在 Variant

19.
无 Release 情况


Auth：

20.
Announcement Admin API 无 token 时拒绝

21.
有效 Admin token 可以 CRUD

22.
公开 latest API 不需要 Admin token

23.
公开 announcement API 不需要 Admin token


前端：

至少保证：

npm/pnpm/yarn build 成功

TypeScript check 成功（如果项目使用）

lint 成功（如果项目已有 lint）

不要为了新增测试引入大型测试 framework。


==================================================
十四、兼容性测试
==================================================

必须确认本次修改没有破坏：

Product 创建

Variant 创建

Release 上传

APK metadata parsing

IPA 流程

文件下载

/d/:releaseID

Local storage

S3/R2 storage abstraction

Admin 登录

Upload token

Release history

Product public page

QR Code

Event tracking


不要求所有功能进行真实外部服务集成测试，
但至少需要确认相关代码路径和 build 没有被本次修改破坏。


==================================================
十五、数据库升级测试
==================================================

这个测试很重要。

创建一个“修改前”的 Fenfa SQLite 数据库。

然后使用修改后的程序启动。

确认：

migration 自动完成。

旧数据仍然存在。

旧 Release：

force_update = false。

Announcement table 被正常创建。

不需要人工运行 SQL。

不需要删除 fenfa.db。


然后重启服务一次。

确认 migration 是幂等的。


==================================================
十六、Docker 构建
==================================================

代码功能完成以后：

使用项目原有 Dockerfile 构建。

不要重新设计 Docker 架构。


执行类似：

docker build -t fenfa-custom:dev .

然后启动测试。


端口：

开发环境可以正常映射。

最终生产 Compose 推荐：

127.0.0.1:8000:8000

因为最终会通过 1Panel OpenResty 反向代理。


不要在本次开发中增加 Caddy/Nginx。

生产环境使用 1Panel。


==================================================
十七、Cloudflare R2
==================================================

不要重新实现 R2。

Fenfa 已经支持 S3-compatible storage。

只需要验证：

现有 S3/R2 storage configuration 仍然正常。


本次代码中禁止新增：

Cloudflare SDK

R2 SDK

Workers

KV

D1

新的 object storage abstraction。


客户端 API 返回：

Fenfa 原有 download URL。

不要直接暴露 storage credentials。


==================================================
十八、Git 提交要求
==================================================

不要把整个项目最后一次性提交。

按逻辑拆分 commit。

建议：

1.
chore: establish fenfa customization baseline

仅在确实需要 baseline 文件时提交。


2.
feat(release): add force update flag


3.
feat(announcement): add announcement model and admin api


4.
feat(admin): add announcement management ui


5.
feat(api): add public latest release endpoint


6.
feat(api): add public announcements endpoint


7.
test: cover release update and announcements


不要提交：

node_modules

.env

数据库

R2 credentials

Admin Token

Upload Token

临时 build 文件

IDE 私有配置


==================================================
十九、文档
==================================================

开发完成后新增：

docs/customization.md

或者如果项目 docs 结构不同，则使用合适位置。


文档说明：

当前 fork 相比 upstream 增加了什么：

1. Release.force_update

2. Product Announcement

3. Announcement Admin CRUD

4. Public latest API

5. Public announcement API


另外记录：

upstream repository

upstream baseline commit hash

custom branch

数据库 migration 信息

API 示例

Docker build 方法


不要重写原项目全部文档。


==================================================
二十、最终交付报告
==================================================

全部完成后，请输出一份详细总结。

格式：

# Fenfa 二开完成报告

## 1. Upstream baseline

repository:
commit:
branch:

## 2. 架构确认

Backend:
Frontend:
Database:
Storage:
Router:
Migration:

## 3. 修改文件

逐个列出修改文件，并说明原因。

## 4. 新增文件

逐个列出。

## 5. 数据库变化

Release:
Announcement:

## 6. API

列出：

method
path
auth
request
response

## 7. Admin UI

说明增加了哪些页面/字段。

## 8. 测试

列出所有：

pass
fail
baseline fail

## 9. Docker

说明 Docker build 是否成功。

## 10. 数据库升级

说明旧数据库 migration 是否验证成功。

## 11. 尚未完成

必须明确列出所有未完成事项。

禁止假装完成。

## 12. 风险

列出目前仍存在的风险。

## 13. Production deployment readiness

最后明确给出：

READY

或者：

NOT READY

如果 NOT READY：

必须说明阻塞原因。


==================================================
二十一、非常重要的执行规则
==================================================

不要一开始直接修改代码。

严格执行顺序：

Step 1
检查 workspace / clone

Step 2
建立开发分支

Step 3
记录 upstream commit

Step 4
运行原版项目

Step 5
完成 baseline test/build

Step 6
分析项目真实代码结构

Step 7
输出 implementation plan

Step 8
实现 force_update

Step 9
测试

Step 10
实现 Announcement backend

Step 11
测试

Step 12
实现 Announcement Admin UI

Step 13
测试/build

Step 14
实现 public latest API

Step 15
测试

Step 16
实现 public announcement API

Step 17
测试

Step 18
执行数据库升级测试

Step 19
执行完整回归

Step 20
Docker build

Step 21
文档

Step 22
最终报告


如果实际代码结构与我在 Prompt 中假设的不一致：

以实际代码为准。

不要为了满足 Prompt 中的文件名猜测而创建重复架构。


如果发现实现某个功能需要“大范围重构”：

停止扩大修改范围。

先说明：

为什么现有架构无法直接实现

预计需要修改哪些模块

然后选择侵入更小的实现。


核心目标始终是：

“在 Fenfa 原有 Product → Variant → Release 架构上，以最少代码增加 force_update、Product 公告，以及两个客户端查询 API。”

而不是：

“重新设计 Fenfa。”