package enforce

import (
	"errors"
	"reflect"
	"sync"

	"k10webprotection/internal/config"
	"k10webprotection/internal/hosts"
	"k10webprotection/internal/proxy"
)

type Store interface {
	Save() error
	PolicyView() config.PolicyView
	FocusBlocks(host string) bool
	LoadFailed() bool
}

type Proxy interface {
	SetPolicy(pol proxy.Policy) (closed int)
}

type Hosts interface {
	Apply(plan hosts.Plan) (hosts.Result, error)
}

type HostsFunc func(plan hosts.Plan) (hosts.Result, error)

func (f HostsFunc) Apply(plan hosts.Plan) (hosts.Result, error) { return f(plan) }

var (
	ErrProtectionOff      = errors.New("hosts not updated: protection is off")
	ErrSettingsUnreadable = errors.New("hosts not updated: the settings file could not be read")
)

type ApplyStatus struct {
	View          config.PolicyView
	ConfigErr     error
	HostsErr      error
	HostsSkipped  error // why the hosts file was deliberately left alone
	HostsApplied  bool  // false when the hosts file was left alone
	ClosedTunnels int
	Hosts         hosts.Result
}

func (s ApplyStatus) Err() error { return errors.Join(s.ConfigErr, s.HostsErr) }

type Applier struct {
	mu      sync.Mutex
	store   Store
	proxy   Proxy
	hosts   Hosts
	enabled func() bool
	// declined is the plan whose UAC prompt was refused; it is not prompted for again.
	declined *hosts.Plan
	clearErr error // a failed ClearHosts, reported until hosts is written again

	lastMu sync.Mutex
	last   ApplyStatus
}

func New(store Store, px Proxy, h Hosts, enabled func() bool) *Applier {
	return &Applier{store: store, proxy: px, hosts: h, enabled: enabled}
}

// Apply saves the config, then pushes it to the proxy and, while protection is on, the hosts file.
func (a *Applier) Apply() ApplyStatus { return a.run(true, false) }

// Enforce is Apply without saving.
func (a *Applier) Enforce() ApplyStatus { return a.run(false, false) }

// Retry is Enforce that also prompts again for a hosts plan whose elevation was declined.
func (a *Applier) Retry() ApplyStatus { return a.run(false, true) }

// SetStore swaps in a reloaded config; call Retry or Enforce afterwards.
func (a *Applier) SetStore(s Store) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.store = s
	a.declined = nil
}

// ClearHosts removes the managed hosts sections when protection is turned off.
func (a *Applier) ClearHosts() error { return a.clearHosts(nil) }

// Disable clears hosts before stopping protection. stop must mark protection
// off, and must not call back into the applier. It runs only after a successful
// cleanup, under the same lock as Apply, so a concurrent rule edit cannot
// restore the hosts entries before protection becomes off.
func (a *Applier) Disable(stop func()) error { return a.clearHosts(stop) }

func (a *Applier) clearHosts(stop func()) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.declined = nil
	_, err := a.hosts.Apply(hosts.Plan{})
	a.clearErr = err
	if err == nil && stop != nil {
		stop()
	}
	a.lastMu.Lock()
	a.last.Hosts, a.last.HostsErr, a.last.HostsApplied = hosts.Result{}, err, false
	a.last.HostsSkipped = nil
	if err == nil || (a.enabled != nil && !a.enabled()) {
		a.last.HostsSkipped = ErrProtectionOff
	}
	a.lastMu.Unlock()
	return err
}

func (a *Applier) Last() ApplyStatus {
	a.lastMu.Lock()
	defer a.lastMu.Unlock()
	return a.last
}

func (a *Applier) run(save, retry bool) ApplyStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	var st ApplyStatus
	if save {
		st.ConfigErr = a.store.Save()
	}
	st.View = a.store.PolicyView()
	st.ClosedTunnels = a.proxy.SetPolicy(PolicyFor(st.View, a.store.FocusBlocks))
	switch plan := PlanFor(st.View); {
	case a.store.LoadFailed():
		// the defaults in memory are not the user's rules; keep the hosts file as it is
		st.HostsSkipped = ErrSettingsUnreadable
	case a.enabled != nil && !a.enabled():
		st.HostsSkipped, st.HostsErr = ErrProtectionOff, a.clearErr
	case !retry && a.declined != nil && reflect.DeepEqual(*a.declined, plan):
		st.HostsErr = hosts.ErrElevationCancelled
	default:
		st.HostsApplied = true
		st.Hosts, st.HostsErr = a.hosts.Apply(plan)
		a.declined, a.clearErr = nil, nil
		if errors.Is(st.HostsErr, hosts.ErrElevationCancelled) {
			a.declined = &plan
		}
	}

	a.lastMu.Lock()
	a.last = st
	a.lastMu.Unlock()
	return st
}

func PolicyFor(v config.PolicyView, focus func(host string) bool) proxy.Policy {
	return proxy.Policy{
		Block:             proxyRules(v.Block),
		Allow:             proxyRules(v.Allow),
		FilterLevel:       v.FilterLevel,
		BlockAdultContent: v.BlockAdultContent,
		BlockImageSearch:  v.BlockImageSearch,
		BlockYouTube:      v.BlockYouTube,
		SafeSearch:        v.SafeSearch,
		Keywords:          append([]string(nil), v.Keywords...),
		Focus:             focus,
	}
}

// PlanFor leaves out names the proxy exempts as built-in services, so hosts never blocks them either.
func PlanFor(v config.PolicyView) hosts.Plan {
	p := hosts.Plan{SafeSearch: v.SafeSearch}
	for _, r := range v.Block {
		if proxy.IsBuiltinAllowed(r.Domain) {
			continue
		}
		p.Block = append(p.Block, hosts.Rule{Host: r.Domain, IncludeSubdomains: r.IncludeSubdomains})
		if www := "www." + r.Domain; r.IncludeSubdomains && proxy.IsBuiltinAllowed(www) {
			p.Allow = append(p.Allow, hosts.Rule{Host: www})
		}
	}
	for _, r := range v.Allow {
		p.Allow = append(p.Allow, hosts.Rule{Host: r.Domain, IncludeSubdomains: r.IncludeSubdomains})
	}
	return p
}

func proxyRules(in []config.DomainRule) []proxy.Rule {
	out := make([]proxy.Rule, 0, len(in))
	for _, r := range in {
		out = append(out, proxy.Rule{Host: r.Domain, IncludeSubdomains: r.IncludeSubdomains})
	}
	return out
}
