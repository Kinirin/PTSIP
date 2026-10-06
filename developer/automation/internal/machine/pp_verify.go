package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

func PPAuthorityFingerprint(state *PPAuthorityState) string {
	entries := Object{}
	for id, row := range state.Entries {
		res := Text(row["resource"])
		entries[id] = Object{"resource": res, "contract": row["contract"], "semantic_profile": state.Semantic[res], "declared_version": state.Declared[res]}
	}
	raw, _ := json.Marshal(Object{"current": state.Current, "entries": entries, "schema_semantic": state.SchemaSemantic})
	return string(raw)
}
func PPComparableParents(states []*PPAuthorityState) error {
	if len(states) < 2 {
		return nil
	}
	first := PPAuthorityFingerprint(states[0])
	for _, state := range states[1:] {
		if PPAuthorityFingerprint(state) != first {
			return fmt.Errorf("DIVERGENT_PARENT_PP_AUTHORITY: %s", state.Label)
		}
	}
	return nil
}
func ValidatePPSnapshot(source PPSnapshot) (*PPAuthorityState, error) {
	state, err := LoadPPAuthority(source)
	if err != nil {
		return nil, err
	}
	canonical, err := source.ReadBytes(PPRegistry)
	if err != nil {
		return nil, err
	}
	embedded, err := source.ReadBytes(PPEmbeddedRegistry)
	if err != nil {
		return nil, err
	}
	if canonical == nil || !bytes.Equal(canonical, embedded) {
		return nil, fmt.Errorf("PP_RUNTIME_REGISTRY_PROJECTION_MISMATCH: %s", source.Label())
	}
	registry, err := ppYAML(canonical, PPRegistry, true)
	if err != nil {
		return nil, err
	}
	if List(registry["contracts"]) == nil || List(registry["transitions"]) == nil {
		return nil, fmt.Errorf("PP_REGISTRY_INVALID")
	}
	rows := []Object{}
	for _, raw := range List(registry["contracts"]) {
		row := Map(raw)
		if row["version"] == state.Current {
			rows = append(rows, row)
		}
	}
	if len(rows) != 1 || rows[0]["lifecycle"] != "CURRENT" {
		return nil, fmt.Errorf("PP_REGISTRY_CURRENT_INVALID")
	}
	schemaPath, baseline := Text(rows[0]["schema"]), Text(rows[0]["baseline"])
	if schemaPath == "" || baseline == "" || !ppSafeAsset(baseline) {
		return nil, fmt.Errorf("PP_REGISTRY_CURRENT_INVALID")
	}
	schema, err := source.ReadBytes(schemaPath)
	if err != nil {
		return nil, err
	}
	embeddedSchema, err := source.ReadBytes("src/ptsip/specdata/" + filepath.Base(schemaPath))
	if err != nil {
		return nil, err
	}
	if schema == nil || !bytes.Equal(schema, embeddedSchema) {
		return nil, fmt.Errorf("PP_SCHEMA_PROJECTION_MISMATCH")
	}
	catalogRaw, err := source.ReadBytes(PPCatalog)
	if err != nil {
		return nil, err
	}
	catalog, err := ppYAML(catalogRaw, PPCatalog, true)
	if err != nil {
		return nil, err
	}
	profiles := List(catalog["profiles"])
	if len(profiles) == 0 {
		return nil, fmt.Errorf("PUBLIC_PROFILE_CATALOG_INVALID")
	}
	registered := []string{}
	for _, raw := range profiles {
		row := Map(raw)
		res := Text(row["resource"])
		if res == "" || row["contract"] != state.Current {
			return nil, fmt.Errorf("PUBLIC_PROFILE_CATALOG_INVALID")
		}
		registered = append(registered, res)
		profile, err := source.ReadBytes("profiles/" + res)
		if err != nil {
			return nil, err
		}
		historical, err := source.ReadBytes(baseline + "/" + res)
		if err != nil {
			return nil, err
		}
		if profile == nil || !bytes.Equal(profile, historical) {
			return nil, fmt.Errorf("CURRENT_PP_BASELINE_MISMATCH: %s", res)
		}
		payload, err := ppYAML(profile, res, true)
		if err != nil {
			return nil, err
		}
		if Map(payload["ptsip"])["version"] != state.Current {
			return nil, fmt.Errorf("PUBLIC_PROFILE_VERSION_MISMATCH")
		}
		if Map(payload["responsibility_map"])["mode"] != row["responsibility_mode"] {
			return nil, fmt.Errorf("PUBLIC_PROFILE_MODE_MISMATCH")
		}
	}
	sort.Strings(registered)
	if !reflect.DeepEqual(registered, state.Resources) {
		return nil, fmt.Errorf("PUBLIC_PROFILE_CATALOG_COVERAGE_MISMATCH")
	}
	return state, nil
}
func (r *Repository) VerifyPPStagedParents() error {
	parents := []string{"HEAD"}
	mergePath, err := ppGit(r.Root, "rev-parse", "--git-path", "MERGE_HEAD")
	if err != nil {
		return err
	}
	path := strings.TrimSpace(string(mergePath))
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.Root, path)
	}
	if raw, err := os.ReadFile(path); err == nil {
		parents = append(parents, strings.Fields(string(raw))...)
	} else if !os.IsNotExist(err) {
		return err
	}
	states := []*PPAuthorityState{}
	for _, parent := range parents {
		state, err := LoadPPAuthority(PPGitSnapshot{Root: r.Root, Revision: parent})
		if err != nil {
			return err
		}
		states = append(states, state)
	}
	return PPComparableParents(states)
}
func (r *Repository) verifyPPTransitionShape(parent, commit string, delta PPDelta) error {
	if !delta.Valid || !delta.Triggered || !delta.Reconciled {
		return fmt.Errorf("PP_TRANSITION_NOT_ATOMIC")
	}
	source, target := Text(delta.BaseCurrent), Text(delta.ExpectedNext)
	candidate := PPGitSnapshot{Root: r.Root, Revision: commit}
	raw, err := candidate.ReadBytes(PPRegistry)
	if err != nil {
		return err
	}
	registry, err := ppYAML(raw, PPRegistry, true)
	if err != nil {
		return err
	}
	sources, targets := []Object{}, []Object{}
	for _, raw := range List(registry["contracts"]) {
		row := Map(raw)
		if row["version"] == source {
			sources = append(sources, row)
		}
		if row["version"] == target {
			targets = append(targets, row)
		}
	}
	if len(sources) != 1 || len(targets) != 1 || sources[0]["lifecycle"] != "SUPERSEDED" || targets[0]["lifecycle"] != "CURRENT" {
		return fmt.Errorf("PP_TRANSITION_REGISTRY_INVALID")
	}
	found := false
	for _, raw := range List(registry["transitions"]) {
		row := Map(raw)
		found = found || (row["from"] == source && row["to"] == target && row["kind"] == "SEMANTIC_MIGRATION")
	}
	if !found {
		return fmt.Errorf("PP_TRANSITION_RECORD_MISSING")
	}
	changed, err := ppGit(r.Root, "diff", "--name-status", parent, commit, "--", "profiles/history")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(changed)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 || !strings.HasPrefix(parts[0], "A") || !strings.HasPrefix(parts[len(parts)-1], "profiles/history/"+target+"/") {
			return fmt.Errorf("HISTORICAL_BASELINE_MUTATION: %s", line)
		}
	}
	base := PPGitSnapshot{Root: r.Root, Revision: parent}
	state, err := LoadPPAuthority(base)
	if err != nil {
		return err
	}
	if old := state.Contracts[source]; old != "" {
		left, err := base.ReadBytes(old)
		if err != nil {
			return err
		}
		right, err := candidate.ReadBytes(old)
		if err != nil {
			return err
		}
		if !bytes.Equal(left, right) {
			return fmt.Errorf("SUPERSEDED_PP_SCHEMA_MUTATION")
		}
	}
	return nil
}
func (r *Repository) VerifyPPCommit(commit string) (Object, error) {
	resolved, err := ppGit(r.Root, "rev-parse", "--verify", commit+"^{commit}")
	if err != nil {
		return nil, err
	}
	commit = strings.TrimSpace(string(resolved))
	parentsRaw, err := ppGit(r.Root, "show", "-s", "--format=%P", commit)
	if err != nil {
		return nil, err
	}
	parents := strings.Fields(string(parentsRaw))
	candidate := PPGitSnapshot{Root: r.Root, Revision: commit}
	state, err := ValidatePPSnapshot(candidate)
	if err != nil {
		return nil, err
	}
	out := Object{"commit": commit, "parents": parents, "classification": "ROOT_COMMIT", "triggered": false, "current": state.Current}
	if len(parents) == 0 {
		return out, nil
	}
	states := []*PPAuthorityState{}
	for _, parent := range parents {
		left, err := LoadPPAuthority(PPGitSnapshot{Root: r.Root, Revision: parent})
		if err != nil {
			return nil, err
		}
		states = append(states, left)
	}
	if err = PPComparableParents(states); err != nil {
		return nil, err
	}
	delta, err := ComparePPSnapshots(PPGitSnapshot{Root: r.Root, Revision: parents[0]}, candidate)
	if err != nil {
		return nil, err
	}
	if !delta.Valid {
		return nil, fmt.Errorf("%s", delta.Classification)
	}
	fingerprint := PPAuthorityFingerprint(state)
	if len(parents) > 1 && fingerprint == PPAuthorityFingerprint(states[0]) {
		out["classification"] = "MERGE_INHERITED_PP_AUTHORITY"
		return out, nil
	}
	if delta.Triggered {
		if err = r.verifyPPTransitionShape(parents[0], commit, delta); err != nil {
			return nil, err
		}
	} else if fingerprint != PPAuthorityFingerprint(states[0]) {
		return nil, fmt.Errorf("UNCLASSIFIED_PP_AUTHORITY_CHANGE")
	}
	if !delta.Triggered && delta.Classification == "NO_T2_AUTHORITY_DELTA" {
		history, err := ppGit(r.Root, "diff", "--name-only", parents[0], commit, "--", "profiles/history")
		if err != nil {
			return nil, err
		}
		if len(bytes.TrimSpace(history)) > 0 {
			return nil, fmt.Errorf("HISTORICAL_BASELINE_MUTATION")
		}
	}
	out["classification"] = delta.Classification
	out["triggered"] = delta.Triggered
	return out, nil
}
func (r *Repository) VerifyPPRange(base, head string) (Object, error) {
	raw, err := ppGit(r.Root, "rev-list", "--reverse", "--topo-order", base+".."+head)
	if err != nil {
		return nil, err
	}
	commits := []any{}
	for _, commit := range strings.Fields(string(raw)) {
		result, err := r.VerifyPPCommit(commit)
		if err != nil {
			return nil, err
		}
		commits = append(commits, result)
	}
	return Object{"status": "PASS", "verified_commit_count": len(commits), "commits": commits}, nil
}
func (r *Repository) VerifyPPRelease(expected string) (Object, error) {
	resolved, err := ppGit(r.Root, "rev-parse", "--verify", expected+"^{commit}")
	if err != nil {
		return nil, err
	}
	head, err := ppGit(r.Root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(resolved, head) {
		return nil, fmt.Errorf("RELEASE_EXACT_SHA_MISMATCH")
	}
	sha := strings.TrimSpace(string(resolved))
	commit, err := r.VerifyPPCommit(sha)
	if err != nil {
		return nil, err
	}
	snapshot := PPGitSnapshot{Root: r.Root, Worktree: true}
	state, err := ValidatePPSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	if failures, err := r.ProfileRegistryErrors(); err != nil || len(failures) > 0 {
		return nil, fmt.Errorf("PP_RELEASE_REGISTRY_INVALID: %v %s", err, strings.Join(failures, "; "))
	}
	immutable := PPGitSnapshot{Root: r.Root, Revision: sha}
	for _, path := range []string{PPRegistry, PPEmbeddedRegistry, PPCatalog, state.Contracts[state.Current], "src/ptsip/specdata/" + filepath.Base(state.Contracts[state.Current])} {
		left, err := snapshot.ReadBytes(path)
		if err != nil {
			return nil, err
		}
		right, err := immutable.ReadBytes(path)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(left, right) {
			return nil, fmt.Errorf("RELEASE_EXACT_ASSET_MISMATCH: %s", path)
		}
	}
	return Object{"status": "PASS", "source_sha": sha, "project_profile": state.Current, "schema": state.Contracts[state.Current], "commit_classification": commit["classification"], "transition_triggered": commit["triggered"]}, nil
}
