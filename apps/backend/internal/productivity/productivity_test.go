package productivity

import "testing"

func TestNormalizeDomain(t *testing.T) {
	cases := map[string]string{
		"https://www.GitHub.com/org/repo?q=secret#x": "github.com",
		"https://example.org?search=secret":          "example.org",
		"github.com/x":                               "github.com",
		"http://user:pw@intranet.local:8080/a":       "intranet.local",
		"docs.google.com.":                           "docs.google.com",
		"about:blank":                                "",
		"":                                           "",
		"   ":                                        "",
	}
	for in, want := range cases {
		if got := NormalizeDomain(in); got != want {
			t.Errorf("NormalizeDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsBrowserApp(t *testing.T) {
	for _, n := range []string{"Google Chrome", "msedge", "Firefox", "Brave Browser", "Arc", "Safari"} {
		if !IsBrowserApp(n) {
			t.Errorf("%q should be a browser", n)
		}
	}
	for _, n := range []string{"Code", "Slack", "Archicad", "Search"} {
		if IsBrowserApp(n) {
			t.Errorf("%q should not be a browser", n)
		}
	}
}

func TestMatcherDomainMostSpecificWins(t *testing.T) {
	m := NewMatcher([]Rule{
		{MatchType: MatchDomain, Pattern: "google.com", Category: Neutral},
		{MatchType: MatchDomain, Pattern: "docs.google.com", Category: Productive},
		{MatchType: MatchApp, Pattern: "slack", Category: Productive},
	})
	if c := m.Domain("docs.google.com"); c != Productive {
		t.Errorf("docs.google.com = %s", c)
	}
	if c := m.Domain("mail.google.com"); c != Neutral {
		t.Errorf("mail.google.com = %s", c)
	}
	if c := m.Domain("notgoogle.com"); c != Unclassified {
		t.Errorf("notgoogle.com = %s (suffix match must respect label boundary)", c)
	}
	if c := m.App("Slack"); c != Productive {
		t.Errorf("Slack = %s (app match is case-insensitive)", c)
	}
}

func TestClassifySplitsBrowserTimeWithoutDoubleCounting(t *testing.T) {
	m := NewMatcher([]Rule{
		{MatchType: MatchApp, Pattern: "code", Category: Productive},
		{MatchType: MatchDomain, Pattern: "github.com", Category: Productive},
		{MatchType: MatchDomain, Pattern: "youtube.com", Category: Unproductive},
	})
	apps := []Usage{{"Code", 3600}, {"Google Chrome", 1200}, {"Spotify", 600}}
	// Extension time (1500s) exceeds browser app time (1200s): it is scaled,
	// not added. 600/1500 github, 600/1500 youtube, 300/1500 unknown.
	domains := []Usage{{"github.com", 600}, {"youtube.com", 600}, {"news.example", 300}}
	got := Classify(m, apps, domains)
	want := Totals{ActiveS: 5400, ProductiveS: 3600 + 480, UnproductiveS: 480, UnclassifiedS: 600 + 240}
	if got != want {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	if sum := got.ProductiveS + got.UnproductiveS + got.NeutralS + got.UnclassifiedS; sum != got.ActiveS {
		t.Errorf("categories sum %d != active %d", sum, got.ActiveS)
	}
}

func TestClassifyBrowserFallbackRule(t *testing.T) {
	m := NewMatcher([]Rule{{MatchType: MatchApp, Pattern: "firefox", Category: Neutral}})
	// No domain data (extension not installed): the browser's app rule applies.
	got := Classify(m, []Usage{{"Firefox", 100}}, nil)
	if got.NeutralS != 100 || got.UnclassifiedS != 0 {
		t.Errorf("fallback not applied: %+v", got)
	}
}

func TestScores(t *testing.T) {
	tot := Totals{ActiveS: 1000, ProductiveS: 600, UnproductiveS: 100, NeutralS: 100, UnclassifiedS: 200}
	if s := tot.ScorePct(); s == nil || *s != 75 {
		t.Errorf("score = %v", s)
	}
	if c := tot.CoveragePct(); c == nil || *c != 80 {
		t.Errorf("coverage = %v", c)
	}
	if p := tot.TargetPct(300, 2); p == nil || *p != 100 {
		t.Errorf("target = %v", p)
	}
	if (Totals{ActiveS: 50, UnclassifiedS: 50}).ScorePct() != nil {
		t.Error("score must be N/A when nothing is classified")
	}
}
