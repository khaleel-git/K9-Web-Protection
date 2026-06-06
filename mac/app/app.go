package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"k10webprotection/internal/config"
	"k10webprotection/internal/database"
	"k10webprotection/internal/hosts"
	"k10webprotection/internal/proxy"
)

// ── Types exposed to frontend ─────────────────────────────────────────────────

type Status struct {
	ProxyRunning      bool                  `json:"proxyRunning"`
	Layer1Active      bool                  `json:"layer1Active"`
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
}

type BlocklistData struct {
	UserAdded      []string `json:"userAdded"`
	BuiltInDomains int      `json:"builtInDomains"`
	BuiltInURLs    int      `json:"builtInUrls"`
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
	DelayHours      int  `json:"delayHours"`
	RequestPending  bool `json:"requestPending"`
	ReadyToDisable  bool `json:"readyToDisable"`
	RemainingSeconds int `json:"remainingSeconds"`
}

// ── App ───────────────────────────────────────────────────────────────────────

type App struct {
	ctx          context.Context
	cfg          *config.Config
	proxy        *proxy.Proxy
	proxyRunning int32 // accessed via sync/atomic; 0=stopped 1=running
	quitAuth     int32 // 1 = quit authorised; lets OnBeforeClose pass through
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	registerShutdownObserver() // macOS shutdown/restart/logout: quit without password prompt
	a.cfg = config.Load()
	a.proxy = proxy.New(a.cfg, func(domain string) {
		cat := database.DB.CategoryFor(domain)
		a.cfg.IncrementBlocked(domain, cat)
		a.cfg.Save()
	})
	// Enforce SafeSearch via /etc/hosts on startup
	if a.cfg.SafeSearch {
		go hosts.SetSafeSearch(true)
	}

	// Auto-install the MITM CA into System keychain on first run (or after reinstall).
	// Runs after a short delay so the app window is visible before the password dialog.
	go func() {
		time.Sleep(2 * time.Second)
		if !isCACertInstalled() {
			a.InstallCACert() //nolint:errcheck
		}
	}()
	// Always clear system proxy first — recovers from a previous force-kill
	// that left the proxy enabled with nothing listening on the port.
	a.setSystemProxy(false)
	if a.cfg.AutoStart {
		go func() {
			if err := a.startProxyAndWait(); err == nil {
				a.setSystemProxy(true)
			}
		}()
	}
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		for range sigs {
			// Always clear the system proxy immediately so internet is not left broken
			// if the process is killed before shutdown() can run.
			a.setSystemProxy(false)
			a.proxy.Stop()
			atomic.StoreInt32(&a.proxyRunning, 0)
			a.cfg.Save()
			os.Exit(0)
		}
	}()
}

func (a *App) shutdown(_ context.Context) {
	a.setSystemProxy(false)
	a.proxy.Stop()
	a.cfg.Save()
}

// ── Status ────────────────────────────────────────────────────────────────────

func (a *App) ClearStats() error {
	a.cfg.Stats.TopBlocked = nil
	a.cfg.Stats.BlockedToday = 0
	a.cfg.Stats.TotalBlocked = 0
	return a.cfg.Save()
}

func (a *App) GetStatus() Status {
	db := database.DB
	rem := int(a.cfg.FocusModeRemaining().Seconds())
	return Status{
		ProxyRunning:   atomic.LoadInt32(&a.proxyRunning) == 1,
		Layer1Active:   hosts.IsActive(),
		BlockedToday:   a.cfg.Stats.BlockedToday,
		TotalBlocked:   a.cfg.Stats.TotalBlocked,
		ProxyPort:      a.cfg.ProxyPort,
		TopBlocked:     a.cfg.Stats.TopBlocked,
		DBDomains:      db.DomainCount(),
		DBURLs:         db.URLCount(),
		DBKeywords:     db.KeywordCount(),
		InFocusMode:       a.cfg.InFocusMode(),
		FocusRemaining:    rem,
		InTimeRestriction: a.cfg.InTimeRestriction(),
	}
}

// ── Protection on/off ─────────────────────────────────────────────────────────

func (a *App) EnableProtection() error {
	if err := hosts.Install(a.cfg.GetUserBlocklist()); err != nil {
		return err
	}
	if err := a.startProxyAndWait(); err != nil {
		return err
	}
	a.setSystemProxy(true)
	return nil
}

func (a *App) DisableProtection(password string) error {
	if a.cfg.PasswordHash != "" && !a.verifyPassword(password) {
		return errors.New("incorrect password")
	}
	if a.cfg.InFocusMode() {
		rem := int(a.cfg.FocusModeRemaining().Minutes())
		return fmt.Errorf("focus mode is active — %d min remaining", rem)
	}
	allowed, remaining := a.cfg.DisableAllowed()
	if !allowed {
		return fmt.Errorf("disable delay active — %.0f hours remaining",
			remaining.Hours())
	}
	a.cfg.ClearDisableRequest()
	a.proxy.Stop()
	atomic.StoreInt32(&a.proxyRunning, 0)
	a.setSystemProxy(false)
	return a.cfg.Save()
}

// RequestDisable registers a disable intent when delay is configured.
func (a *App) RequestDisable() error {
	if a.cfg.DisableDelayHours <= 0 {
		return errors.New("no delay configured")
	}
	a.cfg.RequestDisable()
	return a.cfg.Save()
}

func (a *App) GetDisableDelayStatus() DisableDelayStatus {
	allowed, remaining := a.cfg.DisableAllowed()
	return DisableDelayStatus{
		DelayHours:       a.cfg.DisableDelayHours,
		RequestPending:   a.cfg.DisableRequestedAt != nil,
		ReadyToDisable:   allowed,
		RemainingSeconds: int(remaining.Seconds()),
	}
}

// ── Block list ────────────────────────────────────────────────────────────────

func (a *App) GetBlocklist() BlocklistData {
	db := database.DB
	return BlocklistData{
		UserAdded:      a.cfg.GetUserBlocklist(),
		BuiltInDomains: db.DomainCount(),
		BuiltInURLs:    db.URLCount(),
	}
}

func (a *App) AddToBlocklist(domain string) error {
	domain = cleanDomain(domain)
	if domain == "" {
		return errors.New("invalid domain")
	}
	a.cfg.AddUserBlocklist(domain)
	return a.cfg.Save()
}

func (a *App) RemoveFromBlocklist(domain string) error {
	a.cfg.RemoveUserBlocklist(domain)
	return a.cfg.Save()
}

// ── Allow list ────────────────────────────────────────────────────────────────

func (a *App) GetAllowlist() []string { return a.cfg.GetUserAllowlist() }

func (a *App) AddToAllowlist(domain string) error {
	domain = cleanDomain(domain)
	if domain == "" {
		return errors.New("invalid domain")
	}
	a.cfg.AddUserAllowlist(domain)
	return a.cfg.Save()
}

func (a *App) RemoveFromAllowlist(domain string) error {
	a.cfg.RemoveUserAllowlist(domain)
	return a.cfg.Save()
}

// ── Keywords ──────────────────────────────────────────────────────────────────

func (a *App) GetKeywords() KeywordsData {
	return KeywordsData{
		UserAdded:    a.cfg.GetUserKeywords(),
		BuiltInCount: database.DB.KeywordCount(),
	}
}

func (a *App) AddKeyword(keyword string) error {
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if keyword == "" {
		return errors.New("empty keyword")
	}
	a.cfg.AddUserKeyword(keyword)
	return a.cfg.Save()
}

func (a *App) RemoveKeyword(keyword string) error {
	a.cfg.RemoveUserKeyword(keyword)
	return a.cfg.Save()
}

// ── Content Settings ──────────────────────────────────────────────────────────

func (a *App) GetContentSettings() ContentSettings {
	return ContentSettings{
		FilterLevel:       a.cfg.FilterLevel,
		BlockAdultContent: a.cfg.BlockAdultContent,
		BlockImageSearch:  a.cfg.BlockImageSearch,
		BlockYouTube:      a.cfg.BlockYouTube,
		SafeSearch:        a.cfg.SafeSearch,
	}
}


// ── Advanced Settings ─────────────────────────────────────────────────────────

func (a *App) GetAdvancedSettings() AdvancedSettings {
	return AdvancedSettings{
		DisableDelayHours: a.cfg.DisableDelayHours,
		BlockedMessage:    a.cfg.BlockedMessage,
	}
}

func (a *App) SaveAdvancedSettings(password string, s AdvancedSettings) error {
	if a.cfg.PasswordHash != "" && !a.verifyPassword(password) {
		return errors.New("incorrect password")
	}
	a.cfg.DisableDelayHours = s.DisableDelayHours
	if s.BlockedMessage != "" {
		a.cfg.BlockedMessage = s.BlockedMessage
	}
	return a.cfg.Save()
}

// SetFilterLevel saves a standard filter level (high/default/moderate/minimal)
// without requiring a password. Monitor and Custom require password via SaveContentSettings.
func (a *App) SetFilterLevel(level string) error {
	switch level {
	case "high", "default", "moderate", "minimal":
		// allowed without password
	default:
		return errors.New("use SaveContentSettings for monitor/custom levels")
	}
	a.cfg.FilterLevel = level
	return a.cfg.Save()
}

func (a *App) SaveContentSettings(password string, s ContentSettings) error {
	if a.cfg.PasswordHash != "" && !a.verifyPassword(password) {
		return errors.New("incorrect password")
	}
	a.cfg.FilterLevel = s.FilterLevel
	a.cfg.BlockAdultContent = s.BlockAdultContent
	a.cfg.BlockImageSearch = s.BlockImageSearch
	a.cfg.BlockYouTube = s.BlockYouTube
	a.cfg.SafeSearch = s.SafeSearch
	if err := a.cfg.Save(); err != nil {
		return err
	}
	// Apply or remove SafeSearch enforcement in /etc/hosts (requires admin once)
	go hosts.SetSafeSearch(s.SafeSearch)
	return nil
}

// ── Proxy Settings ────────────────────────────────────────────────────────────

func (a *App) GetProxySettings() ProxySettings {
	return ProxySettings{ProxyPort: a.cfg.ProxyPort, AutoStart: a.cfg.AutoStart}
}

func (a *App) SaveProxySettings(s ProxySettings) error {
	if s.ProxyPort < 1024 || s.ProxyPort > 65535 {
		return errors.New("port must be between 1024 and 65535")
	}
	a.cfg.ProxyPort = s.ProxyPort
	a.cfg.AutoStart = s.AutoStart
	return a.cfg.Save()
}

// ── Focus Mode ────────────────────────────────────────────────────────────────

func (a *App) StartFocusMode(minutes int) error {
	if minutes < 1 || minutes > 1440 {
		return errors.New("duration must be between 1 and 1440 minutes")
	}
	a.cfg.SetFocusMode(minutes)
	return a.cfg.Save()
}

func (a *App) StopFocusMode(password string) error {
	if a.cfg.PasswordHash != "" && !a.verifyPassword(password) {
		return errors.New("incorrect password")
	}
	a.cfg.StopFocusMode()
	return a.cfg.Save()
}

func (a *App) GetFocusMode() FocusModeStatus {
	return FocusModeStatus{
		Active:    a.cfg.InFocusMode(),
		Remaining: int(a.cfg.FocusModeRemaining().Seconds()),
	}
}

// ── Focus Sites ───────────────────────────────────────────────────────────────

func (a *App) GetFocusSites() []config.FocusSite {
	return a.cfg.GetFocusSites()
}

func (a *App) SetFocusSiteActive(domain string, active bool) error {
	a.cfg.SetFocusSiteActive(domain, active)
	return a.cfg.Save()
}

func (a *App) AddFocusSite(domain string) error {
	domain = cleanDomain(domain)
	if domain == "" {
		return errors.New("invalid domain")
	}
	a.cfg.AddFocusSite(domain)
	return a.cfg.Save()
}

func (a *App) RemoveFocusSite(domain string) error {
	a.cfg.RemoveFocusSite(domain)
	return a.cfg.Save()
}

// ── Time Restrictions ─────────────────────────────────────────────────────────

func (a *App) GetTimeRestrictions() config.TimeRestrictions {
	return a.cfg.GetTimeRestrictions()
}

func (a *App) SaveTimeRestrictions(tr config.TimeRestrictions) error {
	a.cfg.SaveTimeRestrictions(tr)
	return a.cfg.Save()
}

// ── Password ──────────────────────────────────────────────────────────────────

func (a *App) HasPassword() bool { return a.cfg.PasswordHash != "" }

func (a *App) VerifyPassword(password string) bool { return a.verifyPassword(password) }

func (a *App) SetPassword(current, newPass string) error {
	if a.cfg.PasswordHash != "" && !a.verifyPassword(current) {
		return errors.New("incorrect current password")
	}
	if newPass == "" {
		a.cfg.PasswordHash = ""
		return a.cfg.Save()
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.cfg.PasswordHash = string(hash)
	return a.cfg.Save()
}

func (a *App) ConfirmQuit(password string) error {
	if !a.verifyPassword(password) {
		return errors.New("incorrect password")
	}
	atomic.StoreInt32(&a.quitAuth, 1)
	wailsruntime.Quit(a.ctx)
	return nil
}

func (a *App) verifyPassword(p string) bool {
	if a.cfg.PasswordHash == "" {
		return true
	}
	return bcrypt.CompareHashAndPassword([]byte(a.cfg.PasswordHash), []byte(p)) == nil
}

// ── Uninstall ─────────────────────────────────────────────────────────────────

func (a *App) Uninstall(password string) error {
	if a.cfg.PasswordHash != "" && !a.verifyPassword(password) {
		return errors.New("incorrect password")
	}

	home, _ := os.UserHomeDir()
	uid := os.Getuid()

	// Write a privileged cleanup script. It runs as root via osascript so it
	// can stop services, unlock uchg files, remove PF rules, and delete the app.
	// The app quits after launching the script; the script runs in the background.
	cleanupScript := fmt.Sprintf(`#!/bin/bash
REAL_UID=%d
APP="/Applications/K10 Web Protection.app"
AGENT="/Library/LaunchAgents/com.k10webprotection.plist"
WATCHDOG_AGENT="/Library/LaunchAgents/com.k10webprotection.watchdog.plist"
WATCHDOG_BIN="/usr/local/bin/k10_watchdog.sh"
PF_ANCHOR="/etc/pf.anchors/k10webprotection"
PF_CONF="/etc/pf.conf"

# Stop watchdog first so it stops re-locking files
launchctl bootout gui/$REAL_UID "$WATCHDOG_AGENT" 2>/dev/null || true
launchctl bootout gui/$REAL_UID "$AGENT"          2>/dev/null || true
sleep 1

# Unlock protected files
chflags -R nouchg "$APP"            2>/dev/null || true
chflags nouchg    "$AGENT"           2>/dev/null || true
chflags nouchg    "$WATCHDOG_AGENT"  2>/dev/null || true
chflags nouchg    "$WATCHDOG_BIN"    2>/dev/null || true

# Remove app and system files
rm -rf "$APP"
rm -f  "$AGENT" "$WATCHDOG_AGENT" "$WATCHDOG_BIN"

# Remove /etc/hosts entries
sed -i '' '/^# K10-Web-Protection START$/,/^# K10-Web-Protection END$/d' /etc/hosts 2>/dev/null || true
sed -i '' '/^# K10-SafeSearch START$/,/^# K10-SafeSearch END$/d'         /etc/hosts 2>/dev/null || true
dscacheutil -flushcache 2>/dev/null || true
killall -HUP mDNSResponder 2>/dev/null || true

# Remove PF rules
pfctl -a k10webprotection -F rules 2>/dev/null || true
sed -i '' '/k10webprotection/d' "$PF_CONF" 2>/dev/null || true
rm -f "$PF_ANCHOR"

# Clear system proxy for all network services
networksetup -listallnetworkservices 2>/dev/null | grep -v '[Aa]sterisk\|^$' | while IFS= read -r svc; do
    networksetup -setwebproxystate       "$svc" off 2>/dev/null || true
    networksetup -setsecurewebproxystate "$svc" off 2>/dev/null || true
done

# Remove user config and CA cert
rm -rf "%s/.k10webprotection"

rm -f "$0"
`, uid, home)

	tmp, err := os.CreateTemp("", "k10-uninstall-*.sh")
	if err != nil {
		return fmt.Errorf("could not prepare uninstall script: %w", err)
	}
	scriptPath := tmp.Name()
	tmp.WriteString(cleanupScript)
	tmp.Close()
	os.Chmod(scriptPath, 0755)

	// Stop proxy and clear system proxy from this process first
	a.proxy.Stop()
	atomic.StoreInt32(&a.proxyRunning, 0)
	a.setSystemProxy(false)

	// Run the cleanup script as root via osascript, detached (nohup + &) so
	// this app can quit before the script finishes removing it.
	osaCmd := fmt.Sprintf(
		`do shell script "nohup bash %s >/tmp/k10-uninstall.log 2>&1 &" with administrator privileges`,
		scriptPath,
	)
	if err := exec.Command("osascript", "-e", osaCmd).Run(); err != nil {
		os.Remove(scriptPath)
		return fmt.Errorf("admin privileges required to complete uninstall: %w", err)
	}

	// Quit the app — the cleanup script removes everything in the background
	go func() {
		time.Sleep(400 * time.Millisecond)
		wailsruntime.Quit(a.ctx)
	}()
	return nil
}

// ── Internals ─────────────────────────────────────────────────────────────────

func (a *App) startProxy() {
	if !atomic.CompareAndSwapInt32(&a.proxyRunning, 0, 1) {
		return
	}
	go func() {
		if err := a.proxy.Start(); err != nil {
			atomic.StoreInt32(&a.proxyRunning, 0)
		}
	}()
}

func (a *App) startProxyAndWait() error {
	if !atomic.CompareAndSwapInt32(&a.proxyRunning, 0, 1) {
		return nil
	}
	errCh := make(chan error, 1)
	go func() {
		if err := a.proxy.Start(); err != nil {
			atomic.StoreInt32(&a.proxyRunning, 0)
			errCh <- err
		}
	}()
	addr := fmt.Sprintf("127.0.0.1:%d", a.cfg.ProxyPort)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			return fmt.Errorf("proxy failed to start: %w", err)
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("proxy did not start on port %d within 5 seconds", a.cfg.ProxyPort)
}

var proxyBypassDomains = []string{
	"localhost", "127.0.0.1", "*.local", "169.254/16",
	"*.apple.com", "*.icloud.com", "*.mzstatic.com",
	"*.itunes.com", "*.push.apple.com", "gateway.icloud.com",
}

func (a *App) setSystemProxy(on bool) {
	services := getNetworkServices()
	port := itoa(a.cfg.ProxyPort)
	for _, svc := range services {
		if on {
			exec.Command("networksetup", "-setwebproxy", svc, "127.0.0.1", port).Run()
			exec.Command("networksetup", "-setsecurewebproxy", svc, "127.0.0.1", port).Run()
			exec.Command("networksetup", "-setwebproxystate", svc, "on").Run()
			exec.Command("networksetup", "-setsecurewebproxystate", svc, "on").Run()
			args := append([]string{"-setproxybypassdomains", svc}, proxyBypassDomains...)
			exec.Command("networksetup", args...).Run()
		} else {
			exec.Command("networksetup", "-setwebproxystate", svc, "off").Run()
			exec.Command("networksetup", "-setsecurewebproxystate", svc, "off").Run()
		}
	}
	if on {
		spawnProxyWatchdog()
	}
}

// spawnProxyWatchdog starts a detached bash process that clears the system proxy
// the moment this process dies — handles SIGKILL and wails dev force-kills.
func spawnProxyWatchdog() {
	pid := os.Getpid()
	script := fmt.Sprintf(`pid=%d
while kill -0 "$pid" 2>/dev/null; do sleep 1; done
networksetup -listallnetworkservices 2>/dev/null | grep -v '[Aa]sterisk\|^$' | while IFS= read -r svc; do
  networksetup -setwebproxystate "$svc" off 2>/dev/null
  networksetup -setsecurewebproxystate "$svc" off 2>/dev/null
done`, pid)
	cmd := exec.Command("bash", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // own process group → survives parent kill
	cmd.Start()                                            //nolint — intentionally fire-and-forget
}

func getNetworkServices() []string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil
	}
	var services []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "asterisk") {
			continue
		}
		services = append(services, line)
	}
	return services
}

func cleanDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	for _, pfx := range []string{"https://", "http://", "www."} {
		d = strings.TrimPrefix(d, pfx)
	}
	return strings.Split(d, "/")[0]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 8)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// CACertPath returns the path to the K10 root CA certificate that must be
// trusted in macOS Keychain for the HTTPS block page to display correctly.
func (a *App) CACertPath() string { return proxy.CACertPath() }

// isCACertInstalled reports whether the K10 CA is present in the System keychain.
func isCACertInstalled() bool {
	out, err := exec.Command("security", "find-certificate",
		"-c", "K10 Web Protection CA",
		"/Library/Keychains/System.keychain",
	).Output()
	return err == nil && len(out) > 0
}

// InstallCACert adds the K10 root CA to the macOS System keychain and marks
// it as a trusted root for TLS.
//
// IMPORTANT: security(1) must be called directly from this GUI process — NOT
// via osascript "do shell script with administrator privileges". When run as a
// root subprocess via osascript, SecTrustSettingsSetTrustSettings cannot show
// its interactive authorization dialog and fails with "no user interaction was
// possible". Run directly, macOS shows its own native password prompt.
func (a *App) InstallCACert() error {
	certPath := proxy.CACertPath()
	if _, err := os.Stat(certPath); err != nil {
		return fmt.Errorf("CA certificate not yet generated — enable protection first")
	}
	out, err := exec.Command(
		"security", "add-trusted-cert",
		"-d", "-r", "trustRoot",
		"-k", "/Library/Keychains/System.keychain",
		certPath,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("install failed — %s", strings.TrimSpace(string(out)))
	}
	return nil
}
