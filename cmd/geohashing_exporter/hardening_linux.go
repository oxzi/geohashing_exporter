// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"log"
	"os"
	"strings"

	_ "unsafe" // go:linkname

	"github.com/landlock-lsm/go-landlock/landlock"
	llsys "github.com/landlock-lsm/go-landlock/landlock/syscall"

	syscallset "github.com/oxzi/syscallset-go"
)

//go:linkname zoneSources time.zoneSources
var zoneSources []string

// toLeastPrivilegeLandlock limits with Linux' Landlock.
//
// Turns out, Golang's time package loads timezone information during its
// initialization functions and sources those from different locations. On top,
// there might be an additional path from downstream distribution patches.
//
//   https://cs.opensource.google/go/go/+/master:src/time/zoneinfo_unix.go;bpv=0;bpt=0
//
// Thus, we access this not exported variable, filter for path validity as
// go-landlock returns an error otherwise. I just want to have unveil(2)..
func toLeastPrivilegeLandlock() {
	_, err := llsys.LandlockGetABIVersion()
	if err != nil {
		log.Printf("Landlock is not supported.")
		return
	}

	allowedZoneSourceDirs := []string{}
	allowedZoneSourceFiles := []string{}
	for _, zoneSource := range zoneSources {
		info, err := os.Stat(zoneSource)
		if err != nil {
			continue
		} else if info.IsDir() {
			allowedZoneSourceDirs = append(allowedZoneSourceDirs, zoneSource)
		} else {
			allowedZoneSourceFiles = append(allowedZoneSourceFiles, zoneSource)
		}
	}

	err = landlock.V2.BestEffort().RestrictPaths(
		// Golang's time package
		landlock.ROFiles("/etc/localtime"),
		landlock.ROFiles(allowedZoneSourceFiles...),
		landlock.RODirs(allowedZoneSourceDirs...),

		// Golang's net/http HTTP server and lookup
		landlock.RODirs("/proc/sys"),
		landlock.ROFiles(
			"/etc/hosts",
			"/etc/nsswitch.conf",
			"/etc/resolv.conf",
		),
	)
	if err != nil {
		log.Fatalf("Cannot apply Landlock filter: %v", err)
	}
}

// toLeastPrivilegeSeccompBpf limits with Linux' seccomp-bpf.
func toLeastPrivilegeSeccompBpf() {
	if !syscallset.IsSupported() {
		log.Print("seccomp-bpf or syscallset is not supported.")
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
		"@system-service",
		"fadvise64 ioctl madvise mremap sysinfo",
	}
	err := syscallset.LimitTo(strings.Join(filter, " "))
	if err != nil {
		log.Fatalf("Cannot apply syscallset resp. seccomp-bpf filter: %v", err)
	}
}

// toLeastPrivilege is achieved on a Linux with Landlock and seccomp-bpf.
func toLeastPrivilege() {
	toLeastPrivilegeLandlock()
	toLeastPrivilegeSeccompBpf()
}
