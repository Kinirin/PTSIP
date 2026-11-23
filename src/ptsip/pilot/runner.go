// Package pilot collects read-only PTSIP repository evidence.
package pilot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
	"github.com/Kinirin/PTSIP/src/ptsip/inspection"
	"github.com/Kinirin/PTSIP/src/ptsip/internal/enginerouter"
	sourcecompat "github.com/Kinirin/PTSIP/src/ptsip/source_compat"
)

type Runner struct {
	caller      enginerouter.Caller
	packageRoot string
	toolVersion func(context.Context) (string, error)
}

func OpenInstalled(cli, packageRoot, scriptsDir string) Runner {
	client := enginerouter.Client{CLI: cli, PackageRoot: packageRoot, ScriptsDir: scriptsDir}
	return Runner{caller: client, packageRoot: packageRoot, toolVersion: client.ToolVersion}
}

type Procedure struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type Snapshot struct {
	Before     json.RawMessage `json:"before"`
	After      json.RawMessage `json:"after"`
	Comparison Comparison      `json:"comparison"`
}

type Comparison struct {
	Status  string   `json:"status"`
	Stable  bool     `json:"stable"`
	Reasons []string `json:"reasons"`
}

type NonIntrusion struct {
	Status                   string   `json:"status"`
	AnalysisReadOnlyByDesign bool     `json:"analysis_read_only_by_design"`
	OutputInsideRepository   bool     `json:"output_inside_repository"`
	Methods                  []string `json:"methods"`
}

type ComponentEvidence struct {
	Status                 string                          `json:"status"`
	ClassificationAsserted bool                            `json:"classification_asserted"`
	Candidates             []inspection.ComponentCandidate `json:"candidates"`
	DeclaredPartition      json.RawMessage                 `json:"declared_partition"`
}

type Classification struct {
	Status                 string                      `json:"status"`
	AllowedClassifications []string                    `json:"allowed_classifications"`
	DecisionStatuses       []inspection.DecisionStatus `json:"decision_statuses"`
}

type Specification struct {
	Name                             string `json:"name"`
	Version                          string `json:"version"`
	Source                           string `json:"source"`
	Revision                         string `json:"revision"`
	DraftRevisionIsNormativeIdentity bool   `json:"draft_revision_is_normative_identity"`
}

type ProfileEvidence struct {
	Path        string   `json:"profile_path"`
	Version     string   `json:"version"`
	Valid       bool     `json:"valid"`
	Diagnostics []string `json:"diagnostics"`
}

type BoundaryEvaluation struct {
	Status                     string `json:"status"`
	Reason                     string `json:"reason"`
	FindingCount               *int   `json:"finding_count"`
	DependencyCoverageComplete bool   `json:"dependency_evidence_coverage_complete"`
}

type Report struct {
	Format         string                        `json:"format"`
	GeneratedAt    time.Time                     `json:"generated_at"`
	Tool           map[string]string             `json:"tool"`
	Specification  Specification                 `json:"specification"`
	Repository     json.RawMessage               `json:"repository"`
	Snapshot       Snapshot                      `json:"snapshot"`
	NonIntrusion   NonIntrusion                  `json:"non_intrusion"`
	Inventory      inspection.Inventory          `json:"inventory"`
	Components     ComponentEvidence             `json:"components"`
	Dependencies   inspection.DependencyScan     `json:"dependencies"`
	Artifacts      Procedure                     `json:"artifacts"`
	Classification Classification                `json:"classification"`
	Profile        *ProfileEvidence              `json:"profile"`
	Evaluation     map[string]BoundaryEvaluation `json:"evaluation"`
	Findings       []any                         `json:"findings"`
	Conformance    Procedure                     `json:"conformance"`
	repositoryRoot string
}

type selectedContract struct {
	ID       string `json:"contract_id"`
	Version  string `json:"version"`
	Resource string `json:"schema_resource"`
	Digest   string `json:"source_sha256"`
	Allowed  bool   `json:"allowed"`
}

// Run orchestrates read-only observations and leaves report persistence to the
// caller. It only derives findings from evidence supplied by registered owners.
func (r Runner) Run(ctx context.Context, path string) (Report, error) {
	var report Report
	if r.caller == nil || r.toolVersion == nil || path == "" {
		return report, fmt.Errorf("installed native runner and a repository path are required")
	}
	var where struct {
		Root string `json:"root"`
	}
	if err := r.caller.Invoke(ctx, "discovery.repository.inspect", map[string]string{"path": path}, &where); err != nil {
		return report, err
	}
	if !filepath.IsAbs(where.Root) {
		return report, fmt.Errorf("native discovery did not resolve an absolute repository root")
	}
	rootInput := map[string]string{"root": where.Root}
	if err := r.caller.Invoke(ctx, "context.repository.snapshot", rootInput, &report.Snapshot.Before); err != nil {
		return report, err
	}
	if err := r.caller.Invoke(ctx, "discovery.repository.inspect", map[string]string{"path": where.Root}, &report.Repository); err != nil {
		return report, err
	}
	inventory, err := inspection.ProjectInventory(report.Repository)
	if err != nil {
		return report, err
	}
	report.Inventory = inventory
	report.Dependencies, err = inspection.ObserveDependencies(where.Root, report.Repository)
	if err != nil {
		return report, err
	}
	var observed struct {
		Source      *sourcecompat.ObservedBinding `json:"source"`
		Diagnostics []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"diagnostics"`
	}
	if err := r.caller.Invoke(ctx, "discovery.profile.candidates", map[string]any{"root": where.Root, "candidate_paths": []string{}}, &observed); err != nil {
		return report, err
	}
	active, err := r.selectProfile(ctx, "")
	if err != nil {
		return report, err
	}
	activeSchema, err := r.readSchema(active)
	if err != nil {
		return report, err
	}
	report.Classification = Classification{Status: "EVIDENCE_ONLY", AllowedClassifications: classifications(activeSchema),
		DecisionStatuses: []inspection.DecisionStatus{inspection.Resolved, inspection.Unknown, inspection.Conflict, inspection.Incomplete}}
	report.Components = ComponentEvidence{Status: "CANDIDATES_ONLY",
		Candidates: inspection.DiscoverComponentCandidates(inventory, report.Dependencies.CandidateAnchors())}
	unavailable := BoundaryEvaluation{Status: "BLOCKED", Reason: "NO_PROFILE",
		DependencyCoverageComplete: report.Dependencies.CoverageComplete}
	if observed.Source != nil {
		profile, coverage, err := r.assessProfile(ctx, where.Root, *observed.Source, report.Repository)
		if err != nil {
			return report, err
		}
		report.Profile = &profile
		unavailable.Reason = "PROFILE_INVALID"
		if profile.Valid {
			unavailable.Reason = "DECLARED_BOUNDARY_EVALUATOR_UNAVAILABLE"
			report.Classification.Status = "DECLARATION_AVAILABLE"
			if coverage != nil {
				report.Components.Status = "DECLARED_AND_CANDIDATES"
				report.Components.DeclaredPartition = coverage
			}
		}
	} else {
		for _, diagnostic := range observed.Diagnostics {
			if diagnostic.Code != "NO_CANONICAL_PROFILE" {
				if report.Profile == nil {
					report.Profile = &ProfileEvidence{Path: "ptsip.yaml"}
				}
				report.Profile.Diagnostics = append(report.Profile.Diagnostics, diagnostic.Code+": "+diagnostic.Message)
				unavailable.Reason = "PROFILE_INVALID"
			}
		}
	}
	if err := r.caller.Invoke(ctx, "context.repository.snapshot", rootInput, &report.Snapshot.After); err != nil {
		return report, err
	}
	if err := r.caller.Invoke(ctx, "validation.repository.snapshot", map[string]json.RawMessage{
		"before": report.Snapshot.Before, "after": report.Snapshot.After,
	}, &report.Snapshot.Comparison); err != nil {
		return report, err
	}
	version, err := r.toolVersion(ctx)
	if err != nil {
		return report, err
	}
	report.Format, report.GeneratedAt, report.Tool = "ptsip-pilot-report/v2", time.Now().UTC(), map[string]string{"version": version}
	report.Specification = Specification{Name: "PTSIP", Version: active.Version, Revision: active.Digest, Source: sourcecompat.SpecificationSource}
	report.NonIntrusion = NonIntrusion{Status: report.observationStatus(), AnalysisReadOnlyByDesign: true,
		Methods: []string{"git-head-before-after", "git-status-fingerprint-before-after", "tracked-content-fingerprint-before-after"}}
	report.Artifacts = Procedure{Status: "NOT_INSPECTED", Reason: "no explicit Product Artifact evidence"}
	report.Evaluation = map[string]BoundaryEvaluation{"declared_dependency_boundaries": unavailable}
	report.Findings = []any{}
	report.Conformance = Procedure{Status: "NOT_EVALUATED", Reason: "Pilot collects repository evidence"}
	report.repositoryRoot = where.Root
	return report, nil
}

func (report Report) observationStatus() string {
	comparison := report.Snapshot.Comparison
	switch {
	case comparison.Stable && len(report.Inventory.ObservationErrors) == 0:
		return "VERIFIED_NO_OBSERVED_CHANGE"
	case comparison.Stable, len(comparison.Reasons) == 1 && comparison.Reasons[0] == "snapshot observation was incomplete":
		return "UNVERIFIED"
	default:
		return "CHANGE_OBSERVED_DURING_ANALYSIS"
	}
}

func (r Runner) assessProfile(ctx context.Context, root string, binding sourcecompat.ObservedBinding, repository json.RawMessage) (ProfileEvidence, json.RawMessage, error) {
	result := ProfileEvidence{Path: binding.Path, Version: binding.DeclaredVersion, Diagnostics: []string{}}
	for _, family := range sourcecompat.SupportedSourceFamilies() {
		if family.Version == binding.DeclaredVersion {
			return r.assessHistorical(ctx, root, binding, repository, result)
		}
	}
	doc, failure := sourcecompat.ReadObservedProfile(ctx, r.caller, root, binding)
	if failure != nil {
		result.Diagnostics = append(result.Diagnostics, failure.Code+": "+failure.Message)
		return result, nil, nil
	}
	contract, err := r.selectProfile(ctx, binding.DeclaredVersion)
	if err != nil {
		if ctx.Err() != nil {
			return result, nil, ctx.Err()
		}
		result.Diagnostics = append(result.Diagnostics, err.Error())
		return result, nil, nil
	}
	schema, err := r.readSchema(contract)
	if err != nil {
		return result, nil, err
	}
	var checked struct {
		Valid       bool     `json:"valid"`
		Diagnostics []string `json:"diagnostics"`
	}
	if err := r.caller.Invoke(ctx, "validation.schema", map[string]any{"schema": schema, "instance": doc}, &checked); err != nil {
		return result, nil, err
	}
	result.Valid, result.Diagnostics = checked.Valid, checked.Diagnostics
	return result, nil, nil
}

func (r Runner) assessHistorical(ctx context.Context, root string, binding sourcecompat.ObservedBinding, repository json.RawMessage, result ProfileEvidence) (ProfileEvidence, json.RawMessage, error) {
	read := sourcecompat.NewReader(r.caller, os.DirFS(r.packageRoot)).ReadSourceProfile(ctx, root, binding)
	if err := ctx.Err(); err != nil {
		return result, nil, err
	}
	for _, issue := range read.Issues {
		result.Diagnostics = append(result.Diagnostics, issue.Code+": "+issue.Message)
	}
	if !read.Complete {
		return result, nil, nil
	}
	result.Valid = true
	var projected struct {
		Coverages json.RawMessage `json:"coverages"`
		Issues    []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"issues"`
	}
	if err := r.caller.Invoke(ctx, "projection.migration.source", read.Profile.ProjectionInput(), &projected); err != nil {
		return result, nil, err
	}
	for _, issue := range projected.Issues {
		result.Diagnostics = append(result.Diagnostics, issue.Code+": "+issue.Message)
	}
	if len(projected.Issues) != 0 {
		return result, nil, nil
	}
	var observed struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(repository, &observed); err != nil {
		return result, nil, err
	}
	var coverage json.RawMessage
	err := r.caller.Invoke(ctx, "projection.migration.coverage", map[string]any{"paths": observed.Files, "coverages": projected.Coverages}, &coverage)
	return result, coverage, err
}

func (r Runner) selectProfile(ctx context.Context, version string) (selectedContract, error) {
	var selected selectedContract
	input := map[string]string{"root": r.packageRoot}
	if version != "" {
		input["contract_version"] = version
	}
	if err := r.caller.Invoke(ctx, "discovery.profile.contract", input, &selected); err != nil {
		return selected, err
	}
	if !selected.Allowed || selected.ID == "" || selected.Version == "" || version != "" && selected.Version != version {
		return selected, fmt.Errorf("no registered Profile contract selected for %q", version)
	}
	return selected, nil
}

func (r Runner) readSchema(selected selectedContract) (json.RawMessage, error) {
	if !fs.ValidPath(selected.Resource) || !strings.HasPrefix(selected.Resource, "schemas/") || strings.ContainsAny(selected.Resource, "\\:") {
		return nil, fmt.Errorf("selected Profile schema is outside the installed schema root")
	}
	file, err := os.DirFS(r.packageRoot).Open(selected.Resource)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > source.MaxBytes {
		return nil, fmt.Errorf("invalid installed Profile schema file %q", selected.Resource)
	}
	data, err := io.ReadAll(io.LimitReader(file, source.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > source.MaxBytes || fmt.Sprintf("%x", sha256.Sum256(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))) != selected.Digest {
		return nil, fmt.Errorf("selected Profile schema differs from its registered digest: %s", selected.Resource)
	}
	return data, nil
}

func classifications(schema json.RawMessage) []string {
	var doc struct {
		Defs struct {
			Component struct {
				Properties struct {
					Classification struct {
						Enum []string `json:"enum"`
					} `json:"classification"`
				} `json:"properties"`
			} `json:"component"`
		} `json:"$defs"`
	}
	if json.Unmarshal(schema, &doc) != nil || doc.Defs.Component.Properties.Classification.Enum == nil {
		return []string{}
	}
	return doc.Defs.Component.Properties.Classification.Enum
}
