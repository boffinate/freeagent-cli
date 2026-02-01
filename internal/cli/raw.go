package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func rawCommand() *cli.Command {
	return &cli.Command{
		Name:  "raw",
		Usage: "Break-glass raw API call",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "method", Value: "GET"},
			&cli.StringFlag{Name: "path", Usage: "Path like /v2/invoices"},
			&cli.StringFlag{Name: "body", Usage: "JSON file to send"},
		},
		Action: func(c *cli.Context) error {
			return fmt.Errorf("raw request not implemented yet")
		},
	}
}
