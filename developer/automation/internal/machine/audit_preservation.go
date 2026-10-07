package machine

import (
	"bytes"
	"os"
	"reflect"
	"sort"
	"strings"
)

func (r *Repository) VerifyPreservedFiles(preserved []any, activation Object) error {
	changes := Map(activation["authorized_preservation_changes"])
	for _, raw := range preserved {
		record := Map(raw)
		relative := Text(record["path"])
		operation := Text(changes[relative])
		path, err := r.Path(relative)
		if err != nil {
			return err
		}
		if operation == "DELETE" {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				return Fail("RETIRED_SOURCE_PRESENT", relative)
			}
			continue
		}
		if operation == "MODIFY" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if operation == "MODIFY_LIFECYCLE_ONLY" {
			content = bytes.Replace(content, []byte("status: RETIRED"), []byte("status: ACTIVE"), 1)
		}
		if LFDigest(content) != Text(record["lf_sha256"]) {
			return Fail("PRESERVED_SOURCE_CHANGED", relative)
		}
	}
	return nil
}

func (r *Repository) ExactAuditScope(record Object, expected []string, check bool) error {
	if !check {
		return nil
	}
	head, err := r.GitOutput("rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != record["base_head"] {
		return Fail("BASE_HEAD_CHANGED", "audit scope base differs from HEAD")
	}
	changed, err := r.GitOutput("diff", "--name-only", "HEAD")
	if err != nil {
		return err
	}
	untracked, err := r.GitOutput("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return err
	}
	actual := []string{}
	for _, path := range strings.Split(changed+"\n"+untracked, "\n") {
		if path != "" {
			actual = append(actual, path)
		}
	}
	actual = UniqueStrings(actual)
	expected = UniqueStrings(expected)
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		return Fail("EXACT_CHANGE_SCOPE_MISMATCH", "worktree paths differ from the approved audit scope")
	}
	return nil
}
