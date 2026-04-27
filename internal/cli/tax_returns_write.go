//go:build !readonly

package cli

import (
	"github.com/urfave/cli/v2"
)

func vatReturnsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "transition",
			Usage: "Apply a transition (mark_as_filed, mark_as_unfiled)",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "period-ends-on", Usage: "Period end date YYYY-MM-DD"},
				&cli.StringFlag{Name: "url", Usage: "VAT return URL"},
				&cli.StringFlag{Name: "name", Required: true, Usage: "Transition name"},
			},
			Action: func(c *cli.Context) error {
				_, _, profile, err := bootstrapClient(c)
				if err != nil {
					return err
				}
				base, err := taxReturnPath(profile.BaseURL, "vat_returns", c.String("period-ends-on"), c.String("url"))
				if err != nil {
					return err
				}
				return applyTransition(c, base, c.String("name"))
			},
		},
		{
			Name:  "payment-transition",
			Usage: "Mark a VAT return payment as paid/unpaid",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "period-ends-on", Required: true, Usage: "Period end date YYYY-MM-DD"},
				&cli.StringFlag{Name: "payment-date", Required: true, Usage: "Payment date YYYY-MM-DD"},
				&cli.StringFlag{Name: "name", Required: true, Usage: "Transition: mark_as_paid or mark_as_unpaid"},
			},
			Action: func(c *cli.Context) error {
				_, _, profile, err := bootstrapClient(c)
				if err != nil {
					return err
				}
				period, err := normalizeResourceURL(profile.BaseURL, "vat_returns", c.String("period-ends-on"))
				if err != nil {
					return err
				}
				return applyTransition(c, period+"/payments/"+c.String("payment-date"), c.String("name"))
			},
		},
	}
}

func corporationTaxReturnsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "transition",
			Usage: "Apply a transition (mark_as_filed, mark_as_unfiled, mark_as_paid, mark_as_unpaid)",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "period-ends-on", Usage: "Period end date YYYY-MM-DD"},
				&cli.StringFlag{Name: "url", Usage: "Corporation Tax return URL"},
				&cli.StringFlag{Name: "name", Required: true, Usage: "Transition name"},
			},
			Action: func(c *cli.Context) error {
				_, _, profile, err := bootstrapClient(c)
				if err != nil {
					return err
				}
				base, err := taxReturnPath(profile.BaseURL, "corporation_tax_returns", c.String("period-ends-on"), c.String("url"))
				if err != nil {
					return err
				}
				return applyTransition(c, base, c.String("name"))
			},
		},
	}
}

func selfAssessmentReturnsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "transition",
			Usage: "Apply a transition (mark_as_filed, mark_as_unfiled)",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "user", Required: true, Usage: "User ID or URL"},
				&cli.StringFlag{Name: "period-ends-on", Required: true, Usage: "Period end date YYYY-MM-DD"},
				&cli.StringFlag{Name: "name", Required: true, Usage: "Transition name"},
			},
			Action: func(c *cli.Context) error {
				_, _, profile, err := bootstrapClient(c)
				if err != nil {
					return err
				}
				base, err := selfAssessmentBasePath(profile.BaseURL, c.String("user"))
				if err != nil {
					return err
				}
				return applyTransition(c, base+"/"+c.String("period-ends-on"), c.String("name"))
			},
		},
		{
			Name:  "payment-transition",
			Usage: "Mark a Self Assessment payment as paid/unpaid",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "user", Required: true, Usage: "User ID or URL"},
				&cli.StringFlag{Name: "period-ends-on", Required: true, Usage: "Period end date YYYY-MM-DD"},
				&cli.StringFlag{Name: "payment-date", Required: true, Usage: "Payment date YYYY-MM-DD"},
				&cli.StringFlag{Name: "name", Required: true, Usage: "Transition: mark_as_paid or mark_as_unpaid"},
			},
			Action: func(c *cli.Context) error {
				_, _, profile, err := bootstrapClient(c)
				if err != nil {
					return err
				}
				base, err := selfAssessmentBasePath(profile.BaseURL, c.String("user"))
				if err != nil {
					return err
				}
				return applyTransition(c, base+"/"+c.String("period-ends-on")+"/payments/"+c.String("payment-date"), c.String("name"))
			},
		},
	}
}

func salesTaxPeriodsWriteSubcommands() []*cli.Command {
	return crudWriteSubcommands(crudSpec{
		resource: "sales_tax_periods",
		wrapper:  "sales_tax_period",
		label:    "sales tax period",
	})
}
