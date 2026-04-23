package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/anjor/freeagent-cli/internal/config"

	"github.com/urfave/cli/v2"
)

func contactsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "contacts",
		Usage: "Manage contacts",
		Subcommands: []*cli.Command{
			contactsListCmd(),
			contactsSearchCmd(),
			contactsGetCmd(),
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, contactsWriteSubcommands()...)
	return cmd
}

func contactsListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List contacts",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "view", Usage: "API view filter (for example: active)"},
			&cli.StringFlag{Name: "sort", Usage: "API sort field"},
			&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
			&cli.StringFlag{Name: "query", Usage: "Local name/email filter"},
		},
		Action: contactsList,
	}
}

func contactsSearchCmd() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search contacts by name or email",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "query", Usage: "Name or email to match"},
			&cli.StringFlag{Name: "view", Usage: "API view filter (for example: active)"},
			&cli.StringFlag{Name: "sort", Usage: "API sort field"},
			&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
		},
		Action: contactsSearch,
	}
}

func contactsGetCmd() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a contact by ID or URL",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Contact ID"},
			&cli.StringFlag{Name: "url", Usage: "Contact URL"},
		},
		Action: contactsGet,
	}
}

func contactsList(c *cli.Context) error {
	return contactsListWithQuery(c, c.String("query"), false)
}

func contactsSearch(c *cli.Context) error {
	query := strings.TrimSpace(c.String("query"))
	if query == "" {
		return fmt.Errorf("query is required")
	}
	return contactsListWithQuery(c, query, true)
}

func contactsListWithQuery(c *cli.Context, query string, requireQuery bool) error {
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

	path := "/contacts"
	if queryParams := buildContactsQuery(c); queryParams != "" {
		path += "?" + queryParams
	}

	resp, _, _, err := client.Do(context.Background(), http.MethodGet, path, nil, "")
	if err != nil {
		return err
	}

	var decoded map[string]any
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return err
	}
	list, _ := decoded["contacts"].([]any)

	filtered := list
	query = strings.TrimSpace(query)
	if query != "" {
		filtered = filterContacts(list, query)
	}
	if requireQuery && query == "" {
		return fmt.Errorf("query is required")
	}

	if rt.JSONOutput {
		if query != "" {
			data, err := json.Marshal(map[string]any{"contacts": filtered})
			if err != nil {
				return err
			}
			return writeJSONOutput(data)
		}
		return writeJSONOutput(resp)
	}

	if len(filtered) == 0 {
		fmt.Fprintln(os.Stdout, "No contacts found")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "Name\tEmail\tURL")
	for _, item := range filtered {
		contact, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(writer, "%v\t%v\t%v\n", contactDisplayName(contact), contactEmail(contact), contact["url"])
	}
	_ = writer.Flush()
	return nil
}

func buildContactsQuery(c *cli.Context) string {
	query := url.Values{}
	if v := c.String("view"); v != "" {
		query.Set("view", v)
	}
	if v := c.String("sort"); v != "" {
		query.Set("sort", v)
	}
	if v := c.String("updated-since"); v != "" {
		query.Set("updated_since", v)
	}
	return query.Encode()
}

func contactsGet(c *cli.Context) error {
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
		path = fmt.Sprintf("/contacts/%s", id)
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
	contact, _ := decoded["contact"].(map[string]any)
	if contact == nil {
		fmt.Fprintln(os.Stdout, string(resp))
		return nil
	}

	fmt.Fprintf(os.Stdout, "Name:     %v\n", contactDisplayName(contact))
	fmt.Fprintf(os.Stdout, "Email:    %v\n", contactEmail(contact))
	fmt.Fprintf(os.Stdout, "URL:      %v\n", contact["url"])
	if v := contact["phone_number"]; v != nil {
		fmt.Fprintf(os.Stdout, "Phone:    %v\n", v)
	}
	if v := contact["mobile"]; v != nil {
		fmt.Fprintf(os.Stdout, "Mobile:   %v\n", v)
	}
	return nil
}

func filterContacts(list []any, query string) []any {
	query = strings.TrimSpace(query)
	if query == "" {
		return list
	}
	var out []any
	lower := strings.ToLower(query)
	for _, item := range list {
		contact, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.ToLower(contactDisplayName(contact))
		email := strings.ToLower(contactEmail(contact))
		if strings.Contains(name, lower) || strings.Contains(email, lower) {
			out = append(out, contact)
		}
	}
	return out
}

func contactDisplayName(contact map[string]any) string {
	if contact == nil {
		return ""
	}
	if name, ok := contact["organisation_name"].(string); ok && name != "" {
		return name
	}
	first, _ := contact["first_name"].(string)
	last, _ := contact["last_name"].(string)
	full := strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
	if full != "" {
		return full
	}
	if name, ok := contact["display_name"].(string); ok && name != "" {
		return name
	}
	if name, ok := contact["name"].(string); ok && name != "" {
		return name
	}
	if url, ok := contact["url"].(string); ok {
		return url
	}
	return ""
}

func contactEmail(contact map[string]any) string {
	if contact == nil {
		return ""
	}
	if email, ok := contact["email"].(string); ok && email != "" {
		return email
	}
	if email, ok := contact["billing_email"].(string); ok && email != "" {
		return email
	}
	return ""
}
