//go:build !darwin

package main

func notifyDarwin(_, _ string) bool {
	return false
}
