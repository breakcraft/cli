package test

import (
	"runtime"
	"strings"
)

// NormalizeRelativePath converts a POSIX-style relative path to the OS-native format.
func NormalizeRelativePath(posixPath string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(posixPath, "/", "\\")
	}
	return posixPath
}
