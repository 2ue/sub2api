# 2ue Fork AI 协作、同步、开发与发布方案

本文定义 `2ue/sub2api` 作为 `Wei-Shaw/sub2api` fork 的长期维护流程。目标是让 `main` 永远保持上游镜像，同时让 `2ue-main` 承载本仓库迭代、AI 自动化、发布和本仓库特有能力。

这份文档属于 `2ue-main` 分支。不要把它合入 `main`，因为 `main` 的职责是镜像上游仓库。

## 分支职责

| 分支 | 职责 | 允许内容 | 禁止内容 |
| --- | --- | --- | --- |
| `main` | 镜像 `upstream/main` | 上游代码的 fast-forward 同步 | 本仓库功能、fork 发布配置、2ue 文档、release tag 回写 |
| `2ue-main` | 本仓库默认集成与发布分支 | 上游代码、本仓库功能、fork 发布 workflow、AI 规则、2ue 文档 | 直接给上游 PR 的混杂提交 |
| `pr/*` 或 `upstream/*` | 给上游仓库提 PR 的干净分支 | 从 `main` 切出的单一上游可接受改动 | 2ue 专属配置、release workflow、fork 镜像名、本仓库私有文档 |
| `feature/2ue-*` | 本仓库特有功能开发分支 | 从 `2ue-main` 切出的本仓库功能 | 直接合入 `main` |

建议将 GitHub 默认分支设置为 `2ue-main`。`main` 仍然存在，并且仍然只用于镜像上游；默认分支切到 `2ue-main` 后，本仓库的 Actions、发布说明和 AI 规则都可以留在产品分支里，不需要污染 `main`。

## 不变量

AI 和维护者都必须遵守以下规则：

1. `main` 只能通过 `upstream/main` fast-forward 更新。
2. 不允许把本仓库功能直接提交或合并到 `main`。
3. 不允许把 `2ue-main` 合回 `main`。
4. 给上游提 PR 的分支必须从 `main` 切出。
5. 本仓库特有功能必须从 `2ue-main` 切出，并且只合回 `2ue-main`。
6. 本仓库发布 tag 必须打在 `2ue-main` 的提交上。
7. 本仓库发布 tag 使用 `vX.Y.Z-2ue.N`，例如 `v0.1.133-2ue.1`。
8. 不允许在 `origin` 推送普通上游 release tag 触发本仓库发布，例如 `v0.1.133`。
9. 不允许 force push `main`、`2ue-main` 或已发布 tag。
10. 如果 `main` 不能 fast-forward 到 `upstream/main`，必须停止并分析，不能自动 reset。

`main` 和 `2ue-main` 产生分歧是预期行为。`2ue-main` 会包含本仓库文档、AI 规则、发布 workflow、fork 专属功能和 release 调整；这些不应出现在 `main`。

## 1. 同步上游到 main

目标：让 `origin/main` 等于 `upstream/main`。

```bash
git fetch upstream --prune
git fetch origin --prune
git switch main
git merge --ff-only upstream/main
git push origin main
```

如果 `git merge --ff-only upstream/main` 失败，说明 `main` 已经偏离上游。此时必须停止，检查 `main` 上是否存在本仓库提交。禁止 AI 自动执行 `git reset --hard` 或强推。

不要使用：

```bash
git push origin --tags
```

上游普通 tag 可能触发本仓库 release workflow。本仓库只推送 `vX.Y.Z-2ue.N` 形式的 fork release tag。

## 2. 合并最新 main 到 2ue-main

目标：把最新上游代码带到本仓库集成分支。

首次同步或 `2ue-main` 没有独有提交时，可以 fast-forward：

```bash
git switch 2ue-main
git pull --ff-only origin 2ue-main
git merge --ff-only origin/main
git push origin 2ue-main
```

当 `2ue-main` 已经有本仓库独有提交后，使用 merge commit 保留历史：

```bash
git switch 2ue-main
git pull --ff-only origin 2ue-main
git merge --no-ff origin/main -m "merge: upstream main into 2ue-main"
```

如果发生冲突：

1. 先理解上游和 `2ue-main` 双方语义。
2. 不允许为了快速解决冲突直接丢弃一边。
3. 上游修复和本仓库增强都要保留，除非明确确认其中一方已经被替代。
4. 解决后说明每个冲突文件的处理策略。

合并完成后运行测试：

```bash
cd backend && make test-unit
cd backend && make test-integration
make test-frontend
```

如果测试失败，禁止发布；需要先修复或明确记录风险。

## 3. 给上游仓库提 PR

目标：向 `Wei-Shaw/sub2api` 提交干净、可接受的通用改动。

```bash
git fetch upstream --prune
git fetch origin --prune
git switch main
git merge --ff-only upstream/main
git push origin main
git switch -c pr/<topic>
```

开发完成后检查：

```bash
git diff --stat upstream/main...HEAD
git log --oneline upstream/main..HEAD
```

PR 分支禁止包含：

- 2ue release workflow
- 2ue 镜像名
- `2ue-main` merge commit
- fork 版本号
- 本仓库私有部署配置
- 本文档或 AI 工作流配置

推送并向上游发 PR：

```bash
git push -u origin pr/<topic>
```

如果本仓库需要提前使用该功能，可以把 `pr/<topic>` merge 或 cherry-pick 到 `2ue-main`。上游 PR 合并后，再通过同步上游流程把改动带回 `main`，然后合并到 `2ue-main`。

如果上游使用 squash merge，`2ue-main` 里提前 cherry-pick 的提交可能与上游新提交不是同一个 hash。同步后需要使用以下命令辅助判断是否重复：

```bash
git cherry -v origin/main 2ue-main
git diff origin/main...2ue-main
```

## 4. 开发本仓库特有功能

目标：开发只属于 `2ue/sub2api` 的功能，不污染上游 PR。

```bash
git fetch origin --prune
git switch 2ue-main
git pull --ff-only origin 2ue-main
git switch -c feature/2ue-<topic>
```

开发完成后：

```bash
git switch 2ue-main
git pull --ff-only origin 2ue-main
git merge --no-ff feature/2ue-<topic> -m "feat(2ue): <topic>"
git push origin 2ue-main
```

这类功能可以长期存在于 `2ue-main`，并且会导致 `main` 与 `2ue-main` 分歧。这是正常状态，不要通过合并 `2ue-main` 到 `main` 来“消除差异”。

## 5. 发布本仓库版本

发布只允许从 `2ue-main` 执行。

发布前检查：

```bash
git fetch origin --prune
git fetch upstream --prune
git switch 2ue-main
git pull --ff-only origin 2ue-main
git status --short
git merge-base --is-ancestor origin/main HEAD
```

必须满足：

- 当前分支是 `2ue-main`
- 工作区干净
- `HEAD` 已经推送到 `origin/2ue-main`
- `origin/main` 已经合入当前 `HEAD`
- 测试通过
- tag 不存在
- tag 格式为 `vX.Y.Z-2ue.N`

版本规则：

```text
上游基础版本: v0.1.133
本仓库第 1 次发布: v0.1.133-2ue.1
本仓库第 2 次发布: v0.1.133-2ue.2
```

打 tag：

```bash
git tag -a v0.1.133-2ue.1 -m "2ue release v0.1.133-2ue.1"
git push origin v0.1.133-2ue.1
```

发布 workflow 必须校验：

1. tag 匹配 `v*-2ue.*`
2. tag commit 属于 `origin/2ue-main`
3. tag commit 包含 `origin/main`
4. workflow 不把 fork 版本写回 `main`

建议将 release workflow 触发器从：

```yaml
on:
  push:
    tags:
      - 'v*'
```

改为：

```yaml
on:
  push:
    tags:
      - 'v*-2ue.*'
```

并在 workflow 中增加 shell 级别校验，避免手动触发或错误 tag 绕过规则。

## 6. AI 执行约束

AI 执行任何 git 工作前必须先输出并检查：

```bash
git status --short --branch
git branch --show-current
git remote -v
```

AI 必须先判断任务类型：

- `sync-upstream`
- `merge-main-into-2ue-main`
- `upstream-pr`
- `2ue-feature`
- `2ue-release`

AI 禁止执行：

- `git push --force`
- `git reset --hard`
- 删除 `main`
- 删除 `2ue-main`
- 删除已发布 tag
- 把 `2ue-main` 合入 `main`
- 在 `main` 上创建本仓库功能提交
- 向 `origin` 推送普通上游 release tag 触发本仓库发布

AI 遇到以下情况必须停止并报告：

- `main` 无法 fast-forward 到 `upstream/main`
- `2ue-main` 合并 `main` 出现冲突且无法判断语义
- 发布 tag 不符合 `vX.Y.Z-2ue.N`
- 发布 commit 不属于 `origin/2ue-main`
- 测试失败
- 工作区存在用户未提交改动

## 7. 建议补充的自动化

为了减少 AI 自由发挥，建议后续添加固定脚本：

```text
scripts/ai/sync-upstream.sh
scripts/ai/merge-main-into-2ue.sh
scripts/ai/start-upstream-pr.sh
scripts/ai/start-2ue-feature.sh
scripts/ai/preflight-release.sh
scripts/ai/release-2ue.sh
```

建议添加 GitHub Actions guard：

```text
guard-main-mirror.yml
guard-2ue-release-tag.yml
guard-release-branch.yml
```

建议添加仓库规则：

- `main` 禁止 force push，禁止删除
- `2ue-main` 禁止 force push，禁止删除
- `v*` tag 禁止删除和覆盖
- release workflow 只允许 `v*-2ue.*`
- GitHub Packages 发布权限必须允许 `packages: write`

## 8. 分歧处理原则

`main` 和 `2ue-main` 的分歧是设计结果：

```text
main:      upstream-only history
2ue-main:  upstream history + 2ue workflow + 2ue features + 2ue releases
```

判断差异时不要用“分支是否完全一致”作为目标。正确目标是：

1. `main` 是否等于最新 `upstream/main`
2. `2ue-main` 是否包含最新 `origin/main`
3. `2ue-main` 的独有改动是否都是本仓库有意维护的内容
4. 给上游 PR 的分支是否从 `main` 切出且保持干净
5. release tag 是否只从 `2ue-main` 创建

因此，永远不要为了消除差异把 `2ue-main` merge 到 `main`。差异应该被理解、归类和测试，而不是被抹平。
