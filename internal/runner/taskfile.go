package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

type TaskfileRunner struct{}

func (t *TaskfileRunner) Name() string { return "task" }
func (t *TaskfileRunner) Bin() string  { return "task" }

type taskListOutput struct {
	Tasks []taskEntry `json:"tasks"`
}

type taskEntry struct {
	Name    string   `json:"name"`
	Desc    string   `json:"desc"`
	Aliases []string `json:"aliases"`
}

func (t *TaskfileRunner) ListTasks(ctx context.Context, dir string) ([]Task, error) {
	cmd := exec.CommandContext(ctx, "task", "--list-all", "--json", "--no-status")
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var out taskListOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(out.Tasks))
	for _, e := range out.Tasks {
		tasks = append(tasks, Task{
			Name:   e.Name,
			Desc:   e.Desc,
			Runner: "task",
		})
	}
	return tasks, nil
}

func (t *TaskfileRunner) CmdString(task string, args []string) string {
	parts := []string{"task", task}
	if len(args) > 0 {
		parts = append(parts, "--")
		parts = append(parts, args...)
	}
	return strings.Join(parts, " ")
}

func (t *TaskfileRunner) Exec(ctx context.Context, dir string, task string, args []string) error {
	cmdArgs := []string{task}
	if len(args) > 0 {
		cmdArgs = append(cmdArgs, "--")
		cmdArgs = append(cmdArgs, args...)
	}
	cmd := exec.CommandContext(ctx, "task", cmdArgs...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return ExecExitCode(cmd.Run())
}
