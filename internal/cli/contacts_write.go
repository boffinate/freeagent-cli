//go:build !readonly

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/boffinate/freeagent-cli/internal/config"
	"github.com/boffinate/freeagent-cli/internal/freeagent"

	"github.com/urfave/cli/v2"
)

func contactsWriteSubcommands() []*cli.Command {
	return []*cli.Command{contactsCreateCmd(), contactsUpdateCmd(), contactsDeleteCmd()}
}

func contactsUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a contact",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Contact ID"},
			&cli.StringFlag{Name: "url", Usage: "Contact URL"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with contact payload or contact object", Required: true},
		},
		Action: contactsUpdate,
	}
}

func contactsDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a contact",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Contact ID"},
			&cli.StringFlag{Name: "url", Usage: "Contact URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: contactsDelete,
	}
}

func contactsUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "contacts", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	contact, err := loadResourceObject(c.String("body"), "contact")
	if err != nil {
		return err
	}
	if len(contact) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"contact": contact})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated contact %s\n", path)
		return nil
	})
}

func contactsDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "contacts", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	if !c.Bool("yes") && !confirmDelete("contact", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted contact %s\n", path)
		return nil
	})
}

func contactsCreateCmd() *cli.Command {
	return &cli.Command{
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
	}
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

	org, _ := contact["organisation_name"].(string)
	first, _ := contact["first_name"].(string)
	last, _ := contact["last_name"].(string)
	hasName := strings.TrimSpace(first) != "" && strings.TrimSpace(last) != ""
	hasOrg := strings.TrimSpace(org) != ""
	if !hasName && !hasOrg {
		return nil, fmt.Errorf("either organisation_name or both first_name and last_name are required (set via flags or --body)")
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
