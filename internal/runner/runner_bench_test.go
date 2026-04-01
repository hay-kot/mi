package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupBenchDir(b *testing.B) string {
	b.Helper()
	dir := b.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("build:\n\techo build\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("version: '3'\ntasks:\n  build:\n    cmds:\n      - echo build\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mise.toml"), []byte("[tasks.build]\nrun = \"echo build\"\n"), 0o644); err != nil {
		b.Fatal(err)
	}

	return dir
}

func setupBenchDirEmpty(b *testing.B) string {
	b.Helper()
	return b.TempDir()
}

// --- Detection (now Available via LookPath) ---

func BenchmarkDetection(b *testing.B) {
	runners := DefaultPriority()
	b.ResetTimer()
	for b.Loop() {
		Available(runners)
	}
}

func BenchmarkDetection_Empty(b *testing.B) {
	runners := DefaultPriority()
	b.ResetTimer()
	for b.Loop() {
		Available(runners)
	}
}

// --- LookPath ---

func BenchmarkLookPath_Task(b *testing.B) {
	for b.Loop() {
		_, _ = exec.LookPath("task")
	}
}

func BenchmarkLookPath_Mise(b *testing.B) {
	for b.Loop() {
		_, _ = exec.LookPath("mise")
	}
}

func BenchmarkLookPath_Make(b *testing.B) {
	for b.Loop() {
		_, _ = exec.LookPath("make")
	}
}

// --- CLI ListTasks ---

func BenchmarkListTasks_CLI_Task(b *testing.B) {
	if _, err := exec.LookPath("task"); err != nil {
		b.Skip("task not installed")
	}
	dir := setupBenchDir(b)
	r := &TaskfileRunner{}
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		_, _ = r.ListTasks(ctx, dir)
	}
}

func BenchmarkListTasks_CLI_Task_NoConfig(b *testing.B) {
	if _, err := exec.LookPath("task"); err != nil {
		b.Skip("task not installed")
	}
	dir := setupBenchDirEmpty(b)
	r := &TaskfileRunner{}
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		_, _ = r.ListTasks(ctx, dir)
	}
}

func BenchmarkListTasks_CLI_Mise(b *testing.B) {
	if _, err := exec.LookPath("mise"); err != nil {
		b.Skip("mise not installed")
	}
	dir := setupBenchDir(b)
	r := &MiseRunner{}
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		_, _ = r.ListTasks(ctx, dir)
	}
}

func BenchmarkListTasks_CLI_Mise_NoConfig(b *testing.B) {
	if _, err := exec.LookPath("mise"); err != nil {
		b.Skip("mise not installed")
	}
	dir := setupBenchDirEmpty(b)
	r := &MiseRunner{}
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		_, _ = r.ListTasks(ctx, dir)
	}
}

func BenchmarkListTasks_Make(b *testing.B) {
	dir := setupBenchDir(b)
	r := &MakeRunner{}
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		_, _ = r.ListTasks(ctx, dir)
	}
}

// --- Full flow: detect/filter → resolve ---

func BenchmarkFullFlow_Resolve(b *testing.B) {
	dir := setupBenchDir(b)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		runners := Available(DefaultPriority())
		_, _ = Resolve(ctx, runners, dir, "build")
	}
}

func BenchmarkFullFlow_ListAll(b *testing.B) {
	dir := setupBenchDir(b)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		runners := Available(DefaultPriority())
		_, _ = ListAll(ctx, runners, dir)
	}
}
