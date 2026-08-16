# 本地 Codex 重开发清单

这个仓库的第一目标是把护栏边界说清楚，再逐步实现。不要因为“防止乱写”而做成另一个不透明的自动 Agent。

## 固定约束

- 单 Go 原生二进制；核心只使用标准库。
- 默认离线、本地、无第二模型、无源码上传。
- Hook 配置兼容 Codex 当前 `~/.codex/hooks.json` 形态；以实际 Codex 版本为准。
- 仅管理命令包含 `codex-shadow-guard hook` 的 Hook group。
- 仅管理 `AGENTS.md` 中 `codex-shadow-guard:begin/end` 标记范围。
- 所有高风险动作由用户确认；工具不替 Codex 或用户做发布、推送、删除决定。
- 本机不跑完整 build/test；以 GitHub Actions 为真实跨平台验证。

## 开发顺序

1. 先让 CI 通过基础安装、卸载、Hook JSON 合并和审查规则单测。
2. 增加真实 Codex Hook payload fixture；根据实际版本更新兼容范围。
3. 增加项目级策略文件，但先只支持确定性字段，例如禁止命令、最大变更文件数、必须运行的验证命令。
4. 增加状态读取和审查报告展示。
5. 只有用户明确选择后，设计脱敏、独立、只读的 AI Reviewer；默认关闭。
6. TUI 可以升级，但不要改变配置的可读性和可逆性。

## 首版验收

- 多次 install 后 Hooks 不重复，AGENTS 只有一个受控区块。
- uninstall 后第三方 Hook 和用户 AGENTS 内容逐字保留。
- `PreToolUse` 对确定性破坏命令返回 `block` JSON。
- `Stop` 在 `git diff --check` 有错误时返回 `block`；正常 diff 不阻断。
- 失败工具调用只产生 warning，不自动阻断。
- GitHub Actions 在 Linux/macOS/Windows 均执行 gofmt、vet、test、build。

## 暂不接受

- 自动把所有会话、diff 或命令发给云端模型。
- 根据关键词随意阻断合法命令。
- 自动修改用户业务代码来“修复”审查结果。
- 复制 Pi Shadow Mind 的内部运行时或声称拥有 Pi 相同的并行 session 能力。
