package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

type PPTransitionPlan struct {
	Status       string            `json:"status"`
	BaseCurrent  string            `json:"base_current"`
	Target       string            `json:"target"`
	Reasons      []string          `json:"reasons"`
	Paths        []string          `json:"outputs"`
	Outputs      map[string][]byte `json:"-"`
	BaseRevision string            `json:"-"`
	IndexTree    string            `json:"-"`
}

func ppSchemaPath(version string) string {
	return "schemas/ptsip-profile-pp-" + strings.TrimPrefix(version, "pp.") + ".schema.json"
}
func ppEmbeddedSchemaPath(version string) string {
	return "src/ptsip/specdata/" + filepath.Base(ppSchemaPath(version))
}
func ppHistory(version string) string { return "profiles/history/" + version }

var ppVersionLine = regexp.MustCompile(`^(\s+version:\s*)("[^"]*"|'[^']*'|[^#\s]+)(\s*(?:#.*)?)$`)

func PPRebindProfile(raw []byte, source, target string) ([]byte, error) {
	payload, err := ppYAML(raw, "profile", true)
	if err != nil {
		return nil, err
	}
	version := Text(Map(payload["ptsip"])["version"])
	if version != source && version != target {
		return nil, fmt.Errorf("PUBLIC_PROFILE_VERSION_MISMATCH")
	}
	if version == target {
		return raw, nil
	}
	lines := strings.SplitAfter(string(raw), "\n")
	inPtsip, replaced := false, false
	for i, line := range lines {
		body := strings.TrimRight(line, "\r\n")
		if body == "ptsip:" {
			inPtsip = true
			continue
		}
		if !inPtsip {
			continue
		}
		if body != "" && !strings.HasPrefix(body, " ") && !strings.HasPrefix(body, "\t") {
			break
		}
		match := ppVersionLine.FindStringSubmatch(body)
		if match == nil {
			continue
		}
		quote := ""
		if strings.HasPrefix(match[2], "\"") {
			quote = "\""
		} else if strings.HasPrefix(match[2], "'") {
			quote = "'"
		}
		lines[i] = match[1] + quote + target + quote + match[3] + line[len(body):]
		replaced = true
		break
	}
	if !replaced {
		return nil, fmt.Errorf("PUBLIC_PROFILE_VERSION_LINE_UNRESOLVED")
	}
	out := []byte(strings.Join(lines, ""))
	verified, err := ppYAML(out, "profile", true)
	if err != nil {
		return nil, err
	}
	if Map(verified["ptsip"])["version"] != target {
		return nil, fmt.Errorf("PUBLIC_PROFILE_REBIND_FAILED")
	}
	return out, nil
}
func ppReidentitySchema(raw []byte, target string) ([]byte, error) {
	var payload Object
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	version := Map(Map(Map(Map(payload["properties"])["ptsip"])["properties"])["version"])
	if version == nil {
		return nil, fmt.Errorf("CURRENT_PP_SCHEMA_INVALID")
	}
	payload["$id"] = "https://raw.githubusercontent.com/Kinirin/PTSIP/main/" + ppSchemaPath(target)
	payload["title"] = "PTSIP Project Profile " + target
	version["const"] = target
	version["description"] = "Canonical Project Profile contract identity."
	out, err := json.MarshalIndent(payload, "", "  ")
	return append(out, '\n'), err
}
func ppBuildRegistry(registry Object, source, target string) ([]byte, error) {
	contracts, transitions := List(registry["contracts"]), List(registry["transitions"])
	if contracts == nil || transitions == nil {
		return nil, fmt.Errorf("PP_CONTRACT_REGISTRY_INVALID")
	}
	matches := []Object{}
	for _, raw := range contracts {
		row := Map(raw)
		if row["version"] == target {
			return nil, fmt.Errorf("PP_TARGET_CONTRACT_ALREADY_EXISTS")
		}
		if row["version"] == source {
			matches = append(matches, row)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("PP_SOURCE_CONTRACT_UNRESOLVED")
	}
	operations := List(matches[0]["operations"])
	if len(operations) == 0 {
		return nil, fmt.Errorf("PP_SOURCE_OPERATIONS_INVALID")
	}
	matches[0]["lifecycle"] = "SUPERSEDED"
	registry["contracts"] = append(contracts, Object{"version": target, "lifecycle": "CURRENT", "operations": operations, "schema": ppSchemaPath(target), "baseline": ppHistory(target)})
	registry["transitions"] = append(transitions, Object{"from": source, "to": target, "kind": "SEMANTIC_MIGRATION"})
	registry["current"] = target
	return yaml.Marshal(registry)
}
func (r *Repository) BuildPPTransition(baseRevision string) (*PPTransitionPlan, error) {
	base := PPGitSnapshot{Root: r.Root, Revision: baseRevision}
	candidate := PPGitSnapshot{Root: r.Root, Staged: true}
	delta, err := ComparePPSnapshots(base, candidate)
	if err != nil {
		return nil, err
	}
	if !delta.Valid {
		return nil, fmt.Errorf("%s", delta.Classification)
	}
	plan := &PPTransitionPlan{BaseCurrent: Text(delta.BaseCurrent), Target: Text(delta.CandidateCurrent), Reasons: delta.Reasons, Paths: []string{}, Outputs: map[string][]byte{}, BaseRevision: baseRevision}
	if !delta.Triggered {
		if delta.Classification != "NO_T2_AUTHORITY_DELTA" {
			return nil, fmt.Errorf("%s", delta.Classification)
		}
		changedHistory, err := ppGit(r.Root, "diff", "--cached", "--name-only", baseRevision, "--", "profiles/history")
		if err != nil {
			return nil, err
		}
		if len(bytes.TrimSpace(changedHistory)) > 0 {
			return nil, fmt.Errorf("HISTORICAL_BASELINE_MUTATION")
		}
		plan.Status = "NO_CHANGE"
		return plan, nil
	}
	plan.Target = Text(delta.ExpectedNext)
	if delta.Reconciled {
		if err = r.VerifyPPReconciled(baseRevision, plan.BaseCurrent, plan.Target); err != nil {
			return nil, err
		}
		plan.Status = "ALREADY_RECONCILED"
		return plan, nil
	}
	historyDiff, err := ppGit(r.Root, "diff", "--cached", "--name-only", baseRevision, "--", "profiles/history")
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(historyDiff)) > 0 {
		return nil, fmt.Errorf("HISTORICAL_BASELINE_MUTATION")
	}
	baseRegistryRaw, err := base.ReadBytes(PPRegistry)
	if err != nil {
		return nil, err
	}
	candidateRegistryRaw, err := candidate.ReadBytes(PPRegistry)
	if err != nil {
		return nil, err
	}
	baseRegistry, err := ppYAML(baseRegistryRaw, PPRegistry, true)
	if err != nil {
		return nil, err
	}
	candidateRegistry, err := ppYAML(candidateRegistryRaw, PPRegistry, true)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(baseRegistry, candidateRegistry) {
		return nil, fmt.Errorf("MANUAL_PP_REGISTRY_MUTATION")
	}
	baseState, err := LoadPPAuthority(base)
	if err != nil {
		return nil, err
	}
	for _, res := range baseState.Resources {
		history, err := base.ReadBytes(ppHistory(plan.BaseCurrent) + "/" + res)
		if err != nil {
			return nil, err
		}
		if history == nil || !bytes.Equal(history, baseState.Raw[res]) {
			return nil, fmt.Errorf("HISTORICAL_BASELINE_INVALID")
		}
	}
	catalogRaw, err := candidate.ReadBytes(PPCatalog)
	if err != nil {
		return nil, err
	}
	catalog, err := ppYAML(catalogRaw, PPCatalog, true)
	if err != nil {
		return nil, err
	}
	if len(List(catalog["profiles"])) == 0 {
		return nil, fmt.Errorf("PUBLIC_PROFILE_CATALOG_INVALID")
	}
	for _, raw := range List(catalog["profiles"]) {
		row := Map(raw)
		res := Text(row["resource"])
		if res == "" || strings.Contains(res, "/") {
			return nil, fmt.Errorf("PUBLIC_PROFILE_CATALOG_INVALID")
		}
		profile, err := candidate.ReadBytes("profiles/" + res)
		if err != nil {
			return nil, err
		}
		if profile == nil {
			return nil, fmt.Errorf("PUBLIC_PROFILE_RESOURCE_MISSING")
		}
		rebound, err := PPRebindProfile(profile, plan.BaseCurrent, plan.Target)
		if err != nil {
			return nil, err
		}
		plan.Outputs["profiles/"+res] = rebound
		plan.Outputs[ppHistory(plan.Target)+"/"+res] = rebound
		row["contract"] = plan.Target
	}
	oldPath := baseState.Contracts[plan.BaseCurrent]
	oldSchema, err := base.ReadBytes(oldPath)
	if err != nil {
		return nil, err
	}
	candidateSchema, err := candidate.ReadBytes(oldPath)
	if err != nil {
		return nil, err
	}
	if oldSchema == nil || candidateSchema == nil {
		return nil, fmt.Errorf("CURRENT_PP_SCHEMA_MISSING")
	}
	schemaSource := oldSchema
	for _, reason := range delta.Reasons {
		if reason == "CURRENT_PP_CANONICAL_SCHEMA_CONTENT" {
			schemaSource = candidateSchema
			plan.Outputs[oldPath] = oldSchema
		}
	}
	schema, err := ppReidentitySchema(schemaSource, plan.Target)
	if err != nil {
		return nil, err
	}
	plan.Outputs[ppSchemaPath(plan.Target)] = schema
	plan.Outputs[ppEmbeddedSchemaPath(plan.Target)] = schema
	plan.Outputs[PPCatalog], err = yaml.Marshal(catalog)
	if err != nil {
		return nil, err
	}
	registry, err := ppBuildRegistry(baseRegistry, plan.BaseCurrent, plan.Target)
	if err != nil {
		return nil, err
	}
	plan.Outputs[PPRegistry] = registry
	plan.Outputs[PPEmbeddedRegistry] = registry
	for path := range plan.Outputs {
		plan.Paths = append(plan.Paths, path)
	}
	sort.Strings(plan.Paths)
	tree, err := ppGit(r.Root, "write-tree")
	if err != nil {
		return nil, err
	}
	plan.IndexTree = strings.TrimSpace(string(tree))
	plan.Status = "RECONCILE"
	return plan, nil
}
func (r *Repository) VerifyPPReconciled(baseRevision, source, target string) error {
	candidate := PPGitSnapshot{Root: r.Root, Staged: true}
	if _, err := ValidatePPSnapshot(candidate); err != nil {
		return err
	}
	base := PPGitSnapshot{Root: r.Root, Revision: baseRevision}
	left, err := LoadPPAuthority(base)
	if err != nil {
		return err
	}
	right, err := LoadPPAuthority(candidate)
	if err != nil {
		return err
	}
	if right.Current != target {
		return fmt.Errorf("RECONCILED_CURRENT_MISMATCH")
	}
	for _, res := range left.Resources {
		old, err := base.ReadBytes(ppHistory(source) + "/" + res)
		if err != nil {
			return err
		}
		new, err := candidate.ReadBytes(ppHistory(source) + "/" + res)
		if err != nil {
			return err
		}
		if !bytes.Equal(old, new) {
			return fmt.Errorf("HISTORICAL_BASELINE_MUTATION")
		}
	}
	history, err := ppGit(r.Root, "diff", "--cached", "--name-status", baseRevision, "--", "profiles/history")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(history)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 || parts[0] != "A" || !strings.HasPrefix(parts[len(parts)-1], ppHistory(target)+"/") {
			return fmt.Errorf("HISTORICAL_BASELINE_MUTATION")
		}
	}
	raw, _ := candidate.ReadBytes(PPRegistry)
	registry, err := ppYAML(raw, PPRegistry, true)
	if err != nil {
		return err
	}
	found := false
	for _, raw := range List(registry["transitions"]) {
		row := Map(raw)
		found = found || (row["from"] == source && row["to"] == target && row["kind"] == "SEMANTIC_MIGRATION")
	}
	if !found {
		return fmt.Errorf("RECONCILED_TRANSITION_MISSING")
	}
	return nil
}
func (r *Repository) ApplyPPTransition(plan *PPTransitionPlan) (*PPTransitionPlan, error) {
	if plan.Status != "RECONCILE" {
		return plan, nil
	}
	currentTree, err := ppGit(r.Root, "write-tree")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(currentTree)) != plan.IndexTree {
		return nil, fmt.Errorf("AUTOMATION_INDEX_STALE")
	}
	before := map[string][]byte{}
	for _, path := range plan.Paths {
		if _, err := r.Path(path); err != nil {
			return nil, err
		}
		status, err := ppGit(r.Root, "status", "--porcelain=v1", "--untracked-files=all", "--", path)
		if err != nil {
			return nil, err
		}
		line := strings.TrimRight(string(status), "\r\n")
		if strings.HasPrefix(line, "??") || (len(line) >= 2 && line[1] != ' ') {
			return nil, fmt.Errorf("AUTOMATION_WRITE_CONFLICT: %s", path)
		}
		before[path], err = (PPGitSnapshot{Root: r.Root, Staged: true}).ReadBytes(path)
		if err != nil {
			return nil, err
		}
	}
	restore := func() {
		for _, path := range plan.Paths {
			target, _ := r.Path(path)
			if before[path] == nil {
				os.Remove(target)
				ppGit(r.Root, "rm", "--cached", "--ignore-unmatch", "--", path)
			} else {
				planningAtomicWrite(target, before[path])
				ppGit(r.Root, "add", "--", path)
			}
		}
	}
	for _, path := range plan.Paths {
		target, _ := r.Path(path)
		if err = planningAtomicWrite(target, plan.Outputs[path]); err != nil {
			restore()
			return nil, err
		}
	}
	args := append([]string{"add", "--"}, plan.Paths...)
	if _, err = ppGit(r.Root, args...); err != nil {
		restore()
		return nil, err
	}
	delta, err := r.ComparePP(plan.BaseRevision, "", true)
	if err != nil || !delta.Valid || !delta.Triggered || !delta.Reconciled {
		restore()
		return nil, fmt.Errorf("POST_RECONCILIATION_DELTA_INVALID: %v", err)
	}
	if err = r.VerifyPPReconciled(plan.BaseRevision, plan.BaseCurrent, plan.Target); err != nil {
		restore()
		return nil, err
	}
	if failures, err := r.ProfileRegistryErrors(); err != nil || len(failures) > 0 {
		restore()
		return nil, fmt.Errorf("POST_RECONCILIATION_REGISTRY_INVALID: %v %s", err, strings.Join(failures, "; "))
	}
	result := *plan
	result.Status = "RECONCILED"
	result.Outputs = map[string][]byte{}
	return &result, nil
}
func (r *Repository) ReconcilePP(apply bool) (*PPTransitionPlan, error) {
	plan, err := r.BuildPPTransition("HEAD")
	if err != nil {
		return nil, err
	}
	if apply {
		return r.ApplyPPTransition(plan)
	}
	return plan, nil
}
func (r *Repository) VerifyPPPreCommit() (Object, error) {
	if err := r.VerifyPPStagedParents(); err != nil {
		return nil, err
	}
	plan, err := r.ReconcilePP(true)
	if err != nil {
		return nil, err
	}
	if _, err = ValidatePPSnapshot(PPGitSnapshot{Root: r.Root, Worktree: true}); err != nil {
		return nil, err
	}
	if failures, err := r.ProfileRegistryErrors(); err != nil || len(failures) > 0 {
		return nil, fmt.Errorf("PP_REGISTRY_VALIDATION_FAILED: %v %s", err, strings.Join(failures, "; "))
	}
	delta, err := r.ComparePP("HEAD", "", true)
	if err != nil {
		return nil, err
	}
	if !delta.Valid || delta.Triggered && !delta.Reconciled {
		return nil, fmt.Errorf("PP_STAGED_DELTA_INVALID")
	}
	return Object{"status": "PASS", "reconciliation": plan.Status, "base_current": delta.BaseCurrent, "candidate_current": delta.CandidateCurrent, "classification": delta.Classification, "triggered": delta.Triggered, "candidate_already_reconciled": delta.Reconciled}, nil
}
