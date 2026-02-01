package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func authCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Authenticate with FreeAgent",
		Subcommands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Start OAuth flow",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "manual",
						Usage: "Use manual code paste instead of local callback",
					},
					&cli.StringFlag{
						Name:  "redirect",
						Usage: "Override redirect URI",
					},
				},
				Action: func(c *cli.Context) error {
					rt := runtimeFrom(c)
					return fmt.Errorf("auth login not implemented yet (profile %s)", rt.Profile)
				},
			},
			{
				Name:  "status",
				Usage: "Show current auth status",
				Action: func(c *cli.Context) error {
					return fmt.Errorf("auth status not implemented yet")
				},
			},
			{
				Name:  "refresh",
				Usage: "Refresh access token",
				Action: func(c *cli.Context) error {
					return fmt.Errorf("auth refresh not implemented yet")
				},
			},
		},
	}
}
