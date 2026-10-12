package enginewheel

import (
	"encoding/json"
	"strings"
	"testing"
)

func gateFixture(t *testing.T, kind string) (GateOptions, gateRun, []gateJob, []gateArtifact) {
	t.Helper()
	opts := GateOptions{Kind: kind, SourceSHA: strings.Repeat("a", 40), RunID: 7}
	run := gateRun{ID: 7, HeadSHA: opts.SourceSHA, Event: "workflow_dispatch", Path: ".github/workflows/tooling-release.yml"}
	selected := kinds
	if kind != "all" {
		selected = []string{kind}
	}
	jobs := []gateJob{{Name: "verification-admission", Status: "completed", Conclusion: "success"}}
	artifacts := []gateArtifact{}
	for _, engine := range selected {
		for _, platform := range []string{"windows-x64", "linux-x64", "macos-arm64"} {
			job := gateJob{Name: "Engine Verification / " + engine + " / " + platform, Status: "completed", Conclusion: "success"}
			for _, name := range []string{"Run only the selected engine's Rust tests", "Build only the selected independent Rust engine", "Assemble local native wheel (unreleased)", "Verify the wheel in Go and install into isolated consumer", "Preserve engine verification evidence"} {
				job.Steps = append(job.Steps, struct{ Name, Conclusion string }{name, "success"})
			}
			if platform == "linux-x64" {
				job.Steps = append(job.Steps, struct{ Name, Conclusion string }{"Verify Linux wheel ABI", "success"})
			}
			jobs = append(jobs, job)
			artifact := gateArtifact{Name: "engine-verification-" + engine + "-" + platform + "-" + opts.SourceSHA, Size: 100}
			artifact.WorkflowRun.ID = opts.RunID
			artifact.WorkflowRun.HeadSHA = opts.SourceSHA
			artifacts = append(artifacts, artifact)
		}
	}
	return opts, run, jobs, artifacts
}

func TestEngineGateAllAndSelectedMatrix(t *testing.T) {
	for _, kind := range []string{"all", "context"} {
		t.Run(kind, func(t *testing.T) {
			opts, run, jobs, artifacts := gateFixture(t, kind)
			result, err := validateGate(opts, run, jobs, artifacts)
			if err != nil || result.Status != "PASS" || result.PublicationAuthorized || result.EnginePlatformsVerified != len(artifacts) {
				t.Fatalf("unexpected gate result: %+v, %v", result, err)
			}
		})
	}
}

func TestEngineGateFailsClosedOnIncompleteOrUnrelatedEvidence(t *testing.T) {
	cases := []string{"source", "run", "workflow", "admission", "missing-platform", "failed-platform", "skipped-check", "duplicate-job", "unexpected-engine", "missing-artifact", "expired-artifact", "empty-artifact", "artifact-source", "artifact-run", "duplicate-artifact"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			opts, run, jobs, artifacts := gateFixture(t, "context")
			switch name {
			case "source":
				run.HeadSHA = strings.Repeat("b", 40)
			case "run":
				run.ID++
			case "workflow":
				run.Path = ".github/workflows/other.yml"
			case "admission":
				jobs[0].Conclusion = "failure"
			case "missing-platform":
				jobs = jobs[:len(jobs)-1]
			case "failed-platform":
				jobs[1].Conclusion = "cancelled"
			case "skipped-check":
				jobs[1].Steps[0].Conclusion = "skipped"
			case "duplicate-job":
				jobs = append(jobs, jobs[1])
			case "unexpected-engine":
				jobs[1].Name = "Engine Verification / storage / windows-x64"
			case "missing-artifact":
				artifacts = artifacts[:len(artifacts)-1]
			case "expired-artifact":
				artifacts[0].Expired = true
			case "empty-artifact":
				artifacts[0].Size = 0
			case "artifact-source":
				artifacts[0].WorkflowRun.HeadSHA = strings.Repeat("b", 40)
			case "artifact-run":
				artifacts[0].WorkflowRun.ID++
			case "duplicate-artifact":
				artifacts = append(artifacts, artifacts[0])
			}
			if result, err := validateGate(opts, run, jobs, artifacts); err == nil {
				t.Fatalf("invalid evidence admitted: %+v", result)
			}
		})
	}
}

func TestGateGitHubWireFormat(t *testing.T) {
	var artifact gateArtifact
	err := json.Unmarshal([]byte(`{"name":"evidence","size_in_bytes":123,"expired":false,"workflow_run":{"id":7,"head_sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`), &artifact)
	if err != nil || artifact.Size != 123 || artifact.WorkflowRun.ID != 7 || !sourcePattern.MatchString(artifact.WorkflowRun.HeadSHA) {
		t.Fatalf("Actions artifact identity lost: %+v, %v", artifact, err)
	}
}
