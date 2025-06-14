package main

import (
	"context"
	"io"

	"github.com/fatih/color"
	"github.com/nekrassov01/lem"
	"github.com/urfave/cli/v3"
)

var red = color.New(color.FgRed).SprintFunc()

type app struct {
	*cli.Command
	config *cli.StringFlag
	stage  *cli.StringFlag
}

func newApp(w, ew io.Writer) *app {
	a := app{}
	a.config = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "set configuration file path",
		Value:   "lem.toml",
	}
	a.stage = &cli.StringFlag{
		Name:    "stage",
		Aliases: []string{"s"},
		Usage:   "set stage context to run",
		Value:   "default",
	}
	a.Command = &cli.Command{
		Name:                  "lem",
		Version:               getVersion(),
		Usage:                 "The local env manager for monorepo",
		HideHelpCommand:       true,
		EnableShellCompletion: true,
		Writer:                w,
		ErrWriter:             ew,
		Metadata:              map[string]any{},
		Commands: []*cli.Command{
			{
				Name:        "init",
				Usage:       "Initialize the configuration file to current directory",
				Description: "Init generates a sample lem.toml in the current directory.\nYou can customize this file for your use.",
				Action:      a.init,
			},
			{
				Name:        "validate",
				Usage:       "Validate that the configuration file is executable",
				Description: "Validate validates whether the configuration file in the current directory is executable.\nIn addition to syntax checks, it also checks whether the path exists.",
				Before:      a.before,
				Action:      a.validate,
				Flags:       []cli.Flag{a.config},
			},
			{
				Name:        "run",
				Usage:       "Deliver env files to the specified directories based on configuration",
				Description: "Run splits the central env based on configuration and distributes it to each directory.\nIt also checks for empty values based on configuration.",
				Before:      a.before,
				Action:      a.run,
				Flags:       []cli.Flag{a.config, a.stage},
			},
			{
				Name:        "watch",
				Usage:       "Watch changes in the central env and run continuously",
				Description: "Watch continuously monitors changes in the central env and synchronizes changes to each directory.",
				Before:      a.before,
				Action:      a.watch,
				Flags:       []cli.Flag{a.config, a.stage},
			},
		},
	}
	return &a
}

func (a *app) before(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	path := cmd.String(a.config.Name)
	cfg, err := lem.Load(path)
	if err != nil {
		return nil, err
	}
	a.Metadata["config"] = cfg
	return ctx, nil
}

func (*app) init(context.Context, *cli.Command) error {
	return lem.Init()
}

func (a *app) validate(context.Context, *cli.Command) error {
	cfg := a.Metadata["config"].(*lem.Config)
	return cfg.Validate()
}

func (a *app) run(_ context.Context, cmd *cli.Command) error {
	cfg := a.Metadata["config"].(*lem.Config)
	if _, err := cfg.Run(cmd.String(a.stage.Name)); err != nil {
		return err
	}
	return nil
}

func (a *app) watch(_ context.Context, cmd *cli.Command) error {
	cfg := a.Metadata["config"].(*lem.Config)
	if _, err := cfg.Watch(cmd.String(a.stage.Name)); err != nil {
		return err
	}
	return nil
}
