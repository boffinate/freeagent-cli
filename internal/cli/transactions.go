package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

// transactionsCommand exposes the FreeAgent accounting transactions resource
// (journal-style ledger entries from /v2/accounting/transactions). This is
// distinct from `bank transactions`, which lists raw bank-feed activity.
func transactionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "transactions",
		Usage: "Accounting transactions (ledger entries)",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List accounting transactions",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "from-date", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to-date", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "nominal-code", Usage: "Filter by nominal code"},
				},
				Action: transactionsList,
			},
			{
				Name:  "get",
				Usage: "Get an accounting transaction by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Transaction ID"},
					&cli.StringFlag{Name: "url", Usage: "Transaction URL"},
				},
				Action: transactionsGet,
			},
		},
	}
}

func transactionsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	params := map[string]string{
		"from_date":    c.String("from-date"),
		"to_date":      c.String("to-date"),
		"nominal_code": c.String("nominal-code"),
	}
	path := appendQuery("/accounting/transactions", buildQueryParams(params))

	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["transactions"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No transactions found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Dated On\tNominal\tCategory\tDebit\tDescription\tURL")
		for _, item := range list {
			txn, _ := item.(map[string]any)
			if txn == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
				txn["dated_on"], txn["nominal_code"], txn["category_name"],
				txn["debit_value"], txn["description"], txn["url"])
		}
		return w.Flush()
	})
}

func transactionsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "accounting/transactions", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		txn, _ := decoded["transaction"].(map[string]any)
		if txn == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "URL:           %v\n", txn["url"])
		fmt.Fprintf(os.Stdout, "Dated On:      %v\n", txn["dated_on"])
		fmt.Fprintf(os.Stdout, "Description:   %v\n", txn["description"])
		fmt.Fprintf(os.Stdout, "Category:      %v\n", txn["category_name"])
		fmt.Fprintf(os.Stdout, "Nominal Code:  %v\n", txn["nominal_code"])
		fmt.Fprintf(os.Stdout, "Debit Value:   %v\n", txn["debit_value"])
		if src, _ := txn["source_item_url"].(string); src != "" {
			fmt.Fprintf(os.Stdout, "Source Item:   %s\n", src)
		}
		if fcd, _ := txn["foreign_currency_data"].(map[string]any); len(fcd) > 0 {
			fmt.Fprintf(os.Stdout, "FX Currency:   %v\n", fcd["currency_code"])
			fmt.Fprintf(os.Stdout, "FX Debit:      %v\n", fcd["debit_value"])
		}
		return nil
	})
}
