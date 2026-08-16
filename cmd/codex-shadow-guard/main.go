package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/luodaoyi/codex-shadow-guard/internal/guard"
)

const help = `codex-shadow-guard - Codex 的本地确定性审查护栏

用法：
  codex-shadow-guard                 启动安装菜单
  codex-shadow-guard install [PATH]  为项目安装 Hooks 与 AGENTS.md 区块
  codex-shadow-guard uninstall [PATH] 移除本工具写入的 Hooks 与 AGENTS.md 区块
  codex-shadow-guard status [PATH]   查看安装状态
  codex-shadow-guard review [PATH]   只读审查当前工作区
  codex-shadow-guard hook            处理 Codex Hook stdin JSON
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "codex-shadow-guard:", err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	if len(args) == 0 {
		return runMenu(input, output)
	}
	if args[0] == "-h" || args[0] == "--help" {
		_, err := fmt.Fprint(output, help)
		return err
	}
	if args[0] == "-V" || args[0] == "--version" {
		_, err := fmt.Fprintln(output, "codex-shadow-guard dev")
		return err
	}
	if args[0] == "hook" {
		return runHook(input, output)
	}
	project, err := projectArgument(args)
	if err != nil {
		return err
	}
	switch args[0] {
	case "install":
		source, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate executable: %w", err)
		}
		paths, err := guard.Install(project, source)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "已安装守卫：%s\nHooks：%s\n请新开 Codex 会话后运行 /hooks 并信任本工具。\n", paths.Binary, paths.Hooks)
	case "uninstall":
		changed, err := guard.Uninstall(project)
		if err != nil {
			return err
		}
		if changed {
			fmt.Fprintln(output, "已移除本工具写入的 Hooks 和 AGENTS.md 区块。")
		} else {
			fmt.Fprintln(output, "未找到本工具管理的配置，未修改用户文件。")
		}
	case "status":
		status, err := guard.InstallationStatus(project)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "AGENTS.md：%s\nPreToolUse：%s\nPostToolUse：%s\nStop：%s\n", yesNo(status.AgentsInstalled), yesNo(status.PreToolUse), yesNo(status.PostToolUse), yesNo(status.Stop))
	case "review":
		findings, err := guard.ReviewProject(project)
		if err != nil {
			return err
		}
		for _, finding := range findings {
			fmt.Fprintf(output, "[%s] %s\n", strings.ToUpper(finding.Level), finding.Message)
		}
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], help)
	}
	return nil
}

func runHook(input io.Reader, output io.Writer) error {
	var payload guard.HookInput
	if err := json.NewDecoder(input).Decode(&payload); err != nil {
		return fmt.Errorf("parse Hook JSON: %w", err)
	}
	var decision *guard.Decision
	switch payload.Event {
	case "PreToolUse":
		decision = guard.EvaluatePreToolUse(payload)
	case "PostToolUse":
		if err := guard.RecordPostToolUse(payload); err != nil {
			return err
		}
	case "Stop":
		findings, err := guard.ReviewProject(payload.CWD)
		if err != nil {
			return err
		}
		if err := guard.WriteReport(payload.CWD, payload.Event, findings); err != nil {
			return err
		}
		decision = guard.BlockingDecision(findings)
	}
	if decision != nil {
		return json.NewEncoder(output).Encode(decision)
	}
	return nil
}

func runMenu(input io.Reader, output io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Fprint(output, "Codex Shadow Guard\n1. 为当前项目安装\n2. 查看当前项目状态\n3. 卸载当前项目\n4. 退出\n选择：")
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	switch strings.TrimSpace(line) {
	case "1":
		return run([]string{"install", cwd}, input, output)
	case "2":
		return run([]string{"status", cwd}, input, output)
	case "3":
		return run([]string{"uninstall", cwd}, input, output)
	case "4", "":
		return nil
	default:
		return fmt.Errorf("invalid selection")
	}
}

func projectArgument(args []string) (string, error) {
	project := "."
	if len(args) > 2 {
		return "", fmt.Errorf("expected at most one project path")
	}
	if len(args) == 2 {
		project = args[1]
	}
	absolute, err := filepath.Abs(project)
	if err != nil {
		return "", err
	}
	return absolute, nil
}

func yesNo(value bool) string {
	if value {
		return "已配置"
	}
	return "未配置"
}
