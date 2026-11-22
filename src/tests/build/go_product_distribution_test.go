package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGoProductDistribution(t *testing.T) {
	root := buildRepositoryRoot(t)
	workspace := t.TempDir()
	binary := filepath.Join(workspace, "ptsip-build")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	command := exec.Command("go", "-C", filepath.Join(root, "src", "build"), "build", "-mod=readonly", "-o", binary, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Go product builder compilation: %v\n%s", err, output)
	}
	t.Run("reproducible_product_source_at_head", func(t *testing.T) {
		first := filepath.Join(workspace, "one.tar.gz")
		second := filepath.Join(workspace, "two.tar.gz")
		output, err := exec.Command(binary, "source", "-root", root, "-output", first).CombinedOutput()
		if err != nil {
			t.Fatalf("source archive: %v\n%s", err, output)
		}
		var report struct {
			SourceRevision string `json:"source_revision"`
			ProductFiles   int    `json:"product_files"`
			ActiveProfile  string `json:"active_profile"`
		}
		if err := json.Unmarshal(output, &report); err != nil {
			t.Fatal(err)
		}
		if head := strings.TrimSpace(distributionGit(t, root, "rev-parse", "HEAD")); report.SourceRevision != head || report.ActiveProfile == "" {
			t.Fatalf("archive receipt does not identify current source and selected Profile: %s", output)
		}
		if output, err = exec.Command(binary, "source", "-root", root, "-output", second).CombinedOutput(); err != nil {
			t.Fatalf("rebuild source archive: %v\n%s", err, output)
		}
		one, err := os.ReadFile(first)
		if err != nil {
			t.Fatal(err)
		}
		two, err := os.ReadFile(second)
		if err != nil || !bytes.Equal(one, two) {
			t.Fatalf("source archive was not reproducible: %v", err)
		}
		reader, err := gzip.NewReader(bytes.NewReader(one))
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		archive := tar.NewReader(reader)
		entries := make(map[string]bool)
		for {
			entry, err := archive.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			name, valid := strings.CutPrefix(entry.Name, "ptsip-product-source/")
			parts := strings.Split(name, "/")
			engineBuildOutput := len(parts) >= 5 && strings.HasPrefix(name, "src/ptsip/engines/") && parts[4] == "target"
			if !valid || strings.Contains("/"+name, "/.ptsip/") || engineBuildOutput || strings.HasPrefix(name, "src/policy/") ||
				retiredPythonMember(strings.TrimPrefix(name, "src/")) ||
				strings.HasPrefix(name, "src/ptsip/compile_sealed_rbac/") || entries[name] {
				t.Fatalf("unexpected product archive entry %q", entry.Name)
			}
			entries[name] = true
		}
		for _, name := range []string{
			"src/ptsip/schemas/config.json",
			"src/ptsip/schemas/history/profile/project-profile-pp.1.01.schema.json",
			"src/ptsip/schemas/history/profile/project-profile-pp.1.02.schema.json",
			"src/ptsip/schemas/history/profile/project-profile-pp.2.00.schema.json",
			"src/ptsip/schemas/history/profile/ptsip-profile-pp-2.00.candidate.schema.json",
			"src/ptsip/clarification/answer.go",
			"src/ptsip/evidence/normalization.go",
			"src/ptsip/inspection/inventory.go",
			"src/ptsip/engines/index.json",
			"src/ptsip/engines/discovery/profile-support.json",
			"src/ptsip/engines/validation/profile-transitions.json",
		} {
			if !entries[name] {
				t.Errorf("required product asset missing from archive: %s", name)
			}
		}
		if len(entries) != report.ProductFiles {
			t.Fatalf("archive receipt counted %d files for %d archive entries", report.ProductFiles, len(entries))
		}
		if output, err := exec.Command(binary, "source", "-root", root, "-output", first).CombinedOutput(); err == nil {
			t.Fatalf("existing output was overwritten: %s", output)
		}
		if current, err := os.ReadFile(first); err != nil || !bytes.Equal(current, one) {
			t.Fatalf("existing output changed: %v", err)
		}
	})
	t.Run("explicit_checkout_root_required", func(t *testing.T) {
		output, err := exec.Command(binary, "check", "-root", filepath.Join(root, "src")).CombinedOutput()
		if err == nil || !strings.Contains(string(output), "root must select the checkout") {
			t.Fatalf("nested checkout was accepted: %v %s", err, output)
		}
	})
	t.Run("stale_engine_pin_blocks_output", func(t *testing.T) {
		checkout := t.TempDir()
		distributionGit(t, checkout, "init", "--template=")
		for name, payload := range map[string]string{
			"README.md":                                  "test\n",
			"pyproject.toml":                             "[project]\nname='test'\n",
			"License-Authority/LICENSE.md":               "test\n",
			"src/VERSION":                                "test\n",
			"src/ptsip/engines/index.json":               `{"official_engine_entries":[{"kind":"PROJECTION"}]}`,
			"src/ptsip/engines/projection/manifest.json": `{"implementation_version":"test"}`,
			"src/ptsip/engines/contracts/engines_registries/derived-snapshot.json": `{"source_refs":[{"path":"src/ptsip/engines/projection/manifest.json","sha256":"` + strings.Repeat("0", 64) + `"}]}`,
		} {
			file := filepath.Join(checkout, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(payload), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		distributionGit(t, checkout, "add", "--", ".")
		distributionGit(t, checkout, "commit", "-m", "source fixture")
		destination := filepath.Join(workspace, "invalid.tar.gz")
		output, err := exec.Command(binary, "source", "-root", checkout, "-output", destination).CombinedOutput()
		if err == nil || !strings.Contains(string(output), "registered SHA-256 mismatch for src/ptsip/engines/projection/manifest.json") {
			t.Fatalf("stale source accepted: %v %s", err, output)
		}
		if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("invalid source produced output: %v", err)
		}
	})
}

func buildRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the build verification module")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func distributionGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	config := []string{"-C", root, "-c", "user.name=PTSIP Test", "-c", "user.email=ptsip@example.test", "-c", "commit.gpgSign=false"}
	output, err := exec.Command("git", append(config, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}
