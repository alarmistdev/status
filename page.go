package status

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"runtime/debug"
	"sort"
)

var (
	//go:embed page.tmpl
	defaultTemplateContent string
)

// Page represents a status page that can be served via HTTP.
type Page struct {
	title       string
	tmpl        *template.Template
	hc          *HealthChecker
	links       []Link
	showVersion bool
}

// PageOption is a function that configures a Page.
type PageOption func(*Page)

// WithTitle sets the title of the status page.
func WithTitle(title string) PageOption {
	return func(p *Page) {
		p.title = title
	}
}

// WithTemplate sets a custom HTML template for the status page.
func WithTemplate(tmpl *template.Template) PageOption {
	return func(p *Page) {
		p.tmpl = tmpl
	}
}

// WithHealthChecker sets the health checker for the status page.
func WithHealthChecker(hc *HealthChecker) PageOption {
	return func(p *Page) {
		p.hc = hc
	}
}

// WithLink adds a navigation link to the status page.
func WithLink(name, url string) PageOption {
	return func(p *Page) {
		p.links = append(p.links, Link{
			Name: name,
			URL:  url,
		})
	}
}

// WithVersion configures whether to show version information on the status page.
func WithVersion(show bool) PageOption {
	return func(p *Page) {
		p.showVersion = show
	}
}

// NewPage creates a new status page with the given options.
func NewPage(opts ...PageOption) *Page {
	p := &Page{
		title:       "System Status",
		tmpl:        parseDefaultTemplate(),
		showVersion: true,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Link represents a navigation link in the status page.
type Link struct {
	Name string
	URL  string
}

// HealthGroup represents a group of health check results.
type HealthGroup struct {
	Name    string
	Results []HealthCheckResult
}

// PageData contains the data that will be rendered in the status page template.
type PageData struct {
	Title         string
	Version       string
	HealthResults []HealthCheckResult
	HealthGroups  []HealthGroup
	Links         []Link
}

// Handler returns an HTTP handler that serves the status page.
func (p *Page) Handler() http.HandlerFunc {
	version := retrieveVersion()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var healthResults []HealthCheckResult
		if p.hc != nil {
			var err error
			healthResults, err = p.hc.Check(r.Context())
			if err != nil {
				http.Error(w, fmt.Sprintf("Error checking health: %v", err), http.StatusInternalServerError)

				return
			}
		}

		groups := groupHealthResults(healthResults)
		ungrouped := getUngroupedResults(healthResults)

		data := PageData{
			Title:         p.title,
			HealthResults: ungrouped,
			HealthGroups:  groups,
			Links:         p.links,
		}

		if p.showVersion {
			data.Version = version
		}

		w.Header().Set("Content-Type", "text/html")
		if err := p.tmpl.Execute(w, data); err != nil {
			http.Error(w, fmt.Sprintf("Error executing template :%v", err), http.StatusInternalServerError)
		}
	})
}

func retrieveVersion() string {
	var version = "unknown"

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}

	return info.Main.Version
}

func parseDefaultTemplate() *template.Template {
	return template.Must(template.New("page").Parse(defaultTemplateContent))
}

// groupHealthResults groups health check results by their Group field.
func groupHealthResults(results []HealthCheckResult) []HealthGroup {
	groupMap := make(map[string][]HealthCheckResult)

	for _, result := range results {
		if result.Target.Group != "" {
			groupMap[result.Target.Group] = append(groupMap[result.Target.Group], result)
		}
	}

	groups := make([]HealthGroup, 0, len(groupMap))
	for name, results := range groupMap {
		groups = append(groups, HealthGroup{
			Name:    name,
			Results: results,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})

	return groups
}

// getUngroupedResults returns health check results that don't have a group assigned.
func getUngroupedResults(results []HealthCheckResult) []HealthCheckResult {
	ungrouped := make([]HealthCheckResult, 0)
	for _, result := range results {
		if result.Target.Group == "" {
			ungrouped = append(ungrouped, result)
		}
	}

	return ungrouped
}
