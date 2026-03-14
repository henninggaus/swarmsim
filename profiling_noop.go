//go:build !profile

package main

// ToggleProfile is a no-op without the profile build tag.
func ToggleProfile() {}

// StartProfile is a no-op without the profile build tag.
func StartProfile() {}

// StopProfile is a no-op without the profile build tag.
func StopProfile() {}
