package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func categoriesCommand() *cli.Command {
	return &cli.Command{
		Name:  "categories",
		Usage: "Accounting categories",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List categories", Action: categoriesList},
		},
	}
}

func priceListItemsCommand() *cli.Command {
	return &cli.Command{
		Name:  "price-list-items",
		Usage: "Price list items",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List price list items", Action: priceListItemsList},
			{
				Name:  "get",
				Usage: "Get a price list item by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Price list item ID"},
					&cli.StringFlag{Name: "url", Usage: "Price list item URL"},
				},
				Action: priceListItemsGet,
			},
		},
	}
}

func stockItemsCommand() *cli.Command {
	return &cli.Command{
		Name:  "stock-items",
		Usage: "Stock items",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List stock items", Action: stockItemsList},
			{
				Name:  "get",
				Usage: "Get a stock item by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Stock item ID"},
					&cli.StringFlag{Name: "url", Usage: "Stock item URL"},
				},
				Action: stockItemsGet,
			},
		},
	}
}

func categoriesList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/categories", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		// The API returns categories grouped by type (admin_expenses_categories,
		// cost_of_sales_categories, income_categories, general_categories).
		groups := []string{
			"admin_expenses_categories",
			"cost_of_sales_categories",
			"income_categories",
			"general_categories",
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Type\tNomCode\tDescription\tURL")
		found := false
		for _, g := range groups {
			list, _ := decoded[g].([]any)
			for _, item := range list {
				cat, _ := item.(map[string]any)
				if cat == nil {
					continue
				}
				found = true
				fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", g, cat["nominal_code"], cat["description"], cat["url"])
			}
		}
		if !found {
			fmt.Fprintln(os.Stdout, "No categories found")
			return nil
		}
		return w.Flush()
	})
}

func priceListItemsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/price_list_items", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["price_list_items"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No price list items found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Code\tDescription\tPrice\tURL")
		for _, item := range list {
			p, _ := item.(map[string]any)
			if p == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", p["code"], p["description"], p["price"], p["url"])
		}
		return w.Flush()
	})
}

func priceListItemsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "price_list_items", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintln(os.Stdout, string(resp))
		return nil
	})
}

func stockItemsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/stock_items", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["stock_items"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No stock items found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Description\tStockOnHand\tURL")
		for _, item := range list {
			s, _ := item.(map[string]any)
			if s == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\n", s["description"], s["stock_on_hand"], s["url"])
		}
		return w.Flush()
	})
}

func stockItemsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "stock_items", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintln(os.Stdout, string(resp))
		return nil
	})
}
