//
// Copyright © 2025 Agora
// This file is part of TEN Framework, an open source project.
// Licensed under the Apache License, Version 2.0, with certain conditions.
// Refer to the "LICENSE" file in the root directory for more information.
//

package main

/*
#include <signal.h>
#include <stdint.h>

void __lsan_do_leak_check(void);

#if defined(_WIN32)
static int install_test_sigsegv_handler(void) {
	return -1;
}

static uintptr_t get_test_sigsegv_handler(void) {
	return 0;
}
#else
static void test_sigsegv_handler(int signo, siginfo_t *info, void *context) {
	(void)signo;
	(void)info;
	(void)context;
}

static int install_test_sigsegv_handler(void) {
	struct sigaction action = {0};
	action.sa_sigaction = test_sigsegv_handler;
	action.sa_flags = SA_SIGINFO;
	sigemptyset(&action.sa_mask);
	return sigaction(SIGSEGV, &action, NULL);
}

static uintptr_t get_test_sigsegv_handler(void) {
	struct sigaction action = {0};
	if (sigaction(SIGSEGV, NULL, &action) != 0) {
		return 0;
	}
	return (uintptr_t)action.sa_sigaction;
}
#endif
*/
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	ten "ten_framework/ten_runtime"
)

type defaultApp struct {
	ten.DefaultApp
}

func (p *defaultApp) OnInit(tenEnv ten.TenEnv) {
	tenEnv.LogDebug("onInit")
	tenEnv.OnInitDone()
}

func (p *defaultApp) OnDeinit(tenEnv ten.TenEnv) {
	tenEnv.LogDebug("onDeinit")
	tenEnv.OnDeinitDone()
}

func main() {
	if os.Getenv("TEN_TEST_SIGSEGV_HANDLER") == "1" {
		if C.install_test_sigsegv_handler() != 0 {
			os.Exit(10)
		}

		before := C.get_test_sigsegv_handler()
		_, err := ten.NewApp(&defaultApp{})
		if err != nil {
			os.Exit(11)
		}

		after := C.get_test_sigsegv_handler()
		if before != after {
			os.Exit(12)
		}
		os.Exit(0)
	}

	// test app
	app, err := ten.NewApp(&defaultApp{})
	if err != nil {
		fmt.Println("Failed to create app.")
	}

	app.Run(true)
	app.Wait()

	// A single GC is not enough; multiple rounds of GC are needed to clean up
	// as thoroughly as possible.
	for i := 0; i < 10; i++ {
		// Explicitly trigger GC to increase the likelihood of finalizer
		// execution.
		debug.FreeOSMemory()
		runtime.GC()

		// Wait for a short period to give the GC time to run.
		runtime.Gosched()
		time.Sleep(1 * time.Second)
	}

	// To detect memory leaks with ASan, need to enable the following cgo code.
	C.__lsan_do_leak_check()
}
