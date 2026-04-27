package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

func NewApp() *cli.App {
	app := &cli.App{
		Name:  "freeagent",
		Usage: "CLI for the FreeAgent API",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				EnvVars: []string{"FREEAGENT_CONFIG"},
				Usage:   "Path to config file",
			},
			&cli.StringFlag{
				Name:    "profile",
				EnvVars: []string{"FREEAGENT_PROFILE"},
				Value:   "default",
				Usage:   "Credential profile name",
			},
			&cli.BoolFlag{
				Name:  "sandbox",
				Usage: "Use FreeAgent sandbox API",
			},
			&cli.StringFlag{
				Name:    "base-url",
				EnvVars: []string{"FREEAGENT_BASE_URL"},
				Usage:   "Override API base URL",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output raw JSON",
			},
		},
		Before: initRuntime,
		Commands: append([]*cli.Command{
			authCommand(),
			apCommand(),
			bankCommand(),
			billsCommand(),
			categoriesCommand(),
			companyCommand(),
			contactsCommand(),
			creditNotesCommand(),
			estimatesCommand(),
			expensesCommand(),
			invoiceCommand(),
			priceListItemsCommand(),
			projectsCommand(),
			reportsCommand(),
			stockItemsCommand(),
			tasksCommand(),
			timeslipsCommand(),
			usersCommand(),
		}, writeCommands()...),
	}

	cli.AppHelpTemplate = strings.ReplaceAll(cli.AppHelpTemplate, "GLOBAL OPTIONS", "GLOBAL FLAGS")
	return app
}

func initRuntime(c *cli.Context) error {
	rt := Runtime{
		ConfigPath: c.String("config"),
		Profile:    c.String("profile"),
		Sandbox:    c.Bool("sandbox"),
		BaseURL:    c.String("base-url"),
		JSONOutput: c.Bool("json"),
	}

	if rt.Profile == "" {
		return errors.New("profile cannot be empty")
	}

	if rt.BaseURL == "" {
		if rt.Sandbox {
			rt.BaseURL = "https://api.sandbox.freeagent.com/v2"
		} else {
			rt.BaseURL = "https://api.freeagent.com/v2"
		}
	}

	c.App.Metadata = map[string]interface{}{
		"runtime": rt,
	}

	if !strings.HasSuffix(rt.BaseURL, "/v2") {
		return fmt.Errorf("base-url must include /v2 (got %s)", rt.BaseURL)
	}

	return nil
}

func runtimeFrom(c *cli.Context) (Runtime, error) {
	if c.App.Metadata == nil {
		return Runtime{}, errors.New("runtime not initialized")
	}
	if v, ok := c.App.Metadata["runtime"]; ok {
		if rt, ok := v.(Runtime); ok {
			return rt, nil
		}
	}
	return Runtime{}, errors.New("runtime missing")
}
