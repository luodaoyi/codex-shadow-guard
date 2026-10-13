package guard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluatePreToolUseBlocksDestructiveCommand(t *testing.T) {
	decision := EvaluatePreToolUse(HookInput{ToolInput: map[string]any{"command": "git reset --hard HEAD"}})
	if decision == nil || decision.Decision != "block" {
		t.Fatalf("expected destructive command to be blocked, got %#v", decision)
	}
}

func TestEvaluatePreToolUseAllowsNormalCommand(t *testing.T) {
	decision := EvaluatePreToolUse(HookInput{ToolInput: map[string]any{"command": "git status --short"}})
	if decision != nil {
		t.Fatalf("normal read-only command unexpectedly blocked: %#v", decision)
	}
}

// Codex sends Bash and apply_patch tool_response as a JSON string, not an object.
func TestRecordPostToolUseAcceptsStringToolResponse(t *testing.T) {
	project := t.TempDir()
	raw, err := json.Marshal(map[string]any{
		"session_id":      "session",
		"cwd":             project,
		"hook_event_name": "PostToolUse",
		"tool_name":       "Bash",
		"tool_use_id":     "call-1",
		"tool_input":      map[string]any{"command": "ls"},
		"tool_response":   "file1\nfile2\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	var input HookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatalf("decode PostToolUse payload with string tool_response: %v", err)
	}
	if err := RecordPostToolUse(input); err != nil {
		t.Fatal(err)
	}
	if failed, err := hasFailedToolRecord(filepath.Join(project, ToolRecordFile)); err != nil || failed {
		t.Fatalf("string tool_response must be recorded as not failed, failed=%v err=%v", failed, err)
	}
}

func TestResponseFailedReadsObjectResponse(t *testing.T) {
	var input HookInput
	if err := json.Unmarshal([]byte(`{"tool_response":{"exit_code":2}}`), &input); err != nil {
		t.Fatal(err)
	}
	if !responseFailed(input.ToolResponse) {
		t.Fatal("object tool_response with non-zero exit_code must be reported as failed")
	}
}

func TestReviewBlocksInvalidDiff(t *testing.T) {
	project := t.TempDir()
	mustWrite(t, filepath.Join(project, "bad.txt"), "line with trailing whitespace \n")
	mustGit(t, project, "init")
	mustGit(t, project, "add", "bad.txt")
	mustGit(t, project, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "base")
	mustWrite(t, filepath.Join(project, "bad.txt"), "changed trailing whitespace \n")

	findings, err := ReviewProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if decision := BlockingDecision(findings); decision == nil {
		t.Fatalf("expected invalid diff to block, got %#v", findings)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustGit(t *testing.T, project string, arguments ...string) {
	t.Helper()
	output, err := git(project, arguments...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
}
