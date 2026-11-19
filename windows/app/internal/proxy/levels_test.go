package proxy

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
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

// sampleDomain returns the first domain of cat that no built-in rule touches
// (allow-lists, communication apps, web-proxy detection, direct IPs) and that
// is not in any category of avoid. It returns "" if there is none.
func sampleDomain(t *testing.T, cat string, avoid []string) string {
	t.Helper()
	all, err := loadDomains()
	if err != nil {
		t.Fatalf("load domains.json: %v", err)
	}
	for _, d := range all[cat] {
		if IsBuiltinAllowed(d) || IsCommunicationAllowed(d) || isWebProxy(d, "https://"+d+"/") || isDirectIP(d) {
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
	r := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: host + ":443"},
		Host:   host + ":443",
	}
	return p.decide(&Policy{FilterLevel: level}, host, r)
}

func TestNamedLevelsBlockEachOfTheirCategories(t *testing.T) {
	p := New(0, nil)
	for _, level := range []string{LevelHigh, LevelDefault, LevelModerate, LevelMinimal} {
		for _, cat := range LevelCategories[level] {
			if cat == "unrated" { // has no domains today
				continue
			}
			d := sampleDomain(t, cat, nil)
			if d == "" {
				t.Errorf("level %s: no usable sample domain in category %q", level, cat)
				continue
			}
			if got := connectDecision(p, level, d); got != verdictBlock {
				t.Errorf("level %s should block %s (category %s), got verdict %d", level, d, cat, got)
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
		t.Errorf("minimal should pass %s, got verdict %d", d, got)
	}
	if got := connectDecision(p, LevelModerate, d); got != verdictBlock {
		t.Errorf("moderate should block %s, got verdict %d", d, got)
	}
}

func TestMonitorBlocksNoDatabaseDomain(t *testing.T) {
	p := New(0, nil)
	d := sampleDomain(t, "pornography", nil)
	if d == "" {
		t.Fatal("no usable pornography sample domain")
	}
	if got := connectDecision(p, LevelMonitor, d); got != verdictPass {
		t.Errorf("monitor should pass %s, got verdict %d", d, got)
	}
}
