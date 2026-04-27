package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func creditNotesCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "credit-notes",
		Usage: "Credit notes",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List credit notes",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
					&cli.StringFlag{Name: "view", Usage: "API view filter"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
				},
				Action: creditNotesList,
			},
			{
				Name:  "get",
				Usage: "Get a credit note by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Credit note ID"},
					&cli.StringFlag{Name: "url", Usage: "Credit note URL"},
				},
				Action: creditNotesGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, creditNotesWriteSubcommands()...)
	return cmd
}

func creditNotesList(c *cli.Context) error {
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

	path := appendQuery("/credit_notes", buildQueryParams(params))

	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["credit_notes"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No credit notes found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Reference\tStatus\tDate\tTotal\tURL")
		for _, item := range list {
			cn, _ := item.(map[string]any)
			if cn == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v %v\t%v\n",
				cn["reference"], cn["status"], cn["dated_on"],
				cn["currency"], cn["total_value"], cn["url"])
		}
		return w.Flush()
	})
}

func creditNotesGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "credit_notes", c.String("id"), c.String("url"))
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
		cn, _ := decoded["credit_note"].(map[string]any)
		if cn == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Reference: %v\n", cn["reference"])
		fmt.Fprintf(os.Stdout, "Contact:   %v\n", cn["contact"])
		fmt.Fprintf(os.Stdout, "Status:    %v\n", cn["status"])
		fmt.Fprintf(os.Stdout, "Dated On:  %v\n", cn["dated_on"])
		fmt.Fprintf(os.Stdout, "Total:     %v %v\n", cn["currency"], cn["total_value"])
		fmt.Fprintf(os.Stdout, "URL:       %v\n", cn["url"])
		return nil
	})
}
