package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func estimatesCommand() *cli.Command {
	return &cli.Command{
		Name:  "estimates",
		Usage: "Estimates",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List estimates",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
					&cli.StringFlag{Name: "view", Usage: "API view filter"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
				},
				Action: estimatesList,
			},
			{
				Name:  "get",
				Usage: "Get an estimate by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Estimate ID"},
					&cli.StringFlag{Name: "url", Usage: "Estimate URL"},
				},
				Action: estimatesGet,
			},
		},
	}
}

func estimatesList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	params := map[string]string{
		"view":          c.String("view"),
		"updated_since": c.String("updated-since"),
	}
	if contact := c.String("contact"); contact != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "contacts", contact)
		if err != nil {
			return err
		}
		params["contact"] = resolved
	}
	path := appendQuery("/estimates", buildQueryParams(params))

	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["estimates"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No estimates found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Reference\tStatus\tDate\tTotal\tURL")
		for _, item := range list {
			e, _ := item.(map[string]any)
			if e == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v %v\t%v\n",
				e["reference"], e["status"], e["dated_on"],
				e["currency"], e["total_value"], e["url"])
		}
		return w.Flush()
	})
}

func estimatesGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "estimates", c.String("id"), c.String("url"))
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
		e, _ := decoded["estimate"].(map[string]any)
		if e == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Reference: %v\n", e["reference"])
		fmt.Fprintf(os.Stdout, "Contact:   %v\n", e["contact"])
		fmt.Fprintf(os.Stdout, "Status:    %v\n", e["status"])
		fmt.Fprintf(os.Stdout, "Dated On:  %v\n", e["dated_on"])
		fmt.Fprintf(os.Stdout, "Total:     %v %v\n", e["currency"], e["total_value"])
		fmt.Fprintf(os.Stdout, "URL:       %v\n", e["url"])
		return nil
	})
}
