package guard

import (
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
