//go:build profile

package main

import (
	"fmt"
	"os"
	"runtime/pprof"
)

var profiling bool
var cpuProfileFile *os.File

// ToggleProfile starts or stops CPU profiling.
func ToggleProfile() {
	if profiling {
		StopProfile()
	} else {
		StartProfile()
	}
}

// StartProfile begins writing a CPU profile to cpu.prof.
func StartProfile() {
	var err error
	cpuProfileFile, err = os.Create("cpu.prof")
	if err != nil {
		fmt.Println("[PROFILE] Error creating cpu.prof:", err)
		return
	}
	if err := pprof.StartCPUProfile(cpuProfileFile); err != nil {
		fmt.Println("[PROFILE] Error starting CPU profile:", err)
		cpuProfileFile.Close()
		cpuProfileFile = nil
		return
	}
	profiling = true
	fmt.Println("[PROFILE] CPU profiling STARTED -> cpu.prof")
}

// StopProfile stops the CPU profile and closes the file.
func StopProfile() {
	if !profiling {
		return
	}
	pprof.StopCPUProfile()
	if err := cpuProfileFile.Close(); err != nil {
		fmt.Println("[PROFILE] Error closing cpu.prof:", err)
	}
	cpuProfileFile = nil
	profiling = false
	fmt.Println("[PROFILE] CPU profiling STOPPED")
}
