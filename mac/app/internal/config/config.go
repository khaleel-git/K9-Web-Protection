package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Stats struct {
	TotalBlocked int            `json:"totalBlocked"`
	BlockedToday int            `json:"blockedToday"`
	LastReset    time.Time      `json:"lastReset"`
	TopBlocked   []BlockedEntry `json:"topBlocked"`
}

type BlockedEntry struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type Config struct {
	mu   sync.RWMutex `json:"-"`
	path string       `json:"-"`

	// User-added lists
	UserBlocklist []string `json:"userBlocklist"`
	UserAllowlist []string `json:"userAllowlist"`
	UserKeywords  []string `json:"userKeywords"`

	// Content blocking toggles
	BlockAdultContent bool `json:"blockAdultContent"`
	BlockImageSearch  bool `json:"blockImageSearch"`
	BlockYouTube      bool `json:"blockYouTube"`
	SafeSearch        bool `json:"safeSearch"`

	// Advanced blocking
	DisableDelayHours  int        `json:"disableDelayHours"` // 0 = off
	DisableRequestedAt *time.Time `json:"disableRequestedAt,omitempty"`
	BlockedMessage     string     `json:"blockedMessage"`

	// Focus Mode
	FocusModeUntil time.Time `json:"focusModeUntil"`

	// System
	PasswordHash string `json:"passwordHash"`
	ProxyPort    int    `json:"proxyPort"`
	AutoStart    bool   `json:"autoStart"`
	Stats        Stats  `json:"stats"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".k9webprotection")
}

func Load() *Config {
	dir := configDir()
	os.MkdirAll(dir, 0700)
	path := filepath.Join(dir, "config.json")

	c := &Config{
		path:              path,
		ProxyPort:         8080,
		AutoStart:         true,
		BlockAdultContent: true,
		BlockImageSearch:  false,
		BlockYouTube:      false,
		SafeSearch:        true,
		BlockedMessage:    "This website has been blocked to help you stay focused and protected.",
		PasswordHash:      "$2a$10$N47D4hTSf6Ftc78KPruW1eSLFRO2rw9UBhA9So.arPPPAV..Qijg2",
		Stats:             Stats{LastReset: time.Now()},
	}

	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, c)
		c.path = path
	}
	return c
}

func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0600)
}

// ── Focus Mode ────────────────────────────────────────────────────────────────

func (c *Config) InFocusMode() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Now().Before(c.FocusModeUntil)
}

func (c *Config) FocusModeRemaining() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Now().Before(c.FocusModeUntil) {
		return time.Until(c.FocusModeUntil)
	}
	return 0
}

func (c *Config) SetFocusMode(minutes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.FocusModeUntil = time.Now().Add(time.Duration(minutes) * time.Minute)
}

// ── Disable delay ─────────────────────────────────────────────────────────────

// DisableAllowed returns true if the user is allowed to disable protection now.
// If DisableDelayHours > 0, a request must have been placed that long ago.
func (c *Config) DisableAllowed() (allowed bool, remaining time.Duration) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Now().Before(c.FocusModeUntil) {
		return false, time.Until(c.FocusModeUntil)
	}
	if c.DisableDelayHours <= 0 || c.DisableRequestedAt == nil {
		return true, 0
	}
	readyAt := c.DisableRequestedAt.Add(time.Duration(c.DisableDelayHours) * time.Hour)
	if time.Now().After(readyAt) {
		return true, 0
	}
	return false, time.Until(readyAt)
}

func (c *Config) RequestDisable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.DisableRequestedAt = &now
}

func (c *Config) ClearDisableRequest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DisableRequestedAt = nil
}

// ── User block list ───────────────────────────────────────────────────────────

func (c *Config) GetUserBlocklist() []string {
	c.mu.RLock(); defer c.mu.RUnlock()
	out := make([]string, len(c.UserBlocklist))
	copy(out, c.UserBlocklist); return out
}
func (c *Config) AddUserBlocklist(domain string) {
	c.mu.Lock(); defer c.mu.Unlock()
	for _, d := range c.UserBlocklist { if d == domain { return } }
	c.UserBlocklist = append(c.UserBlocklist, domain)
}
func (c *Config) RemoveUserBlocklist(domain string) {
	c.mu.Lock(); defer c.mu.Unlock()
	out := c.UserBlocklist[:0]
	for _, d := range c.UserBlocklist { if d != domain { out = append(out, d) } }
	c.UserBlocklist = out
}

// ── User allow list ───────────────────────────────────────────────────────────

func (c *Config) GetUserAllowlist() []string {
	c.mu.RLock(); defer c.mu.RUnlock()
	out := make([]string, len(c.UserAllowlist))
	copy(out, c.UserAllowlist); return out
}
func (c *Config) AddUserAllowlist(domain string) {
	c.mu.Lock(); defer c.mu.Unlock()
	for _, d := range c.UserAllowlist { if d == domain { return } }
	c.UserAllowlist = append(c.UserAllowlist, domain)
}
func (c *Config) RemoveUserAllowlist(domain string) {
	c.mu.Lock(); defer c.mu.Unlock()
	out := c.UserAllowlist[:0]
	for _, d := range c.UserAllowlist { if d != domain { out = append(out, d) } }
	c.UserAllowlist = out
}

// ── User keywords ─────────────────────────────────────────────────────────────

func (c *Config) GetUserKeywords() []string {
	c.mu.RLock(); defer c.mu.RUnlock()
	out := make([]string, len(c.UserKeywords))
	copy(out, c.UserKeywords); return out
}
func (c *Config) AddUserKeyword(kw string) {
	c.mu.Lock(); defer c.mu.Unlock()
	for _, k := range c.UserKeywords { if k == kw { return } }
	c.UserKeywords = append(c.UserKeywords, kw)
}
func (c *Config) RemoveUserKeyword(kw string) {
	c.mu.Lock(); defer c.mu.Unlock()
	out := c.UserKeywords[:0]
	for _, k := range c.UserKeywords { if k != kw { out = append(out, k) } }
	c.UserKeywords = out
}

// ── Proxy helpers ─────────────────────────────────────────────────────────────

func (c *Config) IsAllowed(host string) bool {
	c.mu.RLock(); defer c.mu.RUnlock()
	for _, d := range c.UserAllowlist { if d == host { return true } }
	return false
}
func (c *Config) UserBlocks(host string) bool {
	c.mu.RLock(); defer c.mu.RUnlock()
	for _, d := range c.UserBlocklist {
		if d == host || hasSuffix(host, "."+d) { return true }
	}
	return false
}
func (c *Config) UserKeywordMatch(url string) bool {
	c.mu.RLock(); defer c.mu.RUnlock()
	u := toLower(url)
	for _, kw := range c.UserKeywords {
		if contains(u, toLower(kw)) { return true }
	}
	return false
}

// ── Stats ─────────────────────────────────────────────────────────────────────

func (c *Config) IncrementBlocked(domain string) {
	c.mu.Lock(); defer c.mu.Unlock()
	c.Stats.TotalBlocked++
	now := time.Now()
	if now.Year() != c.Stats.LastReset.Year() || now.YearDay() != c.Stats.LastReset.YearDay() {
		c.Stats.BlockedToday = 0; c.Stats.LastReset = now
	}
	c.Stats.BlockedToday++
	for i, e := range c.Stats.TopBlocked {
		if e.Domain == domain { c.Stats.TopBlocked[i].Count++; return }
	}
	c.Stats.TopBlocked = append(c.Stats.TopBlocked, BlockedEntry{Domain: domain, Count: 1})
	if len(c.Stats.TopBlocked) > 10 { c.Stats.TopBlocked = c.Stats.TopBlocked[:10] }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
func toLower(s string) string {
	b := []byte(s)
	for i, c := range b { if c >= 'A' && c <= 'Z' { b[i] = c + 32 } }
	return string(b)
}
func contains(s, sub string) bool {
	if len(sub) == 0 { return true }
	if len(s) < len(sub) { return false }
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}
