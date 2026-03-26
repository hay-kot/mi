//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoRunners(t *testing.T) {
	h := NewHarness(t)
	out, err := h.Run("build")
	require.Error(t, err)
	assert.Contains(t, out, "no task runner found")
}

func TestListNoRunners(t *testing.T) {
	h := NewHarness(t)
	out, err := h.Run()
	require.NoError(t, err)
	assert.Contains(t, out, "no task runners found")
}

func TestMakefileDetect(t *testing.T) {
	h := NewHarness(t).WithMakefile(`
.PHONY: build test

build: ## Build the project
	@echo "make-build"

test: ## Run tests
	@echo "make-test"
`)

	out, err := h.Run("--ls")
	require.NoError(t, err)
	assert.Contains(t, out, "build")
	assert.Contains(t, out, "test")
	assert.Contains(t, out, "make")
}

func TestMakefileExec(t *testing.T) {
	h := NewHarness(t).WithMakefile(`
.PHONY: hello

hello: ## Say hello
	@echo "hello-from-make"
`)

	out, err := h.Run("hello")
	require.NoError(t, err)
	assert.Contains(t, out, "hello-from-make")
}

func TestMakefileDescription(t *testing.T) {
	h := NewHarness(t).WithMakefile(`
.PHONY: build

build: ## Build the project
	@echo "building"
`)

	out, err := h.Run("--ls")
	require.NoError(t, err)
	assert.Contains(t, out, "Build the project")
}

func TestMiseDetect(t *testing.T) {
	h := NewHarness(t).WithMise(`
[tasks.greet]
description = "Say hello"
run = "echo mise-hello"
`)

	out, err := h.Run("--ls")
	require.NoError(t, err)
	assert.Contains(t, out, "greet")
	assert.Contains(t, out, "mise")
}

func TestMiseExec(t *testing.T) {
	h := NewHarness(t).WithMise(`
[tasks.greet]
description = "Say hello"
run = "echo mise-hello"
`)

	out, err := h.Run("greet")
	require.NoError(t, err)
	assert.Contains(t, out, "mise-hello")
}

func TestTaskfileDetect(t *testing.T) {
	h := NewHarness(t).WithTaskfile(`
version: '3'
tasks:
  greet:
    desc: Say hello
    cmds:
      - echo "task-hello"
`)

	out, err := h.Run("--ls")
	require.NoError(t, err)
	assert.Contains(t, out, "greet")
	assert.Contains(t, out, "task")
}

func TestTaskfileExec(t *testing.T) {
	h := NewHarness(t).WithTaskfile(`
version: '3'
tasks:
  greet:
    desc: Say hello
    cmds:
      - echo "task-hello"
`)

	out, err := h.Run("greet")
	require.NoError(t, err)
	assert.Contains(t, out, "task-hello")
}

func TestPriorityMakeOverMise(t *testing.T) {
	h := NewHarness(t).
		WithMakefile(`
.PHONY: build

build: ## Make build
	@echo "from-make"
`).
		WithMise(`
[tasks.build]
description = "Mise build"
run = "echo from-mise"
`)

	out, err := h.Run("build")
	require.NoError(t, err)
	assert.Contains(t, out, "from-make")
	assert.NotContains(t, out, "from-mise")
}

func TestPriorityMakeOverTaskfile(t *testing.T) {
	h := NewHarness(t).
		WithMakefile(`
.PHONY: build

build: ## Make build
	@echo "from-make"
`).
		WithTaskfile(`
version: '3'
tasks:
  build:
    desc: Task build
    cmds:
      - echo "from-task"
`)

	out, err := h.Run("build")
	require.NoError(t, err)
	assert.Contains(t, out, "from-make")
	assert.NotContains(t, out, "from-task")
}

func TestPriorityTaskfileOverMise(t *testing.T) {
	h := NewHarness(t).
		WithTaskfile(`
version: '3'
tasks:
  build:
    desc: Task build
    cmds:
      - echo "from-task"
`).
		WithMise(`
[tasks.build]
description = "Mise build"
run = "echo from-mise"
`)

	out, err := h.Run("build")
	require.NoError(t, err)
	assert.Contains(t, out, "from-task")
	assert.NotContains(t, out, "from-mise")
}

func TestMergedTaskList(t *testing.T) {
	h := NewHarness(t).
		WithMakefile(`
.PHONY: build

build: ## Make build
	@echo "from-make"
`).
		WithMise(`
[tasks.deploy]
description = "Mise deploy"
run = "echo deploy"
`)

	out, err := h.Run("--ls")
	require.NoError(t, err)
	assert.Contains(t, out, "build")
	assert.Contains(t, out, "make")
	assert.Contains(t, out, "deploy")
	assert.Contains(t, out, "mise")
}

func TestArgForwarding(t *testing.T) {
	h := NewHarness(t).WithMise(`
[tasks.echo]
description = "Echo args"
run = "echo args: $@"
`)

	out, err := h.Run("echo", "--", "one", "two")
	require.NoError(t, err)
	assert.Contains(t, out, "args: one two")
}

func TestExitCodePassthrough(t *testing.T) {
	h := NewHarness(t).WithMakefile(`
.PHONY: fail

fail:
	@exit 42
`)

	_, err := h.Run("fail")
	require.Error(t, err)
}

func TestTaskNotFound(t *testing.T) {
	h := NewHarness(t).WithMakefile(`
.PHONY: build

build:
	@echo "build"
`)

	out, err := h.Run("nonexistent")
	require.Error(t, err)
	assert.Contains(t, out, "not found")
}

func TestEchoPrefix(t *testing.T) {
	h := NewHarness(t).WithMakefile(`
.PHONY: hello

hello: ## Say hi
	@echo "hi"
`)

	out, err := h.Run("hello")
	require.NoError(t, err)
	assert.Contains(t, out, "[mi]")
	assert.Contains(t, out, "make hello")
}

func TestMiseEchoPrefix(t *testing.T) {
	h := NewHarness(t).WithMise(`
[tasks.hello]
description = "Say hello"
run = "echo hi"
`)

	out, err := h.Run("hello")
	require.NoError(t, err)
	assert.Contains(t, out, "[mi]")
	assert.Contains(t, out, "mise run hello")
}

func TestListDefault(t *testing.T) {
	h := NewHarness(t).WithMakefile(`
.PHONY: build test

build: ## Build it
	@echo "build"

test: ## Test it
	@echo "test"
`)

	// no args should list tasks
	out, err := h.Run()
	require.NoError(t, err)
	assert.Contains(t, out, "build")
	assert.Contains(t, out, "test")
}

func TestColonInTaskName(t *testing.T) {
	h := NewHarness(t).WithMise(`
[tasks."test:watch"]
description = "Watch tests"
run = "echo watching"
`)

	out, err := h.Run("--ls")
	require.NoError(t, err)
	assert.Contains(t, out, "test:watch")
}

func TestSeparatorForwarding(t *testing.T) {
	// Verify that -- is preserved when forwarding to runners
	h := NewHarness(t).WithMise(`
[tasks.echo]
description = "Echo args"
run = "echo hello"
`)

	out, err := h.Run("echo", "--", "--verbose")
	require.NoError(t, err)
	// The [mi] prefix should show the full command with --
	lines := strings.Split(out, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "[mi]") && strings.Contains(line, "-- --verbose") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected [mi] echo line to contain '-- --verbose', got: %s", out)
}
