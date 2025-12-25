package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
)

func getHookPath(storage string) string {
	return fmt.Sprintf("%s:snippets/%s", storage, config.ConstantHookScriptFilename)
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

func ensureProxmoxHost() error {
	if _, err := os.Stat(config.ConstantProxmoxConfigDir); os.IsNotExist(err) {
		return fmt.Errorf("this tool must be run on a Proxmox VE host (%s not found)", config.ConstantProxmoxConfigDir)
	}
	return nil
}
