package cmds

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunCmd runs a shell command and waits for it to complete
func RunCmd(cmd string) error {
	cmds := strings.Split(cmd, " ")
	if err := exec.Command(cmds[0], cmds[1:]...).Run(); err != nil {
		return fmt.Errorf("error running shell command:%v", err)
	}

	return nil
}

// RunCmdWithResult runs and waits a shell command and returns newline separated output as []string
func RunCmdWithResult(cmd string) ([]string, error) {
	cmds := strings.Split(cmd, " ")
	out, err := exec.Command(cmds[0], cmds[1:]...).Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(string(out), "\n"), nil
}
