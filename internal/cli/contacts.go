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
	"github.com/anjor/freeagent-cli/internal/freeagent"

	"github.com/urfave/cli/v2"
)

func contactsCommand() *cli.Command {
	return &cli.Command{
		Name:  "contacts",
		Usage: "Manage contacts",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List contacts",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "view", Usage: "API view filter (for example: active)"},
					&cli.StringFlag{Name: "sort", Usage: "API sort field"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "query", Usage: "Local name/email filter"},
				},
				Action: contactsList,
			},
			{
				Name:  "search",
				Usage: "Search contacts by name or email",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "query", Usage: "Name or email to match"},
					&cli.StringFlag{Name: "view", Usage: "API view filter (for example: active)"},
					&cli.StringFlag{Name: "sort", Usage: "API sort field"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
				},
				Action: contactsSearch,
			},
			{
				Name:  "get",
				Usage: "Get a contact by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Contact ID"},
					&cli.StringFlag{Name: "url", Usage: "Contact URL"},
				},
				Action: contactsGet,
			},
			{
				Name:  "create",
				Usage: "Create a contact",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "body", Usage: "JSON file with full contact payload or contact object"},
					&cli.StringFlag{Name: "organisation", Usage: "Organisation name"},
					&cli.StringFlag{Name: "first-name"},
					&cli.StringFlag{Name: "last-name"},
					&cli.StringFlag{Name: "email"},
					&cli.StringFlag{Name: "billing-email"},
					&cli.StringFlag{Name: "phone"},
					&cli.StringFlag{Name: "mobile"},
					&cli.StringFlag{Name: "address1"},
					&cli.StringFlag{Name: "address2"},
					&cli.StringFlag{Name: "address3"},
					&cli.StringFlag{Name: "town"},
					&cli.StringFlag{Name: "region"},
					&cli.StringFlag{Name: "postcode"},
					&cli.StringFlag{Name: "country"},
				},
				Action: contactsCreate,
			},
		},
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

func contactsCreate(c *cli.Context) error {
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

	payload, err := buildContactPayload(c)
	if err != nil {
		return err
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/contacts", payload)
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
	if contact != nil {
		fmt.Fprintf(os.Stdout, "Created contact %v (%v)\n", contactDisplayName(contact), contact["url"])
		return nil
	}
	fmt.Fprintln(os.Stdout, "Contact created")
	return nil
}

func buildContactPayload(c *cli.Context) (map[string]any, error) {
	var contact map[string]any
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
		if v, ok := decoded["contact"].(map[string]any); ok {
			payload = decoded
			contact = v
		} else {
			contact = decoded
			payload["contact"] = contact
		}
	} else {
		contact = map[string]any{}
		payload["contact"] = contact
	}

	if org := strings.TrimSpace(c.String("organisation")); org != "" {
		contact["organisation_name"] = org
	}
	if first := strings.TrimSpace(c.String("first-name")); first != "" {
		contact["first_name"] = first
	}
	if last := strings.TrimSpace(c.String("last-name")); last != "" {
		contact["last_name"] = last
	}
	if email := strings.TrimSpace(c.String("email")); email != "" {
		contact["email"] = email
	}
	if email := strings.TrimSpace(c.String("billing-email")); email != "" {
		contact["billing_email"] = email
	}
	if phone := strings.TrimSpace(c.String("phone")); phone != "" {
		contact["phone_number"] = phone
	}
	if mobile := strings.TrimSpace(c.String("mobile")); mobile != "" {
		contact["mobile"] = mobile
	}

	if addr1 := strings.TrimSpace(c.String("address1")); addr1 != "" {
		contact["address1"] = addr1
	}
	if addr2 := strings.TrimSpace(c.String("address2")); addr2 != "" {
		contact["address2"] = addr2
	}
	if addr3 := strings.TrimSpace(c.String("address3")); addr3 != "" {
		contact["address3"] = addr3
	}
	if town := strings.TrimSpace(c.String("town")); town != "" {
		contact["town"] = town
	}
	if region := strings.TrimSpace(c.String("region")); region != "" {
		contact["region"] = region
	}
	if postcode := strings.TrimSpace(c.String("postcode")); postcode != "" {
		contact["postcode"] = postcode
	}
	if country := strings.TrimSpace(c.String("country")); country != "" {
		contact["country"] = country
	}

	if _, ok := contact["organisation_name"]; !ok {
		first, _ := contact["first_name"].(string)
		last, _ := contact["last_name"].(string)
		if strings.TrimSpace(first) == "" && strings.TrimSpace(last) == "" {
			return nil, fmt.Errorf("organisation or first-name/last-name required (or include in --body)")
		}
	}
	return payload, nil
}

func resolveContactValue(ctx context.Context, client *freeagent.Client, baseURL, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("contact is required")
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "/v2/") || strings.HasPrefix(value, "/") {
		return normalizeResourceURL(baseURL, "contacts", value)
	}
	if isLikelyID(value) {
		return normalizeResourceURL(baseURL, "contacts", value)
	}

	contacts, err := fetchContacts(ctx, client, "")
	if err != nil {
		return "", err
	}
	match, err := resolveContactMatch(contacts, value)
	if err != nil {
		return "", err
	}
	return match, nil
}

func fetchContacts(ctx context.Context, client *freeagent.Client, query string) ([]any, error) {
	path := "/contacts"
	if query != "" {
		path += "?" + query
	}
	resp, _, _, err := client.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return nil, err
	}
	list, _ := decoded["contacts"].([]any)
	return list, nil
}

func resolveContactMatch(contacts []any, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("contact is required")
	}

	exact := matchContacts(contacts, query, true)
	if len(exact) == 1 {
		return contactURL(exact[0]), nil
	}
	if len(exact) > 1 {
		return "", formatContactAmbiguous(query, exact)
	}

	partial := matchContacts(contacts, query, false)
	if len(partial) == 1 {
		return contactURL(partial[0]), nil
	}
	if len(partial) > 1 {
		return "", formatContactAmbiguous(query, partial)
	}

	return "", fmt.Errorf("no contact matches %q", query)
}

func matchContacts(contacts []any, query string, exact bool) []map[string]any {
	query = strings.ToLower(strings.TrimSpace(query))
	var matches []map[string]any
	for _, item := range contacts {
		contact, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(contactDisplayName(contact)))
		email := strings.ToLower(strings.TrimSpace(contactEmail(contact)))
		values := []string{name, email}
		for _, v := range values {
			if v == "" {
				continue
			}
			if exact && v == query {
				matches = append(matches, contact)
				break
			}
			if !exact && strings.Contains(v, query) {
				matches = append(matches, contact)
				break
			}
		}
	}
	return matches
}

func formatContactAmbiguous(query string, matches []map[string]any) error {
	var options []string
	for _, contact := range matches {
		name := contactDisplayName(contact)
		email := contactEmail(contact)
		if email != "" {
			options = append(options, fmt.Sprintf("%s <%s>", name, email))
		} else {
			options = append(options, name)
		}
	}
	return fmt.Errorf("multiple contacts match %q: %s", query, strings.Join(options, "; "))
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

func contactURL(contact map[string]any) string {
	if contact == nil {
		return ""
	}
	if urlValue, ok := contact["url"].(string); ok {
		return urlValue
	}
	return ""
}

func isLikelyID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
