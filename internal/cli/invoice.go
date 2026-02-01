package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func invoiceCommand() *cli.Command {
	return &cli.Command{
		Name:  "invoices",
		Usage: "Create and send invoices",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a draft invoice",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "contact"},
					&cli.StringFlag{Name: "reference"},
					&cli.StringFlag{Name: "currency", Value: "GBP"},
					&cli.StringFlag{Name: "date"},
					&cli.StringFlag{Name: "due"},
					&cli.StringFlag{Name: "lines", Usage: "JSON file with line items"},
				},
				Action: func(c *cli.Context) error {
					return fmt.Errorf("invoice create not implemented yet")
				},
			},
			{
				Name:  "send",
				Usage: "Send an existing draft invoice",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Invoice ID"},
				},
				Action: func(c *cli.Context) error {
					return fmt.Errorf("invoice send not implemented yet")
				},
			},
		},
	}
}
