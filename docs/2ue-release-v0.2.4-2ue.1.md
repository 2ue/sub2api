# v0.2.4-2ue.1 发布说明

基于上游 `v0.2.4` 的 2ue 第 1 次发布。

- GitHub Release: https://github.com/2ue/sub2api/releases/tag/v0.2.4-2ue.1
- 镜像: `ghcr.io/2ue/sub2api:0.2.4-2ue.1`（同时推送了 `latest`）

## 本次内容

### 新增：远程代管模式

本部署的管理面可以查看和管理**另一套 sub2api** 的数据，**目标端无需任何改造**。

设计要点：

- 转发目标只来自启动配置 `remote_proxy.backend_url`，**不可由请求指定**，因此不会形成任意 URL 转发器
- 浏览器始终只访问本站同源地址，由服务端携带 Admin API Key 转发，**全程不涉及跨域**
- 远程会话签发独立的随机令牌，不复用本地 JWT——否则远程会话会与某个本地管理员共用刷新令牌家族（一侧撤销会踢掉另一侧），且会凭空获得本部署的管理员权限
- `/api/v1/remote/info` 不回传目标地址。该接口无需鉴权，回传等于向匿名访问者公开被代管的是哪套部署

### 同步上游

合并上游 main 至 `v0.2.4`，包含 WebAuthn/Passkey、面板限流、模型广场等。

## 配置更新（重点）

新增一组配置项，**全部可选，默认关闭**。不配置时行为与上游完全一致，老部署升级无需改动任何配置。

### 配置文件方式

```yaml
remote_proxy:
  enabled: false
  # 目标部署地址，例如 https://sub2api2.example.com
  # 只取 scheme://host，不要带 /api/v1 等路径前缀
  backend_url: ""
  # 转发超时（秒），0 或留空表示使用默认值 60
  timeout: 60
```

### 环境变量方式

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `REMOTE_PROXY_ENABLED` | `false` | 是否启用远程代管 |
| `REMOTE_PROXY_BACKEND_URL` | 空 | 目标部署地址，`enabled=true` 时**必填** |
| `REMOTE_PROXY_TIMEOUT` | `60` | 转发超时（秒） |

环境变量优先级高于配置文件。命名规则是配置路径的 `.` 换成 `_` 后全大写。

### 校验规则

`enabled=true` 时启动期会校验 `backend_url`，不合法直接 panic（配置错误在启动期暴露，优于运行期才发现管理面不可用）：

- 不能为空
- 必须是合法 URL，scheme 只能是 `http` 或 `https`
- 必须有主机名

配置里即使写了路径（如 `https://x.com/api/v1`），也只会保留 `scheme://host`，路径在转发时按请求路径拼接，避免出现重复前缀。

## 部署

```yaml
# docker-compose.yml
services:
  sub2api:
    image: ghcr.io/2ue/sub2api:0.2.4-2ue.1
    environment:
      - REMOTE_PROXY_ENABLED=true
      - REMOTE_PROXY_BACKEND_URL=https://sub2api2.example.com
      - REMOTE_PROXY_TIMEOUT=60
```

```bash
docker compose up -d
```

## 使用

启用后：

- `/login` **保持上游原样**，本地用户的邮箱密码登录不受影响
- 管理员访问 `/remote-login`，输入**目标部署**的管理员 API Key 登录
- 登录后固定进入 `/admin/dashboard`，看到的是目标部署的数据

`/remote-login` **刻意不做任何入口链接**，不会出现在导航或 `/login` 上，地址需由部署方自行告知。未启用远程代管时访问该路径会跳回 `/login`，不暴露本部署是否具备该能力。

## 注意事项

**启用后本地管理面不可用。** 该模式下 `/admin/*` 全部转发到目标端，不再注册任何本地管理路由（gin 不允许同一前缀既有通配路由又有具体路由），支付与页面路由同样跳过。**要管理本部署自己的数据，需关闭该模式后重启。**

**远程会话没有刷新令牌。** 会话过期后需重新登录，不会自动续期。

**Admin API Key 等同目标端管理员权限。** 该 Key 只在服务端内存中保存，不落库、不回传前端，但仍应按管理员凭据对待。

## 版本号规则

```
上游 v0.2.4 基础上：
  第 1 次发布  v0.2.4-2ue.1
  第 2 次发布  v0.2.4-2ue.2
上游发布 v0.2.5，同步后：
  第 1 次发布  v0.2.5-2ue.1   ← N 归 1
```

`-2ue.N` 按 semver 排序低于对应的上游正式版，永远落在上游两个版本之间的空隙里，不会与上游 tag 冲突。带连字符后缀会被 goreleaser 标记为 GitHub Pre-release（`prerelease: auto`），但 `latest` 镜像标签仍会正常推送。

## 发布流程副作用

发布成功后 `sync-version-file` job 会往**仓库默认分支**（当前是 `main`）推一条 `chore: sync VERSION to <版本> [skip ci]`。本次发布已产生 `a0232fd95`，`main` 因此偏离 `upstream/main`。

下次同步上游时 `git merge --ff-only upstream/main` 会失败，需要：

```bash
git switch main
git reset --hard upstream/main
git push origin main --force-with-lease
```

若要避免，可把 `.github/workflows/release.yml` 中 `sync-version-file` 的 checkout `ref` 与最后一行 push 目标都改为 `2ue-main`。
