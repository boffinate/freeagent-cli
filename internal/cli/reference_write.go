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

func priceListItemsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		priceListItemsCreateCmd(),
		priceListItemsUpdateCmd(),
		priceListItemsDeleteCmd(),
	}
}

func priceListItemsCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a price list item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body", Usage: "JSON file with full price_list_item payload or object"},
			&cli.StringFlag{Name: "code", Usage: "Item code (overrides body)"},
			&cli.StringFlag{Name: "description", Usage: "Description (overrides body)"},
			&cli.StringFlag{Name: "price", Usage: "Price (overrides body)"},
			&cli.StringFlag{Name: "quantity", Usage: "Item quantity (overrides body)"},
			&cli.StringFlag{Name: "item-type", Usage: "Item type, e.g. Hours, Days, Products, Services (overrides body)"},
			&cli.StringFlag{Name: "sales-tax-rate", Usage: "Sales tax rate, e.g. 20.0 (overrides body)"},
		},
		Action: priceListItemsCreate,
	}
}

func priceListItemsUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a price list item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Price list item ID"},
			&cli.StringFlag{Name: "url", Usage: "Price list item URL"},
			&cli.StringFlag{Name: "body", Required: true, Usage: "JSON file with price_list_item payload or object"},
		},
		Action: priceListItemsUpdate,
	}
}

func priceListItemsDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a price list item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Price list item ID"},
			&cli.StringFlag{Name: "url", Usage: "Price list item URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: priceListItemsDelete,
	}
}

func priceListItemsCreate(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	item, err := loadResourceObject(c.String("body"), "price_list_item")
	if err != nil {
		return err
	}
	if v := strings.TrimSpace(c.String("code")); v != "" {
		item["code"] = v
	}
	if v := strings.TrimSpace(c.String("description")); v != "" {
		item["description"] = v
	}
	if v := strings.TrimSpace(c.String("price")); v != "" {
		item["price"] = v
	}
	if v := strings.TrimSpace(c.String("quantity")); v != "" {
		item["quantity"] = v
	}
	if v := strings.TrimSpace(c.String("item-type")); v != "" {
		item["item_type"] = v
	}
	if v := strings.TrimSpace(c.String("sales-tax-rate")); v != "" {
		item["sales_tax_rate"] = v
	}
	for _, field := range []string{"code", "quantity", "item_type", "description", "price"} {
		if _, ok := item[field]; !ok {
			return fmt.Errorf("%s is required (set via flag or --body)", field)
		}
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/price_list_items", map[string]any{"price_list_item": item})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["price_list_item"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Price list item created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created price list item %v (%v)\n", created["code"], created["url"])
		return nil
	})
}

func priceListItemsUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "price_list_items", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	item, err := loadResourceObject(c.String("body"), "price_list_item")
	if err != nil {
		return err
	}
	if len(item) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"price_list_item": item})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated price list item %s\n", path)
		return nil
	})
}

func priceListItemsDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "price_list_items", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	if !c.Bool("yes") && !confirmDelete("price list item", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted price list item %s\n", path)
		return nil
	})
}
