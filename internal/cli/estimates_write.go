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
		estimateItemsCmd(),
	}
}

func estimateItemsCmd() *cli.Command {
	return &cli.Command{
		Name:  "items",
		Usage: "Manage individual estimate line items",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Add a line item to an estimate",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "estimate", Required: true, Usage: "Estimate ID or URL the item belongs to"},
					&cli.StringFlag{Name: "description", Usage: "Item description (overrides body)"},
					&cli.StringFlag{Name: "price", Usage: "Unit price (overrides body)"},
					&cli.StringFlag{Name: "item-type", Usage: "Item type (Hours, Days, Products, Services, Months, Years) (overrides body)"},
					&cli.StringFlag{Name: "quantity", Usage: "Quantity (overrides body)"},
					&cli.StringFlag{Name: "category", Usage: "Category URL (overrides body)"},
					&cli.StringFlag{Name: "body", Usage: "JSON file with estimate_item payload or estimate_item object"},
				},
				Action: estimateItemsCreate,
			},
			{
				Name:  "update",
				Usage: "Update an estimate line item",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Estimate item ID"},
					&cli.StringFlag{Name: "url", Usage: "Estimate item URL"},
					&cli.StringFlag{Name: "description", Usage: "Item description (overrides body)"},
					&cli.StringFlag{Name: "price", Usage: "Unit price (overrides body)"},
					&cli.StringFlag{Name: "item-type", Usage: "Item type (overrides body)"},
					&cli.StringFlag{Name: "quantity", Usage: "Quantity (overrides body)"},
					&cli.StringFlag{Name: "category", Usage: "Category URL (overrides body)"},
					&cli.StringFlag{Name: "body", Usage: "JSON file with estimate_item payload or estimate_item object"},
				},
				Action: estimateItemsUpdate,
			},
			{
				Name:  "delete",
				Usage: "Delete an estimate line item",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Estimate item ID"},
					&cli.StringFlag{Name: "url", Usage: "Estimate item URL"},
					&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
				},
				Action: estimateItemsDelete,
			},
		},
	}
}

func estimateItemsCreate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	estimateURL, err := normalizeResourceURL(profile.BaseURL, "estimates", c.String("estimate"))
	if err != nil {
		return err
	}
	item, err := loadResourceObject(c.String("body"), "estimate_item")
	if err != nil {
		return err
	}
	for _, m := range []struct {
		flag, key string
	}{
		{"description", "description"},
		{"price", "price"},
		{"item-type", "item_type"},
		{"quantity", "quantity"},
		{"category", "category"},
	} {
		if v := strings.TrimSpace(c.String(m.flag)); v != "" {
			item[m.key] = v
		}
	}
	for _, field := range []string{"description", "price", "item_type"} {
		if _, ok := item[field]; !ok {
			return fmt.Errorf("%s is required (set via flag or --body)", field)
		}
	}
	payload := map[string]any{
		"estimate":      estimateURL,
		"estimate_item": item,
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/estimate_items", payload)
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["estimate_item"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Estimate item created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created estimate item %v\n", created["url"])
		return nil
	})
}

func estimateItemsUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "estimate_items", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	item, err := loadResourceObject(c.String("body"), "estimate_item")
	if err != nil {
		return err
	}
	for _, m := range []struct {
		flag, key string
	}{
		{"description", "description"},
		{"price", "price"},
		{"item-type", "item_type"},
		{"quantity", "quantity"},
		{"category", "category"},
	} {
		if v := strings.TrimSpace(c.String(m.flag)); v != "" {
			item[m.key] = v
		}
	}
	if len(item) == 0 {
		return fmt.Errorf("at least one field is required to update (set via flag or --body)")
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"estimate_item": item})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated estimate item %s\n", path)
		return nil
	})
}

func estimateItemsDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "estimate_items", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	if !c.Bool("yes") && !confirmDelete("estimate item", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted estimate item %s\n", path)
		return nil
	})
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
			&cli.StringFlag{Name: "status", Usage: "Estimate status (Draft, Sent, Approved, etc.) (overrides body)"},
			&cli.StringFlag{Name: "estimate-type", Usage: "Estimate type (Estimate, Quote, Proposal) (overrides body)"},
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
	if v := strings.TrimSpace(c.String("status")); v != "" {
		estimate["status"] = v
	}
	if v := strings.TrimSpace(c.String("estimate-type")); v != "" {
		estimate["estimate_type"] = v
	}
	if itemsPath := c.String("items"); itemsPath != "" {
		items, err := loadItemsArray(itemsPath, "estimate_items")
		if err != nil {
			return err
		}
		estimate["estimate_items"] = items
	}
	for _, field := range []string{"contact", "reference", "dated_on", "currency", "status", "estimate_type"} {
		if _, ok := estimate[field]; !ok {
			return fmt.Errorf("%s is required (set via flag or --body)", field)
		}
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
	if !c.Bool("yes") && !confirmDelete("estimate", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
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
