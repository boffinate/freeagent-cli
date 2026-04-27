//go:build !readonly

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func creditNotesWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		creditNotesCreateCmd(),
		creditNotesUpdateCmd(),
		creditNotesDeleteCmd(),
		creditNotesSendCmd(),
		creditNotesTransitionCmd(),
	}
}

func creditNotesCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a draft credit note",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body", Usage: "JSON file with full credit_note payload or credit_note object"},
			&cli.StringFlag{Name: "contact", Usage: "Contact ID, URL, or name (overrides body)"},
			&cli.StringFlag{Name: "reference", Usage: "Reference (overrides body)"},
			&cli.StringFlag{Name: "dated-on", Usage: "Date YYYY-MM-DD (overrides body)"},
			&cli.StringFlag{Name: "currency", Usage: "Currency code (overrides body)"},
			&cli.StringFlag{Name: "items", Usage: "JSON file with credit_note_items array (overrides body)"},
		},
		Action: creditNotesCreate,
	}
}

func creditNotesUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a credit note",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Credit note ID"},
			&cli.StringFlag{Name: "url", Usage: "Credit note URL"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with credit_note payload or credit_note object", Required: true},
		},
		Action: creditNotesUpdate,
	}
}

func creditNotesDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a credit note",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Credit note ID"},
			&cli.StringFlag{Name: "url", Usage: "Credit note URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: creditNotesDelete,
	}
}

func creditNotesSendCmd() *cli.Command {
	return &cli.Command{
		Name:  "send",
		Usage: "Email a credit note",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Credit note ID"},
			&cli.StringFlag{Name: "url", Usage: "Credit note URL"},
			&cli.StringFlag{Name: "email-to", Usage: "Recipient email address"},
			&cli.StringFlag{Name: "cc"},
			&cli.StringFlag{Name: "bcc"},
			&cli.StringFlag{Name: "subject"},
			&cli.StringFlag{Name: "message", Usage: "Email body"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with full send payload"},
		},
		Action: creditNotesSend,
	}
}

func creditNotesTransitionCmd() *cli.Command {
	return &cli.Command{
		Name:  "transition",
		Usage: "Transition a credit note (mark_as_sent, mark_as_draft, mark_as_cancelled)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Credit note ID"},
			&cli.StringFlag{Name: "url", Usage: "Credit note URL"},
			&cli.StringFlag{Name: "name", Required: true, Usage: "Transition name"},
		},
		Action: creditNotesTransition,
	}
}

func creditNotesCreate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	creditNote, err := loadResourceObject(c.String("body"), "credit_note")
	if err != nil {
		return err
	}
	if v := strings.TrimSpace(c.String("contact")); v != "" {
		resolved, err := resolveContactValue(c.Context, client, profile.BaseURL, v)
		if err != nil {
			return err
		}
		creditNote["contact"] = resolved
	}
	if v := strings.TrimSpace(c.String("reference")); v != "" {
		creditNote["reference"] = v
	}
	if v := strings.TrimSpace(c.String("dated-on")); v != "" {
		creditNote["dated_on"] = v
	}
	if v := strings.TrimSpace(c.String("currency")); v != "" {
		creditNote["currency"] = v
	}
	if itemsPath := c.String("items"); itemsPath != "" {
		items, err := loadItemsArray(itemsPath, "credit_note_items")
		if err != nil {
			return err
		}
		creditNote["credit_note_items"] = items
	}
	if _, ok := creditNote["contact"]; !ok {
		return fmt.Errorf("contact is required (set via flag or --body)")
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/credit_notes", map[string]any{"credit_note": creditNote})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["credit_note"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Credit note created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created credit note %v (%v)\n", created["reference"], created["url"])
		return nil
	})
}

func creditNotesUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "credit_notes", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	creditNote, err := loadResourceObject(c.String("body"), "credit_note")
	if err != nil {
		return err
	}
	if len(creditNote) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"credit_note": creditNote})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated credit note %s\n", path)
		return nil
	})
}

func creditNotesDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "credit_notes", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	if !c.Bool("yes") {
		fmt.Fprintf(os.Stdout, "Delete credit note %s? (y/N): ", path)
		var answer string
		_, _ = fmt.Fscanln(os.Stdin, &answer)
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stdout, "Cancelled")
			return nil
		}
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted credit note %s\n", path)
		return nil
	})
}

func creditNotesSend(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "credit_notes", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	payload, err := buildSendEmailPayload(c, "credit_note")
	if err != nil {
		return err
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, path+"/send_email", payload)
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Sent credit note %s\n", path)
		return nil
	})
}

func creditNotesTransition(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "credit_notes", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	transition := strings.TrimSpace(c.String("name"))
	resp, _, _, err := client.Do(context.Background(), http.MethodPut, path+"/transitions/"+transition, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Applied transition %s on %s\n", transition, path)
		return nil
	})
}
