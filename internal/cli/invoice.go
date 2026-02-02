package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"freegant-cli/internal/config"
	"freegant-cli/internal/freeagent"

	"github.com/urfave/cli/v2"
)

func invoiceCommand() *cli.Command {
	return &cli.Command{
		Name:  "invoices",
		Usage: "Create and send invoices",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List invoices",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "view", Usage: "API view filter (for example: recent)"},
					&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "status", Usage: "Invoice status"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
				},
				Action: invoiceList,
			},
			{
				Name:  "get",
				Usage: "Get a single invoice by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Invoice ID"},
					&cli.StringFlag{Name: "url", Usage: "Invoice URL"},
				},
				Action: invoiceGet,
			},
			{
				Name:  "create",
				Usage: "Create a draft invoice",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
					&cli.StringFlag{Name: "reference"},
					&cli.StringFlag{Name: "currency", Value: "GBP"},
					&cli.StringFlag{Name: "date", Usage: "Invoice date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "due", Usage: "Due date (YYYY-MM-DD)"},
					&cli.IntFlag{Name: "payment-terms-days", Value: 30},
					&cli.StringFlag{Name: "lines", Usage: "JSON file with line items"},
					&cli.StringFlag{Name: "body", Usage: "JSON file with full invoice payload or invoice object"},
				},
				Action: invoiceCreate,
			},
			{
				Name:  "send",
				Usage: "Send an existing draft invoice",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Invoice ID"},
					&cli.StringFlag{Name: "url", Usage: "Invoice URL"},
					&cli.StringFlag{Name: "email-to", Usage: "Recipient email address"},
					&cli.StringFlag{Name: "cc"},
					&cli.StringFlag{Name: "bcc"},
					&cli.StringFlag{Name: "subject"},
					&cli.StringFlag{Name: "body", Usage: "JSON file with send payload"},
					&cli.StringFlag{Name: "message", Usage: "Email body"},
				},
				Action: invoiceSend,
			},
		},
	}
}

func invoiceCreate(c *cli.Context) error {
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

	payload, err := buildInvoicePayload(c, profile.BaseURL)
	if err != nil {
		return err
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/invoices", payload)
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
	if invoice != nil {
		fmt.Fprintf(os.Stdout, "Created invoice %v (%v)\n", invoice["reference"], invoice["url"])
		return nil
	}
	fmt.Fprintln(os.Stdout, "Invoice created")
	return nil
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

	query := url.Values{}
	if v := c.String("view"); v != "" {
		query.Set("view", v)
	}
	if v := c.String("status"); v != "" {
		query.Set("status", v)
	}
	if v := c.String("from"); v != "" {
		query.Set("from_date", v)
	}
	if v := c.String("to"); v != "" {
		query.Set("to_date", v)
	}
	if v := c.String("updated-since"); v != "" {
		query.Set("updated_since", v)
	}
	if v := c.String("contact"); v != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "contacts", v)
		if err != nil {
			return err
		}
		query.Set("contact", resolved)
	}

	path := "/invoices"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
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

	list, _ := decoded["invoices"].([]any)
	if len(list) == 0 {
		fmt.Fprintln(os.Stdout, "No invoices found")
		return nil
	}

	for _, item := range list {
		inv, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ref := inv["reference"]
		status := inv["status"]
		url := inv["url"]
		if ref != nil || status != nil || url != nil {
			fmt.Fprintf(os.Stdout, "%v\t%v\t%v\n", ref, status, url)
		}
	}
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

func buildInvoicePayload(c *cli.Context, baseURL string) (map[string]any, error) {
	var invoice map[string]any
	payload := map[string]any{}

	if bodyPath := c.String("body"); bodyPath != "" {
		data, err := os.ReadFile(bodyPath)
		if err != nil {
			return nil, err
		}
		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, err
		}
		if v, ok := decoded["invoice"].(map[string]any); ok {
			payload = decoded
			invoice = v
		} else {
			invoice = decoded
			payload["invoice"] = invoice
		}
	} else {
		invoice = map[string]any{}
		payload["invoice"] = invoice
	}

	if contact := c.String("contact"); contact != "" {
		resolved, err := normalizeResourceURL(baseURL, "contacts", contact)
		if err != nil {
			return nil, err
		}
		invoice["contact"] = resolved
	}

	if ref := c.String("reference"); ref != "" {
		invoice["reference"] = ref
	}
	if currency := c.String("currency"); currency != "" {
		invoice["currency"] = currency
	}

	if date := c.String("date"); date != "" {
		invoice["dated_on"] = date
	} else if _, ok := invoice["dated_on"]; !ok {
		invoice["dated_on"] = time.Now().Format("2006-01-02")
	}

	if due := c.String("due"); due != "" {
		invoice["due_on"] = due
	}

	if _, ok := invoice["payment_terms_in_days"]; !ok {
		invoice["payment_terms_in_days"] = c.Int("payment-terms-days")
	}

	if linesPath := c.String("lines"); linesPath != "" {
		data, err := os.ReadFile(linesPath)
		if err != nil {
			return nil, err
		}
		var decoded any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, err
		}
		switch v := decoded.(type) {
		case map[string]any:
			if items, ok := v["invoice_items"]; ok {
				invoice["invoice_items"] = items
			} else {
				invoice["invoice_items"] = v
			}
		case []any:
			invoice["invoice_items"] = v
		default:
			return nil, fmt.Errorf("lines must be an array or object")
		}
	}

	if _, ok := invoice["contact"]; !ok {
		return nil, fmt.Errorf("contact is required (use --contact or include in --body)")
	}
	return payload, nil
}

func invoiceSend(c *cli.Context) error {
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

	if payloadPath := c.String("body"); payloadPath != "" {
		data, err := os.ReadFile(payloadPath)
		if err != nil {
			return err
		}
		var payload any
		if err := json.Unmarshal(data, &payload); err != nil {
			return err
		}
		resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, path+"/send_email", payload)
		if err != nil {
			return err
		}
		if rt.JSONOutput {
			return writeJSONOutput(resp)
		}
		fmt.Fprintln(os.Stdout, "Sent invoice")
		return nil
	}

	if to := c.String("email-to"); to != "" {
		email := map[string]any{
			"to": to,
		}
		if cc := c.String("cc"); cc != "" {
			email["cc"] = cc
		}
		if bcc := c.String("bcc"); bcc != "" {
			email["bcc"] = bcc
		}
		if subject := c.String("subject"); subject != "" {
			email["subject"] = subject
		}
		if message := c.String("message"); message != "" {
			email["body"] = message
		}
		payload := map[string]any{"invoice": map[string]any{"email": email}}
		resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, path+"/send_email", payload)
		if err != nil {
			return err
		}
		if rt.JSONOutput {
			return writeJSONOutput(resp)
		}
		fmt.Fprintln(os.Stdout, "Sent invoice email")
		return nil
	}

	resp, _, _, err := client.Do(context.Background(), http.MethodPost, path+"/transitions/mark_as_sent", nil, "")
	if err != nil {
		return err
	}
	if rt.JSONOutput {
		return writeJSONOutput(resp)
	}
	fmt.Fprintln(os.Stdout, "Marked invoice as sent")
	return nil
}
