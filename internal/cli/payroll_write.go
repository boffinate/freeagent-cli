//go:build !readonly

package cli

import "github.com/urfave/cli/v2"

func payrollWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "payment-transition",
			Usage: "Mark a payroll payment as paid/unpaid",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "year", Required: true, Usage: "Tax year"},
				&cli.StringFlag{Name: "payment-date", Required: true, Usage: "Payment date YYYY-MM-DD"},
				&cli.StringFlag{Name: "name", Required: true, Usage: "Transition: mark_as_paid or mark_as_unpaid"},
			},
			Action: func(c *cli.Context) error {
				base := "/payroll/" + c.String("year") + "/payments/" + c.String("payment-date")
				return applyTransition(c, base, c.String("name"))
			},
		},
	}
}
