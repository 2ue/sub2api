# Sub2API 邀请奖励与充值返利设计重写分析

日期：2026-04-29  
代码库：`/Users/yuanfeijie/Desktop/procode/sub2api`  
当前提交：`b0a2252e`  
文档范围：基于当前代码事实，重新分析现有邀请返利系统能否支持新的营销方案，并给出不冲突、可审计、可风控的改造方向。  
验证命令：`cd backend && go test -tags=unit ./internal/service ./internal/handler ./internal/payment/...`，结果全部通过。

## 结论

当前代码不能直接、安全、完整地实现这次提出的营销方案。

现有邀请返利系统本质是：

1. 用户拥有唯一 `aff_code`。
2. 新用户注册时可以通过 `aff_code` 绑定一个邀请人。
3. 邀请关系绑定后，只增加邀请人的 `aff_count`。
4. 被邀请人后续完成余额充值订单时，系统按全局比例或邀请人的专属比例给邀请人增加返利额度。
5. 返利额度可以冻结一段时间，冻结到期后转为可用返利。
6. 邀请人需要手动把可用返利转入余额。

现有系统已经支持：

1. 全局邀请返利开关。
2. 全局充值返利比例。
3. 邀请人专属充值返利比例。
4. 邀请人专属邀请码。
5. 充值返利冻结期。
6. 邀请关系有效期。
7. 单个被邀请人的累计充值返利上限。

现有系统不支持：

1. 邀请注册立即奖励。
2. 邀请注册奖励人数上限。
3. 邀请注册奖励金额上限。
4. 累计邀请人数达到阶梯后的额外奖励。
5. 首单充值固定奖励。
6. 前 N 单充值阶梯奖励。
7. 按充值金额阶梯奖励。
8. 邀请人的累计充值奖励总额上限。
9. 已产生充值奖励的被邀请人数上限。
10. 单笔奖励上限。
11. 按支付渠道或 provider instance 设置奖励。
12. 注册奖励和充值奖励之间的独立统计。
13. 多个奖励规则之间的冲突检测。
14. 退款后的返利回滚。

因此，如果只是继续在现有 `affiliate_rebate_rate`、`affiliate_rebate_per_invitee_cap`、`user_affiliates.aff_quota` 和 `user_affiliate_ledger action='accrue'` 上追加逻辑，会出现规则混淆、上限互相污染、重复奖励、并发超发、退款不可回滚、审计不可追踪等问题。

本次方案应改成“事件化奖励系统”：把注册绑定、充值完成、退款回滚、人数里程碑拆成不同事件；每个事件按规则计算奖励；每笔奖励写结构化 ledger；上限通过原子计数占用；不同邀请人的规则通过“唯一生效奖励方案”解决冲突。

## 当前代码事实

### 1. 当前数据模型只表达了邀请关系和返利余额

`user_affiliates` 表定义在 `backend/migrations/130_add_user_affiliates.sql:1-9`：

```sql
CREATE TABLE IF NOT EXISTS user_affiliates (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    aff_code VARCHAR(32) NOT NULL UNIQUE,
    inviter_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    aff_count INTEGER NOT NULL DEFAULT 0,
    aff_quota DECIMAL(20,8) NOT NULL DEFAULT 0,
    aff_history_quota DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

字段语义由同一迁移脚本注释确认：

- `aff_code` 是用户邀请码，见 `backend/migrations/130_add_user_affiliates.sql:16`。
- `inviter_id` 是邀请人用户 ID，见 `backend/migrations/130_add_user_affiliates.sql:17`。
- `aff_count` 是累计邀请人数，见 `backend/migrations/130_add_user_affiliates.sql:18`。
- `aff_quota` 是当前可提取返利金额，见 `backend/migrations/130_add_user_affiliates.sql:19`。
- `aff_history_quota` 是累计返利历史金额，见 `backend/migrations/130_add_user_affiliates.sql:20`。

后续迁移只增加了两个用户级专属字段：

- `aff_rebate_rate_percent`：用户作为邀请人时的专属返利比例，见 `backend/migrations/132_affiliate_custom_settings.sql:5-15`。
- `aff_code_custom`：邀请码是否由管理员改写，见 `backend/migrations/132_affiliate_custom_settings.sql:8-16`。

冻结功能只增加了：

- `user_affiliates.aff_frozen_quota`，见 `backend/migrations/133_affiliate_rebate_freeze.sql:2-5`。
- `user_affiliate_ledger.frozen_until`，见 `backend/migrations/133_affiliate_rebate_freeze.sql:7-12`。

结论：当前表结构没有“奖励规则”“奖励类型”“注册奖励”“首单奖励”“阶梯规则”“订单 ID”“规则 ID”“奖励状态”“回滚关系”等字段。

### 2. 当前 ledger 只能粗略记录 accrue/transfer

`user_affiliate_ledger` 建表见 `backend/migrations/131_affiliate_rebate_hardening.sql:12-21`：

```sql
CREATE TABLE IF NOT EXISTS user_affiliate_ledger (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(32) NOT NULL,
    amount DECIMAL(20,8) NOT NULL,
    source_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

注释只定义了 `action` 为 `accrue|transfer`，见 `backend/migrations/131_affiliate_rebate_hardening.sql:26-27`。

当前充值返利写 ledger 的逻辑在 `backend/internal/repository/affiliate_repo.go:113-127`：

- 冻结时插入 `action='accrue'`、`amount`、`source_user_id`、`frozen_until`。
- 不冻结时插入 `action='accrue'`、`amount`、`source_user_id`。

当前转余额写 ledger 的逻辑在 `backend/internal/repository/affiliate_repo.go:278-280`：

- 插入 `action='transfer'`、`amount`。

结论：当前 ledger 无法区分“注册奖励”和“充值返利”，也无法知道某笔返利来自哪一笔订单、哪一条规则、哪一个奖励方案、是否已回滚、是否因退款被部分扣回。

### 3. 注册绑定只绑定关系，不发奖励

邮箱注册创建用户后，代码会初始化 affiliate profile 并尝试绑定邀请码，见 `backend/internal/service/auth_service.go:227-238`：

- `postAuthUserBootstrap`。
- `EnsureUserAffiliate`。
- `BindInviterByCode`。
- 绑定失败只记录日志，不阻断注册。

OAuth 注册后的绑定逻辑在 `backend/internal/service/auth_service.go:783-797`：

- `bindOAuthAffiliate` 会初始化 affiliate profile。
- 如果存在 affiliate code，则调用 `BindInviterByCode`。
- 失败只记录日志，不阻断注册。

`BindInviterByCode` 的核心逻辑在 `backend/internal/service/affiliate_service.go:195-237`：

- 清理并大写邀请码。
- 总开关关闭时直接忽略，见 `backend/internal/service/affiliate_service.go:203-206`。
- 校验格式，见 `backend/internal/service/affiliate_service.go:207-209`。
- 查找邀请人，见 `backend/internal/service/affiliate_service.go:219-224`。
- 禁止同一 `user_id` 自邀，见 `backend/internal/service/affiliate_service.go:226-228`。
- 调用 repo 绑定邀请人，见 `backend/internal/service/affiliate_service.go:230-236`。

repo 绑定逻辑在 `backend/internal/repository/affiliate_repo.go:51-86`：

- `UPDATE user_affiliates SET inviter_id = $1 ... WHERE user_id = $2 AND inviter_id IS NULL`，见 `backend/internal/repository/affiliate_repo.go:61-64`。
- 绑定成功后 `UPDATE user_affiliates SET aff_count = aff_count + 1 ... WHERE user_id = $1`，见 `backend/internal/repository/affiliate_repo.go:74-79`。

结论：当前“邀请注册成功”只会产生邀请关系和 `aff_count + 1`，不会给邀请人增加 `aff_quota` 或 `aff_frozen_quota`，也不会写任何注册奖励 ledger。

### 4. 充值返利只在余额订单履约里触发

余额订单履约在 `backend/internal/service/payment_fulfillment.go:265-292`：

- 先创建或复用充值码。
- 调用 `redeemService.Redeem` 给用户充值余额，见 `backend/internal/service/payment_fulfillment.go:285-287`。
- 然后调用 `applyAffiliateRebateForOrder`，见 `backend/internal/service/payment_fulfillment.go:288-291`。
- 最后标记订单完成，见 `backend/internal/service/payment_fulfillment.go:291`。

订阅订单履约在 `backend/internal/service/payment_fulfillment.go:339-357`，没有调用 affiliate 返利。

`applyAffiliateRebateForOrder` 只处理余额订单，见 `backend/internal/service/payment_fulfillment.go:368-370`：

```go
if o == nil || o.OrderType != payment.OrderTypeBalance || o.Amount <= 0 {
    return nil
}
```

实际返利调用在 `backend/internal/service/payment_fulfillment.go:397`：

```go
rebateAmount, err := s.affiliateService.AccrueInviteRebate(txCtx, o.UserID, o.Amount)
```

结论：当前充值返利和“余额充值订单履约”强绑定；订阅订单不返利；手工加余额、兑换码加余额、返利转余额也不触发邀请返利。

### 5. 充值返利只接收被邀请人 ID 和金额，不接收订单上下文

`AccrueInviteRebate` 函数签名见 `backend/internal/service/affiliate_service.go:240`：

```go
func (s *AffiliateService) AccrueInviteRebate(ctx context.Context, inviteeUserID int64, baseRechargeAmount float64) (float64, error)
```

它没有接收：

- payment order id。
- order type。
- payment type。
- provider instance id。
- pay amount。
- refund status。
- whether first order。
- order index。
- source rule id。

结论：在当前函数签名下，无法可靠实现首单奖励、前 N 单奖励、金额阶梯、支付渠道条件、订单级幂等和退款回滚。

### 6. 当前返利比例支持全局和邀请人专属

全局 affiliate 配置 key 定义在 `backend/internal/service/domain_constants.go:103-107`：

- `affiliate_enabled`。
- `affiliate_rebate_rate`。
- `affiliate_rebate_freeze_hours`。
- `affiliate_rebate_duration_days`。
- `affiliate_rebate_per_invitee_cap`。

默认值定义在 `backend/internal/service/domain_constants.go:21-31`：

- 默认返利比例 20%。
- 比例范围 0-100%。
- 默认 affiliate 关闭。
- 默认不冻结。
- 默认永久有效。
- 默认没有单个被邀请人返利上限。

读取全局返利比例的逻辑在 `backend/internal/service/setting_service.go:1518-1531`，会解析并 clamp 到范围内。

邀请人专属比例优先于全局比例，见 `backend/internal/service/affiliate_service.go:311-322`：

```go
if inviter != nil && inviter.AffRebateRatePercent != nil {
    ...
    return clampAffiliateRebateRate(v)
}
return s.globalRebateRatePercent(ctx)
```

管理员单用户设置专属比例的请求字段在 `backend/internal/handler/admin/affiliate_handler.go:50-55`。

服务层校验专属比例范围在 `backend/internal/service/affiliate_service.go:417-431`。

repo 写入专属比例在 `backend/internal/repository/affiliate_repo.go:615-640`。

结论：当前确实能针对不同邀请人设置不同充值返利比例，但只能设置一个百分比，不能设置注册奖励、阶梯奖励、金额上限、人数上限等完整方案。

### 7. 当前冻结是时间冻结，不是自动转余额，也没有门槛金额

充值返利入账时，如果 `freezeHours > 0`，会写入 `aff_frozen_quota` 和 `frozen_until`，见 `backend/internal/repository/affiliate_repo.go:96-127`。

冻结到期转换逻辑在 `thawFrozenQuotaTx`，见 `backend/internal/repository/affiliate_repo.go:165-205`：

- 找到 `frozen_until <= NOW()` 的 ledger。
- 把这些 ledger 的 `frozen_until` 置空。
- 把总金额从 `aff_frozen_quota` 移到 `aff_quota`。

触发 thaw 的位置只有：

- 用户查看 affiliate detail 前 best-effort thaw，见 `backend/internal/service/affiliate_service.go:167-172`。
- 用户转余额前 thaw，见 `backend/internal/repository/affiliate_repo.go:216-219`。

转余额是用户手动调用，路由是 `POST /api/v1/user/aff/transfer`，见 `backend/internal/server/routes/user.go:28-29` 和 `backend/internal/handler/user_handler.go:182-200`。

当前转余额逻辑在 `backend/internal/repository/affiliate_repo.go:207-290`：

- 先 thaw 到期冻结额度。
- 锁定并清空 `aff_quota`。
- 把金额加到用户余额。
- 写 `transfer` ledger。

结论：

1. 当前冻结到期不会由后台任务主动执行。
2. 当前冻结到期后只是变成可用返利 `aff_quota`，不会自动进入用户余额。
3. 当前没有“大于多少才能提现/转余额”的最低门槛。
4. 当前也没有“奖励金额低于多少不发”的规则。

## 你提出的营销场景逐项判断

### 场景 1：邀请用户即奖励

需求描述：

1. 被邀请用户注册并绑定邀请人后，邀请人立即获得奖励。
2. 可设置邀请注册奖励上限人数。
3. 可设置邀请注册奖励上限金额。
4. 上限只限制注册奖励，不影响后续继续邀请注册，也不影响后续充值奖励。

当前代码不能直接实现。

原因一：注册绑定路径没有奖励动作。  
`BindInviter` 只做 `inviter_id` 绑定和 `aff_count + 1`，见 `backend/internal/repository/affiliate_repo.go:61-79`。没有调用 `AccrueQuota`，没有写 ledger。

原因二：现有 cap 不是注册奖励 cap。  
`affiliate_rebate_per_invitee_cap` 只在 `AccrueInviteRebate` 里用于充值返利截断，见 `backend/internal/service/affiliate_service.go:280-294`。它不是注册奖励人数上限，也不是注册奖励金额上限。

原因三：现有 ledger 不能区分注册奖励和充值返利。  
如果硬把注册奖励也写成 `action='accrue'`，那么 `GetAccruedRebateFromInvitee` 会把同一个被邀请人的所有 `accrue` 金额都算入 per-invitee cap，见 `backend/internal/repository/affiliate_repo.go:137-152`。这会导致注册奖励挤占充值返利上限，违反“注册奖励不影响后续充值奖励”。

原因四：`aff_count` 不能作为“已奖励邀请人数”。  
`aff_count` 是累计邀请人数，只要绑定成功就加 1，见 `backend/internal/repository/affiliate_repo.go:74-79`。你的规则要求“超过注册奖励人数上限后，仍允许继续邀请和继续充值返利”，所以必须区分：

- 总邀请人数。
- 已发放注册奖励的人数。
- 触发充值奖励的人数。
- 触发里程碑奖励的人数。

推荐实现：

1. 新增事件类型 `invite_signup`。
2. 注册绑定成功后，在同一事务内尝试发放注册奖励。
3. 注册奖励写独立 ledger，必须有 `event_type='invite_signup'`。
4. 注册奖励上限单独统计，不和充值返利共用 cap。
5. 超过注册奖励人数上限或金额上限时，本次注册奖励为 0，但邀请关系仍然绑定，后续充值奖励仍然继续按充值规则计算。

注册奖励建议的业务语义：

```text
if affiliate_enabled=false:
    不绑定邀请关系，也不发奖励

if aff_code 无效:
    当前代码行为是注册不失败，只记录日志；是否改成阻断注册需要产品明确

if invitee 已绑定邀请人:
    不重复发注册奖励

if inviter 达到注册奖励人数上限:
    继续绑定 invitee，注册奖励为 0，后续充值奖励不受影响

if inviter 达到注册奖励金额上限:
    继续绑定 invitee，注册奖励为 0 或截断，后续充值奖励不受影响

if 未达到上限:
    给 inviter 产生 invite_signup 奖励 ledger
```

对“金额上限剩余不足一次奖励”的建议：

- 对固定注册奖励，建议默认跳过，不建议发部分奖励。
- 例如每注册奖励 5，金额上限剩余 3 时，跳过更符合“每人奖励 5”的用户理解。
- 如果业务希望严格用完预算，可以增加 `cap_strategy = skip | truncate`，但必须在后台文案里写清楚。

### 场景 1 衍生：累计邀请人数奖励

你提到“累计邀请人数奖励”，这个不是当前 `aff_count` 直接能做的奖励。

当前 `aff_count` 只记录累计绑定人数，见 `backend/migrations/130_add_user_affiliates.sql:5` 和 `backend/internal/repository/affiliate_repo.go:74-79`。它没有记录某个里程碑是否已经奖励。

如果要做“邀请 10 人奖励 X，邀请 50 人奖励 Y”，需要独立规则：

```text
event_type = invite_count_milestone
milestone = 10 / 50 / 100
reward_amount = X / Y / Z
idempotency_key = invite_count_milestone:<inviter_id>:<rule_id>:<milestone>
```

必须记录已经发放过的里程碑，否则每次查看或每次新注册都可能重复发同一个里程碑奖励。

推荐语义：

1. `aff_count` 继续作为总邀请人数展示。
2. 新增 `rewarded_invite_count_milestones` 或通过 ledger 唯一键记录已奖励里程碑。
3. 里程碑奖励是否与“每注册一个用户奖励”叠加，必须显式配置。
4. 默认可以允许叠加，因为两者是不同奖励类型；如果不允许叠加，需要规则字段 `exclusive_group` 或 `stackable=false`。

### 场景 2：充值首单奖励

需求描述：

- 被邀请人第一次充值时，邀请人获得固定奖励或按比例奖励。

当前代码不能直接实现。

原因一：当前没有首单判断。  
`AccrueInviteRebate` 只拿到 `inviteeUserID` 和金额，见 `backend/internal/service/affiliate_service.go:240`，没有订单 ID 和订单序号。

原因二：当前只要是余额订单，每笔都会按比例返利。  
`applyAffiliateRebateForOrder` 对每个余额订单都会调用 `AccrueInviteRebate`，见 `backend/internal/service/payment_fulfillment.go:368-397`。

原因三：当前 ledger 没有 `payment_order_id`。  
无法可靠判断某笔订单是否已经发过首单奖励，也无法退款时按订单回滚。

推荐实现：

1. 充值奖励事件必须接收完整订单对象，而不是只接收金额。
2. 通过 `payment_orders` 查询该被邀请人的已完成 eligible 充值订单数量，或通过奖励统计表维护 order index。
3. 首单奖励写入 ledger：

```text
event_type = recharge
reward_component = first_order
payment_order_id = 当前订单 ID
order_index = 1
rule_id = 首单规则 ID
idempotency_key = recharge:first_order:<payment_order_id>:<rule_id>
```

4. 同一订单重复回调或 retry 时，依靠 ledger 唯一键保证不重复发。

### 场景 2 衍生：阶梯单奖励

需求描述：

- 第 1 单奖励 X。
- 第 2 单奖励 Y。
- 第 3-5 单奖励 Z。
- 超过 N 单不奖励，或按默认比例奖励。

当前代码不能直接实现。

当前没有：

- eligible order index。
- order tier 规则。
- 每个订单的 `rule_id`。
- 每个订单的 affiliate ledger 记录。
- 多规则命中时的优先级。

推荐规则模型：

```text
event_type = recharge
tier_type = order_index
order_index_from = 1
order_index_to = 1
reward_mode = fixed | percent
reward_value = 10

event_type = recharge
tier_type = order_index
order_index_from = 2
order_index_to = 5
reward_mode = percent
reward_value = 15
```

冲突校验：

1. 同一个奖励方案、同一个 event_type、同一个 order_type、同一个 provider 条件下，订单序号区间不能重叠。
2. 如果允许重叠，必须配置 `priority`，并且只取优先级最高的一条。
3. 默认建议禁止重叠，避免后台误配。

### 场景 2 衍生：阶梯金额奖励

需求描述：

- 充值金额达到不同区间，奖励不同金额或比例。

当前代码不能直接实现。

当前代码中的金额有两个关键字段：

- `PaymentOrder.amount`，字段定义在 `backend/ent/schema/payment_order.go:45-49`。
- `PaymentOrder.pay_amount`，字段定义在 `backend/ent/schema/payment_order.go:45-49`。

余额订单创建时，`orderAmount` 会使用充值倍率计算到账余额，见 `backend/internal/service/payment_order.go:51-58`：

```go
orderAmount = calculateCreditedBalance(req.Amount, cfg.BalanceRechargeMultiplier)
```

`calculateCreditedBalance` 会乘以 `BalanceRechargeMultiplier`，见 `backend/internal/service/payment_amounts.go:18-23`。

当前返利基数使用 `o.Amount`，见 `backend/internal/service/payment_fulfillment.go:397`。

因此，阶梯金额规则上线前必须明确：

1. 按真实支付金额 `pay_amount` 判定阶梯。
2. 按用户输入充值金额判定阶梯。
3. 按到账余额 `amount` 判定阶梯。
4. 按扣除手续费后的净收款判定阶梯。

建议默认按 `pay_amount` 或原始支付金额判定，因为它更接近真实收入。当前代码默认按到账余额返利，在存在充值倍率时会放大奖励。

金额阶梯规则建议：

```text
event_type = recharge
tier_type = amount
amount_base = pay_amount | credited_amount
amount_min = 100
amount_max = 199.99
reward_mode = fixed | percent
reward_value = 10
```

冲突校验：

1. 同一方案内，金额区间不能重叠。
2. 金额边界必须明确是闭区间还是左闭右开。
3. 如果一个订单同时满足“首单奖励”和“金额阶梯奖励”，必须配置是否叠加。

推荐默认：

- `first_order` 和 `amount_tier` 不自动叠加。
- 如果需要叠加，后台必须显式开启 `stackable=true`。
- ledger 必须分别记录两个 `reward_component`，否则后续退款回滚和报表会混乱。

### 场景 2：累计充值奖励上限

需求描述：

- 邀请人通过被邀请人充值产生的奖励，有累计金额上限。

当前代码只支持“单个被邀请人累计返利上限”，不支持“邀请人累计充值奖励总额上限”。

现有单人 cap 的逻辑：

- 读取 `affiliate_rebate_per_invitee_cap`，见 `backend/internal/service/setting_service.go:1567-1579`。
- 查询邀请人从该被邀请人拿到的累计 `accrue`，见 `backend/internal/repository/affiliate_repo.go:137-152`。
- 在 service 层截断本次返利，见 `backend/internal/service/affiliate_service.go:280-294`。

这不是邀请人总额 cap。

推荐新增：

```text
recharge_reward_total_amount_cap
scope = inviter
event_type = recharge
metric = reward_amount
```

语义建议：

1. 只限制充值奖励，不限制注册奖励。
2. 达到上限后，后续充值奖励为 0。
3. 被邀请用户继续充值不受影响。
4. 邀请关系继续存在。
5. 如果剩余额度不足本次奖励：
   - 百分比奖励建议截断到剩余额度。
   - 固定奖励建议默认跳过，也可配置截断。

实现要求：

1. 不能用 `SUM(ledger.amount)` 先查再写的方式做硬上限，因为并发会超发。
2. 必须用统计行锁或单条 SQL 原子占用剩余额度。
3. ledger 仍然要写实际发放金额和原始计算金额。

### 场景 2：累计充值人数奖励上限

这句话有两种可能含义，代码当前两种都不支持。

含义一：最多只有前 N 个产生充值的被邀请人能给邀请人带来充值奖励。

例如：

```text
recharge_reward_paying_invitee_cap = 100
```

语义：

1. 前 100 个完成 eligible 充值的被邀请人可以产生充值奖励。
2. 第 101 个被邀请人充值后，不再给邀请人产生充值奖励。
3. 注册奖励不受影响。
4. 已经被计入前 100 的被邀请人，后续是否还能继续产生充值奖励，需要规则明确。

推荐默认：

- “计入 cap 的被邀请人”后续仍可按充值规则继续产生奖励，直到其他金额/订单上限触发。
- cap 限制的是可产生充值奖励的 distinct invitee 数量，不是订单数量。

实现要求：

1. 需要一张 distinct claim 表，例如 `affiliate_reward_recharge_invitees`。
2. 字段包括 `inviter_id`、`invitee_user_id`、`profile_id`、`first_payment_order_id`、`created_at`。
3. 对 `(inviter_id, invitee_user_id, profile_id)` 加唯一约束。
4. 邀请人维度的 distinct 数量上限必须原子占用，不能并发下超发。

含义二：当累计有 N 个被邀请人完成充值时，给邀请人额外奖励。

例如：

```text
累计 10 个被邀请人首充，奖励 50
累计 50 个被邀请人首充，奖励 300
```

这属于 `recharge_invitee_count_milestone`，需要独立 ledger 幂等键：

```text
idempotency_key = recharge_invitee_milestone:<inviter_id>:<rule_id>:<milestone>
```

结论：这两个含义应在产品配置中分开命名，不能都叫“累计充值人数奖励上限”。建议使用：

- `充值奖励人数上限`：控制最多多少个被邀请人能产生充值奖励。
- `充值人数里程碑奖励`：累计达到 N 个充值用户后额外发奖励。

### 场景 3：针对不同用户设置不同邀请奖励且不能冲突

当前代码只能针对不同邀请人设置“专属充值返利比例”，不能设置完整奖励方案。

当前没有冲突的地方：

1. 全局返利比例是默认值。
2. 如果邀请人设置了 `aff_rebate_rate_percent`，就覆盖全局比例。
3. 一次计算只会使用一个比例。

代码事实见 `backend/internal/service/affiliate_service.go:311-322`。

如果未来支持完整方案，建议不要做“多套规则自动叠加”，否则冲突会非常难控。

推荐模型：

1. 每个邀请人同一时间最多绑定一个 active reward profile。
2. 没有专属 profile 的邀请人使用 global default profile。
3. 专属 profile 和 global profile 默认不是叠加关系，而是专属 profile 覆盖 global profile。
4. 如果要继承 global，需要字段级 `inherit` 语义，而不是两套规则同时命中。
5. 每条 ledger 必须记录 `profile_id` 和 `rule_id`。

推荐表：

```text
affiliate_reward_profiles
- id
- name
- enabled
- is_default
- created_at
- updated_at

affiliate_user_reward_profiles
- inviter_user_id
- profile_id
- active
- starts_at
- ends_at
- created_at
- updated_at
```

约束：

```text
UNIQUE(inviter_user_id) WHERE active = true
```

规则表：

```text
affiliate_reward_rules
- id
- profile_id
- enabled
- event_type
- reward_component
- reward_mode
- reward_value
- amount_base
- order_type
- provider_instance_id
- order_index_from
- order_index_to
- amount_min
- amount_max
- max_reward_per_event
- freeze_hours
- priority
- exclusive_group
- stackable
- starts_at
- ends_at
```

冲突校验：

1. 同一 profile 内，`invite_signup` 固定奖励规则最多一条 active。
2. 同一 profile 内，同一 `exclusive_group` 默认只能命中一条。
3. 阶梯单规则的订单序号区间不能重叠。
4. 阶梯金额规则的金额区间不能重叠。
5. 如果 `provider_instance_id` 为空表示所有渠道；如果同时存在全渠道规则和指定渠道规则，必须明确优先级。
6. 如果规则允许叠加，必须显式 `stackable=true`。
7. 如果一个邀请人同时匹配多个 profile，应拒绝保存配置，而不是运行时随机选。

## 冻结与到账应该怎么定义

当前系统中的“冻结”不是自动到余额。

当前真实语义是：

```text
奖励产生 -> aff_frozen_quota
冻结到期且触发 thaw -> aff_quota
用户手动 transfer -> users.balance
```

其中 thaw 只在用户查看 affiliate detail 或手动 transfer 时触发，见 `backend/internal/service/affiliate_service.go:167-172` 和 `backend/internal/repository/affiliate_repo.go:216-219`。

因此，产品文案不能写成“冻结结束自动到账余额”。更准确的是：

1. 冻结结束后自动转为可用返利。
2. 用户需要手动转入余额。

如果你希望“时间到自动到账到余额”，当前代码不能直接实现，需要新增后台任务或异步 job：

```text
定时任务扫描 frozen_until <= NOW()
把 frozen reward 置为 available
如果 auto_transfer_enabled=true:
    将 available reward 转入 users.balance
    写 transfer ledger
```

是否建议自动转余额：

- 注册奖励不建议默认自动转余额，因为它最容易被批量注册白嫖。
- 充值奖励可以考虑在冻结期大于退款窗口后自动转余额，但必须先实现退款回滚和风控。
- 当前更安全的方式是“冻结期结束变为可转返利，用户手动领取”。

关于“大于多少才行”，建议分成三个不同门槛：

1. `min_reward_event_amount`：单次计算出来的奖励低于多少不发。
2. `min_source_payment_amount`：充值订单实付金额低于多少不参与返利。
3. `min_transfer_amount`：可用返利余额低于多少不能转入余额。

当前三者都没有。

推荐：

- 注册奖励不要设置 `min_reward_event_amount`，因为它本来就是固定奖励。
- 充值奖励建议设置 `min_source_payment_amount`，避免小额订单刷奖励。
- 提现/转余额建议设置 `min_transfer_amount`，例如可用返利满 10 才能转余额。
- 不建议设置“冻结到期但低于某金额不释放”，因为这会让 frozen/available 账务很难解释；门槛应放在转余额环节。

## 推荐的完整奖励事件模型

### 1. 奖励事件类型

建议至少拆成以下事件：

```text
invite_signup
recharge_completed
recharge_refunded
invite_count_milestone
recharge_invitee_count_milestone
```

每个事件独立计算，独立写 ledger，独立应用 cap。

这样可以满足：

1. 注册奖励上限不影响充值奖励。
2. 充值奖励上限不影响注册奖励。
3. 邀请关系继续绑定，不因奖励 cap 达到而停止。
4. 里程碑奖励不会和单次奖励混淆。
5. 退款只回滚和订单相关的充值奖励，不误伤注册奖励。

### 2. 奖励组件

建议把奖励组件拆成：

```text
signup_fixed_reward
signup_count_milestone_reward
recharge_first_order_reward
recharge_order_tier_reward
recharge_amount_tier_reward
recharge_percent_reward
recharge_invitee_count_milestone_reward
```

默认情况下，同一事件只允许一个主组件生效。

如果业务要叠加，例如“首单固定奖励 + 金额阶梯奖励”，必须在规则上显式允许叠加，并在 ledger 中分别记录。

### 3. 统一发奖伪逻辑

推荐把所有奖励都走一个统一入口：

```text
ApplyAffiliateRewardEvent(event):
    1. 校验 affiliate 总开关
    2. 找到 invitee 的 inviter
    3. 解析 inviter 的唯一生效 reward profile
    4. 找出 event 可用规则
    5. 执行冲突选择：exclusive_group / priority / stackable
    6. 计算原始奖励金额
    7. 读取并原子占用对应 cap
    8. 写 reward ledger，带 idempotency_key
    9. 根据 freeze 策略写 frozen 或 available
    10. 更新 user_affiliates 汇总字段
```

### 4. 注册奖励伪逻辑

当前 `BindInviter` 自己开事务并提交，见 `backend/internal/repository/affiliate_repo.go:53-82`。如果在它之后再发注册奖励，会出现“邀请关系已绑定，但奖励发放失败”的半成功状态。

因此应改成同一事务：

```text
BindInviterAndApplySignupReward(userID, affCode):
    tx begin
    ensure user affiliate
    ensure inviter affiliate
    UPDATE user_affiliates
        SET inviter_id = inviterID
        WHERE user_id = userID AND inviter_id IS NULL
    if affected = 0:
        tx commit
        return already_bound

    UPDATE inviter user_affiliates SET aff_count = aff_count + 1

    ApplyAffiliateRewardEventTx(
        event_type = invite_signup,
        inviter_id = inviterID,
        invitee_user_id = userID
    )

    tx commit
```

如果业务希望注册不被奖励系统阻断，可以用 outbox：

```text
tx 内只绑定关系并写 affiliate_reward_events
后台 worker 异步发奖励
失败可重试
```

不建议在绑定事务外直接同步发奖励，因为失败后不好补偿。

### 5. 充值奖励伪逻辑

当前返利发生在余额到账后、订单完成前，见 `backend/internal/service/payment_fulfillment.go:285-291`。如果返利失败，`ExecuteBalanceFulfillment` 会调用 `markFailed`，见 `backend/internal/service/payment_fulfillment.go:234-237` 和 `backend/internal/service/payment_fulfillment.go:504-517`，导致“用户余额已到账但订单 FAILED”。

充值奖励应从主支付履约中解耦：

```text
doBalance:
    1. 创建/兑换充值码，给用户余额到账
    2. 标记 payment order COMPLETED
    3. 写 affiliate_reward_events 或异步任务
    4. affiliate 奖励失败只影响 affiliate 状态，不影响支付订单完成状态
```

充值奖励事件应传完整订单：

```text
event_type = recharge_completed
payment_order_id
invitee_user_id
order_type
payment_type
provider_instance_id
pay_amount
credited_amount
completed_at
```

这样才能支持：

- 首单。
- 前 N 单。
- 金额阶梯。
- 支付渠道条件。
- 订单级幂等。
- 退款回滚。

### 6. 退款回滚伪逻辑

当前退款成功逻辑在 `backend/internal/service/payment_refund.go:376-387`，只更新订单退款状态并写 `REFUND_SUCCESS` audit，不处理 affiliate。

新增充值奖励后必须补：

```text
OnRefundSuccess(payment_order_id, refund_amount):
    找到该订单所有 recharge 奖励 ledger
    按退款比例或全额策略计算应回滚金额
    如果奖励仍 frozen:
        扣 aff_frozen_quota
    else if 奖励 available 未 transfer:
        扣 aff_quota
    else if 已 transfer:
        扣邀请人 balance 或写负余额/待扣款
    写 reversal ledger
    更新 reward counters
```

部分退款时的策略必须明确：

1. 按退款比例回滚。
2. 只要退款就全额回滚。
3. 退款后订单仍满足某个金额阶梯，则重算差额。

推荐默认：

- 固定首单奖励：只要订单退款到低于最低有效金额，就全额回滚。
- 百分比奖励：按退款比例回滚。
- 金额阶梯奖励：按退款后有效支付金额重新计算应得奖励，回滚差额。

## 推荐数据结构

### 方案 A：扩展现有表，改造成本较低

扩展 `user_affiliate_ledger`：

```text
event_type VARCHAR(32)          -- invite_signup / recharge / milestone / reversal
reward_component VARCHAR(64)    -- signup_fixed / recharge_first_order / amount_tier
profile_id BIGINT NULL
rule_id BIGINT NULL
payment_order_id BIGINT NULL
source_amount DECIMAL(20,8) NULL
source_amount_type VARCHAR(32) NULL -- pay_amount / credited_amount
status VARCHAR(32) NOT NULL DEFAULT 'available' -- frozen / available / transferred / reversed
idempotency_key VARCHAR(160) NOT NULL
reversed_ledger_id BIGINT NULL
metadata JSONB
```

新增唯一约束：

```text
UNIQUE(idempotency_key)
```

保留 `user_affiliates.aff_quota`、`aff_frozen_quota`、`aff_history_quota` 作为汇总余额。

优点：

- 改造成本较低。
- 兼容现有用户页面。
- 现有 transfer 逻辑可以复用，但要改成只处理 `status='available'` 的 ledger 汇总。

缺点：

- ledger 会承载更多含义，迁移需要非常谨慎。

### 方案 B：新增完整 reward ledger，长期更清晰

新增 `affiliate_reward_ledger`：

```text
id BIGSERIAL PRIMARY KEY
inviter_user_id BIGINT NOT NULL
invitee_user_id BIGINT NULL
event_type VARCHAR(32) NOT NULL
reward_component VARCHAR(64) NOT NULL
profile_id BIGINT NULL
rule_id BIGINT NULL
payment_order_id BIGINT NULL
source_amount DECIMAL(20,8) NULL
source_amount_type VARCHAR(32) NULL
calculated_amount DECIMAL(20,8) NOT NULL
granted_amount DECIMAL(20,8) NOT NULL
status VARCHAR(32) NOT NULL
frozen_until TIMESTAMPTZ NULL
idempotency_key VARCHAR(160) NOT NULL UNIQUE
reversed_ledger_id BIGINT NULL
metadata JSONB
created_at TIMESTAMPTZ
updated_at TIMESTAMPTZ
```

新增 `affiliate_reward_counters`：

```text
id BIGSERIAL PRIMARY KEY
profile_id BIGINT NULL
rule_id BIGINT NULL
inviter_user_id BIGINT NOT NULL
invitee_user_id BIGINT NULL
event_type VARCHAR(32) NOT NULL
metric VARCHAR(32) NOT NULL
period_start TIMESTAMPTZ NULL
period_end TIMESTAMPTZ NULL
used_count BIGINT NOT NULL DEFAULT 0
used_amount DECIMAL(20,8) NOT NULL DEFAULT 0
updated_at TIMESTAMPTZ
```

对计数行加唯一约束：

```text
UNIQUE(profile_id, rule_id, inviter_user_id, invitee_user_id, event_type, metric, period_start, period_end)
```

优点：

- 结构清晰。
- 注册奖励、充值奖励、里程碑、回滚都能审计。
- cap 可以通过 counter 行锁保证并发安全。

缺点：

- 改造量更大。
- 需要把现有 affiliate 页面和 transfer 逻辑迁移到新 ledger 汇总。

推荐：如果只是短期支持少量配置，采用方案 A；如果要做长期可运营的分佣/营销系统，采用方案 B。

## 上限与并发安全

当前 per-invitee cap 有并发超限风险。

代码事实：

1. `AccrueInviteRebate` 先查询历史累计，见 `backend/internal/service/affiliate_service.go:282-284`。
2. 查询 SQL 是简单 `SUM(amount)`，没有锁，见 `backend/internal/repository/affiliate_repo.go:137-152`。
3. 然后才调用 `AccrueQuota` 写入，见 `backend/internal/service/affiliate_service.go:301-308`。
4. `AccrueQuota` 更新邀请人的 `user_affiliates` 行并写 ledger，但没有重新检查 cap，见 `backend/internal/repository/affiliate_repo.go:94-127`。

新增这些上限时不能复制这个模式：

- 注册奖励人数上限。
- 注册奖励金额上限。
- 邀请人数里程碑。
- 充值奖励总额上限。
- 充值奖励订单数上限。
- 充值奖励人数上限。
- 单笔奖励上限。

推荐使用 counter 行锁：

```text
tx begin
SELECT * FROM affiliate_reward_counters
WHERE inviter_user_id = ?
  AND event_type = ?
  AND metric = ?
FOR UPDATE

计算 remaining
如果 remaining <= 0:
    写 skipped ledger 或不写奖励 ledger
否则:
    占用 used_count / used_amount
    写 reward ledger
    更新 user_affiliates 汇总
tx commit
```

或者使用单条条件 UPDATE：

```sql
UPDATE affiliate_reward_counters
SET used_amount = used_amount + :grant_amount
WHERE id = :counter_id
  AND used_amount + :grant_amount <= :cap
RETURNING used_amount;
```

关键原则：

1. cap 检查和 cap 占用必须在同一事务。
2. 奖励 ledger 和余额汇总也必须在同一事务。
3. 不能先 `SUM` 再写。
4. 必须为每个奖励事件设置 idempotency key。

## 当前方案直接实现的风险点

### R-01 注册即奖有白嫖风险

注册奖励是最高风险功能，因为被邀请人不需要支付成本。

当前代码只禁止同一 `user_id` 自邀，见 `backend/internal/service/affiliate_service.go:226-228`。没有 IP、设备、邮箱域、OAuth identity、支付账号等多账号风控。

如果直接上线注册即奖，攻击路径很明确：

```text
用户 A 获取 aff_code
批量注册用户 B/C/D
B/C/D 绑定 A
A 获得注册奖励
A 转入余额
```

上限能限制总损失，但不能阻止刷量。

建议：

1. 注册奖励金额设置较小。
2. 注册奖励默认冻结。
3. 注册奖励转余额设置最低门槛。
4. 同 IP/同设备/同邮箱域批量注册进入风控。
5. 可选：注册奖励先进入 pending，等被邀请人完成邮箱验证、账号存活 N 小时、或首次有效使用后再冻结/可用。
6. 如果业务坚持“注册即奖励”，也应至少要求邮箱验证成功后发奖。

### R-02 充值奖励不回滚会被退款套利

当前退款成功路径没有 affiliate 回滚。退款逻辑在 `backend/internal/service/payment_refund.go:275-387`，其中没有 affiliate 调用。

如果新增首单奖、阶梯奖、金额阶梯奖而不处理退款，攻击路径会更严重：

```text
A 邀请 B
B 充值高金额触发首单/阶梯奖励
A 获得奖励并转余额
B 申请退款
系统退回 B 的款，但不扣 A 的奖励
```

建议：

1. 充值奖励冻结期至少覆盖可退款窗口。
2. 退款成功必须回滚该订单相关的充值奖励。
3. ledger 必须记录 `payment_order_id`。
4. 若奖励已转余额，必须有负向余额、待扣款或人工审核策略。

### R-03 返利基数使用到账余额会放大奖励

当前余额订单创建时，`amount` 是充值倍率后的到账余额，见 `backend/internal/service/payment_order.go:51-58` 和 `backend/internal/service/payment_amounts.go:18-23`。

当前返利基数使用 `o.Amount`，见 `backend/internal/service/payment_fulfillment.go:397`。

如果充值倍率大于 1，按到账余额发奖励会高于按真实支付金额发奖励。

建议：

1. 规则中显式配置 `amount_base`。
2. 默认用 `pay_amount` 或原始支付金额。
3. 后台展示时同时展示实付金额、到账金额、奖励基数。

### R-04 支付履约不应被奖励失败拖失败

当前 `doBalance` 在余额到账后调用 affiliate，如果 affiliate 返回错误，`ExecuteBalanceFulfillment` 会 `markFailed`，见 `backend/internal/service/payment_fulfillment.go:234-237` 和 `backend/internal/service/payment_fulfillment.go:504-517`。

这会造成：

```text
用户余额已到账
订单却被标记 FAILED
```

新增更复杂规则后，affiliate 失败概率会增加，因此必须解耦：

1. 支付权益到账成功后先完成订单。
2. affiliate 奖励失败只记录 affiliate event failed。
3. 用后台任务重试奖励。

### R-05 多规则冲突会导致重复奖励

如果没有冲突模型，可能出现：

```text
首单固定奖励
首单金额阶梯奖励
默认百分比返利
专属用户百分比返利
全局活动奖励
```

同一笔订单全部命中，导致超预算。

建议默认规则：

1. 专属 profile 覆盖全局 profile，不叠加。
2. 同一 exclusive group 只取一条规则。
3. 同一阶梯维度禁止区间重叠。
4. 叠加必须显式配置。
5. 每笔 ledger 记录 `profile_id`、`rule_id`、`reward_component`。

### R-06 当前 `aff_history_quota` 不能继续作为所有营销统计依据

`aff_history_quota` 当前只是累计返利历史金额，`AccrueQuota` 每次发放都会增加，见 `backend/internal/repository/affiliate_repo.go:99-101`。

如果注册奖励、充值奖励、里程碑奖励、回滚都混入同一字段，它只能作为展示总额，不能作为规则判断依据。

规则判断必须基于结构化 ledger 或 counter。

## 推荐后台配置形态

建议把后台拆成三层。

### 1. 全局默认奖励方案

配置项：

```text
affiliate_enabled
default_reward_profile_id
min_transfer_amount
default_freeze_hours
auto_thaw_enabled
auto_transfer_enabled
```

当前已有 `affiliate_enabled` 和 `affiliate_rebate_freeze_hours`，见 `backend/internal/service/domain_constants.go:103-107`。

缺失：

- 默认 profile。
- 最低转余额金额。
- 自动 thaw job。
- 自动转余额。

### 2. 奖励方案 profile

一个 profile 包含：

```text
注册奖励：
- 是否启用
- 每注册奖励金额
- 注册奖励人数上限
- 注册奖励金额上限
- 注册奖励冻结小时
- 是否需要邮箱验证

累计邀请人数里程碑：
- milestone 人数
- 奖励金额
- 是否与注册奖励叠加

充值奖励：
- 是否启用
- 适用订单类型：balance / subscription / both
- 金额基数：pay_amount / credited_amount
- 最低充值金额
- 首单奖励
- 第 N 单阶梯
- 金额阶梯
- 默认百分比返利
- 单笔奖励上限
- 单个被邀请人累计奖励上限
- 邀请人累计充值奖励上限
- 可产生充值奖励的被邀请人数上限
- 充值奖励冻结小时
- 退款回滚策略
```

### 3. 用户专属方案

当前用户专属配置只有：

- 自定义邀请码。
- 专属充值返利比例。

建议扩展为：

```text
inviter_user_id -> reward_profile_id
```

这样不同用户可以有不同完整方案。

为避免冲突：

1. 一个用户同一时间只能有一个 active profile。
2. 专属 profile 覆盖 global default profile。
3. profile 内部规则保存时做冲突校验。
4. 每个用户页面展示自己当前生效方案的摘要。

## 与现有代码的改造点

### 后端 migration

需要新增 migration：

1. 扩展 `user_affiliate_ledger` 或新增 `affiliate_reward_ledger`。
2. 新增 `affiliate_reward_profiles`。
3. 新增 `affiliate_reward_rules`。
4. 新增 `affiliate_user_reward_profiles`。
5. 新增 `affiliate_reward_counters`。
6. 给幂等 key、profile 分配、订单奖励加唯一约束。

### `AffiliateRepository`

当前接口在 `backend/internal/service/affiliate_service.go:97-113`。

需要新增：

```go
ResolveRewardProfile(ctx, inviterID)
ListRewardRules(ctx, profileID, event)
ApplyRewardLedgerTx(ctx, grant)
ReserveRewardLimitTx(ctx, scope, metric, amount, count)
ReverseRewardsForOrderTx(ctx, paymentOrderID, refundAmount)
GetRechargeOrderIndex(ctx, inviterID, inviteeUserID, orderID)
ClaimRechargeInviteeTx(ctx, inviterID, inviteeUserID, profileID)
```

现有 `AccrueQuota` 只能作为底层“加可用/冻结额度”的子步骤，不应继续承载所有规则。

### `AffiliateService`

当前核心函数是 `AccrueInviteRebate`，见 `backend/internal/service/affiliate_service.go:240-308`。

建议改造为：

```go
ApplySignupReward(ctx, inviterID, inviteeUserID)
ApplyRechargeRewardForOrder(ctx, order)
ApplyMilestoneRewards(ctx, inviterID, event)
ReverseRechargeRewardsForRefund(ctx, order, refund)
```

并新增统一规则评估器：

```go
EvaluateAffiliateRewardRules(profile, event) []RewardGrant
```

### `AuthService`

注册路径当前在用户创建后调用 `BindInviterByCode`，见 `backend/internal/service/auth_service.go:229-238` 和 `backend/internal/service/auth_service.go:783-797`。

如果注册奖励是同步发放，应将绑定和奖励放入同一事务。  
如果注册奖励异步发放，应在绑定事务内写 outbox event。

### `PaymentService`

当前余额履约在 `doBalance` 内直接调用 affiliate，见 `backend/internal/service/payment_fulfillment.go:265-292`。

建议：

1. 支付订单完成和 affiliate 发奖解耦。
2. `applyAffiliateRebateForOrder` 改为写 event 或异步任务。
3. 传完整 order 给 affiliate，不再只传 `o.UserID` 和 `o.Amount`。
4. affiliate 失败不得让 payment order 失败。

### `PaymentRefund`

当前退款成功在 `backend/internal/service/payment_refund.go:376-387`。

需要增加：

```go
affiliateService.ReverseRechargeRewardsForRefund(ctx, p.Order, p.RefundAmount)
```

推荐在退款成功后执行。  
如果 reversal 失败，不能简单把退款失败，因为网关退款可能已成功；应记录待处理补偿任务。

### 管理端接口和前端

当前 admin affiliate handler 只管理用户专属码和比例，见 `backend/internal/handler/admin/affiliate_handler.go:28-155`。

需要新增：

```text
GET    /api/v1/admin/affiliate-reward/profiles
POST   /api/v1/admin/affiliate-reward/profiles
PUT    /api/v1/admin/affiliate-reward/profiles/:id
DELETE /api/v1/admin/affiliate-reward/profiles/:id
POST   /api/v1/admin/affiliate-reward/profiles/:id/rules
PUT    /api/v1/admin/affiliate-reward/rules/:id
POST   /api/v1/admin/affiliate-reward/users/:user_id/profile
DELETE /api/v1/admin/affiliate-reward/users/:user_id/profile
GET    /api/v1/admin/affiliate-reward/ledger
GET    /api/v1/admin/affiliate-reward/stats
```

前端需要新增 profile/rule 管理 UI。  
现有设置页中的 `affiliate_rebate_rate` 可以作为兼容模式保留，也可以迁移成默认 profile 的一条充值百分比规则。

## 迁移策略

### 阶段 1：保持现有行为不变

1. 新增结构化 ledger 字段或新表。
2. 把现有全局比例迁移为默认 profile 的 `recharge_percent_reward` 规则。
3. 把 `aff_rebate_rate_percent` 迁移为用户专属 profile 或用户 override。
4. 保持 `/user/aff` 和 `/user/aff/transfer` 行为不变。

### 阶段 2：实现注册奖励

1. 新增 `invite_signup` 事件。
2. 注册绑定成功后生成注册奖励。
3. 加注册奖励人数 cap 和金额 cap。
4. 加注册奖励冻结。
5. 加注册奖励幂等。

### 阶段 3：实现充值阶梯

1. `ApplyRechargeRewardForOrder` 接收完整订单。
2. 支持首单、订单序号阶梯、金额阶梯。
3. 支持充值奖励总额 cap。
4. 支持充值奖励 distinct invitee cap。
5. 支持单笔 cap。

### 阶段 4：实现退款回滚和风控

1. 退款成功后按订单回滚奖励。
2. 支持部分退款重算。
3. 支持冻结期覆盖退款窗口。
4. 增加同 IP/设备/邮箱域/支付账号风控。

### 阶段 5：可观测性

1. 后台按订单查看奖励。
2. 后台按邀请人查看注册奖励、充值奖励、冻结、可用、已转余额、已回滚。
3. 导出 ledger。
4. 告警高风险邀请人。

## 最低可实现版本

如果要尽快支持你的三个核心场景，最低改造不应少于以下内容：

1. ledger 增加 `event_type`、`payment_order_id`、`rule_id`、`idempotency_key`、`status`。
2. 新增注册奖励配置：
   - `signup_reward_enabled`
   - `signup_reward_amount`
   - `signup_reward_count_cap`
   - `signup_reward_amount_cap`
   - `signup_reward_freeze_hours`
3. 新增充值奖励配置：
   - `recharge_reward_enabled`
   - `recharge_reward_base`
   - `first_order_reward_amount`
   - `recharge_reward_total_amount_cap`
   - `recharge_reward_invitee_count_cap`
4. 暂时不做复杂多阶梯 UI，先支持简单 JSON 配置。
5. 注册奖励和充值奖励分别统计 cap。
6. `PaymentService` 传完整订单给 affiliate。
7. 退款成功后按 `payment_order_id` 回滚充值奖励。
8. 专属用户只允许绑定一个 reward profile。

这个最低版本才能保证：

- 注册奖励不影响充值奖励。
- 充值奖励不影响注册奖励。
- 不同用户可以有不同规则。
- 同一个用户不会同时命中多套互相冲突的规则。
- 订单 retry 不重复发奖励。
- 退款后能找到并回滚奖励。

## 不能建议的实现方式

不建议方式一：在 `BindInviter` 后直接调用现有 `AccrueQuota` 发注册奖励。

原因：

- 注册奖励会写成普通 `accrue`。
- 会污染充值返利 cap。
- 没有注册奖励人数/金额独立上限。
- 绑定成功但发奖失败时状态不一致。

不建议方式二：继续只用 `aff_rebate_rate_percent` 承载用户专属规则。

原因：

- 它只能表示百分比。
- 无法表示固定奖励、阶梯、cap、冻结差异。
- 无法表达多个奖励组件。

不建议方式三：通过查询 ledger `SUM(amount)` 做所有上限。

原因：

- 当前 per-invitee cap 已经是这种模式，存在并发超限风险。
- 新增注册人数 cap、总金额 cap、充值人数 cap 时风险会扩大。

不建议方式四：让 affiliate 奖励失败导致支付订单失败。

原因：

- 当前已经存在“余额到账但订单 FAILED”的风险。
- 规则越复杂，affiliate 失败概率越高。
- 支付履约和营销奖励应分离。

## 最终判断

你的营销方案合理，但它已经超过当前“简单邀请返利”的模型能力。

基于当前代码，能直接复用的部分是：

1. 邀请关系绑定。
2. 唯一邀请码。
3. 邀请人数 `aff_count` 展示。
4. 全局开关。
5. 现有可用/冻结返利余额。
6. 手动转余额。
7. 邀请人专属比例的“覆盖全局”思路。

不能直接复用、必须改造的部分是：

1. 注册即奖。
2. 注册奖励人数/金额上限。
3. 邀请人数里程碑奖励。
4. 首单充值奖励。
5. 订单序号阶梯。
6. 金额阶梯。
7. 邀请人充值奖励总额上限。
8. 已奖励充值人数上限。
9. 不同用户完整奖励方案。
10. 规则冲突检测。
11. 退款回滚。
12. 并发安全 cap。
13. 自动 thaw 或自动到账。

推荐的核心改造方向是：

1. 把 affiliate 从“比例返利函数”升级为“奖励事件引擎”。
2. 把 `user_affiliate_ledger` 升级成结构化奖励 ledger，或新增独立 `affiliate_reward_ledger`。
3. 用 reward profile 解决不同邀请人的规则差异。
4. 用唯一 active profile 和规则互斥校验解决冲突。
5. 用 counter 行锁或原子 SQL 解决上限并发。
6. 用 `payment_order_id` 和 `idempotency_key` 解决订单幂等和退款回滚。
7. 把 affiliate 奖励从支付主履约中解耦，避免营销奖励失败污染支付订单状态。
