//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/idna"

	"k10webprotection/internal/config"
	"k10webprotection/internal/database"
	"k10webprotection/internal/enforce"
	"k10webprotection/internal/hosts"
	"k10webprotection/internal/i18n"
	"k10webprotection/internal/proxy"
	"k10webprotection/internal/tray"
)

// ── Types exposed to frontend ─────────────────────────────────────────────────

type Status struct {
	ProxyRunning      bool                  `json:"proxyRunning"`
	Layer1Active      bool                  `json:"layer1Active"`
	Layer1Idle        bool                  `json:"layer1Idle"` // hosts applied fine but had nothing to write
	BlockedToday      int                   `json:"blockedToday"`
	TotalBlocked      int                   `json:"totalBlocked"`
	ProxyPort         int                   `json:"proxyPort"`
	TopBlocked        []config.BlockedEntry `json:"topBlocked"`
	DBDomains         int                   `json:"dbDomains"`
	DBURLs            int                   `json:"dbUrls"`
	DBKeywords        int                   `json:"dbKeywords"`
	InFocusMode       bool                  `json:"inFocusMode"`
	FocusRemaining    int                   `json:"focusRemaining"` // seconds
	InTimeRestriction bool                  `json:"inTimeRestriction"`
	Diagnostics       DiagnosticsView       `json:"diagnostics"`
}

// ApplyView is the outcome of pushing settings to disk, the proxy and the hosts file.
type ApplyView struct {
	ConfigError   string `json:"configError"`
	HostsError    string `json:"hostsError"`
	HostsInfo     string `json:"hostsInfo"`    // why hosts was deliberately left alone
	HostsApplied  bool   `json:"hostsApplied"` // false while protection is off
	HostsPartial  bool   `json:"hostsPartial"` // hosts can't cover every subdomain; the proxy does
	ClosedTunnels int    `json:"closedTunnels"`
}

type RulesView struct {
	Block   []config.DomainRule `json:"block"`
	Allow   []config.DomainRule `json:"allow"`
	Display map[string]string   `json:"display"` // Unicode form of IDN domains, keyed by domain
	Apply   ApplyView           `json:"apply"`
	Notice  string              `json:"notice"`
}

type DiagnosticsView struct {
	SettingsPath   string    `json:"settingsPath"`
	SettingsSource string    `json:"settingsSource"` // current | legacy | default
	MigratedFrom   string    `json:"migratedFrom"`
	BackupPath     string    `json:"backupPath"`
	LoadError      string    `json:"loadError"`
	ReadOnly       bool      `json:"readOnly"`
	SkippedLegacy  []string  `json:"skippedLegacy"`
	SkippedRules   []string  `json:"skippedRules"`
	LastApply      ApplyView `json:"lastApply"`
}

type KeywordsData struct {
	UserAdded    []string `json:"userAdded"`
	BuiltInCount int      `json:"builtInCount"`
}

type ContentSettings struct {
	FilterLevel       string `json:"filterLevel"`
	BlockAdultContent bool   `json:"blockAdultContent"`
	BlockImageSearch  bool   `json:"blockImageSearch"`
	BlockYouTube      bool   `json:"blockYouTube"`
	SafeSearch        bool   `json:"safeSearch"`
}

type AdvancedSettings struct {
	DisableDelayHours int    `json:"disableDelayHours"`
	BlockedMessage    string `json:"blockedMessage"`
}

type ProxySettings struct {
	ProxyPort int  `json:"proxyPort"`
	AutoStart bool `json:"autoStart"`
}

type FocusModeStatus struct {
	Active    bool `json:"active"`
	Remaining int  `json:"remaining"` // seconds
}

type DisableDelayStatus struct {
	DelayHours       int  `json:"delayHours"`
	RequestPending   bool `json:"requestPending"`
	ReadyToDisable   bool `json:"readyToDisable"`
	RemainingSeconds int  `json:"remainingSeconds"`
}

// ── App ───────────────────────────────────────────────────────────────────────

type App struct {
	ctx          context.Context
	cur          atomic.Pointer[settings]
	loadOpts     config.LoadOptions
	hostsActive  func() bool
	sysProxy     func(on bool) // setSystemProxy; replaced in tests
	proxy        *proxy.Proxy
	applier      *enforce.Applier
	proxyRunning int32 // accessed via sync/atomic; 0=stopped 1=running
	runPort      int32 // port the running proxy listens on; atomic
	reloadMu     sync.Mutex
	quitAuth     int32 // 1 = quit authorised; lets OnBeforeClose pass through

	statsMu    sync.Mutex
	statsTimer *time.Timer
}

// statsSaveDelay batches stats writes so a burst of blocked requests is one save.
const statsSaveDelay = 5 * time.Second

// settings is swapped as a whole by ReloadSettings.
type settings struct {
	cfg     *config.Config
	loadErr error
}

func NewApp(cfg *config.Config, loadErr error, opts config.LoadOptions) *App {
	a := &App{loadOpts: opts, hostsActive: hosts.IsActive}
	a.sysProxy = a.setSystemProxy
	a.cur.Store(&settings{cfg: cfg, loadErr: loadErr})
	return a
}

func (a *App) conf() *config.Config { return a.cur.Load().cfg }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	registerShutdownObserver()
	tray.Run(ctx, func() {
		wailsruntime.WindowShow(a.ctx)
		wailsruntime.WindowUnminimise(a.ctx)
	}, func() {
		// the quit modal lives in the same window; it must be shown before the event reaches it
		wailsruntime.WindowShow(a.ctx)
		wailsruntime.WindowUnminimise(a.ctx)
		wailsruntime.EventsEmit(a.ctx, "quit-requested")
	})

	a.proxy = proxy.New(a.conf().PolicyView().ProxyPort, a.onBlock)
	a.applier = enforce.New(a.conf(), a.proxy, enforce.HostsFunc(hosts.Apply), a.protectionOn)
	a.applier.Enforce() // policy in place before the proxy listens

	// Always clear system proxy first — recovers from a previous force-kill
	// that left the proxy enabled with nothing listening on the port.
	a.sysProxy(false)
	if a.conf().AutoStart {
		go func() {
			if err := a.startProxyAndWait(); err == nil {
				a.sysProxy(true)
				a.applier.Enforce()
			}
		}()
	}
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		for range sigs {
			tray.Stop()
			a.sysProxy(false)
			a.proxy.Stop()
			atomic.StoreInt32(&a.proxyRunning, 0)
			a.flushStats()
			os.Exit(0)
		}
	}()
}

func (a *App) shutdown(_ context.Context) {
	tray.Stop()
	a.sysProxy(false)
	a.proxy.Stop()
	a.flushStats()
}

func (a *App) protectionOn() bool { return atomic.LoadInt32(&a.proxyRunning) == 1 }

func (a *App) onBlock(domain string) {
	a.conf().IncrementBlocked(domain, database.DB.CategoryFor(domain))
	a.statsMu.Lock()
	if a.statsTimer == nil {
		a.statsTimer = time.AfterFunc(statsSaveDelay, a.flushStats)
	}
	a.statsMu.Unlock()
}

func (a *App) flushStats() {
	a.statsMu.Lock()
	if a.statsTimer != nil {
		a.statsTimer.Stop()
		a.statsTimer = nil
	}
	a.statsMu.Unlock()
	a.conf().Save()
}

// ── Status ────────────────────────────────────────────────────────────────────

func (a *App) ClearStats() error {
	a.conf().Update(func(c *config.Config) {
		c.Stats.TopBlocked = nil
		c.Stats.BlockedToday = 0
		c.Stats.TotalBlocked = 0
	})
	return a.saveErr(a.conf().Save())
}

func (a *App) GetStatus() Status {
	db := database.DB
	rem := int(a.conf().FocusModeRemaining().Seconds())
	var stats config.Stats
	var port int
	a.conf().Update(func(c *config.Config) {
		stats = c.Stats
		stats.TopBlocked = append([]config.BlockedEntry(nil), c.Stats.TopBlocked...)
		port = c.ProxyPort
	})
	last := a.applier.Last()
	on := a.protectionOn()
	if on {
		port = a.listenPort()
	}
	ok := last.HostsApplied && last.HostsErr == nil
	empty := len(last.Hosts.BlockedNames)+len(last.Hosts.SafeSearchNames) == 0
	l1 := on && ((ok && !empty) || a.hostsActive())
	return Status{
		ProxyRunning:      on,
		Layer1Active:      l1,
		Layer1Idle:        on && !l1 && ok && empty,
		BlockedToday:      stats.BlockedToday,
		TotalBlocked:      stats.TotalBlocked,
		ProxyPort:         port,
		TopBlocked:        stats.TopBlocked,
		DBDomains:         db.DomainCount(),
		DBURLs:            db.URLCount(),
		DBKeywords:        db.KeywordCount(),
		InFocusMode:       a.conf().InFocusMode(),
		FocusRemaining:    rem,
		InTimeRestriction: a.conf().InTimeRestriction(),
		Diagnostics:       a.diagnostics(last),
	}
}

func (a *App) diagnostics(last enforce.ApplyStatus) DiagnosticsView {
	d := a.conf().Diagnostics()
	v := DiagnosticsView{
		SettingsPath:   d.Path,
		SettingsSource: d.Source,
		MigratedFrom:   d.MigratedFrom,
		BackupPath:     d.BackupPath,
		LoadError:      d.LoadErr,
		ReadOnly:       a.conf().ReadOnly(),
		SkippedLegacy:  nonNil(d.SkippedLegacy),
		SkippedRules:   nonNil(d.SkippedRules),
		LastApply:      applyView(last),
	}
	if le := a.cur.Load().loadErr; v.LoadError == "" && le != nil {
		v.LoadError = le.Error()
	}
	return v
}

// ReloadSettings reads a read-only settings file again and, if it now loads, enforces it.
func (a *App) ReloadSettings() (DiagnosticsView, error) {
	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()
	if !a.conf().ReadOnly() {
		return a.diagnostics(a.applier.Last()), nil
	}
	lang := i18n.Lang()
	cfg, err := config.LoadWith(a.loadOpts)
	if err != nil {
		i18n.SetLang(lang) // a failed load falls back to defaults, language included
		return a.diagnostics(a.applier.Last()), errors.New(i18n.T("err.settingsReloadFailed", err.Error()))
	}
	a.cur.Store(&settings{cfg: cfg}) // a new port takes effect when the proxy next starts
	a.applier.SetStore(cfg)
	return a.diagnostics(a.applier.Retry()), nil
}

// RetryHosts writes the hosts file again, prompting for administrator permission even if it was declined.
func (a *App) RetryHosts() ApplyView { return applyView(a.applier.Retry()) }

// ── Protection on/off ─────────────────────────────────────────────────────────

func (a *App) EnableProtection() error {
	if err := a.startProxyAndWait(); err != nil {
		return err
	}
	a.sysProxy(true)
	st := a.applier.Retry() // turning protection on may prompt for elevation again
	if err := applyErr(st); err != nil {
		return err
	}
	if st.HostsErr != nil {
		return errors.New(hostsErrText(st.HostsErr))
	}
	return nil
}

func (a *App) DisableProtection(password string) error {
	if !a.verifyPassword(password) {
		return errors.New(i18n.T("err.incorrectPassword"))
	}
	if a.conf().InFocusMode() {
		rem := int(a.conf().FocusModeRemaining().Minutes())
		return errors.New(i18n.T("err.focusModeActive", rem))
	}
	allowed, remaining := a.conf().DisableAllowed()
	if !allowed {
		return errors.New(i18n.T("err.disableDelayActive", remaining.Hours()))
	}
	a.conf().ClearDisableRequest()
	a.proxy.Stop()
	atomic.StoreInt32(&a.proxyRunning, 0)
	a.sysProxy(false)
	var msgs []string
	if err := a.applier.ClearHosts(); err != nil {
		msgs = append(msgs, i18n.T("err.hostsClearFailed", hostsErrText(err)))
	}
	if err := a.saveErr(a.conf().Save()); err != nil {
		msgs = append(msgs, err.Error())
	}
	if len(msgs) > 0 {
		return errors.New(strings.Join(msgs, " · "))
	}
	return nil
}

func (a *App) RequestDisable() error {
	if a.conf().DisableDelayHours <= 0 {
		return errors.New(i18n.T("err.noDelayConfigured"))
	}
	a.conf().RequestDisable()
	return a.saveErr(a.conf().Save())
}

func (a *App) GetDisableDelayStatus() DisableDelayStatus {
	allowed, remaining := a.conf().DisableAllowed()
	return DisableDelayStatus{
		DelayHours:       a.conf().DisableDelayHours,
		RequestPending:   a.conf().DisableRequestedAt != nil,
		ReadyToDisable:   allowed,
		RemainingSeconds: int(remaining.Seconds()),
	}
}

// ── Block / allow rules ───────────────────────────────────────────────────────

// GetRules returns the rules as last applied to the proxy.
func (a *App) GetRules() RulesView { return rulesView(a.applier.Last()) }

func (a *App) AddBlockRule(input string, includeSubdomains bool) (RulesView, error) {
	r, err := config.ParseRule(input, includeSubdomains)
	if err != nil {
		return RulesView{}, ruleErr(err)
	}
	// narrowing weakens protection, so it goes through the password-checked remove
	if r, err = a.conf().AddBlockRule(r); errors.Is(err, config.ErrNarrowing) {
		return RulesView{}, errors.New(i18n.T("err.ruleNarrowing", displayDomain(r.Domain)))
	} else if err != nil {
		return RulesView{}, ruleErr(err)
	}
	v := rulesView(a.applier.Apply())
	if proxy.IsBuiltinAllowed(r.Domain) {
		v.Notice = i18n.T("notice.ruleBuiltinExempt", displayDomain(r.Domain))
	}
	return v, nil
}

func (a *App) RemoveBlockRule(password, domain string) (RulesView, error) {
	if !a.verifyPassword(password) {
		return RulesView{}, errors.New(i18n.T("err.incorrectPassword"))
	}
	if !a.conf().RemoveBlockRule(domain) {
		return RulesView{}, errors.New(i18n.T("err.ruleNotFound"))
	}
	return rulesView(a.applier.Apply()), nil
}

func (a *App) AddAllowRule(password, input string, includeSubdomains bool) (RulesView, error) {
	if !a.verifyPassword(password) {
		return RulesView{}, errors.New(i18n.T("err.incorrectPassword"))
	}
	r, err := config.ParseRule(input, includeSubdomains)
	if err != nil {
		return RulesView{}, ruleErr(err)
	}
	if err := a.conf().UpsertAllowRule(r); err != nil {
		return RulesView{}, ruleErr(err)
	}
	return rulesView(a.applier.Apply()), nil
}

func (a *App) RemoveAllowRule(domain string) (RulesView, error) {
	if !a.conf().RemoveAllowRule(domain) {
		return RulesView{}, errors.New(i18n.T("err.ruleNotFound"))
	}
	return rulesView(a.applier.Apply()), nil
}

// ── Keywords ──────────────────────────────────────────────────────────────────

func (a *App) GetKeywords() KeywordsData {
	return KeywordsData{
		UserAdded:    a.conf().GetUserKeywords(),
		BuiltInCount: database.DB.KeywordCount(),
	}
}

func (a *App) AddKeyword(keyword string) error {
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if keyword == "" {
		return errors.New(i18n.T("err.emptyKeyword"))
	}
	a.conf().AddUserKeyword(keyword)
	return applyErr(a.applier.Apply())
}

func (a *App) RemoveKeyword(keyword string) error {
	a.conf().RemoveUserKeyword(keyword)
	return applyErr(a.applier.Apply())
}

// ── Content Settings ──────────────────────────────────────────────────────────

func (a *App) GetContentSettings() ContentSettings {
	v := a.conf().PolicyView()
	return ContentSettings{
		FilterLevel:       v.FilterLevel,
		BlockAdultContent: v.BlockAdultContent,
		BlockImageSearch:  v.BlockImageSearch,
		BlockYouTube:      v.BlockYouTube,
		SafeSearch:        v.SafeSearch,
	}
}

// GetLevelCategories returns the identifiers each standard filter level blocks, in order.
func (a *App) GetLevelCategories() map[string][]string {
	out := make(map[string][]string, len(proxy.LevelCategories))
	for level, cats := range proxy.LevelCategories {
		c := make([]string, len(cats))
		copy(c, cats)
		out[level] = c
	}
	return out
}

// SetFilterLevel saves a standard filter level without requiring a password.
func (a *App) SetFilterLevel(level string) error {
	switch level {
	case "high", "default", "moderate", "minimal":
		// allowed without password
	default:
		return errors.New("use SaveContentSettings for monitor/custom levels")
	}
	a.conf().Update(func(c *config.Config) { c.FilterLevel = level })
	return applyErr(a.applier.Apply())
}

func (a *App) SaveContentSettings(password string, s ContentSettings) error {
	if !a.verifyPassword(password) {
		return errors.New(i18n.T("err.incorrectPassword"))
	}
	a.conf().Update(func(c *config.Config) {
		c.FilterLevel = s.FilterLevel
		c.BlockAdultContent = s.BlockAdultContent
		c.BlockImageSearch = s.BlockImageSearch
		c.BlockYouTube = s.BlockYouTube
		c.SafeSearch = s.SafeSearch
	})
	return applyErr(a.applier.Apply())
}

// SetSafeSearch needs the password only to turn SafeSearch off.
func (a *App) SetSafeSearch(password string, enabled bool) error {
	if !enabled && !a.verifyPassword(password) {
		return errors.New(i18n.T("err.incorrectPassword"))
	}
	a.conf().Update(func(c *config.Config) { c.SafeSearch = enabled })
	return applyErr(a.applier.Apply())
}

// ── Language ──────────────────────────────────────────────────────────────────

// GetLanguage returns the stored language choice; empty means none has been made yet.
func (a *App) GetLanguage() string { return a.conf().Language }

// SetLanguage switches the backend language and persists it.
func (a *App) SetLanguage(lang string) error {
	switch lang {
	case "he", "en":
		// supported
	default:
		return errors.New(i18n.T("err.unsupportedLanguage"))
	}
	i18n.SetLang(lang)
	a.conf().SetLanguage(lang)
	return a.saveErr(a.conf().Save())
}

// ── Advanced Settings ─────────────────────────────────────────────────────────

func (a *App) GetAdvancedSettings() AdvancedSettings {
	var s AdvancedSettings
	a.conf().Update(func(c *config.Config) {
		s = AdvancedSettings{DisableDelayHours: c.DisableDelayHours, BlockedMessage: c.BlockedMessage}
	})
	return s
}

func (a *App) SaveAdvancedSettings(password string, s AdvancedSettings) error {
	if !a.verifyPassword(password) {
		return errors.New(i18n.T("err.incorrectPassword"))
	}
	a.conf().Update(func(c *config.Config) {
		c.DisableDelayHours = s.DisableDelayHours
		if s.BlockedMessage != "" {
			c.BlockedMessage = s.BlockedMessage
		}
	})
	return a.saveErr(a.conf().Save())
}

// ── Proxy Settings ────────────────────────────────────────────────────────────

func (a *App) GetProxySettings() ProxySettings {
	var s ProxySettings
	a.conf().Update(func(c *config.Config) { s = ProxySettings{ProxyPort: c.ProxyPort, AutoStart: c.AutoStart} })
	return s
}

func (a *App) SaveProxySettings(s ProxySettings) error {
	if s.ProxyPort < 1024 || s.ProxyPort > 65535 {
		return errors.New(i18n.T("err.portRange"))
	}
	a.conf().Update(func(c *config.Config) {
		c.ProxyPort = s.ProxyPort
		c.AutoStart = s.AutoStart
	})
	a.proxy.SetPort(s.ProxyPort)
	return a.saveErr(a.conf().Save())
}

// ── Focus Mode ────────────────────────────────────────────────────────────────

func (a *App) StartFocusMode(minutes int) error {
	if minutes < 1 || minutes > 1440 {
		return errors.New(i18n.T("err.focusDurationRange"))
	}
	a.conf().SetFocusMode(minutes)
	return applyErr(a.applier.Apply())
}

func (a *App) StopFocusMode(password string) error {
	if !a.verifyPassword(password) {
		return errors.New(i18n.T("err.incorrectPassword"))
	}
	a.conf().StopFocusMode()
	return applyErr(a.applier.Apply())
}

func (a *App) GetFocusMode() FocusModeStatus {
	return FocusModeStatus{
		Active:    a.conf().InFocusMode(),
		Remaining: int(a.conf().FocusModeRemaining().Seconds()),
	}
}

// ── Focus Sites ───────────────────────────────────────────────────────────────

func (a *App) GetFocusSites() []config.FocusSite {
	return a.conf().GetFocusSites()
}

func (a *App) SetFocusSiteActive(domain string, active bool) error {
	a.conf().SetFocusSiteActive(domain, active)
	return applyErr(a.applier.Apply())
}

func (a *App) AddFocusSite(domain string) error {
	r, err := config.ParseRule(domain, true)
	if err != nil {
		return ruleErr(err)
	}
	a.conf().AddFocusSite(r.Domain)
	return applyErr(a.applier.Apply())
}

func (a *App) RemoveFocusSite(domain string) error {
	a.conf().RemoveFocusSite(domain)
	return applyErr(a.applier.Apply())
}

// ── Time Restrictions ─────────────────────────────────────────────────────────

func (a *App) GetTimeRestrictions() config.TimeRestrictions {
	return a.conf().GetTimeRestrictions()
}

func (a *App) SaveTimeRestrictions(tr config.TimeRestrictions) error {
	a.conf().SaveTimeRestrictions(tr)
	return applyErr(a.applier.Apply())
}

// ── Password ──────────────────────────────────────────────────────────────────

// HasPassword is also true for an unreadable config, so the UI still prompts (and fails closed).
func (a *App) HasPassword() bool { return a.passwordHash() != "" || a.conf().ReadOnly() }

func (a *App) VerifyPassword(password string) bool { return a.verifyPassword(password) }

func (a *App) SetPassword(current, newPass string) error {
	if !a.verifyPassword(current) {
		return errors.New(i18n.T("err.incorrectCurrentPassword"))
	}
	hash := ""
	if newPass != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		hash = string(h)
	}
	a.conf().Update(func(c *config.Config) { c.PasswordHash = hash })
	return a.saveErr(a.conf().Save())
}

func (a *App) ConfirmQuit(password string) error {
	if !a.verifyPassword(password) {
		return errors.New(i18n.T("err.incorrectPassword"))
	}
	atomic.StoreInt32(&a.quitAuth, 1)
	wailsruntime.Quit(a.ctx)
	return nil
}

func (a *App) passwordHash() string {
	var h string
	a.conf().Update(func(c *config.Config) { h = c.PasswordHash })
	return h
}

func (a *App) verifyPassword(p string) bool {
	hash := a.passwordHash()
	if hash == "" {
		return !a.conf().ReadOnly() // an unreadable config has no password to check against
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
}

// ── Uninstall ─────────────────────────────────────────────────────────────────

func (a *App) Uninstall(password string) error {
	if !a.verifyPassword(password) {
		return errors.New(i18n.T("err.incorrectPassword"))
	}

	home, _ := os.UserHomeDir()
	exePath, _ := os.Executable()
	appDir := filepath.Dir(exePath)

	cleanupScript := fmt.Sprintf(`
# Disable proxy
$key = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
Set-ItemProperty -Path $key -Name ProxyEnable -Value 0 -ErrorAction SilentlyContinue

# Notify WinINet
$sig = @'
using System;using System.Runtime.InteropServices;
public class WinINet{[DllImport("wininet.dll")]public static extern bool InternetSetOption(IntPtr h,int o,IntPtr b,int l);}
'@
Add-Type -TypeDefinition $sig -ErrorAction SilentlyContinue
[WinINet]::InternetSetOption([IntPtr]::Zero,39,[IntPtr]::Zero,0)|Out-Null
[WinINet]::InternetSetOption([IntPtr]::Zero,37,[IntPtr]::Zero,0)|Out-Null

# Remove startup entry
Remove-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name 'K10WebProtection' -ErrorAction SilentlyContinue
Remove-ItemProperty -Path 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Run' -Name 'K10WebProtection' -ErrorAction SilentlyContinue

# Remove hosts entries
$hostsPath = "$env:SystemRoot\System32\drivers\etc\hosts"
`+hostsCleanupPS+`
ipconfig /flushdns | Out-Null

# Remove K10 CA from Windows trust stores
Get-ChildItem Cert:\LocalMachine\Root | Where-Object {$_.Subject -like '*K10 Web Protection*'} | Remove-Item -ErrorAction SilentlyContinue
Get-ChildItem Cert:\CurrentUser\Root  | Where-Object {$_.Subject -like '*K10 Web Protection*'} | Remove-Item -ErrorAction SilentlyContinue

# Remove user data
Remove-Item -Path '%s\.k10webprotection' -Recurse -Force -ErrorAction SilentlyContinue

# Remove app directory after delay so the process has exited
Start-Sleep -Seconds 3
Remove-Item -Path '%s' -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path $MyInvocation.MyCommand.Path -Force -ErrorAction SilentlyContinue
`, home, appDir)

	tmp, err := os.CreateTemp("", "k10-uninstall-*.ps1")
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("err.prepareUninstallScript"), err)
	}
	scriptPath := tmp.Name()
	tmp.WriteString(utf8BOM + cleanupScript)
	tmp.Close()

	a.proxy.Stop()
	atomic.StoreInt32(&a.proxyRunning, 0)
	a.sysProxy(false)

	// Run elevated and detached so the app can quit before the script finishes.
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf(`Start-Process powershell -ArgumentList '-ExecutionPolicy Bypass -File "%s"' -Verb RunAs`,
			strings.ReplaceAll(scriptPath, `"`, `\"`),
		),
	)
	if err := cmd.Run(); err != nil {
		os.Remove(scriptPath)
		return fmt.Errorf("%s: %w", i18n.T("err.uninstallNeedsAdmin"), err)
	}

	go func() {
		time.Sleep(400 * time.Millisecond)
		atomic.StoreInt32(&a.quitAuth, 1)
		wailsruntime.Quit(a.ctx)
	}()
	return nil
}

// utf8BOM makes Windows PowerShell 5.1 read a script's non-ASCII paths as UTF-8, not ANSI.
const utf8BOM = "\ufeff"

// hostsCleanupPS removes every K10 and legacy K9 section from $hostsPath, leaving other bytes as they are.
const hostsCleanupPS = `if (Test-Path $hostsPath) {
    $enc = [System.Text.Encoding]::GetEncoding(28591)
    $orig = $enc.GetString([System.IO.File]::ReadAllBytes($hostsPath))
    $c = $orig
    foreach ($m in 'K10-Web-Protection','K10-SafeSearch','K9-Web-Protection') {
        $c = [System.Text.RegularExpressions.Regex]::Replace($c, '(?m)^(\u00EF\u00BB\u00BF)?[ \t]*# ' + $m + ' START[ \t]*\r?\n(?:.*\n)*?[ \t]*# ' + $m + ' END[ \t]*(?:\r?\n|\z)', '$1')
    }
    if ($c -cne $orig) { [System.IO.File]::WriteAllBytes($hostsPath, $enc.GetBytes($c)) }
}`

// ── CA certificate ────────────────────────────────────────────────────────────

func (a *App) CACertPath() string { return proxy.CACertPath() }

// InstallCACert imports the K10 root CA into the Windows Trusted Root store so
// that HTTPS block pages display correctly in Chrome, Edge, and Firefox.
// A UAC elevation prompt will appear asking for administrator credentials.
func (a *App) InstallCACert() error {
	certPath := proxy.CACertPath()
	if _, err := os.Stat(certPath); err != nil {
		return errors.New(i18n.T("err.caCertNotFound"))
	}

	script := fmt.Sprintf(
		`Import-Certificate -FilePath '%s' -CertStoreLocation Cert:\LocalMachine\Root`,
		strings.ReplaceAll(certPath, "'", "''"),
	)
	tmp, err := os.CreateTemp("", "k10-ca-install-*.ps1")
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("err.prepareInstallScript"), err)
	}
	scriptPath := tmp.Name()
	tmp.WriteString(utf8BOM + script)
	tmp.Close()
	defer os.Remove(scriptPath)

	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf(`Start-Process powershell -ArgumentList '-ExecutionPolicy Bypass -File "%s"' -Verb RunAs -Wait`,
			strings.ReplaceAll(scriptPath, `"`, `\"`),
		),
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("err.caInstallNeedsAdmin"), err)
	}
	return nil
}

// ── Internals ─────────────────────────────────────────────────────────────────

func (a *App) startProxyAndWait() error {
	if !atomic.CompareAndSwapInt32(&a.proxyRunning, 0, 1) {
		return nil
	}
	port := a.port()
	a.proxy.SetPort(port)
	atomic.StoreInt32(&a.runPort, int32(port))
	errCh := make(chan error, 1)
	go func() {
		if err := a.proxy.Start(); err != nil {
			atomic.StoreInt32(&a.proxyRunning, 0)
			errCh <- err
		}
	}()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			return fmt.Errorf("%s: %w", i18n.T("err.proxyFailedToStart"), err)
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New(i18n.T("err.proxyStartTimeout", port))
}

// setSystemProxy sets or clears the Windows system-wide HTTP/HTTPS proxy via registry.
func (a *App) setSystemProxy(on bool) {
	const keyPath = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	if on {
		server := fmt.Sprintf("127.0.0.1:%d", a.listenPort())
		exec.Command("reg", "add", keyPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f").Run()
		exec.Command("reg", "add", keyPath, "/v", "ProxyServer", "/t", "REG_SZ", "/d", server, "/f").Run()
		exec.Command("reg", "add", keyPath, "/v", "ProxyOverride", "/t", "REG_SZ", "/d",
			"localhost;127.0.0.1;<local>;*.microsoft.com;*.windowsupdate.com", "/f").Run()
		spawnProxyWatchdog()
	} else {
		exec.Command("reg", "add", keyPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f").Run()
	}
	notifyProxyChange()
}

// notifyProxyChange signals WinINet/WinHTTP that proxy settings have changed
// so that running applications pick up the new settings without restarting.
func notifyProxyChange() {
	wininet := syscall.NewLazyDLL("wininet.dll")
	setOpt := wininet.NewProc("InternetSetOptionW")
	const (
		INTERNET_OPTION_SETTINGS_CHANGED = 39
		INTERNET_OPTION_REFRESH          = 37
	)
	setOpt.Call(0, INTERNET_OPTION_SETTINGS_CHANGED, 0, 0)
	setOpt.Call(0, INTERNET_OPTION_REFRESH, 0, 0)
}

// spawnProxyWatchdog starts a hidden PowerShell process that monitors this
// process and clears the system proxy the moment it exits — handles SIGKILL
// and force-quit scenarios where shutdown() never runs.
func spawnProxyWatchdog() {
	pid := os.Getpid()
	script := fmt.Sprintf(`
$pid = %d
while (Get-Process -Id $pid -ErrorAction SilentlyContinue) { Start-Sleep -Milliseconds 500 }
$k = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
Set-ItemProperty -Path $k -Name ProxyEnable -Value 0 -ErrorAction SilentlyContinue
try {
    Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;public class WI{[DllImport("wininet.dll")]public static extern bool InternetSetOption(IntPtr h,int o,IntPtr b,int l);}' -ErrorAction Stop
    [WI]::InternetSetOption([IntPtr]::Zero,39,[IntPtr]::Zero,0)|Out-Null
    [WI]::InternetSetOption([IntPtr]::Zero,37,[IntPtr]::Zero,0)|Out-Null
} catch {}
`, pid)
	cmd := exec.Command("powershell", "-WindowStyle", "Hidden", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start() //nolint:errcheck — intentionally fire-and-forget
}

func (a *App) port() int {
	var p int
	a.conf().Update(func(c *config.Config) { p = c.ProxyPort })
	return p
}

// listenPort is the running proxy's port, which a changed setting does not move until restart.
func (a *App) listenPort() int {
	if p := atomic.LoadInt32(&a.runPort); a.protectionOn() && p != 0 {
		return int(p)
	}
	return a.port()
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func rulesView(st enforce.ApplyStatus) RulesView {
	v := RulesView{Block: nonNil(st.View.Block), Allow: nonNil(st.View.Allow), Display: map[string]string{}, Apply: applyView(st)}
	for _, r := range append(append([]config.DomainRule(nil), v.Block...), v.Allow...) {
		if d := displayDomain(r.Domain); d != r.Domain {
			v.Display[r.Domain] = d
		}
	}
	return v
}

// displayDomain shows an IDN domain in Unicode; the stored punycode stays the key.
func displayDomain(d string) string {
	if u, err := idna.ToUnicode(d); err == nil && u != "" {
		return u
	}
	return d
}

func applyView(st enforce.ApplyStatus) ApplyView {
	v := ApplyView{HostsApplied: st.HostsApplied, HostsPartial: st.HostsApplied && st.Hosts.Partial, ClosedTunnels: st.ClosedTunnels}
	if st.ConfigErr != nil {
		v.ConfigError = saveErrText(st.ConfigErr)
	}
	if st.HostsErr != nil {
		v.HostsError = hostsErrText(st.HostsErr)
	}
	switch {
	case errors.Is(st.HostsSkipped, enforce.ErrProtectionOff):
		v.HostsInfo = i18n.T("info.hostsProtectionOff")
	case errors.Is(st.HostsSkipped, enforce.ErrSettingsUnreadable):
		v.HostsInfo = i18n.T("info.hostsSettingsUnreadable")
	}
	return v
}

func hostsErrText(err error) string {
	switch {
	case errors.Is(err, hosts.ErrElevationCancelled):
		return i18n.T("err.hostsElevationDeclined")
	case errors.Is(err, hosts.ErrVerifyFailed):
		return i18n.T("err.hostsVerifyFailed")
	default:
		return i18n.T("err.hostsWriteFailed", err.Error())
	}
}

// applyErr reports only settings failures; hosts failures show as warnings.
func applyErr(st enforce.ApplyStatus) error {
	if st.ConfigErr != nil {
		return errors.New(saveErrText(st.ConfigErr))
	}
	return nil
}

func saveErrText(err error) string {
	if errors.Is(err, config.ErrReadOnly) {
		return i18n.T("err.settingsReadOnly")
	}
	return i18n.T("err.settingsSaveFailed", err.Error())
}

func (a *App) saveErr(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(saveErrText(err))
}

func ruleErr(err error) error {
	if errors.Is(err, config.ErrDomainHasPath) {
		return errors.New(i18n.T("err.domainHasPath"))
	}
	return errors.New(i18n.T("err.invalidDomain"))
}