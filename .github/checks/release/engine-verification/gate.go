package enginewheel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type GateOptions struct {
	Repository, SourceSHA, Kind string
	RunID                       int64
}
type gateRun struct {
	ID      int64  `json:"id"`
	HeadSHA string `json:"head_sha"`
	Event   string `json:"event"`
	Path    string `json:"path"`
}
type gateJob struct {
	Name, Status, Conclusion string
	Steps                    []struct{ Name, Conclusion string }
}
type gateArtifact struct {
	Name        string
	Expired     bool
	Size        int64 `json:"size_in_bytes"`
	WorkflowRun struct {
		ID      int64
		HeadSHA string `json:"head_sha"`
	} `json:"workflow_run"`
}
type GateResult struct {
	Status                  string   `json:"status"`
	Scope                   string   `json:"scope"`
	SourceSHA               string   `json:"source_sha"`
	RunID                   int64    `json:"run_id"`
	EnginePlatformsVerified int      `json:"engine_platforms_verified"`
	Artifacts               []string `json:"artifacts"`
	PublicationAuthorized   bool     `json:"publication_authorized"`
}

func validateGate(opts GateOptions, run gateRun, jobs []gateJob, artifacts []gateArtifact) (GateResult, error) {
	if !sourcePattern.MatchString(opts.SourceSHA) || opts.RunID <= 0 || run.ID != opts.RunID || run.HeadSHA != opts.SourceSHA || run.Event != "workflow_dispatch" || run.Path != ".github/workflows/tooling-release.yml" {
		return GateResult{}, fmt.Errorf("workflow/source identity mismatch")
	}
	selected := kinds
	if opts.Kind != "all" {
		if !has(kinds, opts.Kind) {
			return GateResult{}, fmt.Errorf("unregistered engine selection")
		}
		selected = []string{opts.Kind}
	}
	expected := map[string]string{}
	artifactNames := []string{}
	for _, kind := range selected {
		for _, platform := range []string{"windows-x64", "linux-x64", "macos-arm64"} {
			name := "engine-verification-" + kind + "-" + platform + "-" + opts.SourceSHA
			expected["Engine Verification / "+kind+" / "+platform] = name
			artifactNames = append(artifactNames, name)
		}
	}
	requiredSteps := []string{"Run only the selected engine's Rust tests", "Build only the selected independent Rust engine", "Assemble local native wheel (unreleased)", "Verify the wheel in Go and install into isolated consumer", "Preserve engine verification evidence"}
	admission := 0
	seen := map[string]bool{}
	for _, job := range jobs {
		if job.Name == "verification-admission" {
			admission++
			if job.Status != "completed" || job.Conclusion != "success" {
				return GateResult{}, fmt.Errorf("canonical admission did not pass")
			}
		}
		if !strings.HasPrefix(job.Name, "Engine Verification / ") {
			continue
		}
		if _, ok := expected[job.Name]; !ok || seen[job.Name] || job.Status != "completed" || job.Conclusion != "success" {
			return GateResult{}, fmt.Errorf("missing, duplicate or unsuccessful selected engine job: %s", job.Name)
		}
		for _, required := range requiredSteps {
			passed := 0
			for _, step := range job.Steps {
				if step.Name == required && step.Conclusion == "success" {
					passed++
				}
			}
			if passed != 1 {
				return GateResult{}, fmt.Errorf("required check not passed: %s / %s", job.Name, required)
			}
		}
		seen[job.Name] = true
	}
	if admission != 1 || len(seen) != len(expected) {
		return GateResult{}, fmt.Errorf("incomplete canonical admission or platform matrix")
	}
	for _, name := range artifactNames {
		matches := 0
		for _, artifact := range artifacts {
			if artifact.Name != name {
				continue
			}
			matches++
			if artifact.Expired || artifact.Size <= 0 || artifact.WorkflowRun.ID != opts.RunID || artifact.WorkflowRun.HeadSHA != opts.SourceSHA {
				return GateResult{}, fmt.Errorf("expired, empty or unrelated artifact: %s", name)
			}
		}
		if matches != 1 {
			return GateResult{}, fmt.Errorf("missing or duplicate artifact: %s", name)
		}
	}
	return GateResult{Status: "PASS", Scope: "SELECTED_ENGINE_VERIFICATION", SourceSHA: opts.SourceSHA, RunID: opts.RunID, EnginePlatformsVerified: len(seen), Artifacts: artifactNames, PublicationAuthorized: false}, nil
}

// Gate observes the current run; restored compilation output never counts as PASS evidence.
func Gate(root string, opts GateOptions, token string) (GateResult, error) {
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(opts.Repository) || opts.RunID <= 0 || !sourcePattern.MatchString(opts.SourceSHA) || token == "" {
		return GateResult{}, fmt.Errorf("repository, run, source and Actions read token are required")
	}
	if err := Admit(root); err != nil {
		return GateResult{}, err
	}
	head, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != opts.SourceSHA {
		return GateResult{}, fmt.Errorf("gate checkout does not match the requested source")
	}
	tracked, err := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=no").Output()
	if err != nil || len(tracked) != 0 {
		return GateResult{}, fmt.Errorf("gate checkout contains tracked changes")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 30 * time.Second}
	get := func(endpoint string, out any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+opts.Repository+"/actions/runs/"+strconv.FormatInt(opts.RunID, 10)+endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Actions evidence read returned HTTP %d", resp.StatusCode)
		}
		return json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(out)
	}
	var run gateRun
	if err := get("", &run); err != nil {
		return GateResult{}, err
	}
	jobs := []gateJob{}
	artifacts := []gateArtifact{}
	for page := 1; ; page++ {
		var result struct{ Jobs []gateJob }
		if err := get("/jobs?filter=latest&per_page=100&page="+strconv.Itoa(page), &result); err != nil {
			return GateResult{}, err
		}
		jobs = append(jobs, result.Jobs...)
		if len(result.Jobs) < 100 {
			break
		}
	}
	for page := 1; ; page++ {
		var result struct{ Artifacts []gateArtifact }
		if err := get("/artifacts?per_page=100&page="+strconv.Itoa(page), &result); err != nil {
			return GateResult{}, err
		}
		artifacts = append(artifacts, result.Artifacts...)
		if len(result.Artifacts) < 100 {
			break
		}
	}
	return validateGate(opts, run, jobs, artifacts)
}
