package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/anjor/freeagent-cli/internal/config"
	"github.com/anjor/freeagent-cli/internal/freeagent"

	"github.com/urfave/cli/v2"
)

func bankCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "bank",
		Usage: "Work with bank accounts and transactions",
		Subcommands: []*cli.Command{
			bankAccountsCmd(),
			bankTransactionsCmd(),
			bankExplanationsCmd(),
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, bankWriteSubcommands()...)
	return cmd
}

func bankAccountsCmd() *cli.Command {
	return &cli.Command{
		Name:  "accounts",
		Usage: "Bank accounts",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List bank accounts",
				Flags:  []cli.Flag{&cli.StringFlag{Name: "view", Usage: "API view filter"}},
				Action: bankAccountsList,
			},
			{
				Name:  "get",
				Usage: "Get a bank account by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Bank account ID"},
					&cli.StringFlag{Name: "url", Usage: "Bank account URL"},
				},
				Action: bankAccountsGet,
			},
		},
	}
}

func bankTransactionsCmd() *cli.Command {
	return &cli.Command{
		Name:  "transactions",
		Usage: "Bank transactions",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List bank transactions",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "bank-account", Usage: "Bank account ID or URL (required)"},
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "view", Usage: "API view filter (for example: unexplained)"},
				},
				Action: bankTransactionsList,
			},
		},
	}
}

func bankExplanationsCmd() *cli.Command {
	return &cli.Command{
		Name:  "explanations",
		Usage: "Bank transaction explanations",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List bank transaction explanations",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "bank-account", Usage: "Bank account ID or URL (required)"},
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
				},
				Action: bankExplanationsList,
			},
			{
				Name:  "get",
				Usage: "Get a bank transaction explanation by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Explanation ID"},
					&cli.StringFlag{Name: "url", Usage: "Explanation URL"},
				},
				Action: bankExplanationsGet,
			},
		},
	}
}

func bankAccountsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	query := buildQueryParams(map[string]string{"view": c.String("view")})
	path := appendQuery("/bank_accounts", query)

	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}

	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["bank_accounts"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No bank accounts found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Name\tType\tCurrency\tBalance\tURL")
		for _, item := range list {
			acct, _ := item.(map[string]any)
			if acct == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n",
				acct["name"], acct["type"], acct["currency"],
				acct["current_balance"], acct["url"])
		}
		return w.Flush()
	})
}

func bankAccountsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	path, err := requireIDOrURL(profile.BaseURL, "bank_accounts", c.String("id"), c.String("url"))
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
		acct, _ := decoded["bank_account"].(map[string]any)
		if acct == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Name:     %v\n", acct["name"])
		fmt.Fprintf(os.Stdout, "Type:     %v\n", acct["type"])
		fmt.Fprintf(os.Stdout, "Currency: %v\n", acct["currency"])
		fmt.Fprintf(os.Stdout, "Balance:  %v\n", acct["current_balance"])
		fmt.Fprintf(os.Stdout, "URL:      %v\n", acct["url"])
		return nil
	})
}

func bankTransactionsList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	acct := c.String("bank-account")
	if acct == "" {
		return fmt.Errorf("bank-account is required")
	}
	acctURL, err := normalizeResourceURL(profile.BaseURL, "bank_accounts", acct)
	if err != nil {
		return err
	}

	query := buildQueryParams(map[string]string{
		"bank_account":  acctURL,
		"from_date":     c.String("from"),
		"to_date":       c.String("to"),
		"updated_since": c.String("updated-since"),
		"view":          c.String("view"),
	})
	path := appendQuery("/bank_transactions", query)

	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["bank_transactions"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No transactions found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Date\tAmount\tDescription\tURL")
		for _, item := range list {
			txn, _ := item.(map[string]any)
			if txn == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n",
				txn["dated_on"], txn["amount"], txn["description"], txn["url"])
		}
		return w.Flush()
	})
}

func bankExplanationsList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	acct := c.String("bank-account")
	if acct == "" {
		return fmt.Errorf("bank-account is required")
	}
	acctURL, err := normalizeResourceURL(profile.BaseURL, "bank_accounts", acct)
	if err != nil {
		return err
	}

	params := map[string]string{
		"bank_account":  acctURL,
		"from_date":     c.String("from"),
		"to_date":       c.String("to"),
		"updated_since": c.String("updated-since"),
	}

	path := appendQuery("/bank_transaction_explanations", buildQueryParams(params))

	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["bank_transaction_explanations"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No explanations found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Date\tGross\tCategory\tDescription\tURL")
		for _, item := range list {
			exp, _ := item.(map[string]any)
			if exp == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n",
				exp["dated_on"], exp["gross_value"], exp["category"],
				exp["description"], exp["url"])
		}
		return w.Flush()
	})
}

func bankExplanationsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	path, err := requireIDOrURL(profile.BaseURL, "bank_transaction_explanations", c.String("id"), c.String("url"))
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
		exp, _ := decoded["bank_transaction_explanation"].(map[string]any)
		if exp == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Date:        %v\n", exp["dated_on"])
		fmt.Fprintf(os.Stdout, "Gross:       %v\n", exp["gross_value"])
		fmt.Fprintf(os.Stdout, "Category:    %v\n", exp["category"])
		fmt.Fprintf(os.Stdout, "Description: %v\n", exp["description"])
		fmt.Fprintf(os.Stdout, "URL:         %v\n", exp["url"])
		return nil
	})
}

// bootstrapClient reduces the loadConfig/ensureProfile/newClient boilerplate
// that every command opens with.
func bootstrapClient(c *cli.Context) (Runtime, *freeagent.Client, config.Profile, error) {
	rt, err := runtimeFrom(c)
	if err != nil {
		return Runtime{}, nil, config.Profile{}, err
	}
	cfg, _, err := loadConfig(rt)
	if err != nil {
		return Runtime{}, nil, config.Profile{}, err
	}
	profile := ensureProfile(cfg, rt.Profile, rt, config.Profile{})
	client, _, err := newClient(context.Background(), rt, profile)
	if err != nil {
		return Runtime{}, nil, config.Profile{}, err
	}
	return rt, client, profile, nil
}
