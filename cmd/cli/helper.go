package main

import (
	"fmt"
	"path/filepath"
	"strconv"
)

const hookscriptFile = "proxmox-cpu-affinity-hook"

func getHookPath(storage string) string {
	return fmt.Sprintf("%s:snippets/%s", storage, hookscriptFile)
}

func isValidStorage(s string) bool {
	// Fails if s is empty, is ".", "..", or contains a separator
	if s == "" || s == "." || s == ".." {
		return false
	}
	// If Base(s) changes the string, it meant there were separators
	return filepath.Base(s) == s
}

func isNumeric(s string) bool {
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}
