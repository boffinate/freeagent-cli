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
				Usage: "Profit and loss summary",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "accounting-period", Usage: "Accounting period (YYYY/YY, e.g. 2022/23)"},
				},
				Action: reportsProfitAndLoss,
			},
			{
				Name:  "trial-balance",
				Usage: "Trial balance summary",
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
	return runReport(c, "/accounting/balance_sheet", params)
}

func reportsProfitAndLoss(c *cli.Context) error {
	// P&L summary defaults to the current accounting year to date when no
	// filter is supplied. Don't pre-filter.
	params := map[string]string{
		"from_date":         c.String("from"),
		"to_date":           c.String("to"),
		"accounting_period": c.String("accounting-period"),
	}
	return runReport(c, "/accounting/profit_and_loss/summary", params)
}

func reportsTrialBalance(c *cli.Context) error {
	params := map[string]string{
		"from_date": c.String("from"),
		"to_date":   c.String("to"),
	}
	return runReport(c, "/accounting/trial_balance/summary", params)
}

func reportsCashflow(c *cli.Context) error {
	from := c.String("from")
	to := c.String("to")
	if from == "" || to == "" {
		return fmt.Errorf("--from and --to are both required")
	}
	params := map[string]string{
		"from_date": from,
		"to_date":   to,
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
