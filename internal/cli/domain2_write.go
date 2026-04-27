//go:build !readonly

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func attachmentsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "delete",
			Usage: "Delete an attachment",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "Attachment ID"},
				&cli.StringFlag{Name: "url", Usage: "Attachment URL"},
				&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
			},
			Action: attachmentsDelete,
		},
	}
}

func notesWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "create",
			Usage: "Create a note attached to a contact or project",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
				&cli.StringFlag{Name: "project", Usage: "Project ID or URL"},
				&cli.StringFlag{Name: "note", Usage: "Note text (overrides body)"},
				&cli.StringFlag{Name: "body", Usage: "JSON file with note payload or object"},
			},
			Action: notesCreate,
		},
		{
			Name:  "update",
			Usage: "Update a note",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "Note ID"},
				&cli.StringFlag{Name: "url", Usage: "Note URL"},
				&cli.StringFlag{Name: "note", Usage: "Note text (overrides body)"},
				&cli.StringFlag{Name: "body", Usage: "JSON file with note payload or object"},
			},
			Action: notesUpdate,
		},
		{
			Name:  "delete",
			Usage: "Delete a note",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "Note ID"},
				&cli.StringFlag{Name: "url", Usage: "Note URL"},
				&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
			},
			Action: notesDelete,
		},
	}
}

func journalSetsWriteSubcommands() []*cli.Command {
	return crudWriteSubcommands(crudSpec{
		resource: "journal_sets",
		wrapper:  "journal_set",
		label:    "journal set",
	})
}

func accountLocksWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "set",
			Usage: "Create or update the user account lock",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "body", Required: true, Usage: "JSON file with account_lock payload or object"},
			},
			Action: accountLocksSet,
		},
		{
			Name:  "delete",
			Usage: "Delete the user account lock",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
			},
			Action: accountLocksDelete,
		},
	}
}

func finalAccountsReportsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "transition",
			Usage: "Apply a transition (mark_as_filed, mark_as_unfiled)",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "period-ends-on", Usage: "Period end date YYYY-MM-DD"},
				&cli.StringFlag{Name: "url", Usage: "Final Accounts report URL"},
				&cli.StringFlag{Name: "name", Required: true, Usage: "Transition name"},
			},
			Action: finalAccountsReportsTransition,
		},
	}
}

func attachmentsDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "attachments", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	if !c.Bool("yes") && !confirmDelete("attachment", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted attachment %s\n", path)
		return nil
	})
}

func notesCreate(c *cli.Context) error {
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
	note, err := loadResourceObject(c.String("body"), "note")
	if err != nil {
		return err
	}
	if v := strings.TrimSpace(c.String("note")); v != "" {
		note["note"] = v
	}
	if _, ok := note["note"]; !ok {
		return fmt.Errorf("note text is required (set via --note or --body)")
	}
	path := appendQuery("/notes", q.Encode())
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, path, map[string]any{"note": note})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["note"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Note created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created note %v\n", created["url"])
		return nil
	})
}

func notesUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "notes", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	note, err := loadResourceObject(c.String("body"), "note")
	if err != nil {
		return err
	}
	if v := strings.TrimSpace(c.String("note")); v != "" {
		note["note"] = v
	}
	if len(note) == 0 {
		return fmt.Errorf("--note or --body must contain at least one field")
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"note": note})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated note %s\n", path)
		return nil
	})
}

func notesDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "notes", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	if !c.Bool("yes") && !confirmDelete("note", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted note %s\n", path)
		return nil
	})
}

func accountLocksSet(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	lock, err := loadResourceObject(c.String("body"), "account_lock")
	if err != nil {
		return err
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, "/account_locks", map[string]any{"account_lock": lock})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintln(os.Stdout, "Account lock applied")
		return nil
	})
}

func accountLocksDelete(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	if !c.Bool("yes") && !confirmDelete("account lock", "/v2/account_locks") {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, "/account_locks", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintln(os.Stdout, "Account lock deleted")
		return nil
	})
}

func finalAccountsReportsTransition(c *cli.Context) error {
	_, _, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	base, err := taxReturnPath(profile.BaseURL, "final_accounts_reports", c.String("period-ends-on"), c.String("url"))
	if err != nil {
		return err
	}
	return applyTransition(c, base, c.String("name"))
}

