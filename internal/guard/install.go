package guard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Paths struct {
	CodexHome string
	Binary    string
	Hooks     string
}

type Status struct {
	AgentsInstalled bool
	PreToolUse      bool
	PostToolUse     bool
	Stop            bool
}

func DiscoverPaths() (Paths, error) {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("find home directory: %w", err)
		}
		home = filepath.Join(userHome, ".codex")
	}
	name := "codex-shadow-guard"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return Paths{CodexHome: home, Binary: filepath.Join(home, "bin", name), Hooks: filepath.Join(home, "hooks.json")}, nil
}

// Install copies the native executable, manages only its hooks, and adds an
// explicitly marked project AGENTS.md block.
func Install(project, source string) (Paths, error) {
	paths, err := DiscoverPaths()
	if err != nil {
		return Paths{}, err
	}
	if err := copyBinary(source, paths.Binary); err != nil {
		return Paths{}, err
	}
	if _, err := updateHooks(paths.Hooks, paths.Binary, true); err != nil {
		return Paths{}, err
	}
	if _, _, err := InstallAgents(project); err != nil {
		return Paths{}, err
	}
	return paths, nil
}

func Uninstall(project string) (bool, error) {
	paths, err := DiscoverPaths()
	if err != nil {
		return false, err
	}
	hooksChanged, err := updateHooks(paths.Hooks, paths.Binary, false)
	if err != nil {
		return false, err
	}
	_, agentsChanged, err := UninstallAgents(project)
	return hooksChanged || agentsChanged, err
}

func InstallationStatus(project string) (Status, error) {
	paths, err := DiscoverPaths()
	if err != nil {
		return Status{}, err
	}
	content, err := readOptional(filepath.Join(project, "AGENTS.md"))
	if err != nil {
		return Status{}, err
	}
	root, err := loadHooks(paths.Hooks)
	if err != nil {
		return Status{}, err
	}
	return Status{
		AgentsInstalled: strings.Contains(content, AgentsBegin) && strings.Contains(content, AgentsEnd),
		PreToolUse:      hasManagedHook(root, "PreToolUse", paths.Binary),
		PostToolUse:     hasManagedHook(root, "PostToolUse", paths.Binary),
		Stop:            hasManagedHook(root, "Stop", paths.Binary),
	}, nil
}

func copyBinary(source, target string) error {
	if samePath(source, target) {
		return nil
	}
	contents, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read current executable: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := atomicWrite(target, contents); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		return os.Chmod(target, 0o755)
	}
	return nil
}

func samePath(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo)
}

func loadHooks(path string) (map[string]any, error) {
	content, err := readOptional(path)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return map[string]any{}, nil
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(content), &root); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return root, nil
}

func updateHooks(path, binary string, enabled bool) (bool, error) {
	root, err := loadHooks(path)
	if err != nil {
		return false, err
	}
	hooks, err := hooksObject(root, enabled)
	if err != nil {
		return false, err
	}
	changed := false
	for _, event := range []string{"PreToolUse", "PostToolUse", "Stop"} {
		updated, err := updateEvent(hooks, event, binary, enabled)
		if err != nil {
			return false, err
		}
		changed = changed || updated
	}
	if !enabled && len(hooks) == 0 {
		delete(root, "hooks")
	}
	if !changed {
		return false, nil
	}
	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	return true, atomicWrite(path, append(encoded, '\n'))
}

func hooksObject(root map[string]any, create bool) (map[string]any, error) {
	value, exists := root["hooks"]
	if !exists {
		if !create {
			return map[string]any{}, nil
		}
		hooks := map[string]any{}
		root["hooks"] = hooks
		return hooks, nil
	}
	hooks, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Codex hooks.json field hooks must be an object")
	}
	return hooks, nil
}

func updateEvent(hooks map[string]any, event, binary string, enabled bool) (bool, error) {
	value, exists := hooks[event]
	if !exists {
		if !enabled {
			return false, nil
		}
		hooks[event] = []any{managedGroup(binary)}
		return true, nil
	}
	groups, ok := value.([]any)
	if !ok {
		return false, fmt.Errorf("Codex hooks.json event %s must be an array", event)
	}
	kept := make([]any, 0, len(groups))
	found := false
	for _, group := range groups {
		if isManagedGroup(group, binary) {
			found = true
			continue
		}
		kept = append(kept, group)
	}
	if enabled {
		kept = append(kept, managedGroup(binary))
	}
	if len(kept) == 0 {
		delete(hooks, event)
	} else {
		hooks[event] = kept
	}
	return found || enabled, nil
}

func managedGroup(binary string) map[string]any {
	return map[string]any{"hooks": []any{map[string]any{"type": "command", "command": quoteCommand(binary) + " " + HookCommandArgument, "timeout": 10}}}
}

func quoteCommand(path string) string {
	return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
}

func isManagedGroup(value any, binary string) bool {
	group, ok := value.(map[string]any)
	if !ok {
		return false
	}
	handlers, ok := group["hooks"].([]any)
	if !ok {
		return false
	}
	for _, handler := range handlers {
		entry, ok := handler.(map[string]any)
		if !ok {
			continue
		}
		command, _ := entry["command"].(string)
		if strings.Contains(command, ManagedCommandMarker) || strings.Contains(command, binary) && strings.Contains(command, HookCommandArgument) {
			return true
		}
	}
	return false
}

func hasManagedHook(root map[string]any, event, binary string) bool {
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		return false
	}
	groups, ok := hooks[event].([]any)
	if !ok {
		return false
	}
	for _, group := range groups {
		if isManagedGroup(group, binary) {
			return true
		}
	}
	return false
}
