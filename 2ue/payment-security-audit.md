# Sub2API 支付系统安全审计报告

日期：2026-04-29  
代码库：`/Users/yuanfeijie/Desktop/procode/sub2api`  
审计对象：当前代码库支付创建订单、支付回调、订单查询、履约发放、退款、支付配置、支付前端调用与相关 Ent schema。  
当前提交：`b0a2252e`  
验证命令：`cd backend && go test -tags=unit ./internal/service ./internal/handler ./internal/payment/...`，结果全部通过。

## 总结结论

1. 确认未发现普通用户可以直接绕过真实支付后白嫖余额或订阅的代码路径。当前代码在订单创建、回调签名、金额校验、provider 绑定、履约幂等、退款扣减等关键位置都有服务端校验。
2. 确认未发现支付路径中基于用户输入的 SQL 注入、命令注入或模板注入代码路径。支付核心查询使用 Ent 谓词；支付履约里唯一相关原生 SQL 使用 `$1`、`$2` 参数绑定。
3. 确认存在高风险敏感信息泄漏：支付服务商配置新记录以明文 JSON 存在数据库；同时 admin 创建/更新服务商实例接口会返回包含 `config` 的 Ent 模型，配置内包含 `pkey`、私钥、API v3 key、Stripe secret/webhook secret 等敏感字段。
4. 确认存在中风险信息泄漏：用户侧订单接口直接返回 Ent `PaymentOrder` 模型，仅清空 `provider_snapshot`，仍会把订单拥有者自己的 `user_notes`、`recharge_code`、`payment_trade_no`、`provider_instance_id`、`client_ip`、`src_url` 等内部字段返回给前端。
5. 确认存在低风险匿名订单状态泄漏：`/payment/public/orders/verify` 无认证，仅凭 `out_trade_no` 返回订单金额、支付方式、状态、退款字段和套餐 ID。该接口不会触发上游查询和履约，因此不是支付绕过，但它是明确的信息暴露面。
6. 确认存在两个强化项：`out_trade_no` 使用 `math/rand/v2` 生成而不是 `crypto/rand`；取消订单频率限制查询失败时 fail open。它们没有形成支付白嫖链路，但降低了安全余量。

## 确认存在的风险

### F-01 高风险：支付服务商密钥明文落库，并在 admin 创建/更新响应中返回

结论：风险确认存在。支付服务商配置中的敏感字段被明文 JSON 写入 `payment_provider_instances.config`，数据库、备份、只读数据库账号、SQL 导出文件或应用响应日志拿到该字段后即可获得支付密钥材料。

代码事实：

- `backend/ent/schema/payment_provider_instance.go:39` 定义 `config` 为字符串字段，Postgres 类型是 `text`。
- 同一 schema 注释仍写着 `config 字段存储加密后的密钥信息`，见 `backend/ent/schema/payment_provider_instance.go:20`，但当前实现已经不加密。
- `backend/internal/service/payment_config_providers.go:418-426` 的注释明确写明“New records are plaintext JSON”。
- `backend/internal/service/payment_config_providers.go:461-469` 的 `encryptConfig` 只执行 `json.Marshal(cfg)` 并返回 `string(data)`，没有加密步骤。
- `backend/internal/service/payment_config_providers.go:105-117` 定义的敏感字段包括：
  - EasyPay：`pkey`
  - Alipay：`privateKey`、`publicKey`、`alipayPublicKey`
  - Wxpay：`privateKey`、`apiV3Key`、`publicKey`
  - Stripe：`secretKey`、`webhookSecret`
- `backend/internal/service/payment_config_providers.go:181-204` 的 `CreateProviderInstance` 调用 `encryptConfig(req.Config)` 后直接 `SetConfig(enc)` 保存。
- `backend/internal/handler/admin/payment_handler.go:254-268` 的 `CreateProvider` 将 `CreateProviderInstance` 返回的 Ent 实体直接 `response.Created(c, inst)`。
- `backend/ent/paymentproviderinstance.go:24-25` 的 Ent 模型字段 `Config string` 带有 `json:"config,omitempty"`，因此直接序列化会输出 `config`。
- `backend/internal/service/payment_config_providers.go:303-309` 的更新路径在配置变更时同样 `SetConfig(enc)`。
- `backend/internal/handler/admin/payment_handler.go:271-289` 的 `UpdateProvider` 也把 `UpdateProviderInstance` 返回的 Ent 实体直接 `response.Success(c, inst)`。
- admin 列表接口确实有 masking：`backend/internal/handler/admin/payment_handler.go:243-251` 调用 `ListProviderInstancesWithConfig`；`backend/internal/service/payment_config_providers.go:76-95` 会跳过敏感字段。但该 masking 没有覆盖创建和更新响应。

影响：

- 数据库明文泄漏后，攻击者可以直接读取支付服务商密钥材料。
- EasyPay 的 `pkey` 用于通知签名校验，见 `backend/internal/payment/provider/easypay.go:252` 和 `backend/internal/payment/provider/easypay.go:443-465`。泄漏该字段会破坏 EasyPay 回调签名可信度。
- Stripe 的 `webhookSecret` 用于 `webhook.ConstructEvent` 校验，见 `backend/internal/payment/provider/stripe.go:147-159`。泄漏该字段会破坏 Stripe webhook 真实性校验。
- Alipay/Wxpay 私钥和 API key 泄漏会扩大到支付商户侧风险，包括退款、查询、签名和商户资产安全风险。
- admin 创建/更新响应返回原始 `config`，会把密钥暴露给浏览器响应体、前端调试工具、反向代理响应日志、APM 采样日志和错误抓包链路。

利用前提：

- 读取数据库或数据库备份；或者读取 admin 创建/更新 provider 的 HTTP 响应、代理日志、应用响应日志。
- 该风险不需要普通用户权限；它属于密钥存储和 admin API 响应面泄漏。普通未认证用户不能直接调用 admin provider 接口，因为 `/admin/payment` 路由使用 `adminAuth`，见 `backend/internal/server/routes/payment.go:67-105`。

修复建议：

1. 恢复服务商配置加密存储，使用 KMS、系统密钥环或至少 AES-GCM；加密 key 不得与数据库备份一起存放。
2. 对现有明文 `payment_provider_instances.config` 做迁移：读取明文 JSON、加密后写回、记录迁移版本。
3. `CreateProvider` 和 `UpdateProvider` 不要返回 Ent 实体，改为复用 `ProviderInstanceResponse` 并执行 `decryptAndMaskConfig`，或者直接返回 `{ "id": ... }` / `{ "message": "updated" }`。
4. 修正 `PaymentProviderInstance` schema 注释和 `ProvideEncryptionKey` 相关说明，避免维护者误以为当前仍然加密。
5. 增加单元测试：创建 provider 后数据库中 `config` 不得出现原始 `pkey`/`secretKey`；创建/更新接口响应不得包含敏感字段。

### F-02 中风险：用户侧订单接口返回 Ent 模型，泄漏内部订单字段和 admin-only 用户备注

结论：风险确认存在。用户侧订单接口虽然校验了订单归属，但响应对象过宽，订单拥有者可以看到本不应由用户侧 API 返回的内部字段。

代码事实：

- 认证用户订单接口在 `/api/v1/payment/orders` 下，使用 JWT 和 user guard，见 `backend/internal/server/routes/payment.go:23-44`。
- `VerifyOrder` 调用 `sanitizePaymentOrderForResponse(order)` 返回，见 `backend/internal/handler/payment_handler.go:431-452`。
- `GetMyOrders` 调用 `sanitizePaymentOrdersForResponse(orders)` 返回，见 `backend/internal/handler/payment_handler.go:313-334`。
- `GetOrder` 调用 `sanitizePaymentOrderForResponse(order)` 返回，见 `backend/internal/handler/payment_handler.go:336-355`。
- `sanitizePaymentOrderForResponse` 只复制对象并清空 `ProviderSnapshot`，见 `backend/internal/handler/payment_handler.go:568-574`。
- Ent `PaymentOrder` JSON 字段包含大量内部字段，见 `backend/ent/paymentorder.go:21-99`，包括：
  - `user_id`、`user_email`、`user_name`、`user_notes`
  - `recharge_code`
  - `payment_trade_no`
  - `pay_url`、`qr_code`、`qr_code_img`
  - `provider_instance_id`、`provider_key`
  - `client_ip`、`src_host`、`src_url`
  - `failed_reason`
- 订单创建时会把用户备注写入订单：`SetNillableUserNotes(psNilIfEmpty(user.Notes))`，见 `backend/internal/service/payment_order.go:153-158`。
- 项目自己的 DTO 注释明确把 `Notes` 定义为 admin-only：`UserFromServiceAdmin` 注释写着 “It includes notes - user-facing endpoints must not use this”，见 `backend/internal/handler/dto/mappers.go:58-60`。
- 服务层确实限制了跨用户读取：`GetOrder` 检查 `o.UserID != userID` 后返回 forbidden，见 `backend/internal/service/payment_order.go:674-681`；`GetUserOrders` 使用 `paymentorder.UserIDEQ(userID)` 过滤，见 `backend/internal/service/payment_order.go:693-709`。

影响：

- 订单拥有者可以看到自己的 admin-only `user_notes`。这不是跨用户泄漏，但违反了现有 DTO 注释定义的权限边界。
- `recharge_code`、`payment_trade_no`、`provider_instance_id`、`provider_key`、`client_ip`、`src_url` 等字段会扩大前端可见内部实现细节。当前代码没有证据表明这些字段单独可完成白嫖，但它们会增加后续接口枚举、社工排查、日志关联和支付状态探测的可用信息。
- 前端类型 `frontend/src/types/payment.ts:75-96` 只声明了部分字段，不代表后端没有返回其它 Ent JSON 字段。

利用前提：

- 攻击者需要是订单拥有者并已登录。
- 该问题不会让 A 用户读取 B 用户订单，因为服务层已有归属校验。

修复建议：

1. 用户侧接口不要返回 Ent 模型，定义明确 DTO，例如 `UserPaymentOrderDTO`。
2. 用户 DTO 仅保留业务必需字段：`id`、`out_trade_no`、`amount`、`pay_amount`、`fee_rate`、`payment_type`、`order_type`、`status`、`created_at`、`expires_at`、`paid_at`、`completed_at`、必要退款状态和 `plan_id`。
3. 从用户 DTO 删除 `user_notes`、`recharge_code`、`payment_trade_no`、`client_ip`、`src_host`、`src_url`、`provider_key`、`provider_snapshot`。
4. `provider_instance_id` 当前被前端用来判断用户退款按钮，见 `frontend/src/views/user/UserOrdersView.vue:175-184`。更稳妥的做法是后端直接返回 `can_request_refund: true/false`，避免暴露 provider 实例 ID。
5. 增加 API 测试：用户订单响应不得包含 admin notes、recharge code、client IP、provider snapshot 和 provider internal fields。

### F-03 低风险：匿名 public out_trade_no 查询暴露有限订单状态

结论：风险确认存在。`/payment/public/orders/verify` 是无认证接口，只要知道 `out_trade_no` 就能查询有限订单状态。该接口不会触发上游支付查询，因此不是白嫖入口。

代码事实：

- public payment endpoints 没有挂 JWT，见 `backend/internal/server/routes/payment.go:46-54`。
- 注释明确说明 `/payment/public/orders/verify` 是 legacy anonymous out_trade_no verify endpoint，见 `backend/internal/server/routes/payment.go:46-49`。
- `VerifyOrderPublic` 只接收 `out_trade_no` 并返回 `buildPublicOrderResult(order)`，见 `backend/internal/handler/payment_handler.go:500-515`。
- `buildPublicOrderResult` 返回 `id`、`out_trade_no`、`amount`、`pay_amount`、`fee_rate`、`payment_type`、`order_type`、`status`、`created_at`、`expires_at`、`paid_at`、`completed_at`、退款字段和 `plan_id`，见 `backend/internal/handler/payment_handler.go:454-497`。
- 服务层 `VerifyOrderPublic` 明确只返回已持久化状态，不触发上游查询，见 `backend/internal/service/payment_order_lifecycle.go:283-298`。
- `out_trade_no` 输入会做长度和字符集限制，见 `backend/internal/service/payment_order_lifecycle.go:300-318`。

影响：

- 任何拿到 `out_trade_no` 的人都可以匿名查询订单金额、支付方式、状态、退款状态和套餐 ID。
- `out_trade_no` 会出现在支付跳转返回 URL 中：`buildPaymentReturnURL` 会写入 `out_trade_no`，见 `backend/internal/service/payment_resume_service.go:291-300`。因此浏览器历史、截图、日志、Referer 链路中泄漏该值后，匿名查询成立。
- 该接口不会执行 `checkPaid`，不会向支付上游确认付款，也不会触发履约；代码事实不支持把它认定为支付绕过或白嫖路径。

修复建议：

1. 废弃 legacy public verify，统一使用 `/payment/public/orders/resolve` 的 HMAC resume token。
2. 如果必须保留，返回字段收缩到 `status`、`expires_at`、`paid_at`，移除 `amount`、`pay_amount`、`refund_*`、`plan_id`。
3. 为 public verify 加 IP 和 out_trade_no 维度限流。
4. 对前端 legacy fallback 增加版本过渡期，过渡后删除匿名查询。

### F-04 低风险强化项：订单号随机源和取消限流 fail open 降低安全余量

结论：这两项风险确认存在于代码，但未形成支付白嫖链路。

代码事实：

- `generateOutTradeNo` 格式为 `sub2_` + 日期 + 8 位随机字符，见 `backend/internal/service/payment_service.go:55-60`。
- 随机字符使用 `math/rand/v2` 的 `rand.IntN`，见 `backend/internal/service/payment_service.go:63-69`。
- `allocateOutTradeNo` 在数据库里检查唯一性，最多重试 5 次，见 `backend/internal/service/payment_order.go:200-212`。
- 取消频率限制查询 audit log 失败时记录错误后 `return nil`，明确 fail open，见 `backend/internal/service/payment_order_lifecycle.go:31-46`。
- 即使取消限流 fail open，订单创建仍有 pending 上限，见 `backend/internal/service/payment_order.go:215-227`。

影响：

- `math/rand/v2` 不是加密随机源。当前订单号空间为 62^8，再叠加日期和数据库唯一性检查；本次审计没有代码证据证明可被实际预测并绕过支付，但 public verify 存在时，订单号安全性更重要。
- 取消限流 fail open 会在 audit log 查询异常时允许继续创建/取消订单，属于运营层滥用风险，不是免费履约风险。

修复建议：

1. 将 `generateRandomString` 改为 `crypto/rand`。
2. public verify 保留期间增加查询限流，降低订单号探测价值。
3. 取消限流查询失败时改为 fail closed，或至少降级到更严格的 pending order 限制。

## 确认未发现的支付绕过和白嫖路径

### 1. 普通用户支付接口需要认证

- 用户支付接口 `/payment` 挂载 `jwtAuth` 和 `BackendModeUserGuard`，见 `backend/internal/server/routes/payment.go:23-44`。
- admin 支付接口 `/admin/payment` 挂载 `adminAuth`，见 `backend/internal/server/routes/payment.go:67-105`。
- webhook 和 public endpoints 不走登录态，但 webhook 依赖 provider 签名，public verify 不触发履约。

结论：确认未发现未登录普通用户直接创建订单、取消他人订单、查询他人完整订单、申请退款或执行 admin 支付操作的路由路径。

### 2. 客户端不能决定实际套餐价格和实际付款金额

- 创建订单时读取支付配置并检查支付系统开启，见 `backend/internal/service/payment_order.go:30-36`。
- 余额充值要求 `amount > 0` 且非 NaN/Inf，并检查最小/最大金额，见 `backend/internal/service/payment_order.go:90-104`。
- 订阅订单要求 `plan_id`，读取服务端套餐并要求 `ForSale`，见 `backend/internal/service/payment_order.go:107-114`。
- 订阅订单金额使用 `plan.Price` 覆盖客户端金额，见 `backend/internal/service/payment_order.go:51-55`。
- 余额订单入账金额由服务端按 `BalanceRechargeMultiplier` 计算，见 `backend/internal/service/payment_order.go:56-58` 和 `backend/internal/service/payment_amounts.go:18-23`。
- 实际支付金额使用 `limitAmount` 和服务端费率计算，见 `backend/internal/service/payment_order.go:59-61` 和 `backend/internal/payment/fee.go:11-18`。
- provider 创建支付请求使用服务端生成的 `outTradeNo` 和 `payAmountStr`，见 `backend/internal/service/payment_order.go:406-413` 和 `backend/internal/service/payment_order.go:448-459`。

结论：确认未发现客户端通过篡改 `amount`、`plan_id` 或 `fee` 直接降低应付金额并获得更高权益的代码路径。

### 3. 支付回调必须通过 provider 签名校验

- webhook handler 读取原始 body 后先解析 provider candidates，再调用 `provider.VerifyNotification`，见 `backend/internal/handler/payment_webhook_handler.go:63-137`。
- EasyPay 校验 `sign`，用配置 `pkey` 重新计算签名并用 `hmac.Equal` 比较，见 `backend/internal/payment/provider/easypay.go:238-272` 和 `backend/internal/payment/provider/easypay.go:443-465`。
- Alipay 调用 SDK 的 `client.DecodeNotification` 解码并验签，见 `backend/internal/payment/provider/alipay.go:262-312`。
- Wxpay 调用官方 notify handler `ParseNotifyRequest`，见 `backend/internal/payment/provider/wxpay.go:426-457`。
- Stripe 要求 `webhookSecret` 和 `stripe-signature`，使用 `webhook.ConstructEvent`，见 `backend/internal/payment/provider/stripe.go:143-170`。

结论：确认未发现绕过 provider 签名直接伪造成功支付回调的代码路径。F-01 中的密钥明文存储会破坏这一前提，所以它是高风险。

### 4. 回调绑定原订单 provider 实例和商户身份

- webhook provider 解析优先按 `out_trade_no` 查订单，再使用订单绑定的 provider instance，见 `backend/internal/service/payment_webhook_provider.go:32-66`。
- 对 pinned provider instance 的订单，`getPinnedOrderProvider` 强制加载原实例，见 `backend/internal/service/payment_webhook_provider.go:85-94`。
- 非 Wxpay 的 fallback 只有在同 provider key 启用实例数量 `<= 1` 时允许，见 `backend/internal/service/payment_webhook_provider.go:96-113`。
- 订单创建时快照 provider instance、provider key、payment mode、商户 appid/mchid/pid 等信息，见 `backend/internal/service/payment_order.go:230-277`。
- 回调确认时检查 expected provider key，不一致会写审计并返回错误，见 `backend/internal/service/payment_fulfillment.go:76-88`。
- 回调确认时检查 snapshot metadata，例如 Wxpay appid/mchid/currency/trade_state、Alipay app_id、EasyPay pid，见 `backend/internal/service/payment_order_provider_snapshot.go:129-194`。

结论：确认未发现把 A provider、A 商户或 A 实例的成功回调套到 B provider、B 商户或 B 实例订单上的代码路径。

### 5. 履约前校验支付金额，防止少付多得

- `confirmPayment` 要求 provider 金额大于 0 且非 NaN/Inf，见 `backend/internal/service/payment_fulfillment.go:96-113`。
- provider 已付金额和订单 `PayAmount` 差额超过 `0.01` 即拒绝，见 `backend/internal/service/payment_fulfillment.go:104-107`。
- 主动查询上游 `checkPaid` 也会拒绝无效金额，并做一次重查，见 `backend/internal/service/payment_order_lifecycle.go:152-166`。
- public legacy verify 不触发上游查询，见 `backend/internal/service/payment_order_lifecycle.go:283-298`。
- resume token public resolve 可以触发上游查询，但 token 有 HMAC 签名、过期时间和订单/provider 匹配校验，见 `backend/internal/service/payment_resume_lookup.go:12-59` 和 `backend/internal/service/payment_resume_service.go:332-476`。

结论：确认未发现少付、0 元支付、NaN/Inf 金额或 public verify 触发履约的支付白嫖路径。

### 6. 履约具备幂等锁，重复回调不会重复发放

- `toPaid` 只允许 `PENDING`、`CANCELLED`、近期 `EXPIRED` 状态更新成 `PAID`，见 `backend/internal/service/payment_fulfillment.go:134-171`。
- 如果更新影响行数为 0，会进入 `alreadyProcessed`，已完成或已退款直接返回，见 `backend/internal/service/payment_fulfillment.go:173-199`。
- 余额履约要求订单状态为 `PAID` 或 `FAILED`，并通过 `PAID/FAILED -> RECHARGING` 更新作为锁，见 `backend/internal/service/payment_fulfillment.go:213-239`。
- 订阅履约同样使用 `PAID/FAILED -> RECHARGING` 更新作为锁，见 `backend/internal/service/payment_fulfillment.go:308-337`。
- 余额履约使用订单的 `RechargeCode` 创建兑换码并兑换，见 `backend/internal/service/payment_fulfillment.go:265-292`。
- 兑换码表 `code` 唯一，见 `backend/ent/schema/redeem_code.go:38-41`。
- `Redeem` 在事务内先用 `WHERE status = unused` 把兑换码标记为已用，再更新用户余额，见 `backend/internal/service/redeem_service.go:296-367` 和 `backend/internal/repository/redeem_code_repo.go:209-225`。
- 订阅履约用 `SUBSCRIPTION_SUCCESS` 审计日志判断是否已经分配，防止重试时重复延期，见 `backend/internal/service/payment_fulfillment.go:339-357`。

结论：确认未发现重复 webhook、重复 verify 或并发履约导致余额/订阅重复发放的代码路径。

### 7. 订单读取、取消和用户退款申请有归属校验

- `GetOrder` 检查 `o.UserID != userID` 后拒绝，见 `backend/internal/service/payment_order.go:674-681`。
- `GetUserOrders` 使用 `paymentorder.UserIDEQ(userID)` 过滤，见 `backend/internal/service/payment_order.go:693-709`。
- `VerifyOrderByOutTradeNo` 查到订单后检查 `o.UserID != userID`，见 `backend/internal/service/payment_order_lifecycle.go:255-268`。
- `CancelOrder` 检查订单归属和 `PENDING` 状态，见 `backend/internal/service/payment_order_lifecycle.go:93-105`。
- `RequestRefund` 先调用 `validateRefundRequest`，后者检查订单归属、余额订单、`COMPLETED` 状态和 provider 是否允许用户退款，见 `backend/internal/service/payment_refund.go:150-199`。
- 用户退款申请只把状态改成 `REFUND_REQUESTED`，真正网关退款由 admin 路径执行，admin 路由受 `adminAuth` 保护，见 `backend/internal/server/routes/payment.go:67-105` 和 `backend/internal/service/payment_refund.go:201-247`。

结论：确认未发现普通用户通过订单 ID 或 `out_trade_no` 操作他人订单、取消他人订单或替他人申请退款的代码路径。

### 8. 退款路径限制原 provider 实例、金额和余额扣减

- 用户退款申请仅允许已完成的余额订单，并要求用户余额不低于订单入账金额，见 `backend/internal/service/payment_refund.go:150-165`。
- admin 退款只允许 `COMPLETED`、`REFUND_REQUESTED`、`REFUND_FAILED`，见 `backend/internal/service/payment_refund.go:201-209`。
- admin 退款要求订单有明确 provider instance 且该实例开启退款，见 `backend/internal/service/payment_refund.go:210-222`。
- 退款金额拒绝 NaN/Inf，默认全额，且不能超过订单入账金额，见 `backend/internal/service/payment_refund.go:223-240`。
- 网关退款金额按入账金额和支付金额比例换算，见 `backend/internal/service/payment_amounts.go:25-37`。
- 网关退款使用订单原始 provider instance，并再次校验 provider snapshot metadata，见 `backend/internal/service/payment_refund.go:324-361`。
- 网关退款失败后会回滚余额/订阅扣减，见 `backend/internal/service/payment_refund.go:364-405`。

结论：确认未发现普通用户通过退款接口直接拿回网关款项、超额退款或退款后不扣权益的代码路径。

### 9. return_url 开放重定向控制有效

- `CanonicalizeReturnURL` 要求 `return_url` 是绝对 URL，scheme 只能是 `http` 或 `https`，见 `backend/internal/service/payment_resume_service.go:235-246`。
- 路径必须精确等于 `/payment/result`，见 `backend/internal/service/payment_resume_service.go:247-253`。
- host 必须等于当前 request host 或 Referer host，见 `backend/internal/service/payment_resume_service.go:254-274`。
- 构造支付返回 URL 时清除 fragment，并只追加订单字段和 `status=success`，见 `backend/internal/service/payment_resume_service.go:276-305`。
- resume token 使用 HMAC-SHA256 签名，解析时校验签名和过期时间，见 `backend/internal/service/payment_resume_service.go:332-476`。

结论：确认未发现支付 `return_url` 被普通用户设置为外部钓鱼站点的开放重定向路径。

## 注入风险结论

结论：确认未发现支付系统中的 SQL 注入、命令注入和模板注入路径。

代码事实：

- 支付订单查询、更新、列表过滤主要使用 Ent 谓词，例如：
  - `paymentorder.OutTradeNo(...)`：`backend/internal/service/payment_fulfillment.go:36`
  - `paymentorder.UserIDEQ(userID)`：`backend/internal/service/payment_order.go:693-709`
  - `paymentorder.OutTradeNoContainsFold(p.Keyword)`：`backend/internal/service/payment_order.go:731-736`
  - `paymentproviderinstance.ProviderKeyEQ(providerKey)`：`backend/internal/service/payment_webhook_provider.go:102-107`
- 支付履约中唯一相关原生 SQL 在 `tryClaimAffiliateRebateAudit`，SQL 文本固定，参数通过 `$1`、`$2` 绑定，见 `backend/internal/service/payment_fulfillment.go:443-462`。
- `out_trade_no` public lookup 做了长度和字符集校验，见 `backend/internal/service/payment_order_lifecycle.go:300-318`。
- provider 请求参数使用 SDK 或 `url.Values`/map 传参；本次审计未发现把支付用户输入拼接到 shell 命令、SQL 字符串或模板执行中的代码路径。
- 仓库中存在 `exec.CommandContext`，但命中点是更新、备份、测试等非支付路径；支付路径未调用系统命令。

## 测试结果

已执行：

```bash
cd /Users/yuanfeijie/Desktop/procode/sub2api/backend
go test -tags=unit ./internal/service ./internal/handler ./internal/payment/...
```

结果：

- `github.com/Wei-Shaw/sub2api/internal/service`：ok
- `github.com/Wei-Shaw/sub2api/internal/handler`：ok
- `github.com/Wei-Shaw/sub2api/internal/payment`：ok
- `github.com/Wei-Shaw/sub2api/internal/payment/provider`：ok

这些测试覆盖了 payment fulfillment、provider snapshot、webhook provider 解析、resume token、public order lookup、refund 等路径。测试通过不等于不存在漏洞；本报告的漏洞结论以上文列出的代码事实为准。

## 优先级修复清单

1. 最高优先级：修复 provider config 明文落库和 create/update 响应泄漏。先停止 admin 响应返回原始 Ent provider instance，再做数据库加密迁移。
2. 高优先级：用户订单接口改成显式 DTO，删除 `user_notes`、`recharge_code`、`payment_trade_no`、`client_ip`、`src_url`、provider 内部字段。
3. 中优先级：废弃或收缩 `/payment/public/orders/verify`，保留期间加限流，并改用 signed resume token 作为公开恢复路径。
4. 中优先级：把 `out_trade_no` 随机源改为 `crypto/rand`，并增加订单号生成测试。
5. 中优先级：取消限流查询失败时不要 fail open，或者在失败时进入更严格的订单创建保护。
6. 测试补强：新增 provider config 加密/响应脱敏测试、用户订单 DTO 字段白名单测试、public verify 响应字段白名单测试。

