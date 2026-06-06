package main

// This file uses CGO to observe NSWorkspaceWillPowerOffNotification, which
// macOS fires before applicationShouldTerminate: during shutdown, restart,
// and logout. Setting the flag here lets OnBeforeClose in main.go exit
// immediately instead of showing the password modal during system events.

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#import <stdatomic.h>

static atomic_int g_shutdown_pending = 0;

void k10_register_shutdown_observer(void) {
    [[[NSWorkspace sharedWorkspace] notificationCenter]
        addObserverForName:NSWorkspaceWillPowerOffNotification
        object:nil
        queue:nil
        usingBlock:^(NSNotification *note) {
            atomic_store_explicit(&g_shutdown_pending, 1, memory_order_release);
        }];
}

int k10_shutdown_pending(void) {
    return atomic_load_explicit(&g_shutdown_pending, memory_order_acquire);
}
*/
import "C"

func registerShutdownObserver() {
	C.k10_register_shutdown_observer()
}

func systemShuttingDown() bool {
	return C.k10_shutdown_pending() != 0
}
