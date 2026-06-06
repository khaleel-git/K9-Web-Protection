//go:build windows

package main

import (
	"sync/atomic"
	"syscall"
)

var g_shutdownPending int32

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleCtrlHandler = kernel32.NewProc("SetConsoleCtrlHandler")
)

// systemShuttingDown returns true when Windows is logging off or shutting down.
func systemShuttingDown() bool {
	return atomic.LoadInt32(&g_shutdownPending) == 1
}

// registerShutdownObserver installs a Windows console control handler so that
// CTRL_SHUTDOWN_EVENT (6) and CTRL_LOGOFF_EVENT (5) set g_shutdownPending,
// allowing the app to close without prompting for a password.
func registerShutdownObserver() {
	const (
		CTRL_SHUTDOWN_EVENT = 6
		CTRL_LOGOFF_EVENT   = 5
	)
	cb := syscall.NewCallback(func(ctrlType uint32) uintptr {
		if ctrlType == CTRL_SHUTDOWN_EVENT || ctrlType == CTRL_LOGOFF_EVENT {
			atomic.StoreInt32(&g_shutdownPending, 1)
			return 1
		}
		return 0
	})
	procSetConsoleCtrlHandler.Call(cb, 1)
}
