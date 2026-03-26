package commands

import (
	"context"
	"fmt"
	"os"
	"sort"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/urfave/cli/v3"

	"github.com/hay-kot/mi/internal/runner"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D75F6B"))
	taskStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#C5ADF9"))
	runnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A3A3A3"))
	descStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))
)

type LsCmd struct {
	flags *Flags
}

func NewLsCmd(flags *Flags) *LsCmd {
	return &LsCmd{flags: flags}
}

func (cmd *LsCmd) Register(app *cli.Command) *cli.Command {
	app.Commands = append(app.Commands, &cli.Command{
		Name:   "ls",
		Usage:  "list available tasks across all runners",
		Action: cmd.run,
	})
	return app
}

func (cmd *LsCmd) run(ctx context.Context, c *cli.Command) error {
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

	renderTaskTable(tasks)
	return nil
}

func renderTaskTable(tasks []runner.Task) {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Name < tasks[j].Name })

	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, []string{t.Name, t.Runner, t.Desc})
	}

	t := table.New().
		Headers("TASK", "RUNNER", "DESCRIPTION").
		Rows(rows...).
		Border(lipgloss.HiddenBorder()).
		BorderHeader(false).
		BorderRow(false).
		BorderColumn(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle.PaddingRight(2)
			}
			switch col {
			case 0:
				return taskStyle.PaddingRight(2)
			case 1:
				return runnerStyle.PaddingRight(2)
			default:
				return descStyle
			}
		})

	fmt.Println(t.Render())
}
