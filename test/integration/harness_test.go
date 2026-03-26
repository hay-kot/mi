//go:build integration

package integration

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const cmdTimeout = 10 * time.Second

// Harness provides an isolated environment for integration tests.
// Each test gets its own temp directory with runner config files.
type Harness struct {
	t   *testing.T
	dir string // working directory for the test
}

// NewHarness creates a new test harness with an empty temp directory.
func NewHarness(t *testing.T) *Harness {
	t.Helper()
	return &Harness{
		t:   t,
		dir: t.TempDir(),
	}
}

// WithMakefile writes a Makefile to the test directory.
func (h *Harness) WithMakefile(content string) *Harness {
	h.t.Helper()
	h.writeFile("Makefile", content)
	return h
}

// WithTaskfile writes a Taskfile.yml to the test directory.
func (h *Harness) WithTaskfile(content string) *Harness {
	h.t.Helper()
	h.writeFile("Taskfile.yml", content)
	return h
}

// WithMise writes a mise.toml to the test directory.
func (h *Harness) WithMise(content string) *Harness {
	h.t.Helper()
	h.writeFile("mise.toml", content)
	return h
}

// Run executes mi with the given args in the test directory.
// Returns combined stdout+stderr.
func (h *Harness) Run(args ...string) (string, error) {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, miBin, args...)
	cmd.Dir = h.dir
	cmd.Env = h.env()

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	return buf.String(), err
}

// RunStdout executes mi and returns only stdout.
func (h *Harness) RunStdout(args ...string) (string, error) {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, miBin, args...)
	cmd.Dir = h.dir
	cmd.Env = h.env()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		h.t.Logf("stderr: %s", stderr.String())
	}
	return stdout.String(), err
}

func (h *Harness) writeFile(name, content string) {
	h.t.Helper()
	path := filepath.Join(h.dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		h.t.Fatalf("writing %s: %v", name, err)
	}
}

func (h *Harness) env() []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + h.dir,
		"TMPDIR=" + os.Getenv("TMPDIR"),
		"NO_COLOR=1",
		"LOG_LEVEL=error",
		"MISE_TRUSTED_CONFIG_PATHS=" + h.dir,
		"MISE_GLOBAL_CONFIG_FILE=/dev/null",
		"MISE_YES=1",
	}
}
