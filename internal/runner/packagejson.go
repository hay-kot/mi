package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/tailscale/hujson"
)

// maxScriptDescLen keeps script bodies from widening every row of the task table.
const maxScriptDescLen = 60

var packageManagerMajorRe = regexp.MustCompile(`\d+`)

// packageManagerMarkers is ordered most specific first, matching ni so both
// tools choose the same package manager when a repo has several of these files.
var packageManagerMarkers = []struct{ file, name string }{
	{"deno.json", "deno"},
	{"deno.jsonc", "deno"},
	{"bun.lock", "bun"},
	{"bun.lockb", "bun"},
	{"deno.lock", "deno"},
	{"pnpm-lock.yaml", "pnpm"},
	{"pnpm-workspace.yaml", "pnpm"},
	{"yarn.lock", "yarn"},
	{"package-lock.json", "npm"},
	{"npm-shrinkwrap.json", "npm"},
}

// lifecycleScripts run as a side effect of install and publish, so they are
// noise in a task list. Generic lifecycle names like "publish" and "version"
// are left out because projects often define them as ordinary scripts.
var lifecycleScripts = map[string]bool{
	"preinstall":     true,
	"install":        true,
	"postinstall":    true,
	"prepare":        true,
	"prepublish":     true,
	"prepublishOnly": true,
	"prepack":        true,
	"postpack":       true,
}

type PackageJSONRunner struct{}

func (p *PackageJSONRunner) Name() string { return "package.json" }

// Bin returns "" because the binary depends on the directory; Exec reports a
// missing package manager instead.
func (p *PackageJSONRunner) Bin() string { return "" }

type packageManifest struct {
	Scripts        map[string]any `json:"scripts"`
	ScriptsInfo    map[string]any `json:"scripts-info"`
	PackageManager any            `json:"packageManager"`
	DevEngines     struct {
		PackageManager any `json:"packageManager"`
	} `json:"devEngines"`
}

func readPackageManifest(dir string) (packageManifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return packageManifest{}, err
	}

	var manifest packageManifest
	err = json.Unmarshal(data, &manifest)
	if err == nil {
		return manifest, nil
	}

	// Bun reads package.json as JSONC, so its projects can have comments and
	// trailing commas. A manifest that is still broken reports the original error.
	standard, stdErr := hujson.Standardize(data)
	if stdErr != nil {
		return packageManifest{}, err
	}
	manifest = packageManifest{}
	if json.Unmarshal(standard, &manifest) != nil {
		return packageManifest{}, err
	}
	return manifest, nil
}

// declaredPackageManager returns the name and version from the packageManager
// field or devEngines.packageManager. The fields are untyped because devEngines
// also allows an array, which falls through to lockfile detection.
func (m packageManifest) declaredPackageManager() (name, version string) {
	if field, ok := m.PackageManager.(string); ok {
		name, version, _ = strings.Cut(strings.TrimPrefix(field, "^"), "@")
		return name, version
	}
	if engine, ok := m.DevEngines.PackageManager.(map[string]any); ok {
		name, _ = engine["name"].(string)
		version, _ = engine["version"].(string)
	}
	return name, version
}

type packageManager struct {
	bin     string
	runVerb string
	// separator is whether forwarded args need a "--" before them. npm and yarn
	// classic consume flags without one, while pnpm 7+, yarn berry, and deno
	// hand a "--" to the script as a literal argument.
	separator bool
}

func newPackageManager(name, version, dir string) (packageManager, bool) {
	major := majorVersion(version)
	switch name {
	case "npm", "bun":
		return packageManager{bin: name, runVerb: "run", separator: true}, true
	case "pnpm":
		return packageManager{bin: name, runVerb: "run", separator: major >= 0 && major < 7}, true
	case "yarn":
		// A yarn.lock alone doesn't distinguish classic from berry, but only
		// berry reads .yarnrc.yml.
		berry := major >= 2 || pathExists(filepath.Join(dir, ".yarnrc.yml"))
		return packageManager{bin: name, runVerb: "run", separator: !berry}, true
	case "deno":
		return packageManager{bin: name, runVerb: "task"}, true
	}
	return packageManager{}, false
}

// majorVersion returns -1 when version has no number in it.
func majorVersion(version string) int {
	major, err := strconv.Atoi(packageManagerMajorRe.FindString(version))
	if err != nil {
		return -1
	}
	return major
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func packageManagerIn(dir string) (packageManager, bool) {
	if manifest, err := readPackageManifest(dir); err == nil {
		name, version := manifest.declaredPackageManager()
		if pm, ok := newPackageManager(name, version, dir); ok {
			return pm, true
		}
	}
	for _, marker := range packageManagerMarkers {
		if pathExists(filepath.Join(dir, marker.file)) {
			return newPackageManager(marker.name, "", dir)
		}
	}
	return packageManager{}, false
}

// detectPackageManager walks up from dir because a workspace package usually
// has no lockfile of its own. The walk stops at the git root so that a stray
// lockfile in a parent like $HOME can't change how a repo runs.
func detectPackageManager(dir string) packageManager {
	for {
		if pm, ok := packageManagerIn(dir); ok {
			return pm
		}
		parent := filepath.Dir(dir)
		if parent == dir || pathExists(filepath.Join(dir, ".git")) {
			pm, _ := newPackageManager("npm", "", dir)
			return pm
		}
		dir = parent
	}
}

// command normalizes args so the script receives the same argv from every
// package manager. A leading "--" is how mi users separate forwarded args, so
// it is dropped here and added back only where the package manager needs it.
func (pm packageManager) command(task string, args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	cmd := []string{pm.bin, pm.runVerb, task}
	if pm.separator && len(args) > 0 {
		cmd = append(cmd, "--")
	}
	return append(cmd, args...)
}

func (p *PackageJSONRunner) ListTasks(ctx context.Context, dir string) ([]Task, error) {
	manifest, err := readPackageManifest(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	bin := detectPackageManager(dir).bin
	tasks := make([]Task, 0, len(manifest.Scripts))
	for name, body := range manifest.Scripts {
		cmd, ok := body.(string)
		// ni's "?script" convention stores the description of "script".
		if !ok || strings.HasPrefix(name, "?") {
			continue
		}
		tasks = append(tasks, Task{
			Name:   name,
			Desc:   manifest.scriptDesc(name, cmd),
			Runner: bin,
			Hidden: isHiddenScript(name, manifest.Scripts),
		})
	}
	return tasks, nil
}

func (m packageManifest) scriptDesc(name, cmd string) string {
	if desc, ok := m.ScriptsInfo[name].(string); ok && desc != "" {
		return desc
	}
	if desc, ok := m.Scripts["?"+name].(string); ok && desc != "" {
		return desc
	}
	runes := []rune(cmd)
	if len(runes) > maxScriptDescLen {
		return string(runes[:maxScriptDescLen-3]) + "..."
	}
	return cmd
}

// isHiddenScript reports scripts that package managers run implicitly. Matching
// a pre/post prefix against an existing script keeps names like "preview" visible.
func isHiddenScript(name string, scripts map[string]any) bool {
	if lifecycleScripts[name] {
		return true
	}
	for _, prefix := range []string{"pre", "post"} {
		if base, ok := strings.CutPrefix(name, prefix); ok {
			if _, exists := scripts[base]; exists {
				return true
			}
		}
	}
	return false
}

func (p *PackageJSONRunner) CmdString(dir string, task string, args []string) string {
	return strings.Join(detectPackageManager(dir).command(task, args), " ")
}

func (p *PackageJSONRunner) Exec(ctx context.Context, dir string, task string, args []string) error {
	pm := detectPackageManager(dir)
	path, err := exec.LookPath(pm.bin)
	if err != nil {
		return fmt.Errorf("this project uses %s, but %s is not on PATH", pm.bin, pm.bin)
	}

	parts := pm.command(task, args)
	err = p.run(ctx, dir, parts)
	// npm's pnpm package ships a placeholder script with no shebang until its
	// install script replaces it, which ignore-scripts=true skips. Shells run
	// such files but exec doesn't, and ENOEXEC means nothing started, so a retry
	// can't run the script twice.
	if errors.Is(err, syscall.ENOEXEC) {
		return p.run(ctx, dir, append([]string{"sh", path}, parts[1:]...))
	}
	return err
}

func (p *PackageJSONRunner) run(ctx context.Context, dir string, parts []string) error {
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return ExecExitCode(cmd.Run())
}
