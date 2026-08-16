package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstallAgents adds only the managed block and preserves all user content.
func InstallAgents(project string) (string, bool, error) {
	path := filepath.Join(project, "AGENTS.md")
	content, err := readOptional(path)
	if err != nil {
		return "", false, err
	}
	next, changed, err := replaceManagedBlock(content, AgentsBlock)
	if err != nil {
		return "", false, err
	}
	if changed {
		if err := atomicWrite(path, []byte(next)); err != nil {
			return "", false, err
		}
	}
	return path, changed, nil
}

// UninstallAgents removes only the exact managed range.
func UninstallAgents(project string) (string, bool, error) {
	path := filepath.Join(project, "AGENTS.md")
	content, err := readOptional(path)
	if err != nil {
		return "", false, err
	}
	next, changed, err := removeManagedBlock(content)
	if err != nil || !changed {
		return path, changed, err
	}
	if err := atomicWrite(path, []byte(next)); err != nil {
		return "", false, err
	}
	return path, true, nil
}

func replaceManagedBlock(content, block string) (string, bool, error) {
	stripped, removed, err := removeManagedBlock(content)
	if err != nil {
		return "", false, err
	}
	next := strings.TrimRight(stripped, "\n")
	if next != "" {
		next += "\n\n"
	}
	next += block
	return next, !removed || content != next, nil
}

func removeManagedBlock(content string) (string, bool, error) {
	begin := strings.Index(content, AgentsBegin)
	end := strings.Index(content, AgentsEnd)
	if begin < 0 && end < 0 {
		return content, false, nil
	}
	if begin < 0 || end < begin {
		return "", false, fmt.Errorf("AGENTS.md has an invalid codex-shadow-guard managed block")
	}
	end += len(AgentsEnd)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return strings.TrimRight(content[:begin]+content[end:], "\n") + trailingNewline(content[:begin]+content[end:]), true, nil
}

func trailingNewline(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	return "\n"
}

func readOptional(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(bytes), nil
}

func atomicWrite(path string, bytes []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".codex-shadow-guard-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err := temporary.Write(bytes); err != nil {
		temporary.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	// Windows cannot rename over an existing file. The contents were fully
	// prepared before this point, so remove only the exact destination that this
	// operation is replacing and then perform the rename.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
