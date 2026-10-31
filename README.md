# mi

A unified task runner proxy. Run `mi <task>` and it finds the right runner automatically.

Supports **Makefile**, **Taskfile**, **mise**, and **package.json** scripts with a fixed priority order: Makefile > Taskfile > mise > package.json. The first runner that has the task wins.

## Installation

```bash
go install github.com/hay-kot/mi@latest
```

## Usage

```bash
# Run a task (finds the right runner automatically)
mi build
mi test
mi check

# List all available tasks across runners
mi --ls

# Forward arguments to the underlying runner
mi test ./pkg/...
mi run -- --help

# Shell completion
mi --help
```

## Shell Completion

### Zsh

```bash
# Option A: eval in .zshrc
eval "$(mi completion zsh)"

# Option B: save to fpath (faster startup)
mi completion zsh > ~/.zsh/completions/_mi
```

Make sure your completions directory is in `fpath` before `compinit`:

```bash
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit
```

### Bash

```bash
eval "$(mi completion bash)"
```

### Fish

```bash
mi completion fish > ~/.config/fish/completions/mi.fish
```

## How It Works

When you run `mi <task>`, it:

1. Concurrently detects which runners are available in the current directory
2. Walks runners in priority order (Makefile > Taskfile > mise > package.json)
3. Finds the first runner that defines the task
4. Forwards all arguments and executes it, passing through the exit code

```
$ mi tidy
[mi] make tidy
go mod tidy
```

## Runner Detection

| Runner       | Config Files                                     |
|--------------|--------------------------------------------------|
| Makefile     | `Makefile`, `makefile`, `GNUmakefile`            |
| Taskfile     | `Taskfile.yml`, `Taskfile.yaml`, `taskfile.yml`  |
| mise         | `mise.toml`, `.mise.toml`, `.mise/tasks/`        |
| package.json | `package.json` in the current directory          |

## Package Managers

`package.json` scripts run with the project's package manager. `mi` checks the current directory, then each parent up to the git root, and the nearest match wins. Within a directory, it checks in this order:

1. The `packageManager` field, then `devEngines.packageManager`
2. Marker files, in this order:

| Package manager | Marker files                               | Runs with        |
|-----------------|--------------------------------------------|------------------|
| deno            | `deno.json`, `deno.jsonc`                  | `deno task`      |
| bun             | `bun.lock`, `bun.lockb`                    | `bun run`        |
| deno            | `deno.lock`                                | `deno task`      |
| pnpm            | `pnpm-lock.yaml`, `pnpm-workspace.yaml`    | `pnpm run`       |
| yarn            | `yarn.lock`                                | `yarn run`       |
| npm             | `package-lock.json`, `npm-shrinkwrap.json` | `npm run`        |

If nothing matches, `mi` uses `npm`.

Arguments reach the script the same way under every package manager, so `mi test --watch` and `mi test -- --watch` are equivalent. `mi` adds a `--` only where the package manager needs one.

Scripts that package managers run implicitly (`postinstall`, `prepare`, or `prebuild` when `build` exists) are left out of `mi --ls` and completion, but `mi prepare` still runs them.

Task descriptions come from `scripts-info`, then a `?script` entry, then the script itself.

## License

[MIT](LICENSE)
