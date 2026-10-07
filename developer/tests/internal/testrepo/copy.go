package testrepo

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func CopyTree(t *testing.T, destination *Repository, refs ...string) {
	t.Helper()
	source := Open(Root(t))
	for _, ref := range refs {
		root, err := source.Path(ref)
		if err != nil {
			t.Fatal(err)
		}
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			t.Fatalf("registered fixture root %s: %v", ref, err)
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && (entry.Name() == "__pycache__" || entry.Name() == ".cache" || entry.Name() == "legacy") {
				return filepath.SkipDir
			}
			relative, err := filepath.Rel(source.Root, path)
			if err != nil {
				return err
			}
			target, err := destination.Path(filepath.ToSlash(relative))
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, raw, 0644)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
