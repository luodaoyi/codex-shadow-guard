# Codex Shadow Guard

> 为 OpenAI Codex 准备的本地、可审计、可撤销的“防乱写代码”护栏。

`codex-shadow-guard` 不是要替代 Codex，也不会承诺让模型永远写对代码。它将一些本应在事后发现的问题，提前到工具调用前和任务结束前：越界破坏性操作会被阻断，明显的 diff 格式问题会阻止“完成”，其余风险会保存为本地审查报告。

当前为第一版迁移实现。它的重点是一个可靠、低误报的基础设施；后续会在本地使用 Codex 继续迭代。

## 来龙去脉

本项目的设计灵感来自 [Pi Shadow Mind](https://github.com/liuzhengdongfortest/pi-shadow-mind)。Pi Shadow Mind 让主 Agent 编码的同时，运行具有独立职责的 Shadow Minds，例如架构审阅、项目事实核验、文档维护和完成度检查。

它的核心理念值得保留：

- 实现与审查不应完全分离；
- 审查角色应只获得完成职责所需的最小权限；
- 只有带具体证据、可以行动的发现才应介入；
- 用户定义的规则应保持可见、可调整、可撤销。

但是 Pi Shadow Mind 是 Pi 的 TypeScript Extension，依赖 Pi 的 `turn_end`、独立 `AgentSession`、会话注入和 `report_to_main` API。Codex 没有同等的进程内 Extension API，因此**不能直接安装或照搬 Pi 的代码**。

本仓库不复制 Pi Shadow Mind runtime。它以 MIT 许可证允许的方式，借鉴其设计思想，重写为适合 Codex 的本地 Hook 工具。原项目署名与许可证见 [NOTICE.md](NOTICE.md)。

## 我们要仿照什么

| Pi Shadow Mind 的设计 | Codex Shadow Guard 的对应实现 |
| --- | --- |
| 持续、独立的审查职责 | 项目 `AGENTS.md` 规则 + 生命周期 Hook |
| 最小工具权限 | Hook 默认只读审查；只在 `PreToolUse` 拦截确定性危险命令 |
| 主 Agent 工作时同步检查 | `PostToolUse` 记录本地工具证据，`Stop` 进行完成前审查 |
| 只报告有依据的发现 | 第一版不调用第二个模型；仅检查确定性证据 |
| 可配置、可暂停的 Shadow 定义 | 受控 `AGENTS.md` 区块和可逆 `hooks.json` 条目 |

第一版**不迁移** Pi 的随机 heartbeat、并行 Shadow Agent、主会话轨迹复制、跨模型调用或自动回注主会话。这些能力在 Codex 中没有等价且稳定的 Extension 接口；强行模拟会带来上下文泄露、额外成本、不可预测延迟和错误阻断。

## 目标

面向本地 Codex 工作流，减少以下行为：

- 未读取现有实现、测试和项目约定，就大范围重写；
- 一个小修复顺手扩展为无关重构、复杂抽象或依赖升级；
- 执行明显破坏性的 Git 或 shell 命令；
- 存在 `git diff --check` 错误仍声称完成；
- 最近工具命令已经失败，却没有解释或处理；
- 把未验证的猜测包装成完成结论。

它不做这些事：

- 不上传项目源码、diff、提示词或工具输出；
- 不读取或发送密钥；
- 不自动修改业务代码；
- 不替用户批准 `push --force`、删除数据或发布；
- 不把启发式架构意见当成硬性阻断。

## 第一版机制

### 1. 项目规则

`install` 在项目 `AGENTS.md` 加入一个受控区块：

```md
<!-- codex-shadow-guard:begin -->
...
<!-- codex-shadow-guard:end -->
```

它要求 Codex：先读相关实现和测试；保持用户要求的修改范围；避免猜测性 fallback、抽象、依赖和无关重构；结束前检查 diff 并说明验证证据。

安装可重复执行，不会追加重复区块。卸载只移除这个区块，用户自己的 `AGENTS.md` 内容不变。

### 2. Codex Hooks

安装器把自身路径注册进 `~/.codex/hooks.json`：

- `PreToolUse`：拦截确定性危险命令，例如 `rm -rf /`、`git reset --hard`、`git clean -fd`、强制推送和明显数据库销毁命令。
- `PostToolUse`：在项目 `.codex-shadow-guard/tool-events.jsonl` 保存最小的本地工具结果摘要。
- `Stop`：运行只读 `git diff --check`、检查变更路径数量和近期工具失败，写入 `.codex-shadow-guard/latest-report.json`。若存在 diff 格式错误，向 Codex 返回 `block`，要求先修复。

安装器只添加、更新或移除命令指向 `codex-shadow-guard hook` 的 Hook group，不触碰第三方 Hook。每次新装或更新 Hook 后，必须**新开 Codex 会话并执行 `/hooks`，确认信任该命令**。

## 使用

当前推荐从 release 二进制运行；首版 release 发布前也可以在本地从源码构建。

```text
codex-shadow-guard
```

菜单中选择为当前项目安装。也可以使用命令：

```bash
# 当前项目
codex-shadow-guard install

# 指定项目
codex-shadow-guard install /path/to/project

# 手动执行只读审查
codex-shadow-guard review /path/to/project

# 查看安装状态
codex-shadow-guard status /path/to/project

# 可逆卸载；只删除本工具的配置
codex-shadow-guard uninstall /path/to/project
```

## 后续本地重开发方向

第一版先确保确定性规则不会造成噪声。后续在本地 Codex 中开发前，应先确认以下设计，而不是直接堆功能：

1. **范围审查**：如何将用户任务边界表示为可审计的计划，避免仅凭文件数误判。
2. **验证审查**：怎样读取真实 CI / 测试证据，而不把“没有本地测试”误判为失败。
3. **可选 AI Reviewer**：仅在用户明确启用时，对脱敏任务摘要和 diff 进行独立审查；默认禁止自动联网或上传代码。
4. **策略文件**：允许每个项目显式声明风险命令、最大变更范围和必要验证项。
5. **TUI**：将当前标准输入菜单升级为原生终端界面，但保持单二进制和可审计配置。

完整迁移边界和重开发验收标准见 [docs/MIGRATION.md](docs/MIGRATION.md) 与 [docs/REBUILD.md](docs/REBUILD.md)。

## 许可证

本项目使用 [MIT](LICENSE) 许可证。灵感来源 `pi-shadow-mind` 也使用 MIT；本项目保留来源、署名与差异说明，但不包含其 TypeScript runtime 代码。

## 当前状态

- 已实现：项目规则注入、Hook 安装与卸载、危险命令预拦截、工具结果本地记录、完成前只读 diff 审查、基础 CLI 菜单。
- 尚待 GitHub Actions 完成跨平台 `gofmt`、`go vet`、单元测试和构建验证后，才可称为可用首版。
- 未实现：模型驱动的平行审查、自动任务理解、自动改代码或任何云端上传。
