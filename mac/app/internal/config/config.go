package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	Domain   string    `json:"domain"`
	Category string    `json:"category"`
	Count    int       `json:"count"`
	LastSeen time.Time `json:"lastSeen"`
}

// DayRestriction defines the allowed internet window for one weekday.
type DayRestriction struct {
	Day     string `json:"day"`     // "monday" … "sunday"
	From    string `json:"from"`    // "HH:MM" 24-hour
	To      string `json:"to"`      // "HH:MM" 24-hour
	Enabled bool   `json:"enabled"` // false = this day is skipped
}

// TimeRestrictions holds the weekly schedule for auto Focus Mode.
type TimeRestrictions struct {
	Enabled bool             `json:"enabled"`
	Days    []DayRestriction `json:"days"`
}

// FocusSite is a social media domain that gets blocked when Focus Mode is on.
type FocusSite struct {
	Domain  string `json:"domain"`
	Active  bool   `json:"active"`  // block this site while focus mode is running
	Builtin bool   `json:"builtin"` // true = shipped by default, cannot be deleted
}

type Config struct {
	mu   sync.RWMutex `json:"-"`
	path string       `json:"-"`

	// User-added lists
	UserBlocklist []string `json:"userBlocklist"`
	UserAllowlist []string `json:"userAllowlist"`
	UserKeywords  []string `json:"userKeywords"`

	// Content blocking toggles
	FilterLevel       string `json:"filterLevel"` // high|default|moderate|minimal|monitor|custom
	BlockAdultContent bool   `json:"blockAdultContent"`
	BlockImageSearch  bool   `json:"blockImageSearch"`
	BlockYouTube      bool   `json:"blockYouTube"`
	SafeSearch        bool   `json:"safeSearch"`

	// Advanced blocking
	DisableDelayHours  int        `json:"disableDelayHours"` // 0 = off
	DisableRequestedAt *time.Time `json:"disableRequestedAt,omitempty"`
	BlockedMessage     string     `json:"blockedMessage"`

	// Focus Mode
	FocusModeUntil   time.Time        `json:"focusModeUntil"`
	FocusSites       []FocusSite      `json:"focusSites"`
	TimeRestrictions TimeRestrictions `json:"timeRestrictions"`

	// System
	PasswordHash string `json:"passwordHash"`
	ProxyPort    int    `json:"proxyPort"`
	AutoStart    bool   `json:"autoStart"`
	Stats        Stats  `json:"stats"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".k10webprotection")
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
	if len(c.FocusSites) == 0 {
		c.FocusSites = defaultFocusSites()
	}
	if len(c.TimeRestrictions.Days) == 0 {
		c.TimeRestrictions.Days = defaultDayRestrictions()
	}
	return c
}

func defaultDayRestrictions() []DayRestriction {
	type entry struct{ day, from, to string; en bool }
	rows := []entry{
		{"monday", "08:00", "22:00", true},
		{"tuesday", "08:00", "22:00", true},
		{"wednesday", "08:00", "22:00", true},
		{"thursday", "08:00", "22:00", true},
		{"friday", "08:00", "22:00", true},
		{"saturday", "10:00", "22:00", false},
		{"sunday", "10:00", "22:00", false},
	}
	out := make([]DayRestriction, len(rows))
	for i, r := range rows {
		out[i] = DayRestriction{Day: r.day, From: r.from, To: r.to, Enabled: r.en}
	}
	return out
}

func defaultFocusSites() []FocusSite {
	domains := []string{
		"facebook.com", "instagram.com", "x.com", "twitter.com",
		"tiktok.com", "snapchat.com", "reddit.com", "pinterest.com",
		"linkedin.com", "youtube.com", "discord.com", "twitch.tv",
		"threads.net", "tumblr.com", "whatsapp.com", "telegram.org",
		"bereal.com",
	}
	out := make([]FocusSite, len(domains))
	for i, d := range domains {
		out[i] = FocusSite{Domain: d, Active: true, Builtin: true}
	}
	return out
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

func (c *Config) StopFocusMode() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.FocusModeUntil = time.Time{}
}

// ── Focus sites ───────────────────────────────────────────────────────────────

func (c *Config) GetFocusSites() []FocusSite {
	c.mu.RLock(); defer c.mu.RUnlock()
	out := make([]FocusSite, len(c.FocusSites))
	copy(out, c.FocusSites)
	return out
}

func (c *Config) SetFocusSiteActive(domain string, active bool) {
	c.mu.Lock(); defer c.mu.Unlock()
	for i := range c.FocusSites {
		if c.FocusSites[i].Domain == domain {
			c.FocusSites[i].Active = active
			return
		}
	}
}

func (c *Config) AddFocusSite(domain string) {
	c.mu.Lock(); defer c.mu.Unlock()
	for _, s := range c.FocusSites {
		if s.Domain == domain { return }
	}
	c.FocusSites = append(c.FocusSites, FocusSite{Domain: domain, Active: true, Builtin: false})
}

func (c *Config) RemoveFocusSite(domain string) {
	c.mu.Lock(); defer c.mu.Unlock()
	out := c.FocusSites[:0]
	for _, s := range c.FocusSites {
		if s.Domain == domain && !s.Builtin { continue }
		out = append(out, s)
	}
	c.FocusSites = out
}

// FocusBlocks returns true when the host should be blocked — either manual
// focus mode is running or the current time falls inside a time restriction.
func (c *Config) FocusBlocks(host string) bool {
	c.mu.RLock(); defer c.mu.RUnlock()
	if !time.Now().Before(c.FocusModeUntil) && !c.inTimeRestriction() {
		return false
	}
	for _, s := range c.FocusSites {
		if !s.Active { continue }
		if host == s.Domain || hasSuffix(host, "."+s.Domain) {
			return true
		}
	}
	return false
}

// InTimeRestriction reports whether the current time falls inside a scheduled
// restriction window (caller-safe — acquires read lock).
func (c *Config) InTimeRestriction() bool {
	c.mu.RLock(); defer c.mu.RUnlock()
	return c.inTimeRestriction()
}

// inTimeRestriction is the lock-free inner check (call with read lock held).
func (c *Config) inTimeRestriction() bool {
	if !c.TimeRestrictions.Enabled { return false }
	wd := [7]string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	now := time.Now()
	weekday := wd[now.Weekday()]
	cur := now.Hour()*60 + now.Minute()
	for _, d := range c.TimeRestrictions.Days {
		if !d.Enabled || d.Day != weekday { continue }
		from := hhmm(d.From)
		to := hhmm(d.To)
		if from < to {
			if cur >= from && cur < to { return true }
		} else if from > to { // overnight window e.g. 22:00 → 06:00
			if cur >= from || cur < to { return true }
		}
	}
	return false
}

func hhmm(s string) int {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 { return 0 }
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}

// ── Time restrictions CRUD ────────────────────────────────────────────────────

func (c *Config) GetTimeRestrictions() TimeRestrictions {
	c.mu.RLock(); defer c.mu.RUnlock()
	tr := TimeRestrictions{Enabled: c.TimeRestrictions.Enabled}
	tr.Days = make([]DayRestriction, len(c.TimeRestrictions.Days))
	copy(tr.Days, c.TimeRestrictions.Days)
	return tr
}

func (c *Config) SaveTimeRestrictions(tr TimeRestrictions) {
	c.mu.Lock(); defer c.mu.Unlock()
	c.TimeRestrictions = tr
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

// dedupWindow: multiple requests from the same page load all arrive within
// a few hundred milliseconds. Skip counting the same domain again if it was
// already counted within this window — one visit = one count.
const dedupWindow = 5 * time.Second

func (c *Config) IncrementBlocked(domain, category string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if now.Year() != c.Stats.LastReset.Year() || now.YearDay() != c.Stats.LastReset.YearDay() {
		c.Stats.BlockedToday = 0
		c.Stats.LastReset = now
	}

	for i, e := range c.Stats.TopBlocked {
		if e.Domain == domain {
			if now.Sub(e.LastSeen) < dedupWindow {
				return // same page-load burst — skip
			}
			c.Stats.TopBlocked[i].Count++
			c.Stats.TopBlocked[i].LastSeen = now
			if category != "" {
				c.Stats.TopBlocked[i].Category = category
			}
			c.Stats.TotalBlocked++
			c.Stats.BlockedToday++
			// Move to front so the list stays newest-first
			entry := c.Stats.TopBlocked[i]
			c.Stats.TopBlocked = append(c.Stats.TopBlocked[:i], c.Stats.TopBlocked[i+1:]...)
			c.Stats.TopBlocked = append([]BlockedEntry{entry}, c.Stats.TopBlocked...)
			return
		}
	}

	// New domain — prepend so newest is always first
	c.Stats.TotalBlocked++
	c.Stats.BlockedToday++
	c.Stats.TopBlocked = append([]BlockedEntry{{Domain: domain, Category: category, Count: 1, LastSeen: now}}, c.Stats.TopBlocked...)
	if len(c.Stats.TopBlocked) > 20 {
		c.Stats.TopBlocked = c.Stats.TopBlocked[:20]
	}
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
