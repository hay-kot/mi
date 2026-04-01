package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// matches "target:", "target: deps", "target: deps ## description"
	// target names may include '/' for path-style targets (e.g. cmd/foo/bar)
	makeTargetRe     = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9._/-]*):(?:[^=]|$)`)
	makeDescRe       = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*:.*?##\s*(.+)$`)
	makeCommentRe    = regexp.MustCompile(`^#\s*(.+)$`)
	makeLineContinRe = regexp.MustCompile(`\\\n\s*`)
)

type MakeRunner struct{}

func (m *MakeRunner) Name() string { return "make" }
func (m *MakeRunner) Bin() string  { return "make" }

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

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Join line continuations so multi-line targets/deps are treated as one line.
	content := makeLineContinRe.ReplaceAllString(string(data), " ")

	var tasks []Task
	var lastComment string

	for line := range strings.SplitSeq(content, "\n") {
		if strings.TrimSpace(line) == "" {
			lastComment = ""
			continue
		}

		// Capture preceding comment as candidate description.
		if cm := makeCommentRe.FindStringSubmatch(line); cm != nil {
			lastComment = strings.TrimSpace(cm[1])
			continue
		}

		match := makeTargetRe.FindStringSubmatch(line)
		if match == nil {
			lastComment = ""
			continue
		}

		name := match[1]
		// skip hidden/internal targets (.PHONY, .DEFAULT, etc.)
		if strings.HasPrefix(name, ".") {
			lastComment = ""
			continue
		}

		desc := ""
		if dm := makeDescRe.FindStringSubmatch(line); dm != nil {
			// Prefer inline ## description.
			desc = strings.TrimSpace(dm[1])
		} else {
			// Fall back to immediately preceding # comment.
			desc = lastComment
		}
		lastComment = ""

		tasks = append(tasks, Task{
			Name:   name,
			Desc:   desc,
			Runner: "make",
		})
	}
	return tasks, nil
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
