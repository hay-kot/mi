package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/hay-kot/mi/internal/commands"
	"github.com/hay-kot/mi/internal/config"
	"github.com/hay-kot/mi/internal/paths"
	"github.com/hay-kot/mi/internal/runner"
)

var (
	// Build information. Populated at build-time via -ldflags flag.
	version = "dev"
	commit  = "HEAD"
	date    = "now"
)

func build() string {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			version = info.Main.Version
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					commit = s.Value
				case "vcs.time":
					date = s.Value
				}
			}
		}
	}

	short := commit
	if len(commit) > 7 {
		short = commit[:7]
	}

	return fmt.Sprintf("%s (%s) %s", version, short, date)
}

func setupLogger(level string, logFile string, noColor bool) error {
	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %w", err)
	}

	var output io.Writer = zerolog.ConsoleWriter{Out: os.Stderr, NoColor: noColor}

	if logFile != "" {
		// Create log directory if it doesn't exist
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open log file
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		// Write to both console and file
		output = io.MultiWriter(
			zerolog.ConsoleWriter{Out: os.Stderr, NoColor: noColor},
			file,
		)
	}

	log.Logger = log.Output(output).Level(parsedLevel)

	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	flags := &commands.Flags{}

	stopAfterTask := 1
	app := &cli.Command{
		Name:                  "mi",
		Usage:                 `mi is a small cli that wraps makefiles, taskfiles, and equalivents and proxies to the correct task action`,
		Version:               build(),
		EnableShellCompletion: true,
		StopOnNthArg:          &stopAfterTask,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "log level (debug, info, warn, error, fatal, panic)",
				Sources:     cli.EnvVars("LOG_LEVEL"),
				Value:       "info",
				Destination: &flags.LogLevel,
			},
			&cli.BoolFlag{
				Name:        "no-color",
				Usage:       "disable colored output",
				Sources:     cli.EnvVars("NO_COLOR"),
				Destination: &flags.NoColor,
			},
			&cli.StringFlag{
				Name:        "log-file",
				Usage:       "path to log file (optional)",
				Sources:     cli.EnvVars("LOG_FILE"),
				Destination: &flags.LogFile,
			},
			&cli.StringFlag{
				Name:        "config",
				Usage:       "path to config file",
				Sources:     cli.EnvVars("CONFIG_FILE"),
				Destination: &flags.ConfigFile,
			},
			&cli.BoolFlag{
				Name:        "ls",
				Usage:       "list available tasks across all runners",
				Destination: &flags.List,
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			cfg, err := func() (config.Config, error) {
				if flags.ConfigFile != "" {
					return config.ReadFrom(flags.ConfigFile)
				}
				return config.Read()
			}()
			if err != nil {
				return ctx, fmt.Errorf("loading config: %w", err)
			}

			if flags.LogLevel == "info" && cfg.LogLevel != "" {
				flags.LogLevel = cfg.LogLevel
			}
			if flags.LogFile == "" && cfg.LogFile != "" {
				flags.LogFile = cfg.LogFile
			}
			logFile := flags.LogFile
			if logFile == "" {
				logFile = filepath.Join(paths.DataDir(), "mi.log")
			}

			if err := setupLogger(flags.LogLevel, logFile, flags.NoColor); err != nil {
				return ctx, err
			}

			return ctx, nil
		},
	}
	runCmd := commands.NewRunCmd(flags)
	app.Action = runCmd.Action
	app.ShellComplete = runCmd.ShellComplete

	// +scaffold:command:register

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, os.Args); err != nil {
		var exitErr *runner.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.Code
		}

		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#D75F6B"))
		gray := lipgloss.NewStyle().Foreground(lipgloss.Color("#A3A3A3"))
		fmt.Fprintf(os.Stderr, "\n%s\n%s %s\n%s\n",
			red.Render("╭ Error"),
			red.Render("│"),
			gray.Render(err.Error()),
			red.Render("╵"),
		)
		return 1
	}

	return 0
}
