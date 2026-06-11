package commands

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/browser"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"golang.org/x/term"
)

// OutputLink is an http(s) URL found in a run's outputs.
type OutputLink struct {
	Key string
	URL string
}

// primaryOutputKeys are listed first when present, in this order.
var primaryOutputKeys = []string{"html_url", "repository_url", "url"}

// extractOutputLinks collects http(s) URLs from run outputs, recursing into
// nested maps. Primary keys come first, the rest alphabetically by key,
// deduplicated by URL.
func extractOutputLinks(outputs map[string]any) []OutputLink {
	var collected []OutputLink
	collectLinks(outputs, &collected)

	sort.SliceStable(collected, func(i, j int) bool {
		pi, pj := primaryRank(collected[i].Key), primaryRank(collected[j].Key)
		if pi != pj {
			return pi < pj
		}
		return collected[i].Key < collected[j].Key
	})

	seen := map[string]bool{}
	links := collected[:0]
	for _, l := range collected {
		if seen[l.URL] {
			continue
		}
		seen[l.URL] = true
		links = append(links, l)
	}
	return links
}

func collectLinks(m map[string]any, out *[]OutputLink) {
	for key, val := range m {
		switch v := val.(type) {
		case string:
			if isHTTPURL(v) {
				*out = append(*out, OutputLink{Key: key, URL: v})
			}
		case map[string]any:
			collectLinks(v, out)
		}
	}
}

func primaryRank(key string) int {
	for i, p := range primaryOutputKeys {
		if key == p {
			return i
		}
	}
	return len(primaryOutputKeys)
}

func isHTTPURL(s string) bool {
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && u.Host != ""
}

// runResultSections builds the detail sections for a finished run,
// including its outputs and any links found in them.
func runResultSections(run *api.ForgeRun) []tui.DetailSection {
	sections := []tui.DetailSection{
		{
			Fields: []tui.Field{
				{Label: "Run ID", Value: run.ID},
				{Label: "Mold", Value: run.MoldSlug},
				{Label: "Action", Value: run.Action},
				{Label: "Status", Value: tui.StatusColor(run.Status).Render(run.Status)},
			},
		},
	}
	if run.Error != "" {
		sections[0].Fields = append(sections[0].Fields,
			tui.Field{Label: "Error", Value: tui.ErrorStyle.Render(run.Error)})
	}

	links := extractOutputLinks(run.Outputs)
	if len(links) > 0 {
		fields := make([]tui.Field, len(links))
		for i, l := range links {
			fields[i] = tui.Field{Label: l.Key, Value: l.URL}
		}
		sections = append(sections, tui.DetailSection{
			Title:  "Links",
			Fields: fields,
		})
	}

	var scalarFields []tui.Field
	for _, key := range sortedOutputKeys(run.Outputs) {
		val := run.Outputs[key]
		if s, ok := val.(string); ok && isHTTPURL(s) {
			continue
		}
		switch val.(type) {
		case string, bool, float64, int, int64:
			scalarFields = append(scalarFields, tui.Field{Label: key, Value: fmt.Sprintf("%v", val)})
		}
	}
	if len(scalarFields) > 0 {
		sections = append(sections, tui.DetailSection{
			Title:  "Outputs",
			Fields: scalarFields,
		})
	}

	return sections
}

func sortedOutputKeys(outputs map[string]any) []string {
	keys := make([]string, 0, len(outputs))
	for k := range outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// offerOpenLinks opens the primary link (--open) or asks the user when the
// session is interactive. Non-TTY sessions only get the printed links.
func offerOpenLinks(links []OutputLink, openFlag bool) {
	if len(links) == 0 {
		return
	}

	if openFlag {
		openLink(links[0].URL)
		return
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	ok, err := tui.Confirm(fmt.Sprintf("Open %s in your browser?", links[0].URL), false, os.Stdin)
	if err != nil || !ok {
		return
	}
	openLink(links[0].URL)
}

func openLink(url string) {
	if err := browser.Open(url); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Opened %s\n", url)
}
