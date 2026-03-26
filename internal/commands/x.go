package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// XCmd implements the x command
type XCmd struct {
	flags *Flags
}

// NewXCmd creates a new x command
func NewXCmd(flags *Flags) *XCmd {
	return &XCmd{flags: flags}
}

// Register adds the x command to the application
func (cmd *XCmd) Register(app *cli.Command) *cli.Command {
	app.Commands = append(app.Commands, &cli.Command{
		Name:  "x",
		Usage: "x command",
		Flags: []cli.Flag{
			// Add command-specific flags here
		},
		Action: cmd.run,
	})

	return app
}

func (cmd *XCmd) run(ctx context.Context, c *cli.Command) error {
	log.Info().Msg("running x command")

	fmt.Println("Hello World!")

	return nil
}
