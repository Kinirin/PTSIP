// Package localstate selects the existing per-repository PTSIP state directory.
package localstate

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/text/cases"
)

func Directory(root, area string) (string, error) {
	if root == "" || !fs.ValidPath(area) || area == "." || strings.ContainsAny(area, "/\\:") {
		return "", fmt.Errorf("invalid repository root or state area")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository root must be a directory")
	}
	home, err := stateHome()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(cases.Fold().String(resolved)))
	return filepath.Join(home, area, fmt.Sprintf("%x", digest[:10])), nil
}

func stateHome() (string, error) {
	if override := os.Getenv("PTSIP_HOME"); override != "" {
		if override == "~" || strings.HasPrefix(override, "~/") || strings.HasPrefix(override, `~\`) {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			override = filepath.Join(home, strings.TrimLeft(override[1:], "/\\"))
		}
		return filepath.Abs(override)
	}
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, "PTSIP"), nil
		}
	}
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		if base := os.Getenv("XDG_STATE_HOME"); base != "" {
			return filepath.Join(base, "ptsip"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Local", "PTSIP"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "PTSIP"), nil
	default:
		return filepath.Join(home, ".local", "state", "ptsip"), nil
	}
}
