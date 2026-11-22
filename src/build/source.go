package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const archivePrefix = "ptsip-product-source/"

type sourceFile struct {
	Data []byte
	Mode int64
}

type productSource struct {
	Revision string
	Files    map[string]sourceFile
}

func productPath(name string) bool {
	if path.Clean(name) != name || strings.ContainsAny(name, "\\:") || path.IsAbs(name) {
		return false
	}
	if name == "README.md" || name == "pyproject.toml" || name == "src/VERSION" {
		return true
	}
	parts := strings.Split(name, "/")
	if slices.Contains(parts, "..") || slices.Contains(parts, ".ptsip") || slices.Contains(parts, "__pycache__") {
		return false
	}
	if strings.HasPrefix(name, "License-Authority/") || strings.HasPrefix(name, "src/build/") {
		return true
	}
	if !strings.HasPrefix(name, "src/ptsip/") {
		return false
	}
	if strings.HasPrefix(name, "src/ptsip/compile_sealed_rbac/") {
		return false
	}
	if len(parts) >= 5 && parts[2] == "engines" && parts[4] == "target" {
		return false
	}
	return true
}

func runGit(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", root, "--no-replace-objects"}, args...)...)
	out, err := command.Output()
	if err != nil {
		var failure *exec.ExitError
		if errors.As(err, &failure) {
			return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(failure.Stderr)))
		}
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return out, nil
}

func loadGitSource(root string) (productSource, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return productSource{}, err
	}
	if top, err := runGit(root, "rev-parse", "--show-toplevel"); err != nil {
		return productSource{}, err
	} else {
		checkout := strings.TrimSpace(string(top))
		relative, err := filepath.Rel(root, checkout)
		if err != nil || relative != "." {
			return productSource{}, fmt.Errorf("root must select the checkout at %s", checkout)
		}
	}
	revisionBytes, err := runGit(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return productSource{}, err
	}
	revision := strings.TrimSpace(string(revisionBytes))
	arguments := []string{"archive", "--format=tar", revision, "--", "src/ptsip", "src/VERSION", "README.md", "License-Authority", "pyproject.toml"}
	build, err := runGit(root, "ls-tree", "-d", "--name-only", revision, "--", "src/build")
	if err != nil {
		return productSource{}, err
	}
	if strings.TrimSpace(string(build)) == "src/build" {
		arguments = append(arguments, "src/build")
	}
	archive, err := runGit(root, append(arguments, ":(exclude)src/ptsip/compile_sealed_rbac")...)
	if err != nil {
		return productSource{}, err
	}
	files, err := readSourceTar(bytes.NewReader(archive))
	return productSource{Revision: revision, Files: files}, err
}

func readSourceTar(reader io.Reader) (map[string]sourceFile, error) {
	files := make(map[string]sourceFile)
	archive := tar.NewReader(reader)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeDir || header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if !productPath(header.Name) || header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("unexpected product archive entry %q", header.Name)
		}
		if _, exists := files[header.Name]; exists {
			return nil, fmt.Errorf("duplicate product archive entry %q", header.Name)
		}
		data, err := io.ReadAll(archive)
		if err != nil {
			return nil, err
		}
		files[header.Name] = sourceFile{Data: data, Mode: header.Mode & 0o777}
	}
}

func writeSourceArchive(source productSource, destination string) error {
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	archiveErr := archiveSource(file, source)
	if err := errors.Join(archiveErr, file.Close()); err != nil {
		return errors.Join(err, os.Remove(destination))
	}
	return nil
}

func archiveSource(writer io.Writer, source productSource) error {
	compressed := gzip.NewWriter(writer)
	archive := tar.NewWriter(compressed)
	names := make([]string, 0, len(source.Files))
	for name := range source.Files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		file := source.Files[name]
		header := &tar.Header{Name: archivePrefix + name, Mode: file.Mode, Size: int64(len(file.Data)), Typeflag: tar.TypeReg}
		if err := archive.WriteHeader(header); err != nil {
			return err
		}
		if _, err := archive.Write(file.Data); err != nil {
			return err
		}
	}
	return errors.Join(archive.Close(), compressed.Close())
}
