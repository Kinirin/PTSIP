package releasetest

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func toolSource(root, source string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "ptsip-tool-source-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	archivePath := filepath.Join(dir, "source.zip")
	if err = command(root, nil, "git", "-c", "core.autocrlf=false", "-c", "core.eol=lf", "archive", "--format=zip", "--output", archivePath, source); err != nil {
		cleanup()
		return "", func() {}, err
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	defer archive.Close()
	stage := filepath.Join(dir, "checkout")
	if err = os.MkdirAll(stage, 0755); err != nil {
		cleanup()
		return "", func() {}, err
	}
	for _, f := range archive.File {
		ref := strings.TrimSuffix(f.Name, "/")
		if ref == "" {
			continue
		}
		p, e := release.SafePath(stage, ref)
		if e != nil || f.Mode()&os.ModeSymlink != 0 {
			cleanup()
			return "", func() {}, fmt.Errorf("unsafe canonical Tool source entry")
		}
		if f.FileInfo().IsDir() {
			e = os.MkdirAll(p, 0755)
		} else {
			if f.UncompressedSize64 > 128*1024*1024 {
				cleanup()
				return "", func() {}, fmt.Errorf("Tool source entry too large")
			}
			e = os.MkdirAll(filepath.Dir(p), 0755)
			if e == nil {
				stream, x := f.Open()
				if x != nil {
					cleanup()
					return "", func() {}, x
				}
				b, x := io.ReadAll(io.LimitReader(stream, 128*1024*1024+1))
				stream.Close()
				if x != nil {
					cleanup()
					return "", func() {}, x
				}
				e = os.WriteFile(p, b, 0644)
			}
		}
		if e != nil {
			cleanup()
			return "", func() {}, e
		}
	}
	// Archive materialization and build hosts cannot own license identity.
	// Materialize this owner plane directly from the pinned Git blobs, including
	// on Windows with a checkout configured for automatic CRLF conversion.
	refs, err := release.Run(root, "git", "ls-tree", "-r", "--name-only", source, "--", "License-Authority")
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	for _, ref := range strings.Fields(string(refs)) {
		p, e := release.SafePath(stage, ref)
		if e != nil {
			cleanup()
			return "", func() {}, e
		}
		b, e := release.Run(root, "git", "show", source+":"+ref)
		if e != nil {
			cleanup()
			return "", func() {}, e
		}
		if e = os.MkdirAll(filepath.Dir(p), 0755); e == nil {
			e = os.WriteFile(p, b, 0644)
		}
		if e != nil {
			cleanup()
			return "", func() {}, e
		}
	}
	return stage, cleanup, nil
}

func verifyTool(root, out, version, source string) error {
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	wheel := ""
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".whl") {
			if wheel != "" {
				return fmt.Errorf("multiple Tool wheels")
			}
			wheel = filepath.Join(out, e.Name())
		}
	}
	if wheel == "" {
		return fmt.Errorf("Tool wheel missing")
	}
	r, err := zip.OpenReader(wheel)
	if err != nil {
		return err
	}
	defer r.Close()
	license, err := release.Run(root, "git", "show", source+":License-Authority/LICENSE.md")
	if err != nil {
		return err
	}
	foundMetadata, foundLicense := false, false
	for _, f := range r.File {
		if forbiddenToolMember(f.Name) {
			return fmt.Errorf("Tool contains history or native build output: %s", f.Name)
		}
		if strings.HasSuffix(f.Name, ".dist-info/METADATA") || strings.HasSuffix(f.Name, "/licenses/License-Authority/LICENSE.md") {
			stream, e := f.Open()
			if e != nil {
				return e
			}
			b, e := io.ReadAll(stream)
			stream.Close()
			if e != nil {
				return e
			}
			if strings.HasSuffix(f.Name, ".dist-info/METADATA") {
				foundMetadata = toolMetadataMatches(b, version)
			} else {
				foundLicense = bytes.Equal(b, license)
			}
		}
	}
	if !foundMetadata || !foundLicense {
		return fmt.Errorf("Tool metadata or canonical license bytes differ (metadata=%t license=%t)", foundMetadata, foundLicense)
	}
	env, err := os.MkdirTemp("", "ptsip-tool-consumer-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(env)
	if err = command(root, nil, "python", "-m", "venv", env); err != nil {
		return err
	}
	python, cli := filepath.Join(env, "bin/python"), filepath.Join(env, "bin/ptsip")
	if runtime.GOOS == "windows" {
		python = filepath.Join(env, "Scripts/python.exe")
		cli = filepath.Join(env, "Scripts/ptsip.exe")
	}
	if err = command(root, nil, python, "-m", "pip", "install", "--disable-pip-version-check", wheel); err != nil {
		return err
	}
	data, err := release.Run(env, cli, "--version")
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), version) {
		return fmt.Errorf("installed CLI version differs")
	}
	_, err = release.Run(env, cli, "spec")
	return err
}

func forbiddenToolMember(name string) bool {
	return strings.HasPrefix(name, "ptsip/profiles/history/") || strings.HasSuffix(name, ".exe") || (strings.HasPrefix(name, "ptsip/engines/") && strings.Contains(name, "/target/"))
}

func toolMetadataMatches(data []byte, version string) bool {
	message, err := mail.ReadMessage(bytes.NewReader(data))
	return err == nil && strings.EqualFold(message.Header.Get("Name"), "PTSIP") && message.Header.Get("Version") == version
}
