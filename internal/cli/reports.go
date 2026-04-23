package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func reportsCommand() *cli.Command {
	return &cli.Command{
		Name:  "reports",
		Usage: "Accounting reports",
		Subcommands: []*cli.Command{
			{
				Name:  "balance-sheet",
				Usage: "Balance sheet snapshot",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "as-at", Usage: "Snapshot date (YYYY-MM-DD)"},
				},
				Action: reportsBalanceSheet,
			},
			{
				Name:  "profit-and-loss",
				Usage: "Profit and loss report",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "accounting-year", Usage: "Accounting year (YYYY) — alternative to --from/--to"},
					&cli.StringFlag{Name: "accounting-period", Usage: "Accounting period name — used with --accounting-year"},
				},
				Action: reportsProfitAndLoss,
			},
			{
				Name:  "trial-balance",
				Usage: "Trial balance",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
				},
				Action: reportsTrialBalance,
			},
			{
				Name:  "cashflow",
				Usage: "Cashflow report",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
				},
				Action: reportsCashflow,
			},
		},
	}
}

func reportsBalanceSheet(c *cli.Context) error {
	params := map[string]string{"as_at_date": c.String("as-at")}
	return runReport(c, "/balance_sheet", params)
}

func reportsProfitAndLoss(c *cli.Context) error {
	params := map[string]string{
		"from_date":         c.String("from"),
		"to_date":           c.String("to"),
		"accounting_year":   c.String("accounting-year"),
		"accounting_period": c.String("accounting-period"),
	}
	if params["from_date"] == "" && params["to_date"] == "" && params["accounting_year"] == "" {
		return fmt.Errorf("provide --from/--to or --accounting-year")
	}
	return runReport(c, "/profit_and_loss", params)
}

func reportsTrialBalance(c *cli.Context) error {
	params := map[string]string{
		"from_date": c.String("from"),
		"to_date":   c.String("to"),
	}
	return runReport(c, "/trial_balance", params)
}

func reportsCashflow(c *cli.Context) error {
	params := map[string]string{
		"from_date": c.String("from"),
		"to_date":   c.String("to"),
	}
	return runReport(c, "/cashflow", params)
}

// runReport issues the GET and always prints raw JSON. Reports have deeply
// nested shapes that vary by endpoint; rendering a bespoke table per report
// would cost more than it's worth for the read-only scope.
func runReport(c *cli.Context, basePath string, params map[string]string) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path := appendQuery(basePath, buildQueryParams(params))
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	if rt.JSONOutput {
		return writeJSONOutput(resp)
	}
	// Non-JSON fallback: pretty-print the raw JSON so the shape is readable
	// without forcing --json on every invocation.
	_, _ = os.Stdout.Write(append(resp, '\n'))
	return nil
}
