# Sub2API 系统优化深度分析

分析日期：2026-05-30

当前分支：`2ue-main`

分析方式：基于当前仓库静态阅读、路由与文件规模扫描、发布链路检查、部署配置检查。本文不假设已经在浏览器中完整走查所有页面，因此涉及真实点击体验的结论按“代码结构可推断的问题和优化机会”表达。

## 1. 结论摘要

Sub2API 当前已经不是一个单纯的 API 转发项目，而是一个围绕 AI 订阅账号、API Key 分发、计费、风控、支付、运维监控、渠道管理、后台管理和自部署安装的完整网关平台。系统功能面很宽，后端能力也较完整；真正的优化重点不在“补几个小功能”，而在三个方向：

1. 降低使用复杂度

   管理后台入口很多，路由和设置项覆盖面大。对新管理员、普通用户、运维人员来说，系统现在更像“功能全集”，还没有完全整理成“按任务完成工作”的产品体验。需要从信息架构、导航、页面状态、操作流程、错误恢复和新手引导上重构。

2. 降低维护复杂度

   前端存在多个超大 Vue 文件，例如 `frontend/src/views/admin/SettingsView.vue` 接近 1 万行，`CreateAccountModal.vue` 超过 5000 行，`GroupsView.vue` 和 `EditAccountModal.vue` 超过 4000 行。后端也存在 `gateway_service.go`、`openai_gateway_service.go` 等超大服务文件。长期看，这会影响 AI 自动改代码的可靠性，也会增加合并上游的冲突成本。

3. 让 2ue fork 的发布链路和产品策略独立成体系

   当前仓库具备 GitHub Actions 发布、GoReleaser、GHCR 和可选 DockerHub 发布能力。但 release workflow 仍是上游语义，`v*` tag 都会触发发布，部署文档和 Docker Compose 也大量指向 `Wei-Shaw/sub2api` 或 `weishaw/sub2api:latest`。这和你希望 `main` 镜像上游、`2ue-main` 发布 fork 版本的策略存在冲突。必须把发布触发、镜像名、安装脚本、文档、版本号和默认分支策略收束到 `2ue-main`。

最高优先级不是先加新功能，而是先建立“可持续迭代的产品和工程约束”：统一错误反馈、重整后台导航、约束 release tag、拆分巨型文件、建立 AI 改动边界和回归测试清单。

## 2. 系统当前形态

从 README 和代码结构看，当前系统的核心定位是：

```text
用户 / 客户端
  |
  | API Key / OAuth / Web UI
  v
Sub2API 网关层
  |
  | 分组 / 渠道 / 账号调度 / 限流 / 并发 / 风控
  v
上游 AI 服务账号
  |
  | 响应 / token / 图片 / usage
  v
计费 / 使用记录 / 看板 / 运维监控 / 告警
```

实际产品里同时存在三类主要角色：

| 角色 | 主要目标 | 现在的复杂点 |
| --- | --- | --- |
| 普通用户 | 创建 API Key、查看额度和用量、充值或兑换、查看可用渠道 | 入口多，失败原因不够可解释，用量和额度之间的关系需要更直观 |
| 管理员 | 管账号、管分组、管渠道、管用户、管订阅、管风控、管支付、管运维 | 管理菜单密度大，页面和设置项过多，跨模块因果关系不够显性 |
| 维护者 / 发布者 | 同步上游、开发 fork 功能、发布 Docker 镜像和 Release | 现有 workflow 还没有完全按 `2ue-main` 发布策略收紧 |

系统能力已经很强，但“能力强”带来的副作用是：用户需要理解太多内部概念才能稳定使用，包括账户、分组、渠道、模型映射、倍率、额度、订阅、余额、风控、并发、调度、代理、TLS 指纹、错误透传、定时测试、渠道监控、Ops 监控等。

下一阶段的优化目标应该是把这些能力产品化，而不是继续简单堆叠能力。

## 3. 主要证据

本次分析重点阅读和扫描了以下位置：

- 产品说明：`README.md`、`README_CN.md`、`docs/PAYMENT.md`
- fork 工作流：`docs/2ue-ai-fork-workflow.md`
- 前端路由：`frontend/src/router/index.ts`
- 前端布局：`frontend/src/components/layout/AppLayout.vue`、`AppSidebar.vue`、`AppHeader.vue`
- 通用表格：`frontend/src/components/common/DataTable.vue`
- 前端状态与认证：`frontend/src/stores/auth.ts`、`frontend/src/api/client.ts`
- 用户页面：`frontend/src/views/user/DashboardView.vue`、`KeysView.vue`、`UsageView.vue`、`PaymentView.vue`
- 管理页面：`frontend/src/views/admin/SettingsView.vue`、`AccountsView.vue`、`GroupsView.vue`、`RiskControlView.vue`、`ops/OpsDashboard.vue`
- 后端路由：`backend/internal/server/routes/admin.go`、`backend/internal/server/routes/gateway.go`
- 后端核心服务：`backend/internal/service/gateway_service.go`、`openai_gateway_service.go`、`billing_service.go`、`content_moderation.go`
- 后端配置：`backend/internal/config/config.go`
- 部署和发布：`.github/workflows/release.yml`、`.github/workflows/backend-ci.yml`、`.goreleaser.yaml`、`.goreleaser.simple.yaml`、`Dockerfile`、`Dockerfile.goreleaser`、`deploy/docker-compose.yml`、`deploy/docker-deploy.sh`、`deploy/README.md`

扫描结果中值得关注的量化信号：

| 信号 | 观察 |
| --- | --- |
| 前端超大文件 | `SettingsView.vue` 约 9751 行，`CreateAccountModal.vue` 约 5519 行，`GroupsView.vue` 约 4351 行，`EditAccountModal.vue` 约 4241 行 |
| 后端超大文件 | `gateway_service.go` 约 9955 行，`openai_gateway_service.go` 约 6880 行，`setting_service.go` 约 4869 行，`usage_log_repo.go` 约 4582 行 |
| 静默错误 | 前端大量 `console.error`，很多页面没有统一用户可见错误态和重试入口 |
| 客户端 token | `auth_token`、`refresh_token` 等存储在 `localStorage`，便利但扩大 XSS 后果 |
| 发布触发 | `.github/workflows/release.yml` 触发 `v*` tag，和 fork 专属 tag 策略不一致 |
| 文档和镜像名 | README、部署脚本、Docker 文档大量硬编码上游仓库和 `weishaw/sub2api:latest` |

## 4. 使用体验优化

### 4.1 新用户安装与初始化

当前能力：

- 支持 Docker Compose 自动初始化。
- 支持二进制安装和 Web Setup Wizard。
- Docker 方案会自动生成密钥和密码。
- `deploy/.env.example` 对大量配置给了注释。

问题：

1. 安装路径太多，但“推荐路径”不够强。

   README、`deploy/README.md`、Docker 文档、脚本安装、Docker Compose、Setup Wizard 并存。对新用户来说，最关键的问题不是“有没有文档”，而是“我应该选哪条路”。当前文档更像能力列表，而不是决策树。

2. Docker 自动初始化和 Web Setup Wizard 的边界需要更清晰。

   Docker Compose 设置 `AUTO_SETUP=true` 后无需 Setup Wizard；二进制安装使用 Web Setup Wizard。这个区别对第一次部署的人很重要，应该在首页用非常明确的路径图展示。

3. 密钥和密码生成后的长期风险提示还可以更产品化。

   `.env.example` 已经说明 JWT 和 TOTP 加密 key 必须固定。但用户真正容易犯错的点是：重启后登录失效、2FA 失效、忘记保存初始管理员密码、迁移时漏掉 `.env`。建议把这些检查做成安装后健康检查。

建议：

- 做一个“部署方式选择器”文档：

  ```text
  只想最快跑起来 -> Docker Compose local directory
  已有 PostgreSQL/Redis -> Binary install or Docker app-only
  需要可迁移备份 -> docker-compose.local.yml
  需要系统服务 -> install.sh + systemd
  需要 2ue fork 镜像 -> ghcr.io/2ue/sub2api:<version>
  ```

- Web Setup Wizard 增加“部署模式说明”：检测到 Docker 自动初始化时，不展示不适用的引导。
- 启动后首页或管理员面板增加“安装体检”：
  - `JWT_SECRET` 是否固定
  - `TOTP_ENCRYPTION_KEY` 是否固定
  - 数据库连接池是否过大
  - Redis 是否有密码
  - 是否配置 `SERVER_FRONTEND_URL`
  - 是否使用默认管理员邮箱
  - 是否启用 URL allowlist

### 4.2 普通用户工作流

普通用户最常见路径应该是：

```text
注册 / 登录
  -> 查看余额 / 订阅 / 平台额度
  -> 创建 API Key
  -> 复制使用方式
  -> 查看调用是否成功
  -> 出问题时知道该找谁、原因是什么、怎么恢复
```

当前用户侧页面有 Dashboard、Keys、Usage、Redeem、Affiliate、Available Channels、Profile、Subscriptions、Purchase、Orders、Payment、Channel Status、自定义页面等。能力覆盖很完整，但还可以更围绕“第一次成功调用”优化。

问题：

1. Dashboard 代码显示多个接口加载失败时主要 `console.error`，页面缺少局部错误态。

   `frontend/src/views/user/DashboardView.vue` 里的 `loadStats`、`loadCharts`、`loadRecent` 都是 catch 后打印日志。用户看到的可能是空白、旧数据或 loading 消失，但不知道失败原因。

2. API Key 创建后，用户是否能立刻完成一次验证调用，是产品体验关键。

   当前系统有 `UseKeyModal`、用量页、测试组件，但需要把“创建 key -> 复制 base url -> 选择客户端 -> 试调用 -> 查看第一条 usage”做成闭环。

3. 用量、余额、订阅、平台额度之间的关系需要更直观。

   用户最关心的问题是：
   - 我还能用多久？
   - 为什么这次请求失败？
   - 是余额不足、订阅过期、平台额度限制、模型不支持，还是风控拦截？

建议：

- 在用户 Dashboard 增加“状态总览”：
  - API Key 数量
  - 最近一次成功请求时间
  - 最近一次失败原因
  - 当前可用平台
  - 余额/订阅/平台额度的综合状态
- 在 API Key 页面增加“第一次调用向导”：
  - 选择客户端：OpenAI SDK、Claude Code、Codex CLI、Gemini CLI、curl
  - 自动生成 base URL 和 key 示例
  - 提供“复制”和“测试请求”
  - 测试结果链接到对应 usage 或 request log
- 请求失败时，用户可见错误要转成业务语言：
  - `quota_exceeded` -> 额度不足，展示充值/兑换入口
  - `group_not_assigned` -> 未分配可用分组，展示联系管理员
  - `model_not_allowed` -> 当前 Key 不支持该模型，展示可用模型
  - `upstream_unavailable` -> 上游账号暂不可用，展示稍后重试
  - `risk_blocked` -> 风控拦截，展示申诉或联系入口

### 4.3 管理员工作流

管理员最核心路径应该是：

```text
配置系统基础能力
  -> 添加上游账号
  -> 配置分组 / 渠道 / 模型 / 计费
  -> 创建或审核用户
  -> 观察流量和错误
  -> 处理告警和风险
  -> 备份、升级、发布
```

当前后台能力非常多，但入口过于平铺。`frontend/src/router/index.ts` 和 `AppSidebar.vue` 显示管理员区包含 Dashboard、Ops、Users、Groups、Channels、Accounts、Announcements、Proxies、Redeem、Promo、Settings、Risk Control、Usage、Affiliates、Orders、Plans 等。

问题：

1. 后台菜单需要按任务聚类，而不是按数据库实体聚类。

   例如“账号、分组、渠道、模型映射、定时测试、代理、TLS 指纹”实际都属于“上游供给与调度”；“用户、余额、订阅、兑换、支付、优惠码、返利”属于“用户与商业化”；“Ops、Usage、错误日志、系统日志、备份、数据管理”属于“运维与审计”。

2. 管理员很难判断操作影响面。

   修改分组倍率、渠道价格、模型映射、风控策略、错误透传规则，都可能直接影响用户请求和计费。页面需要更强的影响预览、保存确认和变更审计。

3. 设置页面过大。

   `SettingsView.vue` 接近 1 万行，说明系统设置承载了太多异质功能。这样的页面很难维护，也很难保证交互一致。

建议重组后台信息架构：

```text
概览
  - Dashboard
  - 今日流量
  - 今日成本/收入
  - 活跃用户
  - 当前告警

供给与调度
  - 账号
  - 分组
  - 渠道
  - 模型映射
  - 代理
  - TLS 指纹
  - 定时测试

用户与商业化
  - 用户
  - API Key
  - 订阅
  - 订单
  - 套餐
  - 兑换码
  - 优惠码
  - 邀请返利

安全与风控
  - 风控策略
  - 风控日志
  - URL 白名单
  - OAuth / 登录方式
  - 2FA / 会话策略

运维
  - Ops Dashboard
  - 请求错误
  - 上游错误
  - 系统日志
  - 使用记录
  - 备份恢复
  - 数据清理
  - 升级发布

系统配置
  - 站点与品牌
  - 邮件通知
  - 支付渠道
  - 价格源
  - 自定义菜单
  - 实验功能
```

建议新增：

- 后台全局搜索：搜索页面、用户、账号、分组、API Key、订单、错误 ID。
- 最近访问：对管理员高频页面非常有价值。
- 收藏页面：运维人员和客服人员关注点不同。
- 操作影响提示：保存前展示“此改动会影响哪些用户/分组/模型”。
- 变更审计：记录谁在什么时候改了价格、风控、模型映射、账号状态、支付配置。

## 5. 交互与前端体验优化

### 5.1 错误态、空态、加载态

当前通用表格有 loading skeleton 和 empty slot，这是好基础。但大量页面请求失败后只写 `console.error`。这对管理员尤其危险，因为后台操作失败如果没有明确反馈，会导致重复点击、错误判断或误以为配置已经生效。

建议建立统一页面状态模型：

```text
loading
  -> success(data)
  -> empty(reason, action)
  -> partial_success(data, failedPanels)
  -> error(message, retry, traceId)
```

每个页面至少应该有：

- 顶部错误 banner 或局部错误卡片。
- 重试按钮。
- 可复制的错误 ID 或 request ID。
- 用户可理解的失败原因。
- 管理员可跳转的日志链接。

优先改造页面：

1. 用户 Dashboard
2. 用户 Keys
3. 用户 Usage
4. 管理员 Accounts
5. 管理员 Groups
6. 管理员 Settings
7. 管理员 Ops
8. 支付相关页面

### 5.2 表格与移动端

`DataTable.vue` 已经实现了桌面表格和移动端卡片模式，并使用虚拟滚动。这说明项目已经注意到了列表性能和响应式体验。

可优化点：

1. 移动端卡片模式容易信息过密。

   对于账号、用户、订单、错误日志这类列很多的表格，所有字段按 label/value 展示会很长。建议支持“移动端主字段、次字段、折叠字段、主要操作”配置。

2. 操作列需要更稳定的模式。

   当前表格会检测操作列是否需要展开。建议沉淀成统一的 `ActionMenu`：
   - 高频操作直接露出
   - 危险操作放二级菜单
   - 批量操作放表头
   - 移动端统一进入底部抽屉或菜单

3. 空态需要从“无数据”升级为“下一步”。

   例如：
   - 账号列表为空：提示添加第一个上游账号
   - API Key 为空：提示创建 API Key
   - 用量为空：提示复制示例调用
   - 风控日志为空：说明当前没有命中记录，而不是单纯“无数据”

4. 表格需要列管理。

   管理员表格列很多，建议支持：
   - 显示/隐藏列
   - 固定列
   - 保存个人偏好
   - 导出当前筛选结果
   - 清除筛选

### 5.3 Header 和 Sidebar

当前 `AppHeader.vue` 顶部右侧聚合了公告、文档、语言、订阅进度、余额、用户菜单。桌面还好，窄屏会隐藏一部分信息。

建议：

- 顶部只保留跨页面高频状态：告警/公告、余额或订阅状态、用户菜单。
- 文档入口移到帮助菜单。
- 用户余额在普通用户场景保留，在管理员场景可换成“系统健康/告警数量”。
- 管理后台侧边栏增加分组和搜索，而不是无限扩展一级菜单。
- 对不同角色提供不同默认导航：
  - 管理员：Dashboard、供给、用户、运维、安全、配置
  - 普通用户：Dashboard、API Keys、Usage、充值/订阅、Profile

### 5.4 设置页重构

`SettingsView.vue` 是当前前端最大的维护风险之一。设置页的优化建议不是简单拆组件，而是按“配置域”和“发布风险”拆。

建议拆分：

| 新模块 | 内容 |
| --- | --- |
| `settings/site` | 站点名称、Logo、文档链接、联系信息、自定义菜单 |
| `settings/auth` | 登录注册、OAuth、OIDC、微信、钉钉、Turnstile、邮箱验证 |
| `settings/gateway` | 网关协议、Codex、OpenAI、Gemini、Anthropic、Sora、图片 |
| `settings/billing` | 计费模式、价格源、倍率、余额策略 |
| `settings/payment` | 支付开关、支付实例、回调、安全 |
| `settings/security` | URL allowlist、CSP、响应头、API Key IP 限制 |
| `settings/ops` | 日志、告警、采样、清理、更新 |
| `settings/experiments` | 实验功能和回滚开关 |

每个设置模块都应该有：

- 当前值
- 默认值
- 来源：环境变量、配置文件、数据库
- 是否可热更新
- 保存后是否需要重启
- 影响面
- 最近修改人和时间
- 恢复默认

### 5.5 可访问性与可用性

建议补充：

- 所有图标按钮需要稳定 `aria-label`。
- 表格排序头需要键盘可操作和排序状态描述。
- Modal 打开后焦点锁定，关闭后焦点回到触发按钮。
- 危险操作二次确认要说明对象名和影响面。
- 颜色状态不能只靠颜色区分，需文字或图标辅助。
- 长 token、URL、错误信息要支持复制、换行和脱敏展示。

## 6. 功能层优化

### 6.1 账号、分组、渠道、模型的心智模型

当前系统里几个核心概念的关系非常强：

```text
User
  -> API Key
    -> Group
      -> Platform / Model rules / Rate multipliers / Quotas
        -> Channel
          -> Account pool
            -> Upstream provider
```

管理员真正需要的是“这个用户用这个 Key 调这个模型时，为什么会走这个账号，怎么计费，为什么失败”。建议新增一个“路由解释器”或“请求模拟器”：

输入：

- 用户
- API Key
- 请求协议
- 模型
- endpoint
- 是否流式
- 预估 token 或图片参数

输出：

- 命中的分组
- 允许/禁止原因
- 候选渠道
- 候选账号
- 预计价格
- 预计并发限制
- 风控是否会拦截
- 如果失败，最可能原因

这个功能对管理员、客服和 AI 维护都非常关键，可以显著降低排障成本。

### 6.2 定价与计费

系统已经有 token 级用量、模型价格、分组倍率、渠道价格、支付订单等能力。风险点在于计费链路必须可解释、可审计、可回滚。

建议：

- 每条 usage log 记录价格来源：
  - 内置价格
  - 远程价格
  - 渠道覆盖
  - 分组倍率
  - 特殊模型映射
  - 订阅包规则
- 管理端展示价格版本和更新时间。
- 价格更新要有 diff 预览。
- 价格源异常时要告警，而不是静默使用 fallback。
- 对高风险改价提供“试算”功能：
  - 选过去一天 usage
  - 用新价格重算
  - 展示收入/成本变化
- 对订单、余额、返利、兑换、退款等资金变动建立统一 ledger 视图。

代码层观察：

- `billing_service.go` 和相关 pricing 配置已经承载大量逻辑。
- `usage_log_repo.go` 存在 TODO：总成本聚合扫描所有 `usage_logs`，这是明显的性能优化点。
- 支付、订阅、返利、兑换、余额历史是不同模块，建议最终统一到资金账本模型。

### 6.3 风控中心

`content_moderation.go` 和后台 `RiskControlView.vue` 表明系统风控已经比较完整，包括配置、API Key 池、日志、封禁、hash 管理等。

优化重点：

1. 风控策略需要“模拟模式”。

   管理员改策略前应能用样例 prompt 或历史请求回放，看会不会误伤。

2. 风控日志需要解释。

   不只是展示“命中”，还要展示：
   - 命中规则
   - 阈值
   - 模型判断
   - 最终动作
   - 是否封禁
   - 申诉或解封入口

3. 策略需要模板。

   例如：
   - 宽松模式
   - 默认模式
   - 严格模式
   - 企业合规模板
   - 只记录不拦截

4. 需要防误操作。

   清空 hash、删除策略、开启封禁等操作应提供影响确认和审计。

### 6.4 Ops 运维监控

后端已经有 Ops 路由：并发、账号可用性、实时流量、告警、系统日志、请求错误、上游错误、dashboard snapshot 等。这是很好的基础。

下一步应从“有数据”升级到“可处理事件”：

```text
告警出现
  -> 自动分类
  -> 展示影响范围
  -> 给出建议动作
  -> 一键进入相关账号/渠道/用户/日志
  -> 记录处理结果
```

建议：

- Ops 首页按 SLO 展示：
  - 成功率
  - P95/P99 延迟
  - 上游错误率
  - 本地拒绝率
  - 计费写入延迟
  - usage 队列积压
  - 可用账号数
- 错误列表按影响排序：
  - 影响用户数
  - 影响请求数
  - 是否仍在发生
  - 是否有自动恢复
- 请求详情增加“完整链路”：
  - inbound request ID
  - API Key
  - user
  - group
  - channel
  - account
  - upstream request ID
  - billing record
  - ops error
- 告警规则增加推荐阈值和默认模板。

### 6.5 支付与商业化

当前支付页面、订单、支付计划、Stripe、Airwallex、微信回调、PaymentResult 等文件说明支付链路已经很完整。

建议优化：

- 用户付款流程增加“状态恢复”说明和更强的异常恢复。
- 后台订单详情增加支付回调原始摘要、幂等状态、重试记录。
- 支付配置页增加“测试回调”和“沙箱模式”。
- 订阅、余额、返利、优惠码、兑换码的关系要用统一账本解释。
- 用户端订单失败要给明确动作：重新支付、换支付方式、联系客服、刷新状态。

## 7. 后端架构优化

### 7.1 网关路由重复和协议分流

`backend/internal/server/routes/gateway.go` 中对 OpenAI/Anthropic/Gemini/Antigravity 的分流逻辑比较直接，但存在重复模式：

- 判断 group platform
- 不支持则返回 404 或协议格式错误
- 调用对应 handler
- 多个无 `/v1` 前缀别名重复中间件链

建议抽象：

- `GatewayEndpoint` 描述 endpoint、支持平台、错误格式、handler。
- `PlatformGate` 统一处理平台支持与错误响应。
- `MiddlewareBundle` 统一定义 Anthropic/OpenAI/Gemini 的中间件组。
- 给路由生成添加测试，确保新增 endpoint 不漏鉴权、不漏 body limit、不漏 ops logger。

收益：

- 新协议和新 endpoint 更容易添加。
- 减少人工复制导致的中间件遗漏。
- AI 修改路由时更不容易破坏安全边界。

### 7.2 超大服务文件拆分

后端最大的维护风险来自核心服务超大文件：

- `gateway_service.go`
- `openai_gateway_service.go`
- `setting_service.go`
- `usage_log_repo.go`
- `admin_service.go`

建议拆分原则：

```text
gateway_service.go
  -> request_normalization.go
  -> account_selection.go
  -> upstream_forward.go
  -> stream_handling.go
  -> usage_recording.go
  -> error_mapping.go
  -> model_mapping.go
  -> platform_capabilities.go
```

不要为了拆而拆。每次拆分必须配合：

- 不改变导出 API 或只做小范围内部 API。
- 先加 characterization tests。
- 每次拆一个责任域。
- 拆完跑对应单元测试和 integration tests。

### 7.3 配置系统

`backend/internal/config/config.go` 配置项很多，说明系统可调能力很强。但配置越多，越需要：

- 默认值可解释
- 配置来源可追踪
- 配置是否热更新可见
- 配置校验和危险提示明确

建议：

- 建立配置 schema 文档生成。
- 管理端显示每个设置的来源：
  - env
  - config file
  - database setting
  - compiled default
- 启动时输出配置体检摘要，敏感值脱敏。
- 对危险配置 fail closed：
  - URL allowlist
  - private host
  - insecure HTTP
  - trust forwarded IP
  - debug headers
- 对高风险配置建立启动告警。

### 7.4 安全与会话

当前前端把 access token、refresh token、用户信息、过期时间存储在 `localStorage`。这对 SPA 很常见，但如果页面出现 XSS，refresh token 被窃取的风险较高。

建议按阶段优化：

P0：

- 强化 CSP，确保 nonce 策略实际覆盖前端。
- 所有富文本、Markdown、自定义页面继续使用 DOMPurify，并审查配置。
- refresh token 轮换和失效策略明确。
- 管理端提供“退出所有设备”。
- 管理端提供“当前会话/最近登录设备”。

P1：

- 支持 httpOnly secure sameSite cookie 模式作为可选部署模式。
- access token 只保存在内存，refresh token 走 cookie。
- 对 API Key 展示只显示一次，之后只显示尾号和创建时间。

P2：

- 管理后台细粒度 RBAC。
- 操作审计和审批流。
- 高危操作要求二次验证或 TOTP。

其他安全点：

- `backend/internal/pkg/ip/ip.go` 对无效 CIDR 使用 panic。若输入全是内部常量影响不大，但如果未来接入动态配置，应改成返回错误并在配置校验阶段处理。
- `TrustedProxies`、`TrustForwardedIPForAPIKeyACL`、URL allowlist、proxy fallback 都是安全敏感配置，应该在 UI 和文档里显著提示。

### 7.5 数据与性能

明显优化点：

- `backend/internal/repository/usage_log_repo.go` 有 TODO：总成本聚合扫描所有 `usage_logs`。
- usage log、dashboard aggregation、channel monitor rollup 已经有迁移基础，应该把大查询迁移到增量聚合表。
- 高并发下使用记录写入、计费、队列溢出策略要在 Ops 面板显示。

建议：

- 给 usage 相关查询建立查询预算：
  - 用户最近 usage
  - 管理端 usage 列表
  - dashboard trend
  - cost aggregation
- 对慢查询增加日志和指标。
- 对后台导出增加异步任务，避免大查询阻塞请求。
- 对 usage cleanup 任务展示进度、扫描范围、预计影响。
- 建立数据保留策略向导：
  - 原始 usage 保留 N 天
  - 聚合数据保留 N 月
  - 错误日志保留 N 天
  - 系统日志保留 N 天

## 8. 发布、部署与 fork 策略优化

### 8.1 当前能否发布 Docker 镜像

当前仓库具备发布 Docker 镜像的基础：

- `.github/workflows/release.yml` 在 tag 或手动触发时执行。
- 使用 GoReleaser。
- 登录 GHCR 使用 `GITHUB_TOKEN`，具备 `packages: write` 权限。
- 如果配置 `DOCKERHUB_USERNAME` 和 `DOCKERHUB_TOKEN`，也可以发布 DockerHub。
- `.goreleaser.yaml` 支持 amd64/arm64 多架构镜像和 manifest。
- `.goreleaser.simple.yaml` 支持只发布 x86_64 GHCR 镜像。

发布链路有两条：

| 链路 | 触发 | 产物 |
| --- | --- | --- |
| 完整 release | push `v*` tag 或手动 workflow | GitHub Release、二进制包、checksum、多架构 GHCR、可选 DockerHub |
| simple release | 手动设置 `simple_release=true` 或 repo variable | x86_64 GHCR 镜像，跳过二进制包 |

主要问题不是“不能发布”，而是“当前发布策略还没有完全适配 2ue fork”。

### 8.2 和 2ue 发布策略冲突的点

你希望：

- `main` 永远跟上游一致。
- `2ue-main` 承载本仓库迭代。
- 本仓库发版基于 `2ue-main` tag。
- 版本号使用 fork 后缀，例如 `v0.1.133-2ue.1`。

当前冲突：

1. release workflow 触发 `v*`，上游普通 tag 也可能触发。

   应改成只触发：

   ```yaml
   on:
     push:
       tags:
         - 'v*-2ue.*'
   ```

2. workflow 没校验 tag commit 是否属于 `2ue-main`。

   应加入：

   ```bash
   git fetch origin main 2ue-main --depth=1
   git branch -r --contains "$GITHUB_SHA" | grep -q 'origin/2ue-main'
   git merge-base --is-ancestor origin/main "$GITHUB_SHA"
   ```

3. `update-version` 只是 artifact，不写回仓库，这点相对安全。但 release 内容里的安装命令仍指向 `main`。

   对 fork 发布，Release footer 应指向：

   ```text
   https://raw.githubusercontent.com/2ue/sub2api/2ue-main/deploy/install.sh
   ghcr.io/2ue/sub2api:<version>
   ```

4. Dockerfile 和 Dockerfile.goreleaser label 仍是上游 source。

   对 fork 镜像应改为动态 label 或 fork 专属 label。

5. README 和 deploy 脚本硬编码 `Wei-Shaw/sub2api`、`weishaw/sub2api:latest`。

   这在 `main` 上是正确的，但在 `2ue-main` 的 fork 发布文档里应有 2ue 专属入口，避免用户部署到上游镜像。

### 8.3 推荐发布规则

建议继续使用：

```text
上游版本：v0.1.133
2ue 第一次发布：v0.1.133-2ue.1
2ue 第二次发布：v0.1.133-2ue.2
```

不要使用 `v0.1.133.001` 作为长期规则。原因：

- 四段数字不是 SemVer 标准。
- 很多 release 工具、包管理器、版本比较器对四段数字行为不一致。
- `v0.1.133-2ue.1` 明确表达这是基于上游 `v0.1.133` 的 fork 版本。
- GoReleaser 对 prerelease 后缀更友好。

如果担心 `-2ue.1` 被识别为 prerelease，可以接受，因为 fork 发布本质就是带发行渠道标识的版本。若未来要表达稳定渠道，可以用：

```text
v0.1.133+2ue.1
```

但 Docker tag 和部分工具对 `+` 支持不如 `-` 普遍，因此当前更推荐 `v0.1.133-2ue.1`。

### 8.4 部署文档优化

建议在 `2ue-main` 新增 fork 专属部署文档，而不是修改 `main` 上游文档：

- `docs/2ue-deploy.md`
- `docs/2ue-release.md`
- `deploy/2ue-docker-compose.yml` 或在文档里提供替换镜像方式

内容应明确：

```yaml
services:
  sub2api:
    image: ghcr.io/2ue/sub2api:v0.1.133-2ue.1
```

如果仍保留上游部署文档，必须在 2ue 文档里说明：

- `README.md` 主要跟随上游。
- 2ue fork 用户应优先看 `docs/2ue-deploy.md`。
- 不要直接拉 `weishaw/sub2api:latest`，否则不是 2ue 版本。

## 9. AI 全自动协作约束

你明确说这一套大多由 AI 完成，因此必须把流程变成机器可检查的不变量。

### 9.1 AI 执行前置检查

任何 AI 修改前必须先输出：

```bash
git status --short --branch
git branch --show-current
git remote -v
```

如果任务涉及发布或同步，还必须检查：

```bash
git fetch origin --prune
git fetch upstream --prune
git log --oneline --decorate -n 10
git branch -vv
```

### 9.2 AI 任务分类

每次任务必须先判定类型：

| 类型 | 基准分支 | 允许合入 | 禁止 |
| --- | --- | --- | --- |
| 同步上游 | `main` | `upstream/main` fast-forward | 任何本仓库提交 |
| 合并上游到 2ue | `2ue-main` | `origin/main` | 把 `2ue-main` 合回 `main` |
| 给上游 PR | 从 `main` 新切 | 上游可接受通用改动 | 2ue 文档、镜像、release 配置 |
| 2ue 功能 | 从 `2ue-main` 新切 | `2ue-main` | 合入 `main` |
| 2ue 发布 | `2ue-main` | tag | 普通 `vX.Y.Z` tag |

### 9.3 AI 修改代码约束

建议建立 `docs/2ue-ai-coding-rules.md`：

- 大文件禁止随意整文件重写。
- 超过 1000 行文件改动必须先说明影响范围。
- 修改 gateway、billing、auth、payment、migration、release workflow 必须列出回归测试。
- 修改 UI 必须检查移动端、空态、错误态、loading 态。
- 修改配置必须更新 `.env.example`、`config.example.yaml`、文档和测试。
- 修改发布必须检查 tag、branch、image、release note、安装命令。
- 禁止无解释地删除上游逻辑。
- 禁止为了合并冲突直接选择一边。

### 9.4 AI 可执行检查脚本

建议新增脚本：

- `scripts/check-2ue-branch-invariants.sh`
  - `main` 是否等于 `upstream/main`
  - `2ue-main` 是否包含 `origin/main`
  - 当前 tag 是否在 `2ue-main`
  - 是否误推普通 `v*` tag

- `scripts/check-release-readiness.sh`
  - 工作区干净
  - 测试通过
  - tag 格式正确
  - GoReleaser dry run
  - Docker 镜像名正确

- `scripts/check-doc-drift.sh`
  - README Go 版本与 `go.mod` 是否一致
  - Docker Compose 镜像名是否符合当前发行渠道
  - release footer 是否指向正确分支

## 10. 优先级路线图

### P0：先保证不会发错、不会用错、不会看不懂错误

1. 收紧 release workflow

   - 只触发 `v*-2ue.*`
   - 校验 tag 属于 `2ue-main`
   - 校验 tag commit 包含 `origin/main`
   - Release 内容指向 2ue 镜像和 2ue 文档

2. 建立 2ue 部署文档

   - 明确 GHCR 镜像
   - 明确不要使用上游 `weishaw/sub2api:latest`
   - 给 Docker Compose 示例

3. 统一前端错误反馈

   - 先改用户 Dashboard、Keys、Usage
   - 再改管理员 Accounts、Groups、Settings
   - 所有 catch 不能只 `console.error`

4. 后台导航重组第一版

   - 先不大改页面，只重组 Sidebar 分组
   - 增加搜索入口
   - 隐藏未启用功能，保留可发现的开启入口

5. 安装体检

   - 展示 JWT/TOTP/Redis/DB/URL allowlist/默认管理员风险
   - 给出具体修复命令或文档链接

### P1：降低长期维护成本

1. 拆分 `SettingsView.vue`

   先按 tabs/components 拆，不改变视觉和 API。

2. 拆分账号弹窗

   `CreateAccountModal.vue` 和 `EditAccountModal.vue` 按平台、OAuth、代理、测试、调度配置拆。

3. 后端 gateway 分流抽象

   把重复 endpoint gate 和中间件组合收束。

4. 建立请求模拟器

   管理员输入用户、Key、模型，系统解释路由、计费和失败原因。

5. 价格来源和试算

   管理端展示价格版本、来源、更新时间和新价格试算影响。

6. usage 聚合性能修复

   处理 `usage_log_repo.go` 里总成本聚合全表扫描问题。

### P2：产品化和企业化

1. 细粒度管理员权限

   区分超级管理员、运维、客服、财务、只读审计。

2. 操作审计与审批

   风控、价格、支付、渠道、账号、模型映射都应有审计。

3. 运维事件工作台

   告警从“数据展示”升级为“处理流程”。

4. 多环境发布

   dev/staging/prod 镜像、staging tag、生产手动批准。

5. 设计系统

   统一按钮、表格、表单、提示、确认、空态、错误态、Badge、Status。

6. SLO 和容量模型

   建立容量估算、并发预算、数据库连接预算、Redis 连接预算、上游账号池预算。

## 11. 可以先落地的具体任务包

### 任务包 A：2ue 发布安全

目标：防止发错 tag、发错分支、发错镜像。

改动：

- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `.goreleaser.simple.yaml`
- `docs/2ue-release.md`
- `docs/2ue-deploy.md`

验收：

- `v0.1.133` 不触发 2ue 发布。
- `v0.1.133-2ue.1` 可以发布。
- tag 不在 `2ue-main` 时 workflow 失败。
- Release 中 Docker pull 是 `ghcr.io/2ue/sub2api:<version>`。

### 任务包 B：用户首次成功调用

目标：让普通用户 5 分钟内完成第一次调用。

改动：

- `frontend/src/views/user/KeysView.vue`
- `frontend/src/components/keys/UseKeyModal.vue`
- `frontend/src/views/user/DashboardView.vue`
- `backend/internal/handler/gateway_handler.go` 或相关测试接口

验收：

- 创建 Key 后能看到明确下一步。
- 可复制 OpenAI/Claude/Codex/Gemini 示例。
- 可发起一次测试请求。
- 测试失败时展示业务原因。
- 成功后可跳到 usage 记录。

### 任务包 C：统一页面错误状态

目标：消灭关键页面只 `console.error` 的体验。

改动：

- 新增 `useAsyncPanel` 或 `usePageRequestState`
- 新增 `ErrorState.vue`
- 改用户 Dashboard、Usage、Keys
- 改管理员 Accounts、Groups、Settings 的关键请求

验收：

- loading、empty、error、partial success 都可见。
- 错误有重试。
- 错误包含 request ID 或可复制诊断信息。

### 任务包 D：后台导航重组

目标：从功能堆叠变成任务导航。

改动：

- `AppSidebar.vue`
- 路由 meta 增加 group/category
- 增加后台搜索或快速跳转

验收：

- 管理员入口按“概览、供给、用户、商业化、安全、运维、系统”分组。
- 未启用功能不干扰日常操作。
- 高频页面可收藏或最近访问。

### 任务包 E：设置页拆分

目标：降低最大前端维护风险。

改动：

- `SettingsView.vue`
- `frontend/src/views/admin/settings/*`
- `frontend/src/api/admin/settings.ts`

验收：

- 行数显著降低。
- 每个设置域有独立组件和测试。
- 保存行为不变。
- 对 fork 合并上游冲突更容易处理。

## 12. 风险和取舍

### 不建议现在做的事

1. 不建议立即大规模重写 UI。

   当前系统功能多，直接重写容易引入回归。应先统一状态模型和导航，再逐步拆页面。

2. 不建议把 2ue 修改合入 `main`。

   你的分支策略是正确的：`main` 镜像上游，`2ue-main` 承载 fork。分歧是正常状态。

3. 不建议把版本号做成四段数字。

   推荐 `vX.Y.Z-2ue.N`，更符合工具生态。

4. 不建议先追求功能数量。

   现在最影响增长和稳定的是可理解性、可诊断性和可发布性。

### 必须接受的现实

- `main` 和 `2ue-main` 会长期分歧。
- 上游同步会经常带来冲突，尤其是 release、docs、settings、gateway、admin 页面。
- AI 自动化越多，越需要脚本化检查和小步提交。
- 前端巨型文件会显著降低 AI 修改可靠性，应尽早拆。

## 13. 最终建议

建议把下一阶段目标定义为：

```text
让 Sub2API 从“功能完整的网关系统”
升级为
“可被普通用户顺利使用、可被管理员稳定运营、可被 2ue fork 安全发布的产品化平台”
```

执行顺序：

1. 先修发布链路和 fork 文档，避免发错版本。
2. 再修用户首次调用和错误反馈，降低使用阻力。
3. 再重组后台导航，降低管理员认知负担。
4. 再拆最大文件，降低 AI 和上游合并成本。
5. 最后做请求模拟器、价格试算、运维事件工作台这些产品化能力。

如果只能选一个短期突破口，我建议先做“2ue 发布安全 + 用户首次成功调用 + 统一错误态”。这三个任务投入相对可控，但会同时提升发布可靠性、用户转化和日常排障效率。
