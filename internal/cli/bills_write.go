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

func billsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		billsCreateCmd(),
		billsUpdateCmd(),
		billsDeleteCmd(),
	}
}

func billsCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a bill",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body", Usage: "JSON file with full bill payload or bill object"},
			&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL (overrides body)"},
			&cli.StringFlag{Name: "reference", Usage: "Bill reference (overrides body)"},
			&cli.StringFlag{Name: "dated-on", Usage: "Bill date YYYY-MM-DD (overrides body)"},
			&cli.StringFlag{Name: "due-on", Usage: "Due date YYYY-MM-DD (overrides body)"},
			&cli.StringFlag{Name: "items", Usage: "JSON file with bill_items array (overrides body)"},
		},
		Action: billsCreate,
	}
}

func billsUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a bill",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Bill ID"},
			&cli.StringFlag{Name: "url", Usage: "Bill URL"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with bill payload or bill object", Required: true},
		},
		Action: billsUpdate,
	}
}

func billsDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a bill",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Bill ID"},
			&cli.StringFlag{Name: "url", Usage: "Bill URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: billsDelete,
	}
}

func billsCreate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	bill, err := loadResourceObject(c.String("body"), "bill")
	if err != nil {
		return err
	}

	if v := strings.TrimSpace(c.String("contact")); v != "" {
		resolved, err := resolveContactValue(c.Context, client, profile.BaseURL, v)
		if err != nil {
			return err
		}
		bill["contact"] = resolved
	}
	if v := strings.TrimSpace(c.String("reference")); v != "" {
		bill["reference"] = v
	}
	if v := strings.TrimSpace(c.String("dated-on")); v != "" {
		bill["dated_on"] = v
	}
	if v := strings.TrimSpace(c.String("due-on")); v != "" {
		bill["due_on"] = v
	}
	if itemsPath := c.String("items"); itemsPath != "" {
		items, err := loadItemsArray(itemsPath, "bill_items")
		if err != nil {
			return err
		}
		bill["bill_items"] = items
	}

	for _, field := range []string{"contact", "reference", "dated_on", "due_on"} {
		if _, ok := bill[field]; !ok {
			return fmt.Errorf("%s is required (set via flag or --body)", field)
		}
	}
	items, _ := bill["bill_items"].([]any)
	if len(items) == 0 {
		return fmt.Errorf("bill_items is required (set via --items or --body) and must contain at least one item")
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/bills", map[string]any{"bill": bill})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["bill"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Bill created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created bill %v (%v)\n", created["reference"], created["url"])
		return nil
	})
}

func billsUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "bills", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}

	bill, err := loadResourceObject(c.String("body"), "bill")
	if err != nil {
		return err
	}
	if len(bill) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"bill": bill})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated bill %s\n", path)
		return nil
	})
}

func billsDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "bills", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}

	if !c.Bool("yes") && !confirmDelete("bill", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}

	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted bill %s\n", path)
		return nil
	})
}
