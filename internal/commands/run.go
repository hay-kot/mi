package commands

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/urfave/cli/v3"

	"github.com/hay-kot/mi/internal/runner"
)

// RunCmd implements the default root action that dispatches to task runners.
type RunCmd struct {
	flags *Flags
}

func NewRunCmd(flags *Flags) *RunCmd {
	return &RunCmd{flags: flags}
}

func (cmd *RunCmd) Action(ctx context.Context, c *cli.Command) error {
	if cmd.flags.List {
		return cmd.list(ctx)
	}

	taskName := c.Args().First()
	if taskName == "" {
		return cli.ShowAppHelp(c)
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	runners := runner.DetectAll(runner.DefaultPriority(), dir)
	if len(runners) == 0 {
		return fmt.Errorf("no task runner found (looked for Makefile, Taskfile, mise.toml)")
	}

	r, err := runner.Resolve(ctx, runners, dir, taskName)
	if err != nil {
		return err
	}

	// Extract args from os.Args directly to preserve "--" separators
	// that urfave/cli strips from c.Args().
	args := argsAfterTask(os.Args, taskName)

	prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#C5ADF9")).Render("[mi]")
	cmdStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#A3A3A3")).Render(r.CmdString(taskName, args))
	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, cmdStr)

	return r.Exec(ctx, dir, taskName, args)
}

func (cmd *RunCmd) list(ctx context.Context) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	runners := runner.DetectAll(runner.DefaultPriority(), dir)
	if len(runners) == 0 {
		fmt.Println("no task runners found")
		return nil
	}

	tasks, err := runner.ListAll(ctx, runners, dir)
	if err != nil {
		return fmt.Errorf("listing tasks: %w", err)
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Name < tasks[j].Name })

	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, []string{t.Name, t.Runner, t.Desc})
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D75F6B")).PaddingRight(2)
	taskStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#C5ADF9")).PaddingRight(2)
	runnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A3A3A3")).PaddingRight(2)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))

	t := table.New().
		Headers("TASK", "RUNNER", "DESCRIPTION").
		Rows(rows...).
		Border(lipgloss.HiddenBorder()).
		BorderHeader(false).
		BorderRow(false).
		BorderColumn(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			switch col {
			case 0:
				return taskStyle
			case 1:
				return runnerStyle
			default:
				return descStyle
			}
		})

	fmt.Println(t.Render())
	return nil
}

func (cmd *RunCmd) ShellComplete(ctx context.Context, c *cli.Command) {
	if c.NArg() > 0 {
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	runners := runner.DetectAll(runner.DefaultPriority(), dir)
	tasks, err := runner.ListAll(ctx, runners, dir)
	if err != nil {
		return
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Name < tasks[j].Name })
	for _, t := range tasks {
		// Escape colons — zsh uses ':' as a descriptor separator
		name := strings.ReplaceAll(t.Name, ":", "\\:")
		fmt.Println(name)
	}
}

// argsAfterTask finds the task name in os.Args and returns everything after it,
// preserving "--" separators that urfave/cli would otherwise strip.
func argsAfterTask(osArgs []string, taskName string) []string {
	for i, arg := range osArgs {
		if arg == taskName {
			return osArgs[i+1:]
		}
	}
	return nil
}
