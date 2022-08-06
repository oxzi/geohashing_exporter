// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"log"
	"strings"

	syscallset "github.com/oxzi/syscallset-go"
)

// toLeastPrivilege with the help of Linux' seccomp-bpf.
func toLeastPrivilege() {
	if !syscallset.IsSupported() {
		log.Print("seccomp-bpf or syscallset is not supported")
		return
	}

	filter := []string{
		"@basic-io",
		"@file-system", // required for /etc/localtime and some /proc/sys/net/…
		"@io-event",
		"@ipc",
		"@network-io",
		"@process ~execve ~execveat ~fork ~kill",
		"@signal",
		"fadvise64 ioctl madvise mremap sysinfo",
	}
	err := syscallset.LimitTo(strings.Join(filter, " "))
	if err != nil {
		log.Fatalf("Cannot apply syscallset resp. seccomp-bpf filter: %v", err)
	}
}
