package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

// Domain 5: fixed assets. capital_assets and hire_purchases are read-only per
// the API; capital_asset_types and properties expose full CRUD.

func capitalAssetsCommand() *cli.Command {
	return &cli.Command{
		Name:  "capital-assets",
		Usage: "Capital assets",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List capital assets",
				Flags: append([]cli.Flag{
					&cli.StringFlag{Name: "view", Usage: "API view filter"},
					&cli.BoolFlag{Name: "include-history", Usage: "Include capital asset history"},
				}, paginationFlags()...),
				Action: capitalAssetsList,
			},
			{
				Name:  "get",
				Usage: "Get a capital asset by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Capital asset ID"},
					&cli.StringFlag{Name: "url", Usage: "Capital asset URL"},
				},
				Action: capitalAssetsGet,
			},
		},
	}
}

func capitalAssetTypesCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "capital-asset-types",
		Usage: "Capital asset types",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List capital asset types", Flags: paginationFlags(), Action: capitalAssetTypesList},
			{
				Name:  "get",
				Usage: "Get a capital asset type by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Capital asset type ID"},
					&cli.StringFlag{Name: "url", Usage: "Capital asset type URL"},
				},
				Action: capitalAssetTypesGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, capitalAssetTypesWriteSubcommands()...)
	return cmd
}

func hirePurchasesCommand() *cli.Command {
	return &cli.Command{
		Name:  "hire-purchases",
		Usage: "Hire purchases",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List hire purchases", Flags: paginationFlags(), Action: hirePurchasesList},
			{
				Name:  "get",
				Usage: "Get a hire purchase by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Hire purchase ID"},
					&cli.StringFlag{Name: "url", Usage: "Hire purchase URL"},
				},
				Action: hirePurchasesGet,
			},
		},
	}
}

func propertiesCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "properties",
		Usage: "Properties (rental income)",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List properties", Flags: paginationFlags(), Action: propertiesList},
			{
				Name:  "get",
				Usage: "Get a property by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Property ID"},
					&cli.StringFlag{Name: "url", Usage: "Property URL"},
				},
				Action: propertiesGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, propertiesWriteSubcommands()...)
	return cmd
}

func capitalAssetsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	params := map[string]string{"view": c.String("view")}
	if c.Bool("include-history") {
		params["include_history"] = "true"
	}
	resp, err := listAll(context.Background(), client, "/capital_assets", params, "capital_assets", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["capital_assets"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No capital assets found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Description\tDatedOn\tBookValue\tURL")
		for _, item := range list {
			a, _ := item.(map[string]any)
			if a == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", a["description"], a["dated_on"], a["book_value"], a["url"])
		}
		return w.Flush()
	})
}

func capitalAssetsGet(c *cli.Context) error {
	return readResource(c, "capital_assets")
}

func capitalAssetTypesList(c *cli.Context) error {
	return paginatedListPassthrough(c, "/capital_asset_types", "capital_asset_types")
}

func capitalAssetTypesGet(c *cli.Context) error {
	return readResource(c, "capital_asset_types")
}

func hirePurchasesList(c *cli.Context) error {
	return paginatedListPassthrough(c, "/hire_purchases", "hire_purchases")
}

func hirePurchasesGet(c *cli.Context) error {
	return readResource(c, "hire_purchases")
}

func propertiesList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, err := listAll(context.Background(), client, "/properties", nil, "properties", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["properties"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No properties found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Address\tType\tURL")
		for _, item := range list {
			p, _ := item.(map[string]any)
			if p == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\n", p["address"], p["property_type"], p["url"])
		}
		return w.Flush()
	})
}

func propertiesGet(c *cli.Context) error {
	return readResource(c, "properties")
}

func readResource(c *cli.Context, resource string) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, resource, c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

// paginatedListPassthrough fetches a list endpoint, auto-paginating, then
// emits the merged JSON verbatim. Used for resources that don't have a custom
// table renderer.
func paginatedListPassthrough(c *cli.Context, path, wrapper string) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, err := listAll(context.Background(), client, path, nil, wrapper, paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}
