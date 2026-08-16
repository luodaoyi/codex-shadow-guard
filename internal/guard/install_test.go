package guard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateHooksPreservesThirdPartyGroups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	original := `{
  "hooks": {
    "Stop": [{"hooks": [{"type": "command", "command": "other-tool"}]}]
  }
}`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := updateHooks(path, "/tmp/codex-shadow-guard", true)
	if err != nil || !changed {
		t.Fatalf("install hooks: changed=%v err=%v", changed, err)
	}
	root, err := loadHooks(path)
	if err != nil {
		t.Fatal(err)
	}
	if !hasManagedHook(root, "PreToolUse", "/tmp/codex-shadow-guard") || !hasManagedHook(root, "PostToolUse", "/tmp/codex-shadow-guard") || !hasManagedHook(root, "Stop", "/tmp/codex-shadow-guard") {
		t.Fatalf("missing managed hooks: %#v", root)
	}
	changed, err = updateHooks(path, "/tmp/codex-shadow-guard", false)
	if err != nil || !changed {
		t.Fatalf("remove hooks: changed=%v err=%v", changed, err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(contents, &result); err != nil {
		t.Fatal(err)
	}
	if hasManagedHook(result, "Stop", "/tmp/codex-shadow-guard") {
		t.Fatalf("managed hook remains: %#v", result)
	}
	hooks := result["hooks"].(map[string]any)
	if _, ok := hooks["Stop"]; !ok {
		t.Fatalf("third-party Stop hook was removed: %#v", result)
	}
}

func TestQuoteCommandHandlesWindowsStylePath(t *testing.T) {
	got := quoteCommand(`C:\Program Files\Codex\guard.exe`)
	if got != `"C:\Program Files\Codex\guard.exe"` {
		t.Fatalf("quoted command = %q", got)
	}
}
