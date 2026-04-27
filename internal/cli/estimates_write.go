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

func estimatesWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		estimatesCreateCmd(),
		estimatesUpdateCmd(),
		estimatesDeleteCmd(),
		estimatesSendCmd(),
		estimatesTransitionCmd(),
		estimatesDuplicateCmd(),
	}
}

func estimatesCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a draft estimate",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body", Usage: "JSON file with full estimate payload or estimate object"},
			&cli.StringFlag{Name: "contact", Usage: "Contact ID, URL, or name (overrides body)"},
			&cli.StringFlag{Name: "reference", Usage: "Reference (overrides body)"},
			&cli.StringFlag{Name: "dated-on", Usage: "Date YYYY-MM-DD (overrides body)"},
			&cli.StringFlag{Name: "currency", Usage: "Currency code (overrides body)"},
			&cli.StringFlag{Name: "items", Usage: "JSON file with estimate_items array (overrides body)"},
		},
		Action: estimatesCreate,
	}
}

func estimatesUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update an estimate",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Estimate ID"},
			&cli.StringFlag{Name: "url", Usage: "Estimate URL"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with estimate payload or estimate object", Required: true},
		},
		Action: estimatesUpdate,
	}
}

func estimatesDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an estimate",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Estimate ID"},
			&cli.StringFlag{Name: "url", Usage: "Estimate URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: estimatesDelete,
	}
}

func estimatesSendCmd() *cli.Command {
	return &cli.Command{
		Name:  "send",
		Usage: "Email an estimate",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Estimate ID"},
			&cli.StringFlag{Name: "url", Usage: "Estimate URL"},
			&cli.StringFlag{Name: "email-to", Usage: "Recipient email address"},
			&cli.StringFlag{Name: "cc"},
			&cli.StringFlag{Name: "bcc"},
			&cli.StringFlag{Name: "subject"},
			&cli.StringFlag{Name: "message", Usage: "Email body"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with full send payload"},
		},
		Action: estimatesSend,
	}
}

func estimatesTransitionCmd() *cli.Command {
	return &cli.Command{
		Name:  "transition",
		Usage: "Transition an estimate (mark_as_sent, mark_as_draft, mark_as_approved, mark_as_rejected)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Estimate ID"},
			&cli.StringFlag{Name: "url", Usage: "Estimate URL"},
			&cli.StringFlag{Name: "name", Required: true, Usage: "Transition name"},
		},
		Action: estimatesTransition,
	}
}

func estimatesDuplicateCmd() *cli.Command {
	return &cli.Command{
		Name:  "duplicate",
		Usage: "Duplicate an estimate (returns a new draft)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Estimate ID"},
			&cli.StringFlag{Name: "url", Usage: "Estimate URL"},
		},
		Action: estimatesDuplicate,
	}
}

func estimatesCreate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	estimate, err := loadResourceObject(c.String("body"), "estimate")
	if err != nil {
		return err
	}
	if v := strings.TrimSpace(c.String("contact")); v != "" {
		resolved, err := resolveContactValue(c.Context, client, profile.BaseURL, v)
		if err != nil {
			return err
		}
		estimate["contact"] = resolved
	}
	if v := strings.TrimSpace(c.String("reference")); v != "" {
		estimate["reference"] = v
	}
	if v := strings.TrimSpace(c.String("dated-on")); v != "" {
		estimate["dated_on"] = v
	}
	if v := strings.TrimSpace(c.String("currency")); v != "" {
		estimate["currency"] = v
	}
	if itemsPath := c.String("items"); itemsPath != "" {
		items, err := loadItemsArray(itemsPath, "estimate_items")
		if err != nil {
			return err
		}
		estimate["estimate_items"] = items
	}
	if _, ok := estimate["contact"]; !ok {
		return fmt.Errorf("contact is required (set via flag or --body)")
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/estimates", map[string]any{"estimate": estimate})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["estimate"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Estimate created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created estimate %v (%v)\n", created["reference"], created["url"])
		return nil
	})
}

func estimatesUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "estimates", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	estimate, err := loadResourceObject(c.String("body"), "estimate")
	if err != nil {
		return err
	}
	if len(estimate) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"estimate": estimate})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated estimate %s\n", path)
		return nil
	})
}

func estimatesDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "estimates", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	if !c.Bool("yes") {
		fmt.Fprintf(os.Stdout, "Delete estimate %s? (y/N): ", path)
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
		fmt.Fprintf(os.Stdout, "Deleted estimate %s\n", path)
		return nil
	})
}

func estimatesSend(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "estimates", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	payload, err := buildSendEmailPayload(c, "estimate")
	if err != nil {
		return err
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, path+"/send_email", payload)
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Sent estimate %s\n", path)
		return nil
	})
}

func estimatesTransition(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "estimates", c.String("id"), c.String("url"))
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

func estimatesDuplicate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "estimates", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodPost, path+"/duplicate", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["estimate"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Duplicated estimate")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Duplicated to %v\n", created["url"])
		return nil
	})
}
