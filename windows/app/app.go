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

	"golang.org/x/crypto/bcrypt"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"k9webprotection/internal/config"
	"k9webprotection/internal/database"
	"k9webprotection/internal/hosts"
	"k9webprotection/internal/proxy"
)

// ── Types exposed to frontend ─────────────────────────────────────────────────

type Status struct {
	ProxyRunning    bool                  `json:"proxyRunning"`
	Layer1Active    bool                  `json:"layer1Active"`
	BlockedToday    int                   `json:"blockedToday"`
	TotalBlocked    int                   `json:"totalBlocked"`
	ProxyPort       int                   `json:"proxyPort"`
	TopBlocked      []config.BlockedEntry `json:"topBlocked"`
	DBDomains       int                   `json:"dbDomains"`
	DBURLs          int                   `json:"dbUrls"`
	DBKeywords      int                   `json:"dbKeywords"`
	InFocusMode     bool                  `json:"inFocusMode"`
	FocusRemaining  int                   `json:"focusRemaining"` // seconds
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
	BlockAdultContent bool `json:"blockAdultContent"`
	BlockImageSearch  bool `json:"blockImageSearch"`
	BlockYouTube      bool `json:"blockYouTube"`
	SafeSearch        bool `json:"safeSearch"`
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
	cfg          *config.Config
	proxy        *proxy.Proxy
	proxyRunning int32 // accessed via sync/atomic; 0=stopped 1=running

	blockMu      sync.Mutex
	recentBlocks map[string]time.Time // domain → last counted time (cooldown dedup)
}

func NewApp() *App { return &App{recentBlocks: make(map[string]time.Time)} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = config.Load()
	a.proxy = proxy.New(a.cfg, func(domain string) {
		// Deduplicate: count the same domain at most once per 60 seconds
		// so that one page visit (many sub-requests) = one block count.
		a.blockMu.Lock()
		last, seen := a.recentBlocks[domain]
		if seen && time.Since(last) < 60*time.Second {
			a.blockMu.Unlock()
			return
		}
		a.recentBlocks[domain] = time.Now()
		a.blockMu.Unlock()
		a.cfg.IncrementBlocked(domain)
		a.cfg.Save()
	})
	if a.cfg.AutoStart {
		go func() {
			if err := a.startProxyAndWait(); err == nil {
				a.setSystemProxy(true)
			}
		}()
	} else {
		a.setSystemProxy(false)
	}
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		for range sigs {
			if a.cfg.PasswordHash != "" {
				wailsruntime.EventsEmit(a.ctx, "kill-requested")
			} else {
				wailsruntime.Quit(a.ctx)
			}
		}
	}()
}

func (a *App) shutdown(_ context.Context) {
	a.setSystemProxy(false)
	a.proxy.Stop()
	a.cfg.Save()
}

// ── Status ────────────────────────────────────────────────────────────────────

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
		InFocusMode:    a.cfg.InFocusMode(),
		FocusRemaining: rem,
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

func (a *App) SaveContentSettings(password string, s ContentSettings) error {
	if a.cfg.PasswordHash != "" && !a.verifyPassword(password) {
		return errors.New("incorrect password")
	}
	a.cfg.BlockAdultContent = s.BlockAdultContent
	a.cfg.BlockImageSearch = s.BlockImageSearch
	a.cfg.BlockYouTube = s.BlockYouTube
	a.cfg.SafeSearch = s.SafeSearch
	return a.cfg.Save()
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

func (a *App) GetFocusMode() FocusModeStatus {
	return FocusModeStatus{
		Active:    a.cfg.InFocusMode(),
		Remaining: int(a.cfg.FocusModeRemaining().Seconds()),
	}
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

	// Clean up protection immediately.
	a.proxy.Stop()
	atomic.StoreInt32(&a.proxyRunning, 0)
	a.setSystemProxy(false)
	hosts.Remove()

	// Launch the NSIS uninstaller silently via a detached cmd.exe that waits
	// 2 seconds for us to exit before running.  The NSIS uninstaller handles
	// removing files, shortcuts, registry entries, and the config directory.
	if exePath, err := os.Executable(); err == nil {
		uninstaller := filepath.Join(filepath.Dir(exePath), "Uninstall.exe")
		if _, statErr := os.Stat(uninstaller); statErr == nil {
			cmd := exec.Command("cmd", "/c",
				fmt.Sprintf(`timeout /t 2 /nobreak >nul && "%s" /S`, uninstaller))
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			cmd.Start() //nolint:errcheck — fire-and-forget
		} else {
			// Fallback when not installed via NSIS (dev / portable build).
			exec.Command("reg", "delete",
				`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`,
				"/v", "K9WebProtection", "/f").Run()
			appData := os.Getenv("APPDATA")
			os.RemoveAll(filepath.Join(appData, "K9WebProtection"))
		}
	}

	wailsruntime.Quit(a.ctx)
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

// setSystemProxy configures the Windows system-wide HTTP/HTTPS proxy via registry.
func (a *App) setSystemProxy(on bool) {
	const keyPath = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	if on {
		server := fmt.Sprintf("127.0.0.1:%d", a.cfg.ProxyPort)
		exec.Command("reg", "add", keyPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f").Run()
		exec.Command("reg", "add", keyPath, "/v", "ProxyServer", "/t", "REG_SZ", "/d", server, "/f").Run()
		exec.Command("reg", "add", keyPath, "/v", "ProxyOverride", "/t", "REG_SZ", "/d", "localhost;127.0.0.1;<local>", "/f").Run()
	} else {
		exec.Command("reg", "add", keyPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f").Run()
	}
	// Notify WinINet that proxy settings changed so running apps pick it up
	exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		`try { Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;public class WinINet{[DllImport("wininet.dll")]public static extern bool InternetSetOption(IntPtr h,int o,IntPtr b,int l);}' -ErrorAction Stop } catch {}; [WinINet]::InternetSetOption([IntPtr]::Zero,39,[IntPtr]::Zero,0); [WinINet]::InternetSetOption([IntPtr]::Zero,37,[IntPtr]::Zero,0)`,
	).Run()
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
