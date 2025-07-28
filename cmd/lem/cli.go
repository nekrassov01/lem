package main

import (
	"context"
	"io"

	"github.com/fatih/color"
	"github.com/nekrassov01/lem"
	"github.com/nekrassov01/mintab"
	"github.com/urfave/cli/v3"
)

var red = color.New(color.FgRed).SprintFunc()

func newCmd(w, ew io.Writer) *cli.Command {
	config := &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "set configuration file path",
		Value:   "lem.toml",
	}
	before := func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		path := cmd.String(config.Name)
		cfg, err := lem.Load(path)
		if err != nil {
			return nil, err
		}
		cmd.Metadata["config"] = cfg
		return ctx, nil
	}
	return &cli.Command{
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return lem.Init()
				},
			},
			{
				Name:        "validate",
				Usage:       "Validate that the configuration file is executable",
				Description: "Validate validates whether the configuration file in the current directory is executable.\nIn addition to syntax checks, it also checks whether the path exists.",
				Before:      before,
				Flags:       []cli.Flag{config},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := cmd.Metadata["config"].(*lem.Config)
					return cfg.Validate()
				},
			},
			{
				Name:        "stage",
				Usage:       "Show the current stage context",
				Description: "Stage displays the current stage context based on the configuration.",
				Before:      before,
				Flags:       []cli.Flag{config},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := cmd.Metadata["config"].(*lem.Config)
					return cfg.Current()
				},
			},
			{
				Name:        "switch",
				Usage:       "Toggles the current stage to the specified stage",
				Description: "Switch changes the current stage to the specified stage based on the state file.\nIf there is no state file, it will be created.",
				Before:      before,
				Flags:       []cli.Flag{config},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := cmd.Metadata["config"].(*lem.Config)
					if err := cfg.Switch(cmd.Args().Get(0)); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:        "list",
				Usage:       "Show the env file entries in the current stage",
				Description: "List resolves and displays a list of env file entries for the current stage based on the configuration.",
				Before:      before,
				Flags:       []cli.Flag{config},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := cmd.Metadata["config"].(*lem.Config)
					entries, err := cfg.List()
					if err != nil {
						return err
					}
					table := mintab.New(cmd.Writer,
						mintab.WithFormat(mintab.CompressedTextFormat),
						mintab.WithMergeFields([]int{0, 1}),
					)
					if err := table.Load(entries); err != nil {
						return err
					}
					table.Render()
					return nil
				},
			},
			{
				Name:        "run",
				Usage:       "Deliver env files to the specified directories based on configuration",
				Description: "Run splits the central env based on configuration and distributes it to each directory.\nIt also checks for empty values based on configuration.",
				Before:      before,
				Flags:       []cli.Flag{config},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := cmd.Metadata["config"].(*lem.Config)
					if _, err := cfg.Run(); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:        "watch",
				Usage:       "Watch changes in the central env and run continuously",
				Description: "Watch continuously monitors changes in the central env and synchronizes changes to each directory.",
				Before:      before,
				Flags:       []cli.Flag{config},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := cmd.Metadata["config"].(*lem.Config)
					if _, err := cfg.Watch(); err != nil {
						return err
					}
					return nil
				},
			},
		},
	}
}
