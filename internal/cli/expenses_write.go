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

func expensesWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		expensesCreateCmd(),
		expensesUpdateCmd(),
		expensesDeleteCmd(),
	}
}

func expensesCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create an expense",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body", Usage: "JSON file with full expense payload or expense object"},
			&cli.StringFlag{Name: "user", Usage: "User ID or URL (overrides body)"},
			&cli.StringFlag{Name: "category", Usage: "Category ID, URL, or 'Mileage' (overrides body)"},
			&cli.StringFlag{Name: "dated-on", Usage: "Expense date YYYY-MM-DD (overrides body)"},
			&cli.StringFlag{Name: "gross-value", Usage: "Gross value (overrides body)"},
			&cli.StringFlag{Name: "currency", Usage: "Currency code (overrides body)"},
			&cli.StringFlag{Name: "description", Usage: "Description (overrides body)"},
			&cli.StringFlag{Name: "project", Usage: "Project ID or URL (overrides body)"},
		},
		Action: expensesCreate,
	}
}

func expensesUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update an expense",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Expense ID"},
			&cli.StringFlag{Name: "url", Usage: "Expense URL"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with expense payload or expense object", Required: true},
		},
		Action: expensesUpdate,
	}
}

func expensesDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an expense",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Expense ID"},
			&cli.StringFlag{Name: "url", Usage: "Expense URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: expensesDelete,
	}
}

func expensesCreate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	expense, err := loadResourceObject(c.String("body"), "expense")
	if err != nil {
		return err
	}

	if v := strings.TrimSpace(c.String("user")); v != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "users", v)
		if err != nil {
			return err
		}
		expense["user"] = resolved
	}
	if v := strings.TrimSpace(c.String("category")); v != "" {
		if strings.EqualFold(v, "Mileage") {
			expense["category"] = "Mileage"
		} else {
			resolved, err := normalizeResourceURL(profile.BaseURL, "categories", v)
			if err != nil {
				return err
			}
			expense["category"] = resolved
		}
	}
	if v := strings.TrimSpace(c.String("dated-on")); v != "" {
		expense["dated_on"] = v
	}
	if v := strings.TrimSpace(c.String("gross-value")); v != "" {
		expense["gross_value"] = v
	}
	if v := strings.TrimSpace(c.String("currency")); v != "" {
		expense["currency"] = v
	}
	if v := strings.TrimSpace(c.String("description")); v != "" {
		expense["description"] = v
	}
	if v := strings.TrimSpace(c.String("project")); v != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "projects", v)
		if err != nil {
			return err
		}
		expense["project"] = resolved
	}

	for _, field := range []string{"user", "category", "dated_on"} {
		if _, ok := expense[field]; !ok {
			return fmt.Errorf("%s is required (set via flag or --body)", field)
		}
	}
	if cat, _ := expense["category"].(string); !strings.EqualFold(cat, "Mileage") {
		if _, ok := expense["gross_value"]; !ok {
			return fmt.Errorf("gross_value is required when category is not Mileage")
		}
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/expenses", map[string]any{"expense": expense})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["expense"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Expense created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created expense %v\n", created["url"])
		return nil
	})
}

func expensesUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "expenses", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	expense, err := loadResourceObject(c.String("body"), "expense")
	if err != nil {
		return err
	}
	if len(expense) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"expense": expense})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated expense %s\n", path)
		return nil
	})
}

func expensesDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "expenses", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}

	if !c.Bool("yes") {
		fmt.Fprintf(os.Stdout, "Delete expense %s? (y/N): ", path)
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
		fmt.Fprintf(os.Stdout, "Deleted expense %s\n", path)
		return nil
	})
}
