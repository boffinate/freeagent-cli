package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func expensesCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "expenses",
		Usage: "Out-of-pocket expenses",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List expenses",
				Flags: append([]cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "User ID or URL"},
					&cli.StringFlag{Name: "category", Usage: "Category ID or URL"},
					&cli.StringFlag{Name: "project", Usage: "Project ID or URL"},
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "view", Usage: "API view filter"},
				}, paginationFlags()...),
				Action: expensesList,
			},
			{
				Name:  "get",
				Usage: "Get an expense by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Expense ID"},
					&cli.StringFlag{Name: "url", Usage: "Expense URL"},
				},
				Action: expensesGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, expensesWriteSubcommands()...)
	return cmd
}

func expensesList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	params := map[string]string{
		"from_date":     c.String("from"),
		"to_date":       c.String("to"),
		"updated_since": c.String("updated-since"),
		"view":          c.String("view"),
	}
	for flagName, key := range map[string]string{"user": "user", "category": "category", "project": "project"} {
		if raw := c.String(flagName); raw != "" {
			resource := map[string]string{"user": "users", "category": "categories", "project": "projects"}[flagName]
			resolved, err := normalizeResourceURL(profile.BaseURL, resource, raw)
			if err != nil {
				return err
			}
			params[key] = resolved
		}
	}

	resp, err := listAll(context.Background(), client, "/expenses", params, "expenses", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["expenses"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No expenses found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Date\tDescription\tGross\tCategory\tURL")
		for _, item := range list {
			exp, _ := item.(map[string]any)
			if exp == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v %v\t%v\t%v\n",
				exp["dated_on"], exp["description"], exp["currency"],
				exp["gross_value"], exp["category"], exp["url"])
		}
		return w.Flush()
	})
}

func expensesGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "expenses", c.String("id"), c.String("url"))
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
		exp, _ := decoded["expense"].(map[string]any)
		if exp == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Date:        %v\n", exp["dated_on"])
		fmt.Fprintf(os.Stdout, "Description: %v\n", exp["description"])
		fmt.Fprintf(os.Stdout, "Category:    %v\n", exp["category"])
		fmt.Fprintf(os.Stdout, "Gross:       %v %v\n", exp["currency"], exp["gross_value"])
		fmt.Fprintf(os.Stdout, "User:        %v\n", exp["user"])
		fmt.Fprintf(os.Stdout, "Project:     %v\n", exp["project"])
		fmt.Fprintf(os.Stdout, "URL:         %v\n", exp["url"])
		return nil
	})
}
