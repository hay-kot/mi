package runner

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// matches "target: ..." or "target: ... ## description"
	makeTargetRe = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9._-]*):[^=]`)
	makeDescRe   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*:.*?##\s*(.+)$`)
)

type MakeRunner struct{}

func (m *MakeRunner) Name() string { return "make" }

func (m *MakeRunner) Detect(dir string) bool {
	for _, name := range []string{"Makefile", "makefile", "GNUmakefile"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func (m *MakeRunner) findMakefile(dir string) string {
	for _, name := range []string{"Makefile", "makefile", "GNUmakefile"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (m *MakeRunner) ListTasks(ctx context.Context, dir string) ([]Task, error) {
	path := m.findMakefile(dir)
	if path == "" {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tasks []Task
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		match := makeTargetRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		name := match[1]
		// skip hidden/internal targets
		if strings.HasPrefix(name, ".") {
			continue
		}

		desc := ""
		if dm := makeDescRe.FindStringSubmatch(line); dm != nil {
			desc = strings.TrimSpace(dm[1])
		}

		tasks = append(tasks, Task{
			Name:   name,
			Desc:   desc,
			Runner: "make",
		})
	}
	return tasks, scanner.Err()
}

func (m *MakeRunner) CmdString(task string, args []string) string {
	return strings.Join(append([]string{"make", task}, args...), " ")
}

func (m *MakeRunner) Exec(ctx context.Context, dir string, task string, args []string) error {
	cmdArgs := append([]string{task}, args...)
	cmd := exec.CommandContext(ctx, "make", cmdArgs...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return ExecExitCode(cmd.Run())
}
