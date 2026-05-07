package tools

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/sashabaranov/go-openai"
)

func (r *Registry) registerBash() {
	r.registerTool(openai.FunctionDefinition{
		Name:        "Bash",
		Description: "Execute a shell command",
		Parameters:  registerBashSchema(),
	}, func(rawArgs string) (string, error) {
		type props struct {
			Command string `json:"command"`
		}
		var p props
		if err := mustUnmarshal(rawArgs, &p); err != nil {
			r.writeError("tools.Bash.mustUnmarshal", err)
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if p.Command == "" {
			r.writeError("tools.Bash", fmt.Errorf("command must not be empty"))
			return "", fmt.Errorf("command must not be empty")
		}

		cmd := buildShellCommand(p.Command)
		out, err := cmd.CombinedOutput()
		if err != nil {
			r.writeError("tools.Bash.cmd.CombinedOutput", err)
			return fmt.Sprintf("Command failed: %v\nOutput: %s", err, string(out)), nil
		}
		return fmt.Sprintf("Command executed successfully:\n%s", string(out)), nil
	})
}

func buildShellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		// PowerShell is present by default on Windows 10.
		return exec.Command("powershell", "-NoProfile", "-Command", command)
	}
	return exec.Command("sh", "-lc", command)
}
