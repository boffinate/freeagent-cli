//go:build !readonly

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anjor/freeagent-cli/internal/config"
	"github.com/anjor/freeagent-cli/internal/freeagent"

	"github.com/urfave/cli/v2"
)

func invoiceWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		invoiceDeleteCmd(),
		invoiceCreateCmd(),
		invoiceSendCmd(),
	}
}

func invoiceDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a draft invoice",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Invoice ID"},
			&cli.StringFlag{Name: "url", Usage: "Invoice URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
			&cli.BoolFlag{Name: "force", Usage: "Allow delete even if not Draft"},
		},
		Action: invoiceDelete,
	}
}

func invoiceCreateCmd() *cli.Command {
	return &cli.Command{
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
	}
}

func invoiceSendCmd() *cli.Command {
	return &cli.Command{
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

	payload, err := buildInvoicePayload(c, client, profile.BaseURL)
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

func invoiceDelete(c *cli.Context) error {
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

	var decoded map[string]any
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return err
	}
	invoice, _ := decoded["invoice"].(map[string]any)
	status := ""
	reference := ""
	if invoice != nil {
		if v, ok := invoice["status"].(string); ok {
			status = v
		}
		if v, ok := invoice["reference"].(string); ok {
			reference = v
		}
	}

	if !c.Bool("force") && status != "" && !strings.EqualFold(status, "Draft") {
		return fmt.Errorf("invoice status is %s; use --force to delete anyway", status)
	}

	if !c.Bool("yes") {
		label := path
		if reference != "" {
			label = fmt.Sprintf("%s (%s)", reference, path)
		}
		if !confirmDelete("invoice", label) {
			fmt.Fprintln(os.Stdout, "Cancelled")
			return nil
		}
	}

	resp, _, _, err = client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}

	if rt.JSONOutput {
		if len(resp) == 0 {
			return writeJSONOutput([]byte(`{"status":"ok"}`))
		}
		return writeJSONOutput(resp)
	}

	if reference != "" {
		fmt.Fprintf(os.Stdout, "Deleted invoice %s\n", reference)
		return nil
	}
	fmt.Fprintln(os.Stdout, "Deleted invoice")
	return nil
}

func buildInvoicePayload(c *cli.Context, client *freeagent.Client, baseURL string) (map[string]any, error) {
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
		resolved, err := resolveContactValue(c.Context, client, baseURL, contact)
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
