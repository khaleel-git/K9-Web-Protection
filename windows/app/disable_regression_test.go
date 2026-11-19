//go:build windows

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"k10webprotection/internal/config"
	"k10webprotection/internal/hosts"
)

func TestDisableCleanupFailureKeepsProtectionRetryable(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"elevation declined", hosts.ErrElevationCancelled},
		{"permission denied", os.ErrPermission},
		{"verification failed", hosts.ErrVerifyFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			a, h := newTestApp(t, path)
			if _, err := a.AddBlockRule("google.com", false); err != nil {
				t.Fatal(err)
			}
			requested := time.Now().Add(-2 * time.Hour)
			a.conf().Update(func(c *config.Config) {
				c.DisableDelayHours = 1
				c.DisableRequestedAt = &requested
			})
			if err := a.conf().Save(); err != nil {
				t.Fatal(err)
			}
			systemProxyOn := true
			a.sysProxy = func(on bool) {
				if !on && h.active() {
					t.Error("system proxy disabled before hosts cleanup completed")
				}
				systemProxyOn = on
			}

			h.err = tc.err
			if err := a.DisableProtection(testPassword); err == nil {
				t.Fatal("failed cleanup must fail Disable")
			}
			s := a.GetStatus()
			if !s.ProxyRunning || !s.Layer1Active || !systemProxyOn {
				t.Fatalf("failed cleanup must keep protection active: %+v, system proxy=%v", s, systemProxyOn)
			}
			if s.Diagnostics.LastApply.HostsError == "" || s.Diagnostics.LastApply.HostsInfo != "" {
				t.Fatalf("cleanup error must be visible without claiming protection is off: %+v", s.Diagnostics.LastApply)
			}
			if got := a.conf().DisableRequestedAt; got == nil || !got.Equal(requested) {
				t.Fatal("failed cleanup consumed the pending disable request")
			}
			if h.names()["google.com"] != "0.0.0.0" {
				t.Fatal("fixture must retain the block after the failed write")
			}

			// A second Disable is possible without enabling protection again or
			// waiting through the configured delay for a second time.
			h.err = nil
			if err := a.DisableProtection(testPassword); err != nil {
				t.Fatal(err)
			}
			s = a.GetStatus()
			if s.ProxyRunning || s.Layer1Active || systemProxyOn || h.active() {
				t.Fatalf("successful retry must disable both layers: %+v", s)
			}
			if s.Diagnostics.LastApply.HostsError != "" {
				t.Fatal("successful retry retained the previous cleanup error")
			}
			if a.conf().DisableRequestedAt != nil {
				t.Fatal("successful disable did not consume the request")
			}
			if !reflect.DeepEqual(h.names(), map[string]string{"localhost": "127.0.0.1"}) {
				t.Fatalf("cleanup changed unrelated hosts entries: %v", h.names())
			}
			loaded, err := config.LoadWith(config.LoadOptions{Path: path, LegacyPaths: []string{}})
			if err != nil {
				t.Fatal(err)
			}
			if loaded.DisableRequestedAt != nil || len(loaded.GetBlockRules()) != 1 {
				t.Fatal("disable must save the consumed request without deleting the configured rules")
			}
		})
	}
}

func TestDisableGuardsRunBeforeHostsCleanup(t *testing.T) {
	for _, name := range []string{"password", "focus", "delay"} {
		t.Run(name, func(t *testing.T) {
			a, h := newTestApp(t, filepath.Join(t.TempDir(), "config.json"))
			password := testPassword
			switch name {
			case "password":
				password = "incorrect"
			case "focus":
				a.conf().SetFocusMode(5)
			case "delay":
				now := time.Now()
				a.conf().Update(func(c *config.Config) {
					c.DisableDelayHours = 1
					c.DisableRequestedAt = &now
				})
			}
			calls := h.calls
			a.sysProxy = func(bool) { t.Error("rejected Disable changed the system proxy") }
			if err := a.DisableProtection(password); err == nil {
				t.Fatal("guard must reject Disable")
			}
			if h.calls != calls || !a.protectionOn() {
				t.Fatal("rejected Disable changed enforcement")
			}
		})
	}
}

func TestRemoveGmailBlockKeepsAllowAndOtherRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	a, h := newTestApp(t, path)
	for _, domain := range []string{"gmail.com", "google.com"} {
		if _, err := a.AddBlockRule(domain, false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.AddAllowRule(testPassword, "gmail.com", false); err != nil {
		t.Fatal(err)
	}
	allow := a.GetRules().Allow
	v, err := a.RemoveBlockRule(testPassword, "gmail.com")
	if err != nil || v.Apply.ConfigError != "" || v.Apply.HostsError != "" {
		t.Fatalf("remove Gmail: %+v, %v", v, err)
	}
	want := []config.DomainRule{{Domain: "google.com"}}
	if !reflect.DeepEqual(v.Block, want) || !reflect.DeepEqual(v.Allow, allow) {
		t.Fatalf("removal must change only the requested block: %+v", v)
	}
	if h.names()["gmail.com"] != "" || h.names()["google.com"] != "0.0.0.0" {
		t.Fatalf("hosts not synchronized after removal: %v", h.names())
	}
	loaded, err := config.LoadWith(config.LoadOptions{Path: path, LegacyPaths: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.GetBlockRules(), want) || !reflect.DeepEqual(loaded.GetAllowRules(), allow) {
		t.Fatal("Gmail removal or the unchanged allowlist did not persist")
	}
	// Prove that Gmail is really removed, rather than merely masked by Allow.
	if _, err := a.RemoveAllowRule("gmail.com"); err != nil {
		t.Fatal(err)
	}
	if a.proxy.HostBlocked("gmail.com") {
		t.Fatal("Gmail block reappeared after removing the temporary allow")
	}
}
