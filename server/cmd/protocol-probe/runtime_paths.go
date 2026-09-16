package main

import (
	"os"
	"path/filepath"
	"strings"
)

const serverRootEnvironment = "NINEYIN_SERVER_ROOT"

var (
	runtimeProjectRoot       = locateRuntimeProjectRoot()
	defaultModernShareRoot   = runtimeProjectPath("resources", "modern", "share")
	defaultModernActionRoot  = runtimeProjectPath("resources", "modern", "ini", "action")
	defaultModernTextRoot    = runtimeProjectPath("resources", "modern", "text")
	defaultLegacyNPCFuncPath = runtimeProjectPath("resources", "legacy", "Npc_Func.xml")
)

func runtimeProjectPath(parts ...string) string {
	all := append([]string{runtimeProjectRoot}, parts...)
	return filepath.Join(all...)
}
func locateRuntimeProjectRoot() string {
	if root := strings.TrimSpace(os.Getenv("NINEYIN_SERVER_ROOT")); root != "" {
		return filepath.Clean(root)
	}
	starts := make([]string, 0, 2)
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	for _, start := range starts {
		if root, ok := findRuntimeProjectRoot(start); ok {
			return root
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Clean(cwd)
	}
	return "."
}
func findRuntimeProjectRoot(start string) (string, bool) {
	current := filepath.Clean(start)
	for {
		if isRuntimeProjectRoot(current) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}
func isRuntimeProjectRoot(path string) bool {
	if info, err := os.Stat(filepath.Join(path, "resources", "modern", "share")); err == nil && info.IsDir() {
		return true
	}
	info, err := os.Stat(filepath.Join(path, "go.mod"))
	return err == nil && !info.IsDir()
}
