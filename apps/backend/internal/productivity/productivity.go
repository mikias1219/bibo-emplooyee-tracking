// Package productivity classifies tracked time as productive / unproductive /
// neutral / unclassified from owner-defined rules, and computes the performance
// figures shown in the admin. It is pure (no DB) so the rules are unit-testable.
//
// Definitions (engineering reference §10.1):
//   - Active time comes from the desktop's app intervals (idle already excluded).
//   - Browser time is split by the domains visited that day instead of being
//     added on top, so app time and website time are never double counted.
//   - score = productive / (productive + unproductive + neutral); unclassified
//     time is reported separately and never counted as unproductive.
//   - target = productive / (daily target × days with activity).
package productivity

import (
	"net/url"
	"strings"
)

// Category is a rule's classification of an app or domain.
type Category string

const (
	Productive   Category = "productive"
	Unproductive Category = "unproductive"
	Neutral      Category = "neutral"
	Unclassified Category = "unclassified" // no rule — reported, never stored
)

// ValidRuleCategory reports whether c can be stored on a rule.
func ValidRuleCategory(c Category) bool {
	return c == Productive || c == Unproductive || c == Neutral
}

// MatchType selects what a rule's pattern is compared against.
type MatchType string

const (
	MatchApp    MatchType = "app"
	MatchDomain MatchType = "domain"
)

// Rule is one owner-defined classification.
type Rule struct {
	ID        string    `json:"id"`
	MatchType MatchType `json:"match_type"`
	Pattern   string    `json:"pattern"`
	Category  Category  `json:"category"`
}

// NormalizeApp is the stored/compared form of an app name.
func NormalizeApp(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// NormalizeDomain reduces user input ("https://www.GitHub.com/org?q=1") or a
// stored URL to a bare lowercase host ("github.com"). Returns "" when no host
// can be found (about:blank, file paths, garbage).
func NormalizeDomain(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Bare hosts ("github.com/x") have no scheme; give them one so url.Parse
	// treats the first segment as the host rather than a path.
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	h := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	h = strings.TrimPrefix(h, "www.")
	if h == "" || strings.ContainsAny(h, " /\\") {
		return ""
	}
	return h
}

// browserApps are substrings of the app names browsers report on Windows,
// macOS and Linux. Their active time is split by the domains visited.
var browserApps = []string{
	"chrome", "chromium", "msedge", "microsoft edge", "firefox", "brave",
	"safari", "opera", "vivaldi", "arc", "zen browser", "librewolf", "yandex",
	"epiphany", "gnome web",
}

// IsBrowserApp reports whether an app name looks like a web browser.
func IsBrowserApp(name string) bool {
	n := NormalizeApp(name)
	for _, b := range browserApps {
		if n == b || strings.Contains(n, b) {
			// "arc" is short; require an exact or word match to avoid e.g. "Archicad".
			if b == "arc" && n != "arc" && !strings.HasPrefix(n, "arc ") {
				continue
			}
			return true
		}
	}
	return false
}

// Matcher resolves apps and domains to categories.
type Matcher struct {
	apps    map[string]Category
	domains map[string]Category
}

// NewMatcher builds a matcher from rules (patterns already normalised).
func NewMatcher(rules []Rule) *Matcher {
	m := &Matcher{apps: map[string]Category{}, domains: map[string]Category{}}
	for _, r := range rules {
		switch r.MatchType {
		case MatchApp:
			m.apps[NormalizeApp(r.Pattern)] = r.Category
		case MatchDomain:
			if d := NormalizeDomain(r.Pattern); d != "" {
				m.domains[d] = r.Category
			}
		}
	}
	return m
}

// App returns the category for an app name, or Unclassified.
func (m *Matcher) App(name string) Category {
	if c, ok := m.apps[NormalizeApp(name)]; ok {
		return c
	}
	return Unclassified
}

// Domain returns the category for a host. The most specific rule wins: for
// "docs.google.com" a "docs.google.com" rule beats a "google.com" rule.
func (m *Matcher) Domain(host string) Category {
	h := NormalizeDomain(host)
	for h != "" {
		if c, ok := m.domains[h]; ok {
			return c
		}
		i := strings.IndexByte(h, '.')
		if i < 0 {
			break
		}
		h = h[i+1:]
	}
	return Unclassified
}

// Totals is time per category, in seconds.
type Totals struct {
	ActiveS       int64 `json:"active_s"`
	ProductiveS   int64 `json:"productive_s"`
	UnproductiveS int64 `json:"unproductive_s"`
	NeutralS      int64 `json:"neutral_s"`
	UnclassifiedS int64 `json:"unclassified_s"`
}

func (t *Totals) add(c Category, s int64) {
	switch c {
	case Productive:
		t.ProductiveS += s
	case Unproductive:
		t.UnproductiveS += s
	case Neutral:
		t.NeutralS += s
	default:
		t.UnclassifiedS += s
	}
}

// Add accumulates another total into t.
func (t *Totals) Add(o Totals) {
	t.ActiveS += o.ActiveS
	t.ProductiveS += o.ProductiveS
	t.UnproductiveS += o.UnproductiveS
	t.NeutralS += o.NeutralS
	t.UnclassifiedS += o.UnclassifiedS
}

// ScorePct is productive share of classified time (0–100), nil when nothing
// is classified yet.
func (t Totals) ScorePct() *float64 {
	den := t.ProductiveS + t.UnproductiveS + t.NeutralS
	if den <= 0 {
		return nil
	}
	v := round1(100 * float64(t.ProductiveS) / float64(den))
	return &v
}

// CoveragePct is the classified share of active time (0–100), nil when idle.
func (t Totals) CoveragePct() *float64 {
	if t.ActiveS <= 0 {
		return nil
	}
	v := round1(100 * float64(t.ActiveS-t.UnclassifiedS) / float64(t.ActiveS))
	return &v
}

// TargetPct is productive time as a share of the expected productive time
// for the days worked (can exceed 100), nil when there are no working days.
func (t Totals) TargetPct(targetPerDayS int64, days int) *float64 {
	if targetPerDayS <= 0 || days <= 0 {
		return nil
	}
	v := round1(100 * float64(t.ProductiveS) / float64(targetPerDayS*int64(days)))
	return &v
}

func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}

// Usage is tracked seconds for one app or domain.
type Usage struct {
	Name      string
	DurationS int64
}

// Classify computes one employee-day's totals from its app intervals and
// browser-domain visits. Non-browser apps take their app rule. Browser time
// is split across that day's domain categories in proportion to the domain
// durations; the unclassified share falls back to the browser's own app rule.
// Domain time never adds to active time — only app time does.
func Classify(m *Matcher, apps []Usage, domains []Usage) Totals {
	var t Totals
	var browserS int64
	var browserFallback = Unclassified
	for _, a := range apps {
		t.ActiveS += a.DurationS
		if IsBrowserApp(a.Name) {
			browserS += a.DurationS
			if c := m.App(a.Name); c != Unclassified {
				browserFallback = c
			}
			continue
		}
		t.add(m.App(a.Name), a.DurationS)
	}
	if browserS == 0 {
		return t
	}

	var byCat = map[Category]int64{}
	var domainTotal int64
	for _, d := range domains {
		if d.DurationS <= 0 {
			continue
		}
		byCat[m.Domain(d.Name)] += d.DurationS
		domainTotal += d.DurationS
	}
	if domainTotal == 0 {
		t.add(browserFallback, browserS)
		return t
	}
	var allocated int64
	for _, c := range []Category{Productive, Unproductive, Neutral} {
		s := browserS * byCat[c] / domainTotal
		t.add(c, s)
		allocated += s
	}
	// Remainder (unclassified domains + integer rounding) takes the browser's
	// own rule, or stays unclassified.
	t.add(browserFallback, browserS-allocated)
	return t
}
