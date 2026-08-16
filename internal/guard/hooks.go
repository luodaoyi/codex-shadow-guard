package guard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EvaluatePreToolUse blocks only commands with a clear destructive meaning.
// Architectural judgment belongs in the report; it must not become a noisy gate.
func EvaluatePreToolUse(input HookInput) *Decision {
	command := strings.ToLower(extractCommand(input.ToolInput))
	for _, pattern := range []string{
		"rm -rf /", "rm -rf ~", "git reset --hard", "git clean -fd",
		"git push --force", "git push -f", "drop database", "truncate table",
	} {
		if strings.Contains(command, pattern) {
			return &Decision{Decision: "block", Reason: "Shadow Guard blocked a destructive command: " + pattern + ". Ask the user for explicit confirmation before continuing."}
		}
	}
	return nil
}

// RecordPostToolUse persists concise local evidence for the next Stop review.
func RecordPostToolUse(input HookInput) error {
	if input.CWD == "" {
		return nil
	}
	path := filepath.Join(input.CWD, ToolRecordFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create guard state: %w", err)
	}
	entry := map[string]any{
		"tool_name":  input.ToolName,
		"tool_input": compactInput(input.ToolInput),
		"failed":     responseFailed(input.ToolResponse),
	}
	bytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal tool record: %w", err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open tool record: %w", err)
	}
	defer file.Close()
	_, err = file.Write(append(bytes, '\n'))
	return err
}

// ReviewProject runs only read-only git commands. It never calls a model or uploads data.
func ReviewProject(project string) ([]Finding, error) {
	findings := make([]Finding, 0)
	if !isGitRepository(project) {
		return append(findings, Finding{Level: "info", Message: "Not a Git repository; skipped diff review."}), nil
	}
	if output, err := git(project, "diff", "--check"); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		findings = append(findings, Finding{Level: "block", Message: "git diff --check failed: " + message})
	}
	if output, err := git(project, "status", "--porcelain"); err == nil {
		files := countLines(string(output))
		if files > 20 {
			findings = append(findings, Finding{Level: "warn", Message: fmt.Sprintf("The working tree has %d changed paths. Confirm this scope matches the request.", files)})
		}
	}
	if failed, err := hasFailedToolRecord(filepath.Join(project, ToolRecordFile)); err == nil && failed {
		findings = append(findings, Finding{Level: "warn", Message: "A recent tool call reported failure. Do not claim completion without addressing or explaining it."})
	}
	if len(findings) == 0 {
		findings = append(findings, Finding{Level: "ok", Message: "No deterministic Shadow Guard findings."})
	}
	return findings, nil
}

func WriteReport(project, event string, findings []Finding) error {
	if project == "" {
		return nil
	}
	bytes, err := json.MarshalIndent(Report{Version: 1, Event: event, Findings: findings}, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(project, ReportFile), append(bytes, '\n'))
}

func BlockingDecision(findings []Finding) *Decision {
	for _, finding := range findings {
		if finding.Level == "block" {
			return &Decision{Decision: "block", Reason: finding.Message}
		}
	}
	return nil
}

func extractCommand(input map[string]any) string {
	for _, key := range []string{"command", "cmd", "script"} {
		if value, ok := input[key].(string); ok {
			return value
		}
	}
	return ""
}

func compactInput(input map[string]any) map[string]string {
	result := map[string]string{}
	if command := extractCommand(input); command != "" {
		result["command"] = command
	}
	if path, ok := input["path"].(string); ok {
		result["path"] = path
	}
	return result
}

func responseFailed(response map[string]any) bool {
	for _, key := range []string{"is_error", "error", "failed"} {
		if value, ok := response[key].(bool); ok && value {
			return true
		}
	}
	if code, ok := response["exit_code"].(float64); ok && code != 0 {
		return true
	}
	return false
}

func hasFailedToolRecord(path string) (bool, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	for scanner := bufio.NewScanner(file); scanner.Scan(); {
		var record struct {
			Failed bool `json:"failed"`
		}
		if json.Unmarshal(scanner.Bytes(), &record) == nil && record.Failed {
			return true, nil
		}
	}
	return false, scanner.Err()
}

func countLines(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}
