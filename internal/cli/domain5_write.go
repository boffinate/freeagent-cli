//go:build !readonly

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"
)

func capitalAssetTypesWriteSubcommands() []*cli.Command {
	return crudWriteSubcommands(crudSpec{
		resource: "capital_asset_types",
		wrapper:  "capital_asset_type",
		label:    "capital asset type",
	})
}

func propertiesWriteSubcommands() []*cli.Command {
	return crudWriteSubcommands(crudSpec{
		resource: "properties",
		wrapper:  "property",
		label:    "property",
	})
}

// crudSpec parameterises the boilerplate create/update/delete handlers shared
// by Domain 5 (and similar domains): the URL collection segment, the JSON
// wrapper key, and a human-readable label for prompts and stdout messages.
type crudSpec struct {
	resource string // e.g. "properties" — used in URL paths
	wrapper  string // e.g. "property"   — root JSON key
	label    string // e.g. "property"   — prompt/output label
}

// crudWriteSubcommands returns the standard `create`, `update`, `delete` trio
// for a resource that follows the body-or-flags merge convention used across
// the CLI. Both the `create` and `update` commands accept --body and require
// at least one field; `delete` prompts unless --yes is supplied.
func crudWriteSubcommands(spec crudSpec) []*cli.Command {
	return []*cli.Command{
		{
			Name:  "create",
			Usage: "Create a " + spec.label,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "body", Required: true, Usage: "JSON file with " + spec.wrapper + " payload or object"},
			},
			Action: crudCreate(spec),
		},
		{
			Name:  "update",
			Usage: "Update a " + spec.label,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: spec.label + " ID"},
				&cli.StringFlag{Name: "url", Usage: spec.label + " URL"},
				&cli.StringFlag{Name: "body", Required: true, Usage: "JSON file with " + spec.wrapper + " payload or object"},
			},
			Action: crudUpdate(spec),
		},
		{
			Name:  "delete",
			Usage: "Delete a " + spec.label,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: spec.label + " ID"},
				&cli.StringFlag{Name: "url", Usage: spec.label + " URL"},
				&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
			},
			Action: crudDelete(spec),
		},
	}
}

func crudCreate(spec crudSpec) cli.ActionFunc {
	return func(c *cli.Context) error {
		rt, client, _, err := bootstrapClient(c)
		if err != nil {
			return err
		}
		obj, err := loadResourceObject(c.String("body"), spec.wrapper)
		if err != nil {
			return err
		}
		resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/"+spec.resource, map[string]any{spec.wrapper: obj})
		if err != nil {
			return err
		}
		return printOrJSON(rt, resp, func() error {
			var decoded map[string]any
			if err := json.Unmarshal(resp, &decoded); err != nil {
				return err
			}
			created, _ := decoded[spec.wrapper].(map[string]any)
			if created == nil {
				fmt.Fprintf(os.Stdout, "%s created\n", spec.label)
				return nil
			}
			fmt.Fprintf(os.Stdout, "Created %s %v\n", spec.label, created["url"])
			return nil
		})
	}
}

func crudUpdate(spec crudSpec) cli.ActionFunc {
	return func(c *cli.Context) error {
		rt, client, profile, err := bootstrapClient(c)
		if err != nil {
			return err
		}
		path, err := requireIDOrURL(profile.BaseURL, spec.resource, c.String("id"), c.String("url"))
		if err != nil {
			return err
		}
		obj, err := loadResourceObject(c.String("body"), spec.wrapper)
		if err != nil {
			return err
		}
		if len(obj) == 0 {
			return fmt.Errorf("body must contain at least one field to update")
		}
		resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{spec.wrapper: obj})
		if err != nil {
			return err
		}
		return printOrJSON(rt, resp, func() error {
			fmt.Fprintf(os.Stdout, "Updated %s %s\n", spec.label, path)
			return nil
		})
	}
}

func crudDelete(spec crudSpec) cli.ActionFunc {
	return func(c *cli.Context) error {
		rt, client, profile, err := bootstrapClient(c)
		if err != nil {
			return err
		}
		path, err := requireIDOrURL(profile.BaseURL, spec.resource, c.String("id"), c.String("url"))
		if err != nil {
			return err
		}
		if !c.Bool("yes") && !confirmDelete(spec.label, path) {
			fmt.Fprintln(os.Stdout, "Cancelled")
			return nil
		}
		resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
		if err != nil {
			return err
		}
		return printOrJSON(rt, resp, func() error {
			fmt.Fprintf(os.Stdout, "Deleted %s %s\n", spec.label, path)
			return nil
		})
	}
}
