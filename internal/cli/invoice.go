package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/anjor/freeagent-cli/internal/config"
	"github.com/anjor/freeagent-cli/internal/freeagent"

	"github.com/urfave/cli/v2"
)

func invoiceCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "invoices",
		Usage: "Create and send invoices",
		Subcommands: []*cli.Command{
			invoiceListCmd(),
			invoiceGetCmd(),
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, invoiceWriteSubcommands()...)
	return cmd
}

func invoiceListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List invoices",
		Flags: withPagination(
			&cli.StringFlag{Name: "view", Usage: "API view filter (for example: recent)"},
			&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
			&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
			&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
			&cli.StringFlag{Name: "status", Usage: "Invoice status"},
			&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
		),
		Action: invoiceList,
	}
}

func invoiceGetCmd() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a single invoice by ID or URL",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Invoice ID"},
			&cli.StringFlag{Name: "url", Usage: "Invoice URL"},
		},
		Action: invoiceGet,
	}
}

func invoiceList(c *cli.Context) error {
	rt, err := runtimeFrom(c)
	if err != nil {
		return err
	}

	cfg, _, err := loadConfig(rt)
	if err != nil {
		return err
	}
	profile := ensureProfile(cfg, rt.Profile, rt, config.Profile{})

	client, _, err := newClient(context.Background(), rt, profile)
	if err != nil {
		return err
	}

	params := map[string]string{
		"view":          c.String("view"),
		"status":        c.String("status"),
		"from_date":     c.String("from"),
		"to_date":       c.String("to"),
		"updated_since": c.String("updated-since"),
	}
	if v := c.String("contact"); v != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "contacts", v)
		if err != nil {
			return err
		}
		params["contact"] = resolved
	}

	resp, err := listAll(context.Background(), client, "/invoices", params, "invoices", paginationOptsFrom(c))
	if err != nil {
		return err
	}

	if rt.JSONOutput {
		return writeJSONOutput(resp)
	}

	var decoded map[string]any
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return err
	}

	list, _ := decoded["invoices"].([]any)
	if len(list) == 0 {
		fmt.Fprintln(os.Stdout, "No invoices found")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "Reference\tStatus\tContact\tAmount\tURL")

	contactCache := map[string]string{}

	for _, item := range list {
		inv, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ref := inv["reference"]
		status := inv["status"]
		url := inv["url"]
		amount := inv["total_value"]
		currency := inv["currency"]
		contactDisplay := inv["contact"]
		if contactURL, ok := inv["contact"].(string); ok && contactURL != "" {
			if cached, ok := contactCache[contactURL]; ok {
				contactDisplay = cached
			} else if contactName, err := fetchContactName(client, contactURL); err == nil && contactName != "" {
				contactDisplay = contactName
				contactCache[contactURL] = contactName
			}
		}
		if ref != nil || status != nil || url != nil {
			if currency != nil && amount != nil {
				fmt.Fprintf(writer, "%v\t%v\t%v\t%v %v\t%v\n", ref, status, contactDisplay, currency, amount, url)
			} else {
				fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\n", ref, status, contactDisplay, "-", url)
			}
		}
	}
	_ = writer.Flush()
	return nil
}

func invoiceGet(c *cli.Context) error {
	rt, err := runtimeFrom(c)
	if err != nil {
		return err
	}

	cfg, _, err := loadConfig(rt)
	if err != nil {
		return err
	}
	profile := ensureProfile(cfg, rt.Profile, rt, config.Profile{})

	client, _, err := newClient(context.Background(), rt, profile)
	if err != nil {
		return err
	}

	id := c.String("id")
	urlValue := c.String("url")
	if id == "" && urlValue == "" {
		return fmt.Errorf("id or url required")
	}

	path := ""
	if urlValue != "" {
		path = urlValue
	} else {
		path = fmt.Sprintf("/invoices/%s", id)
	}

	resp, _, _, err := client.Do(context.Background(), http.MethodGet, path, nil, "")
	if err != nil {
		return err
	}

	if rt.JSONOutput {
		return writeJSONOutput(resp)
	}

	var decoded map[string]any
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return err
	}
	invoice, _ := decoded["invoice"].(map[string]any)
	if invoice == nil {
		fmt.Fprintln(os.Stdout, string(resp))
		return nil
	}

	contactDisplay := invoice["contact"]
	if contactURL, ok := invoice["contact"].(string); ok && contactURL != "" {
		if contactName, err := fetchContactName(client, contactURL); err == nil && contactName != "" {
			contactDisplay = fmt.Sprintf("%s (%s)", contactName, contactURL)
		}
	}

	fmt.Fprintf(os.Stdout, "Reference: %v\n", invoice["reference"])
	fmt.Fprintf(os.Stdout, "Status:    %v\n", invoice["status"])
	fmt.Fprintf(os.Stdout, "URL:       %v\n", invoice["url"])
	fmt.Fprintf(os.Stdout, "Contact:   %v\n", contactDisplay)
	fmt.Fprintf(os.Stdout, "Dated On:  %v\n", invoice["dated_on"])
	fmt.Fprintf(os.Stdout, "Due On:    %v\n", invoice["due_on"])
	fmt.Fprintf(os.Stdout, "Total:     %v %v\n", invoice["currency"], invoice["total_value"])
	return nil
}

func fetchContactName(client *freeagent.Client, contactURL string) (string, error) {
	resp, _, _, err := client.Do(context.Background(), http.MethodGet, contactURL, nil, "")
	if err != nil {
		return "", err
	}
	var decoded map[string]any
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return "", err
	}
	contact, _ := decoded["contact"].(map[string]any)
	if contact == nil {
		return "", nil
	}
	if name, ok := contact["organisation_name"].(string); ok && name != "" {
		return name, nil
	}
	if name, ok := contact["display_name"].(string); ok && name != "" {
		return name, nil
	}
	if name, ok := contact["name"].(string); ok && name != "" {
		return name, nil
	}
	return "", nil
}
