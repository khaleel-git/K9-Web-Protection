package enforce

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"k10webprotection/internal/hosts"
)

func TestDisableWaitsForSuccessfulCleanup(t *testing.T) {
	for _, cause := range []error{hosts.ErrElevationCancelled, hosts.ErrVerifyFailed, errors.New("write failed")} {
		t.Run(cause.Error(), func(t *testing.T) {
			rec, _, _, h, a := setup(true)
			enabled := true
			a.enabled = func() bool { return enabled }
			a.Apply()
			rec.calls = nil
			h.err = cause
			stops := 0
			stop := func() {
				rec.add("stop")
				stops++
				enabled = false
			}
			if err := a.Disable(stop); !errors.Is(err, cause) {
				t.Fatalf("cleanup error lost: %v", err)
			}
			if stops != 0 || !enabled || !reflect.DeepEqual(rec.calls, []string{"hosts"}) {
				t.Fatalf("stopped before a successful cleanup: stops=%d, calls=%v", stops, rec.calls)
			}
			last := a.Last()
			if !errors.Is(last.HostsErr, cause) || last.HostsSkipped != nil || last.HostsApplied {
				t.Fatalf("failed disable status is misleading: %+v", last)
			}
			if p := h.plans[len(h.plans)-1]; !reflect.DeepEqual(p, hosts.Plan{}) {
				t.Fatalf("Disable did not request an empty plan: %+v", p)
			}

			h.err = nil
			rec.calls = nil
			if err := a.Disable(stop); err != nil {
				t.Fatal(err)
			}
			if stops != 1 || enabled || !reflect.DeepEqual(rec.calls, []string{"hosts", "stop"}) {
				t.Fatalf("retry order: stops=%d, calls=%v", stops, rec.calls)
			}
			last = a.Last()
			if last.HostsErr != nil || !errors.Is(last.HostsSkipped, ErrProtectionOff) {
				t.Fatalf("successful retry status: %+v", last)
			}
			calls := len(h.plans)
			if st := a.Apply(); st.HostsErr != nil || len(h.plans) != calls {
				t.Fatalf("a later rule edit restored hosts or the stale error: %+v", st)
			}
		})
	}
}

func TestDisableSerializesCleanupAndStopWithApply(t *testing.T) {
	_, _, _, h, a := setup(true)
	var enabled atomic.Bool
	enabled.Store(true)
	a.enabled = enabled.Load
	a.Apply()
	enteredStop := make(chan struct{})
	releaseStop := make(chan struct{})
	disabled := make(chan error, 1)
	go func() {
		disabled <- a.Disable(func() {
			// Assert the lock invariant directly rather than depending on a sleep
			// to make the competing Apply run inside the vulnerable window.
			if a.mu.TryLock() {
				a.mu.Unlock()
				t.Error("stop callback ran outside the enforcement lock")
			}
			close(enteredStop)
			<-releaseStop
			enabled.Store(false)
		})
	}()
	select {
	case <-enteredStop:
	case <-time.After(5 * time.Second):
		close(releaseStop)
		t.Fatal("Disable did not reach the stop callback")
	}
	applying := make(chan struct{})
	applied := make(chan ApplyStatus, 1)
	go func() {
		close(applying)
		applied <- a.Apply()
	}()
	<-applying
	close(releaseStop)
	select {
	case err := <-disabled:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Disable deadlocked")
	}
	select {
	case st := <-applied:
		if st.HostsApplied || !errors.Is(st.HostsSkipped, ErrProtectionOff) {
			t.Fatalf("Apply restored hosts after cleanup: %+v", st)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Apply deadlocked")
	}
	if len(h.plans) != 2 || !reflect.DeepEqual(h.plans[1], hosts.Plan{}) {
		t.Fatalf("expected the original plan followed only by cleanup: %+v", h.plans)
	}
}
