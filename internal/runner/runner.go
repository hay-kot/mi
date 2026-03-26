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
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &ExitError{Code: exitErr.ExitCode()}
	}
	return err
}

// Task represents a single runnable task from any backend.
type Task struct {
	Name   string
	Desc   string
	Runner string // which backend owns this task
}

// Runner is a task runner backend (make, task, mise).
type Runner interface {
	// Name returns the runner identifier (e.g. "make", "task", "mise").
	Name() string
	// Detect returns true if this runner's config file exists in dir.
	Detect(dir string) bool
	// ListTasks returns all tasks available in dir.
	ListTasks(ctx context.Context, dir string) ([]Task, error)
	// CmdString returns the command that would be executed (for display).
	CmdString(task string, args []string) string
	// Exec runs the named task with args in dir, inheriting stdio.
	Exec(ctx context.Context, dir string, task string, args []string) error
}

// DefaultPriority returns runners in default priority order: make > task > mise.
func DefaultPriority() []Runner {
	return []Runner{
		&MakeRunner{},
		&TaskfileRunner{},
		&MiseRunner{},
	}
}

// DetectAll runs Detect concurrently on all runners and returns those present, preserving priority order.
func DetectAll(runners []Runner, dir string) []Runner {
	present := make([]bool, len(runners))
	var wg sync.WaitGroup
	wg.Add(len(runners))
	for i, r := range runners {
		go func(i int, r Runner) {
			defer wg.Done()
			present[i] = r.Detect(dir)
		}(i, r)
	}
	wg.Wait()

	var result []Runner
	for i, r := range runners {
		if present[i] {
			result = append(result, r)
		}
	}
	return result
}

// Resolve finds the first runner (by priority) that has the named task.
func Resolve(ctx context.Context, runners []Runner, dir string, task string) (Runner, error) {
	for _, r := range runners {
		tasks, err := r.ListTasks(ctx, dir)
		if err != nil {
			continue
		}
		for _, t := range tasks {
			if t.Name == task {
				return r, nil
			}
		}
	}
	return nil, fmt.Errorf("task %q not found in any runner", task)
}

// ListAll collects tasks from all runners. Higher-priority runner wins on name conflicts.
func ListAll(ctx context.Context, runners []Runner, dir string) ([]Task, error) {
	seen := make(map[string]bool)
	var all []Task
	for _, r := range runners {
		tasks, err := r.ListTasks(ctx, dir)
		if err != nil {
			continue
		}
		for _, t := range tasks {
			if seen[t.Name] {
				continue
			}
			seen[t.Name] = true
			all = append(all, t)
		}
	}
	return all, nil
}
