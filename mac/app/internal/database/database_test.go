package database

import (
	"encoding/json"
	"testing"
)

func TestEmbeddedBlockerFlowListSizes(t *testing.T) {
	cases := []struct {
		cat      string
		min, max int
	}{
		{"pornography", 800_000, 3_000_000},
		{"proxy-avoidance", 15_000, 100_000},
	}
	for _, c := range cases {
		n := DB.CategoryCount(c.cat)
		if n < c.min || n > c.max {
			t.Errorf("CategoryCount(%q) = %d, want between %d and %d", c.cat, n, c.min, c.max)
		}
	}
}

func TestDomainsJSONIsCategoryKeyed(t *testing.T) {
	data, err := fs.ReadFile("domains.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["pornography"]; !ok {
		t.Error("domains.json has no \"pornography\" key")
	}
	for k := range raw {
		if len(k) == 1 {
			t.Errorf("domains.json has single-letter key %q (old letter-keyed format)", k)
		}
	}
}

func TestEmbeddedKeywordsKeepCommonTerms(t *testing.T) {
	if n := DB.KeywordCount(); n < 10_000 {
		t.Errorf("KeywordCount() = %d, want >= 10000", n)
	}
	if !DB.BlocksKeyword("http://example.com/porn") {
		t.Error("BlocksKeyword(\"http://example.com/porn\") = false, want true")
	}
}

// BlockerFlow ships "www.b-cdn.net" (one site) but not "b-cdn.net" (every BunnyCDN pull zone). K9
// keeps pornography and proxy-avoidance entries exactly as shipped, so the www. prefix is not
// stripped, which would widen the entry to the whole parent domain and its subdomains.
func TestPornographyKeepsWwwEntriesAsShipped(t *testing.T) {
	if DB.BlocksDomainInCategories("x.b-cdn.net", []string{"pornography"}) {
		t.Error("x.b-cdn.net is blocked in pornography: the www. entry was widened to b-cdn.net")
	}
	if !DB.BlocksDomainInCategories("www.b-cdn.net", []string{"pornography"}) {
		t.Error("www.b-cdn.net is not blocked in pornography, want blocked")
	}
}
