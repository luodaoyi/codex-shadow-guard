package guard

import (
	"os/exec"
)

func isGitRepository(project string) bool {
	_, err := git(project, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

func git(project string, arguments ...string) ([]byte, error) {
	command := exec.Command("git", arguments...)
	command.Dir = project
	return command.CombinedOutput()
}
