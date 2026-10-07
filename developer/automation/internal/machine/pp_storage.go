package machine

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const PPSourceRoot = "src/ptsip/profiles"

type ppStorage struct{ Root, Catalog, HistoryRoot string }

func (s ppStorage) History(version string) string { return s.HistoryRoot + "/" + version }

// Registries predating the source move use the original v1 layout. New layout
// selection is explicit registry metadata, never filesystem discovery.
func ppStorageForRegistry(registry Object) (ppStorage, error) {
	legacy := ppStorage{"profiles", PPCatalog, "profiles/history"}
	value, present := registry["profile_source"]
	if !present {
		return legacy, nil
	}
	expected := Object{"root": PPSourceRoot, "catalog": PPSourceRoot + "/index.yaml", "history_root": PPSourceRoot + "/history", "history_distribution": "SOURCE_ONLY"}
	if !reflect.DeepEqual(Map(value), expected) {
		return ppStorage{}, fmt.Errorf("PP_STORAGE_BINDING_INVALID")
	}
	storage := ppStorage{PPSourceRoot, PPSourceRoot + "/index.yaml", PPSourceRoot + "/history"}
	for _, value := range List(registry["contracts"]) {
		row := Map(value)
		if baseline := Text(row["baseline"]); baseline != "" && baseline != storage.History(Text(row["version"])) {
			return ppStorage{}, fmt.Errorf("PP_BASELINE_STORAGE_MISMATCH")
		}
	}
	return storage, nil
}

func ppSnapshotStorage(source PPSnapshot) (ppStorage, error) {
	raw, err := source.ReadBytes(PPRegistry)
	if err != nil {
		return ppStorage{}, err
	}
	registry, err := ppYAML(raw, PPRegistry, false)
	if err != nil {
		return ppStorage{}, err
	}
	return ppStorageForRegistry(registry)
}

func (s PPGitSnapshot) files(root string) ([]string, error) {
	if !ppSafeAsset(root) {
		return nil, fmt.Errorf("PP_ASSET_PATH_ESCAPE: %s", root)
	}
	files := []string{}
	if s.Worktree {
		base := filepath.Join(s.Root, filepath.FromSlash(root))
		if _, err := os.Stat(base); os.IsNotExist(err) {
			return files, nil
		} else if err != nil {
			return nil, err
		}
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				rel, err := filepath.Rel(s.Root, path)
				if err != nil {
					return err
				}
				files = append(files, filepath.ToSlash(rel))
			}
			return nil
		})
		return files, err
	}
	var raw []byte
	var err error
	if s.Staged {
		raw, err = ppGit(s.Root, "ls-files", "-z", "--cached", "--", root)
	} else {
		raw, err = ppGit(s.Root, "ls-tree", "-r", "-z", "--name-only", s.Revision, "--", root)
	}
	if err != nil {
		return nil, err
	}
	for _, path := range strings.Split(string(raw), "\x00") {
		if path != "" {
			files = append(files, path)
		}
	}
	sort.Strings(files)
	return files, nil
}

// A storage move may change only the registered physical paths. Contract
// identity, lifecycle, operations, transitions and catalog entries stay exact.
func ppVerifyStorageMove(base, candidate PPGitSnapshot, left, right ppStorage) error {
	if left == right {
		return nil
	}
	if left.Root != "profiles" || right.Root != PPSourceRoot {
		return fmt.Errorf("PP_STORAGE_RELOCATION_INVALID")
	}
	oldRaw, err := base.ReadBytes(PPRegistry)
	if err != nil {
		return err
	}
	newRaw, err := candidate.ReadBytes(PPRegistry)
	if err != nil {
		return err
	}
	oldRegistry, err := ppYAML(oldRaw, PPRegistry, true)
	if err != nil {
		return err
	}
	newRegistry, err := ppYAML(newRaw, PPRegistry, true)
	if err != nil {
		return err
	}
	delete(newRegistry, "profile_source")
	for _, value := range List(newRegistry["contracts"]) {
		row := Map(value)
		if Text(row["baseline"]) != "" {
			row["baseline"] = left.History(Text(row["version"]))
		}
	}
	if !reflect.DeepEqual(oldRegistry, newRegistry) {
		return fmt.Errorf("PP_STORAGE_RELOCATION_SEMANTICS_CHANGED")
	}
	oldRaw, err = base.ReadBytes(left.Catalog)
	if err != nil {
		return err
	}
	newRaw, err = candidate.ReadBytes(right.Catalog)
	if err != nil {
		return err
	}
	oldCatalog, err := ppYAML(oldRaw, left.Catalog, true)
	if err != nil {
		return err
	}
	newCatalog, err := ppYAML(newRaw, right.Catalog, true)
	if err != nil {
		return err
	}
	if oldCatalog["root"] != left.Root || newCatalog["root"] != right.Root {
		return fmt.Errorf("PP_CATALOG_STORAGE_MISMATCH")
	}
	newCatalog["root"] = left.Root
	if !reflect.DeepEqual(oldCatalog, newCatalog) {
		return fmt.Errorf("PP_STORAGE_RELOCATION_SEMANTICS_CHANGED")
	}
	return nil
}

func ppVerifyHistory(base, candidate PPGitSnapshot, allowedVersion string) error {
	left, err := ppSnapshotStorage(base)
	if err != nil {
		return err
	}
	right, err := ppSnapshotStorage(candidate)
	if err != nil {
		return err
	}
	if err = ppVerifyStorageMove(base, candidate, left, right); err != nil {
		return err
	}
	oldFiles, err := base.files(left.HistoryRoot)
	if err != nil {
		return err
	}
	newFiles, err := candidate.files(right.HistoryRoot)
	if err != nil {
		return err
	}
	old := map[string]string{}
	for _, path := range oldFiles {
		old[strings.TrimPrefix(path, left.HistoryRoot+"/")] = path
	}
	for _, path := range newFiles {
		rel := strings.TrimPrefix(path, right.HistoryRoot+"/")
		prior, exists := old[rel]
		if !exists {
			if allowedVersion == "" || !strings.HasPrefix(rel, allowedVersion+"/") || strings.Contains(strings.TrimPrefix(rel, allowedVersion+"/"), "/") || !strings.HasSuffix(rel, ".ptsip.yaml") {
				return fmt.Errorf("HISTORICAL_BASELINE_MUTATION: %s", path)
			}
			continue
		}
		before, err := base.ReadBytes(prior)
		if err != nil {
			return err
		}
		after, err := candidate.ReadBytes(path)
		if err != nil {
			return err
		}
		if before == nil || after == nil || !bytes.Equal(before, after) {
			return fmt.Errorf("HISTORICAL_BASELINE_MUTATION: %s", path)
		}
		delete(old, rel)
	}
	if len(old) > 0 {
		return fmt.Errorf("HISTORICAL_BASELINE_MUTATION: removed history")
	}
	if right.Root == PPSourceRoot {
		obsolete, err := candidate.files("profiles/history")
		if err != nil {
			return err
		}
		if len(obsolete) > 0 {
			return fmt.Errorf("PP_OBSOLETE_HISTORY_PRESENT")
		}
	}
	return nil
}
