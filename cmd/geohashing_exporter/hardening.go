// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !linux

package main

import (
	"log"
	"runtime"
)

// toLeastPrivilege drops privileges by some OS-specific method.
func toLeastPrivilege() {
	log.Printf("Cannot reduce privileges on %s/%s", runtime.GOOS, runtime.GOARCH)
}
