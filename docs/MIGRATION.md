# 从 Pi Shadow Mind 到 Codex Shadow Guard

## 审计结论

上游 [Pi Shadow Mind](https://github.com/liuzhengdongfortest/pi-shadow-mind) 是一个 MIT 许可的 TypeScript Pi Extension。审计基线为上游提交 `ba75a67092024053f6529ef574d0cd81006ba6b1`。

它在 Pi 的 `turn_end` 后按 heartbeat 调度独立 Shadow Session。Shadow 继承主 system prompt，接收去除主 Agent reasoning、压缩工具结果的轨迹；可按自己的只读工具权限复查，并通过 `report_to_main` 将结论使用 Pi 的 steer/follow-up 机制回送给主 Agent。

这个设计适合 Pi，但其关键 API 是 Pi 私有运行时能力：`ExtensionAPI` 事件、`createAgentSession`、会话轨迹读取、`sendMessage`、自定义消息渲染和 `/shadow` 命令。Codex 不提供对应进程内 API。

## 迁移原则

- 不复制上游 TypeScript runtime；采用 MIT 许可允许的独立重写。
- 不伪造 Codex 子会话或读取/转发完整聊天轨迹。
- 默认没有第二个模型，没有网络请求，也没有源码上传。
- 优先实现确定性、可解释、低误报的防护。
- 一切安装均可逆：Hooks 与 `AGENTS.md` 都只管理自己的标记/命令。

## 功能映射

- **Architecture / project-grounding Shadow** → `AGENTS.md` 中要求先读真实实现、测试和项目规则；未来才考虑用户启用的 AI Reviewer。
- **最小工具权限** → `PreToolUse` 只检查命令字符串；`Stop` 只运行 `git` 的只读审查。
- **Shadow 报告** → `.codex-shadow-guard/latest-report.json`，供用户和 Codex 查看。
- **运行轨迹工具摘要** → `PostToolUse` 的本地 JSONL 极简记录，仅保存命令/路径和失败状态。
- **完成前审阅** → Codex `Stop` Hook；`git diff --check` 有错误时返回 Hook `block`。
- **实体 Markdown + runtime 调度** → 第一版不迁移。未来若实现策略文件，应让项目显式配置，不能隐藏随机调度。
- **随机 heartbeat / 并行影子 Agent** → 不迁移。Codex 缺少稳定的 session injection；模拟它会带来额外模型成本和上下文边界风险。

## 安全与数据边界

Hook 不读取 `.env`，不上传 Git diff，不调用外部 API，也不会执行自动修复。它写入的项目状态均位于 `.codex-shadow-guard/`，建议加入项目 `.gitignore`；工具自身仓库已经如此配置。

`PreToolUse` 的阻断规则只处理语义明确的命令。架构质量、测试选择、变更范围等问题默认是 warning/report，而非硬阻断，避免工具因猜测妨碍正常开发。

## 本地重开发验收

任何新增策略都应满足：

1. 有明确、可复现的触发条件和单元测试。
2. 不读取或上传秘密；日志必须最小化。
3. 默认不自动写业务文件。
4. 不影响第三方 Codex Hooks。
5. 卸载后用户 `AGENTS.md` 其余内容和第三方 Hooks 保持原样。
6. 新的硬阻断规则必须优先证明误报风险可接受。
7. 跨 Windows、macOS、Linux 的格式、vet、测试和构建均由 GitHub Actions 通过。
