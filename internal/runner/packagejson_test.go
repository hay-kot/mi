package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	manifestFile = "package.json"
	pnpmLock     = "pnpm-lock.yaml"
	yarnLock     = "yarn.lock"
	watchFlag    = "--watch"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
}

// repoDir returns a directory whose .git keeps detection from walking into
// the real parents of the temp directory.
func repoDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	writeFiles(t, dir, files)
	return dir
}

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"defaults to npm", nil, "npm"},
		{"package-lock", map[string]string{"package-lock.json": ""}, "npm"},
		{"shrinkwrap", map[string]string{"npm-shrinkwrap.json": ""}, "npm"},
		{"pnpm lock", map[string]string{pnpmLock: ""}, "pnpm"},
		{"pnpm workspace", map[string]string{"pnpm-workspace.yaml": ""}, "pnpm"},
		{"yarn lock", map[string]string{yarnLock: ""}, "yarn"},
		{"bun text lock", map[string]string{"bun.lock": ""}, "bun"},
		{"bun binary lock", map[string]string{"bun.lockb": ""}, "bun"},
		{"deno config", map[string]string{"deno.json": ""}, "deno"},
		{"deno lock", map[string]string{"deno.lock": ""}, "deno"},
		{"stray npm lock loses to pnpm", map[string]string{"package-lock.json": "", pnpmLock: ""}, "pnpm"},
		{"bun beats pnpm like ni", map[string]string{"bun.lock": "", pnpmLock: ""}, "bun"},
		{
			"packageManager beats lockfile",
			map[string]string{manifestFile: `{"packageManager":"pnpm@9.1.0+sha512.abc"}`, yarnLock: ""},
			"pnpm",
		},
		{
			"caret packageManager",
			map[string]string{manifestFile: `{"packageManager":"^bun@1.2.0"}`},
			"bun",
		},
		{
			"devEngines beats lockfile",
			map[string]string{manifestFile: `{"devEngines":{"packageManager":{"name":"yarn","version":"^4.0.0"}}}`, pnpmLock: ""},
			"yarn",
		},
		{
			"packageManager beats devEngines",
			map[string]string{manifestFile: `{"packageManager":"pnpm@9.0.0","devEngines":{"packageManager":{"name":"yarn"}}}`},
			"pnpm",
		},
		{
			"devEngines array falls back to lockfile",
			map[string]string{manifestFile: `{"devEngines":{"packageManager":[{"name":"yarn"}]}}`, pnpmLock: ""},
			"pnpm",
		},
		{
			"unknown packageManager falls back to lockfile",
			map[string]string{manifestFile: `{"packageManager":"cnpm@1.0.0"}`, pnpmLock: ""},
			"pnpm",
		},
		{
			"non-string packageManager falls back to lockfile",
			map[string]string{manifestFile: `{"packageManager":true}`, pnpmLock: ""},
			"pnpm",
		},
		{
			"invalid manifest falls back to lockfile",
			map[string]string{manifestFile: `{nope`, yarnLock: ""},
			"yarn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detectPackageManager(repoDir(t, tt.files)).bin)
		})
	}
}

func TestDetectPackageManagerWalksUp(t *testing.T) {
	t.Run("workspace package uses root lockfile", func(t *testing.T) {
		root := repoDir(t, map[string]string{
			pnpmLock:                    "",
			"packages/foo/package.json": `{"scripts":{"test":"x"}}`,
		})
		assert.Equal(t, "pnpm", detectPackageManager(filepath.Join(root, "packages/foo")).bin)
	})

	t.Run("nearest directory wins", func(t *testing.T) {
		root := repoDir(t, map[string]string{
			pnpmLock:           "",
			"nested/yarn.lock": "",
		})
		assert.Equal(t, "yarn", detectPackageManager(filepath.Join(root, "nested")).bin)
	})

	t.Run("stops at the git root", func(t *testing.T) {
		parent := t.TempDir()
		writeFiles(t, parent, map[string]string{yarnLock: "", "repo/.git/HEAD": ""})
		assert.Equal(t, "npm", detectPackageManager(filepath.Join(parent, "repo")).bin)
	})

	t.Run("treats a worktree .git file as the root", func(t *testing.T) {
		parent := t.TempDir()
		writeFiles(t, parent, map[string]string{yarnLock: "", "repo/.git": "gitdir: elsewhere"})
		assert.Equal(t, "npm", detectPackageManager(filepath.Join(parent, "repo")).bin)
	})
}

func TestPackageManagerCommand(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		args  []string
		want  string
	}{
		{"npm no args", nil, nil, "npm run test"},
		{"npm adds separator", nil, []string{watchFlag}, "npm run test -- --watch"},
		{"npm keeps one separator", nil, []string{"--", watchFlag}, "npm run test -- --watch"},
		{"npm keeps literal separator", nil, []string{"--", "--", "x"}, "npm run test -- -- x"},
		{"npm bare separator", nil, []string{"--"}, "npm run test"},
		{"pnpm has no separator", map[string]string{pnpmLock: ""}, []string{watchFlag}, "pnpm run test --watch"},
		{"pnpm strips separator", map[string]string{pnpmLock: ""}, []string{"--", watchFlag}, "pnpm run test --watch"},
		{"pnpm keeps inner separator", map[string]string{pnpmLock: ""}, []string{"--foo", "--", "bar"}, "pnpm run test --foo -- bar"},
		{"pnpm 6 needs separator", map[string]string{manifestFile: `{"packageManager":"pnpm@6.32.0"}`}, []string{watchFlag}, "pnpm run test -- --watch"},
		{"pnpm 7 has no separator", map[string]string{manifestFile: `{"packageManager":"pnpm@7.0.0"}`}, []string{watchFlag}, "pnpm run test --watch"},
		{"yarn classic adds separator", map[string]string{yarnLock: ""}, []string{"--foo", "--", "bar"}, "yarn run test -- --foo -- bar"},
		{"yarn classic by version", map[string]string{manifestFile: `{"packageManager":"yarn@1.22.22"}`}, []string{watchFlag}, "yarn run test -- --watch"},
		{"yarn berry by version", map[string]string{manifestFile: `{"packageManager":"yarn@4.1.0"}`}, []string{"--", watchFlag}, "yarn run test --watch"},
		{"yarn berry by yarnrc", map[string]string{yarnLock: "", ".yarnrc.yml": ""}, []string{watchFlag}, "yarn run test --watch"},
		{"bun adds separator", map[string]string{"bun.lock": ""}, []string{"--foo", "--", "bar"}, "bun run test -- --foo -- bar"},
		{"deno uses task", map[string]string{"deno.json": "{}"}, []string{"--", watchFlag}, "deno task test --watch"},
	}

	r := &PackageJSONRunner{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, r.CmdString(repoDir(t, tt.files), "test", tt.args))
		})
	}
}

func listScripts(t *testing.T, dir string) map[string]Task {
	t.Helper()
	tasks, err := (&PackageJSONRunner{}).ListTasks(context.Background(), dir)
	require.NoError(t, err)
	byName := make(map[string]Task, len(tasks))
	for _, task := range tasks {
		byName[task.Name] = task
	}
	return byName
}

func TestListTasksDescriptions(t *testing.T) {
	long := strings.Repeat("x", maxScriptDescLen+1)
	tasks := listScripts(t, repoDir(t, map[string]string{manifestFile: `{
		"scripts": {
			"plain": "vite build",
			"info": "vite",
			"marked": "vitest",
			"?marked": "Run the tests",
			"long": "` + long + `",
			"bad": 1
		},
		"scripts-info": {"info": "Start the dev server"}
	}`}))

	assert.Equal(t, "vite build", tasks["plain"].Desc)
	assert.Equal(t, "Start the dev server", tasks["info"].Desc)
	assert.Equal(t, "Run the tests", tasks["marked"].Desc)
	assert.Len(t, tasks["long"].Desc, maxScriptDescLen)
	assert.True(t, strings.HasSuffix(tasks["long"].Desc, "..."))
	assert.NotContains(t, tasks, "?marked")
	assert.NotContains(t, tasks, "bad")
}

func TestListTasksHidesImplicitScripts(t *testing.T) {
	tasks := listScripts(t, repoDir(t, map[string]string{manifestFile: `{"scripts": {
		"build": "x", "prebuild": "x", "postbuild": "x",
		"preview": "x", "postcss": "x", "publish": "x",
		"postinstall": "x", "prepare": "x"
	}}`}))

	for _, name := range []string{"prebuild", "postbuild", "postinstall", "prepare"} {
		assert.True(t, tasks[name].Hidden, name)
	}
	for _, name := range []string{"build", "preview", "postcss", "publish"} {
		assert.False(t, tasks[name].Hidden, name)
	}
}

func TestHiddenScriptsResolveButDoNotList(t *testing.T) {
	dir := repoDir(t, map[string]string{manifestFile: `{"scripts":{"prepare":"x","build":"x"}}`})
	ctx := context.Background()
	runners := []Runner{&PackageJSONRunner{}}

	r, err := Resolve(ctx, runners, dir, "prepare")
	require.NoError(t, err)
	assert.Equal(t, manifestFile, r.Name())

	tasks, err := ListAll(ctx, runners, dir)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "build", tasks[0].Name)
}

func TestListTasksManifests(t *testing.T) {
	t.Run("no manifest", func(t *testing.T) {
		assert.Empty(t, listScripts(t, repoDir(t, nil)))
	})

	t.Run("only reads the current directory", func(t *testing.T) {
		root := repoDir(t, map[string]string{manifestFile: `{"scripts":{"build":"x"}}`, "sub/file": ""})
		assert.Empty(t, listScripts(t, filepath.Join(root, "sub")))
	})

	t.Run("no scripts", func(t *testing.T) {
		assert.Empty(t, listScripts(t, repoDir(t, map[string]string{manifestFile: `{"name":"x"}`})))
	})

	t.Run("runner column shows the package manager", func(t *testing.T) {
		tasks := listScripts(t, repoDir(t, map[string]string{manifestFile: `{"scripts":{"build":"x"}}`, pnpmLock: ""}))
		assert.Equal(t, "pnpm", tasks["build"].Runner)
	})

	t.Run("jsonc", func(t *testing.T) {
		tasks := listScripts(t, repoDir(t, map[string]string{manifestFile: `{
			// comments and trailing commas are valid in a Bun project
			"scripts": {"build": "x",},
		}`}))
		assert.Contains(t, tasks, "build")
	})

	t.Run("invalid json reports the error", func(t *testing.T) {
		dir := repoDir(t, map[string]string{manifestFile: `{nope`})
		_, err := (&PackageJSONRunner{}).ListTasks(context.Background(), dir)
		assert.Error(t, err)
	})
}

func TestExecMissingPackageManager(t *testing.T) {
	dir := repoDir(t, map[string]string{manifestFile: `{"scripts":{"build":"x"}}`, pnpmLock: ""})
	t.Setenv("PATH", t.TempDir())

	err := (&PackageJSONRunner{}).Exec(context.Background(), dir, "build", nil)
	assert.ErrorContains(t, err, "pnpm is not on PATH")
}

func TestExecRunsScriptWithoutShebang(t *testing.T) {
	binDir := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	writeFiles(t, binDir, map[string]string{"npm": `echo "$@" > "$STUB_OUT"` + "\n"})
	require.NoError(t, os.Chmod(filepath.Join(binDir, "npm"), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("STUB_OUT", out)

	dir := repoDir(t, map[string]string{manifestFile: `{"scripts":{"build":"x"}}`})
	require.NoError(t, (&PackageJSONRunner{}).Exec(context.Background(), dir, "build", []string{watchFlag}))

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "run build -- --watch\n", string(got))
}
