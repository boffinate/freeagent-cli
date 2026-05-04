package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/anjor/freeagent-cli/internal/freeagent"
	"github.com/urfave/cli/v2"
)

// listAll auto-paginates a /v2 list endpoint, walking RFC 5988 Link rel="next"
// until exhausted (or opts.MaxPages is reached). When the first response has
// no rel="next" the upstream bytes are returned verbatim so single-page
// responses preserve any side fields (meta, etc.) beyond the wrapper key.
// Multi-page responses are merged into {"<wrapper>": [...]}; side fields
// from page 2..N are dropped because they don't compose meaningfully across
// pages.
//
// opts.Page or opts.NoFollow short-circuit to a single upstream request.
//
// Default per_page is 100 (the FreeAgent API cap for most endpoints); some
// endpoints — e.g. /clients with minimal_data — allow higher values, which
// callers can pass via --per-page and the server enforces its own ceiling.
func listAll(ctx context.Context, client *freeagent.Client, basePath string, params map[string]string, wrapper string, opts paginateOpts) ([]byte, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = defaultMaxPages
	}

	merged := make(map[string]string, len(params)+2)
	maps.Copy(merged, params)
	merged["per_page"] = strconv.Itoa(perPage)
	if opts.Page > 0 {
		merged["page"] = strconv.Itoa(opts.Page)
	}
	firstURL := appendQuery(basePath, buildQueryParams(merged))

	resp, _, headers, err := client.Do(ctx, http.MethodGet, firstURL, nil, "")
	if err != nil {
		return resp, err
	}
	nextURL := nextLinkURL(headers)
	if opts.Page > 0 || opts.NoFollow || nextURL == "" {
		return resp, nil
	}

	if wrapper == "" {
		return nil, fmt.Errorf("listAll: wrapper required for multi-page response from %s", basePath)
	}
	var first map[string]any
	if err := json.Unmarshal(resp, &first); err != nil {
		return nil, fmt.Errorf("listAll: decode page 1: %w", err)
	}
	items, _ := first[wrapper].([]any)
	pages := 1

	for nextURL != "" {
		if pages >= maxPages {
			fmt.Fprintf(os.Stderr,
				"warning: %s: reached --max-pages=%d before exhausting results; output is truncated. Narrow filters or pass --max-pages.\n",
				basePath, maxPages)
			break
		}
		resp, _, headers, err := client.Do(ctx, http.MethodGet, nextURL, nil, "")
		if err != nil {
			return nil, err
		}
		pages++
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return nil, fmt.Errorf("listAll: decode page %d: %w", pages, err)
		}
		page, _ := decoded[wrapper].([]any)
		items = append(items, page...)
		nextURL = nextLinkURL(headers)
	}

	return json.Marshal(map[string]any{wrapper: items})
}

type paginateOpts struct {
	PerPage  int
	Page     int
	MaxPages int
	NoFollow bool
}

const (
	defaultPerPage  = 100
	defaultMaxPages = 50
)

var nextLinkRE = regexp.MustCompile(`<([^>]+)>\s*;\s*rel="next"`)

// nextLinkURL extracts the rel="next" target from an RFC 5988 Link header.
// Returns "" when the header is absent or has no next link.
func nextLinkURL(h http.Header) string {
	link := h.Get("Link")
	if link == "" {
		return ""
	}
	m := nextLinkRE.FindStringSubmatch(link)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// paginationFlags is the standard flag set added to every list command. The
// CLI auto-paginates by default; these flags exist only as opt-outs.
func paginationFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{Name: "per-page", Usage: "Items per page when paginating (default 100, server cap)"},
		&cli.IntFlag{Name: "page", Usage: "Fetch a single page instead of auto-paginating"},
		&cli.IntFlag{Name: "max-pages", Usage: "Cap on auto-pagination (default 50)"},
		&cli.BoolFlag{Name: "no-paginate", Usage: "Disable auto-pagination (return only the first page)"},
	}
}

// withPagination returns the command-specific flags concatenated with the
// standard pagination flag set. Use it for every list command's Flags field.
func withPagination(extra ...cli.Flag) []cli.Flag {
	return append(extra, paginationFlags()...)
}

// paginationOptsFrom reads the flags installed by paginationFlags from a
// urfave/cli context.
func paginationOptsFrom(c *cli.Context) paginateOpts {
	return paginateOpts{
		PerPage:  c.Int("per-page"),
		Page:     c.Int("page"),
		MaxPages: c.Int("max-pages"),
		NoFollow: c.Bool("no-paginate"),
	}
}
