package runner

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
)

// ExitError wraps a runner's non-zero exit code so the caller can
// pass it through without printing extra error output.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// ExecExitCode checks if err is an exec.ExitError and wraps it as an ExitError.
func ExecExitCode(err error) error {
	if err == nil {
		return nil
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return &ExitError{Code: exitErr.ExitCode()}
	}
	return err
}

// Task represents a single runnable task from any backend.
type Task struct {
	Name   string
	Desc   string
	Runner string // which backend owns this task
	Hidden bool   // runnable by name, but left out of listings and completion
}

// Runner is a task runner backend (make, task, mise, package.json).
type Runner interface {
	// Name returns the runner identifier (e.g. "make", "task", "mise").
	Name() string
	// Bin returns the binary name used by exec.LookPath to check availability.
	// Return "" for runners that don't require an external binary.
	Bin() string
	// ListTasks returns all tasks available in dir.
	ListTasks(ctx context.Context, dir string) ([]Task, error)
	// CmdString returns the command that would be executed (for display).
	CmdString(dir string, task string, args []string) string
	// Exec runs the named task with args in dir, inheriting stdio.
	Exec(ctx context.Context, dir string, task string, args []string) error
}

// DefaultPriority returns runners in default priority order: make > task > mise > package.json.
func DefaultPriority() []Runner {
	return []Runner{
		&MakeRunner{},
		&TaskfileRunner{},
		&MiseRunner{},
		&PackageJSONRunner{},
	}
}

// Available filters runners to those whose binary is on PATH.
// Runners with an empty Bin() are always included.
func Available(runners []Runner) []Runner {
	var result []Runner
	for _, r := range runners {
		bin := r.Bin()
		if bin == "" {
			result = append(result, r)
			continue
		}
		if _, err := exec.LookPath(bin); err == nil {
			result = append(result, r)
		}
	}
	return result
}

// fetchAll calls ListTasks on all runners concurrently, returning results
// indexed by runner position to preserve priority order.
func fetchAll(ctx context.Context, runners []Runner, dir string) [][]Task {
	results := make([][]Task, len(runners))
	var wg sync.WaitGroup
	wg.Add(len(runners))
	for i, r := range runners {
		go func(i int, r Runner) {
			defer wg.Done()
			tasks, err := r.ListTasks(ctx, dir)
			if err == nil {
				results[i] = tasks
			}
		}(i, r)
	}
	wg.Wait()
	return results
}

// Resolve finds the first runner (by priority) that has the named task.
func Resolve(ctx context.Context, runners []Runner, dir string, task string) (Runner, error) {
	results := fetchAll(ctx, runners, dir)
	for i, tasks := range results {
		for _, t := range tasks {
			if t.Name == task {
				return runners[i], nil
			}
		}
	}
	return nil, fmt.Errorf("task %q not found in any runner", task)
}

// ListAll collects tasks from all runners. Higher-priority runner wins on name conflicts.
func ListAll(ctx context.Context, runners []Runner, dir string) ([]Task, error) {
	results := fetchAll(ctx, runners, dir)
	seen := make(map[string]bool)
	var all []Task
	for _, tasks := range results {
		for _, t := range tasks {
			if t.Hidden || seen[t.Name] {
				continue
			}
			seen[t.Name] = true
			all = append(all, t)
		}
	}
	return all, nil
}
