package cli

import (
	"context"
	"fmt"
	"net/url"

	"github.com/urfave/cli/v2"
)

// Domain 2 read surface: attachments, notes, email_addresses, journal_sets,
// account_locks, final_accounts_reports, recurring_invoices.
// Writes (where the API exposes them) live in domain2_write.go.

func attachmentsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "attachments",
		Usage: "Attachments",
		Subcommands: []*cli.Command{
			{
				Name:  "get",
				Usage: "Get an attachment by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Attachment ID"},
					&cli.StringFlag{Name: "url", Usage: "Attachment URL"},
				},
				Action: attachmentsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, attachmentsWriteSubcommands()...)
	return cmd
}

func notesCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "notes",
		Usage: "Contact / project notes",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List notes filtered by contact or project",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
					&cli.StringFlag{Name: "project", Usage: "Project ID or URL"},
				},
				Action: notesList,
			},
			{
				Name:  "get",
				Usage: "Get a note by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Note ID"},
					&cli.StringFlag{Name: "url", Usage: "Note URL"},
				},
				Action: notesGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, notesWriteSubcommands()...)
	return cmd
}

func emailAddressesCommand() *cli.Command {
	return &cli.Command{
		Name:  "email-addresses",
		Usage: "Verified sender email addresses",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List verified sender email addresses", Action: emailAddressesList},
		},
	}
}

func journalSetsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "journal-sets",
		Usage: "Journal sets",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List journal sets",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "from", Usage: "Start date YYYY-MM-DD"},
					&cli.StringFlag{Name: "to", Usage: "End date YYYY-MM-DD"},
					&cli.StringFlag{Name: "tag", Usage: "Filter by tag"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since timestamp"},
				},
				Action: journalSetsList,
			},
			{
				Name:  "get",
				Usage: "Get a journal set by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Journal set ID"},
					&cli.StringFlag{Name: "url", Usage: "Journal set URL"},
				},
				Action: journalSetsGet,
			},
			{
				Name:   "opening-balances",
				Usage:  "Get opening balances",
				Action: journalSetsOpeningBalances,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, journalSetsWriteSubcommands()...)
	return cmd
}

func accountLocksCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "account-locks",
		Usage: "Account locks",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List account locks", Action: accountLocksList},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, accountLocksWriteSubcommands()...)
	return cmd
}

func finalAccountsReportsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "final-accounts-reports",
		Usage: "Final Accounts reports",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List Final Accounts reports", Action: finalAccountsReportsList},
			{
				Name:  "get",
				Usage: "Get a Final Accounts report by period_ends_on or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "period-ends-on", Usage: "Period end date YYYY-MM-DD"},
					&cli.StringFlag{Name: "url", Usage: "Final Accounts report URL"},
				},
				Action: finalAccountsReportsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, finalAccountsReportsWriteSubcommands()...)
	return cmd
}

func recurringInvoicesCommand() *cli.Command {
	return &cli.Command{
		Name:  "recurring-invoices",
		Usage: "Recurring invoices",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List recurring invoices",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
					&cli.StringFlag{Name: "view", Usage: "API view filter (draft, recent, ...)"},
				},
				Action: recurringInvoicesList,
			},
			{
				Name:  "get",
				Usage: "Get a recurring invoice by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Recurring invoice ID"},
					&cli.StringFlag{Name: "url", Usage: "Recurring invoice URL"},
				},
				Action: recurringInvoicesGet,
			},
		},
	}
}

func attachmentsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "attachments", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func notesList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	q := url.Values{}
	if v := c.String("contact"); v != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "contacts", v)
		if err != nil {
			return err
		}
		q.Set("contact", resolved)
	}
	if v := c.String("project"); v != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "projects", v)
		if err != nil {
			return err
		}
		q.Set("project", resolved)
	}
	if q.Get("contact") == "" && q.Get("project") == "" {
		return fmt.Errorf("--contact or --project required")
	}
	path := appendQuery("/notes", q.Encode())
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return renderListTable(resp, "notes", []string{"created_at", "note", "url"})
	})
}

func notesGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "notes", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func emailAddressesList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/email_addresses", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return renderListTable(resp, "email_addresses", []string{"email", "status", "url"})
	})
}

func journalSetsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	params := map[string]string{
		"from_date":     c.String("from"),
		"to_date":       c.String("to"),
		"tag":           c.String("tag"),
		"updated_since": c.String("updated-since"),
	}
	path := appendQuery("/journal_sets", buildQueryParams(params))
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func journalSetsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "journal_sets", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func journalSetsOpeningBalances(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/journal_sets/opening_balances", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func accountLocksList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/account_locks", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func finalAccountsReportsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/final_accounts_reports", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return renderListTable(resp, "final_accounts_reports", []string{"period_ends_on", "status", "url"})
	})
}

func finalAccountsReportsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := taxReturnPath(profile.BaseURL, "final_accounts_reports", c.String("period-ends-on"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func recurringInvoicesList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	q := url.Values{}
	if v := c.String("view"); v != "" {
		q.Set("view", v)
	}
	if v := c.String("contact"); v != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "contacts", v)
		if err != nil {
			return err
		}
		q.Set("contact", resolved)
	}
	path := appendQuery("/recurring_invoices", q.Encode())
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return renderListTable(resp, "recurring_invoices", []string{"reference", "status", "next_runs_on", "url"})
	})
}

func recurringInvoicesGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "recurring_invoices", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}
