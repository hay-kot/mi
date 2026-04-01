package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

type MiseRunner struct{}

func (m *MiseRunner) Name() string { return "mise" }
func (m *MiseRunner) Bin() string  { return "mise" }

type miseTaskEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (m *MiseRunner) ListTasks(ctx context.Context, dir string) ([]Task, error) {
	cmd := exec.CommandContext(ctx, "mise", "tasks", "ls", "--json")
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var entries []miseTaskEntry
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(entries))
	for _, e := range entries {
		tasks = append(tasks, Task{
			Name:   e.Name,
			Desc:   e.Description,
			Runner: "mise",
		})
	}
	return tasks, nil
}

func (m *MiseRunner) CmdString(task string, args []string) string {
	return strings.Join(append([]string{"mise", "run", task}, args...), " ")
}

func (m *MiseRunner) Exec(ctx context.Context, dir string, task string, args []string) error {
	cmdArgs := append([]string{"run", task}, args...)
	cmd := exec.CommandContext(ctx, "mise", cmdArgs...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return ExecExitCode(cmd.Run())
}
