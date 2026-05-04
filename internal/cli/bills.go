package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func billsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "bills",
		Usage: "Bills (purchases)",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List bills",
				Flags: append([]cli.Flag{
					&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
					&cli.StringFlag{Name: "view", Usage: "API view filter (open, recent, ...)"},
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
				}, paginationFlags()...),
				Action: billsList,
			},
			{
				Name:  "get",
				Usage: "Get a bill by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Bill ID"},
					&cli.StringFlag{Name: "url", Usage: "Bill URL"},
				},
				Action: billsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, billsWriteSubcommands()...)
	return cmd
}

func billsList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	params := map[string]string{
		"view":          c.String("view"),
		"from_date":     c.String("from"),
		"to_date":       c.String("to"),
		"updated_since": c.String("updated-since"),
	}
	if contact := c.String("contact"); contact != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "contacts", contact)
		if err != nil {
			return err
		}
		params["contact"] = resolved
	}

	resp, err := listAll(context.Background(), client, "/bills", params, "bills", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["bills"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No bills found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Reference\tDue\tStatus\tAmount\tURL")
		for _, item := range list {
			bill, _ := item.(map[string]any)
			if bill == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v %v\t%v\n",
				bill["reference"], bill["due_on"], bill["status"],
				bill["currency"], bill["total_value"], bill["url"])
		}
		return w.Flush()
	})
}

func billsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "bills", c.String("id"), c.String("url"))
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
		bill, _ := decoded["bill"].(map[string]any)
		if bill == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Reference: %v\n", bill["reference"])
		fmt.Fprintf(os.Stdout, "Contact:   %v\n", bill["contact"])
		fmt.Fprintf(os.Stdout, "Dated On:  %v\n", bill["dated_on"])
		fmt.Fprintf(os.Stdout, "Due On:    %v\n", bill["due_on"])
		fmt.Fprintf(os.Stdout, "Status:    %v\n", bill["status"])
		fmt.Fprintf(os.Stdout, "Total:     %v %v\n", bill["currency"], bill["total_value"])
		fmt.Fprintf(os.Stdout, "URL:       %v\n", bill["url"])
		return nil
	})
}
