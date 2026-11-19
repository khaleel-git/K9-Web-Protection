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
