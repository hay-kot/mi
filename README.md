# mi

A unified task runner proxy. Run `mi <task>` and it finds the right runner automatically.

Supports **Makefile**, **Taskfile**, and **mise** with a fixed priority order: Makefile > Taskfile > mise. The first runner that has the task wins.

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
2. Walks runners in priority order (Makefile > Taskfile > mise)
3. Finds the first runner that defines the task
4. Forwards all arguments and executes it, passing through the exit code

```
$ mi tidy
[mi] make tidy
go mod tidy
```

## Runner Detection

| Runner   | Config Files                                     |
|----------|--------------------------------------------------|
| Makefile | `Makefile`, `makefile`, `GNUmakefile`            |
| Taskfile | `Taskfile.yml`, `Taskfile.yaml`, `taskfile.yml`  |
| mise     | `mise.toml`, `.mise.toml`, `.mise/tasks/`        |

## License

[MIT](LICENSE)
