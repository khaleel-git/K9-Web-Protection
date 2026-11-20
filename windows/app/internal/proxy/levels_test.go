package proxy

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"k10webprotection/internal/database"
)

var (
	domainsOnce  sync.Once
	domainsByCat map[string][]string
	domainsErr   error
)

func loadDomains() (map[string][]string, error) {
	domainsOnce.Do(func() {
		data, err := os.ReadFile("../database/domains.json")
		if err != nil {
			domainsErr = err
			return
		}
		domainsErr = json.Unmarshal(data, &domainsByCat)
	})
	return domainsByCat, domainsErr
}

func verdictName(v verdict) string {
	switch v {
	case verdictPass:
		return "pass"
	case verdictBlock:
		return "block"
	case verdictSafeSearch:
		return "safesearch"
	}
	return "unknown"
}

// sampleDomain returns the first plain-hostname domain of cat that no built-in
// rule touches (allow-lists, communication apps, web-proxy detection, direct
// IPs) and that is not in any category of avoid. URL-style entries such as
// "bing.com/images" are skipped. It returns "" if there is none.
func sampleDomain(t *testing.T, cat string, avoid []string) string {
	t.Helper()
	all, err := loadDomains()
	if err != nil {
		t.Fatalf("load domains.json: %v", err)
	}
	for _, d := range all[cat] {
		if d == "" || strings.ContainsAny(d, "/:? \t\r\n") {
			continue
		}
		// decide sees r.URL.String() for a CONNECT request. Skip hosts that decide
		// blocks by its own rules (web proxy, YouTube, image search) before the
		// database lookup, or the check would not exercise the database.
		raw := connectRequest(d + ":443").URL.String()
		if IsBuiltinAllowed(d) || IsCommunicationAllowed(d) || isDirectIP(d) ||
			isWebProxy(d, raw) || isYouTube(d) || isImageSearch(d, raw) {
			continue
		}
		if len(avoid) > 0 && database.DB.BlocksDomainInCategories(d, avoid) {
			continue
		}
		return d
	}
	return ""
}

func connectDecision(p *Proxy, level, host string) verdict {
	return p.decide(&Policy{FilterLevel: level}, host, connectRequest(host+":443"))
}

func TestNamedLevelsBlockEachOfTheirCategories(t *testing.T) {
	p := New(0, nil)
	for _, level := range []string{LevelHigh, LevelDefault, LevelModerate, LevelMinimal} {
		for _, cat := range LevelCategories[level] {
			if cat == "unrated" { // has no domains today
				continue
			}
			var others []string
			for _, c := range LevelCategories[level] {
				if c != cat {
					others = append(others, c)
				}
			}
			d := sampleDomain(t, cat, others)
			if d == "" {
				d = sampleDomain(t, cat, nil)
				if d == "" {
					t.Errorf("level %s: no usable sample domain in category %q", level, cat)
					continue
				}
				t.Logf("level %s: category %q check is not isolated (no exclusive sample); using %s", level, cat, d)
			}
			if got := connectDecision(p, level, d); got != verdictBlock {
				t.Errorf("level %s should block %s (category %s), got %v", level, d, cat, verdictName(got))
			}
		}
	}
}

func TestMinimalLeavesGamblingAlone(t *testing.T) {
	p := New(0, nil)
	d := sampleDomain(t, "gambling", LevelCategories[LevelMinimal])
	if d == "" {
		t.Fatal("no usable gambling sample domain outside the minimal categories")
	}
	if got := connectDecision(p, LevelMinimal, d); got != verdictPass {
		t.Errorf("minimal should pass %s, got %v", d, verdictName(got))
	}
	if got := connectDecision(p, LevelModerate, d); got != verdictBlock {
		t.Errorf("moderate should block %s, got %v", d, verdictName(got))
	}
}

func TestMonitorBlocksNoDatabaseDomain(t *testing.T) {
	p := New(0, nil)
	d := sampleDomain(t, "pornography", nil)
	if d == "" {
		t.Fatal("no usable pornography sample domain")
	}
	if got := connectDecision(p, LevelDefault, d); got != verdictBlock {
		t.Fatalf("control: default should block %s, got %v", d, verdictName(got))
	}
	if got := connectDecision(p, LevelMonitor, d); got != verdictPass {
		t.Errorf("monitor should pass %s, got %v", d, verdictName(got))
	}
}
