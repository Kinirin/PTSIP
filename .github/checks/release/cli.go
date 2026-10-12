package release

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

func CLI(args []string) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("registered release command is required")
	}
	command := args[0]
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	root := f.String("root", ".", "repository root")
	targets := f.String("targets", "all", "comma-separated registered targets")
	disabled := f.String("disabled", "", "switched-off targets")
	base := f.String("package-base", "", "owner-selected Package base")
	selection := f.String("selection", ".artifacts/release/selection.json", "exact-source selection file")
	output := f.String("output", ".artifacts/release/output", "artifact output directory")
	input := f.String("input", ".artifacts/release/components", "component artifact input directory")
	id := f.String("target", "", "registered component")
	platform := f.String("platform", "", "native host platform")
	repository := f.String("github-repository", os.Getenv("GITHUB_REPOSITORY"), "owner/name")
	api := f.String("github-api-url", os.Getenv("GITHUB_API_URL"), "GitHub API root")
	offline := f.Bool("offline", false, "local selection without claiming publication state")
	publish := f.Bool("publish", false, "publish assembled Package")
	eventTag := f.String("release-tag", "", "published immutable tag event")
	if err := f.Parse(args[1:]); err != nil {
		return nil, err
	}
	if len(f.Args()) != 0 {
		return nil, fmt.Errorf("unexpected positional arguments")
	}
	repo, err := machine.Open(*root)
	if err != nil {
		return nil, err
	}
	catalog, err := loadCatalog(repo)
	if err != nil {
		return nil, err
	}
	admitted := false
	for _, v := range catalog.Commands {
		admitted = admitted || v == command
	}
	if !admitted {
		return nil, fmt.Errorf("unregistered release command: %s", command)
	}
	switch command {
	case "sync-versions":
		return SyncEngineVersions(repo, *targets)
	case "selection":
		return readSelection(repo, *selection)
	case "contracts":
		s, err := readSelection(repo, *selection)
		if err != nil {
			return nil, err
		}
		c, err := selectedComponent(s, *id)
		if err != nil || c.Kind != "ENGINE" {
			return nil, fmt.Errorf("contracts helper requires one registered engine")
		}
		files, err := contractFiles(repo, c)
		if err != nil {
			return nil, err
		}
		dir, err := repo.Path(*output)
		if err != nil {
			return nil, err
		}
		for name, source := range files {
			path := filepath.Join(dir, filepath.FromSlash(name))
			if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return nil, err
			}
			data, e := os.ReadFile(source)
			if e != nil {
				return nil, e
			}
			if err = os.WriteFile(path, data, 0644); err != nil {
				return nil, err
			}
		}
		return machine.Object{"status": "PASS", "target": *id, "artifact_role": "NEUTRAL_CONTRACT_PROJECTION", "executables_created": false}, nil
	case "plan":
		if *eventTag != "" {
			*targets = ""
			for _, candidate := range append([]string{"tool", "profile"}, catalog.EngineIDs...) {
				prefix := catalog.TagPrefixes[candidate]
				if candidate != "tool" && candidate != "profile" {
					prefix = catalog.TagPrefixes["engine"] + candidate + "-v"
				}
				if strings.HasPrefix(*eventTag, prefix) {
					*targets = candidate
					break
				}
			}
			if *targets == "" {
				return nil, fmt.Errorf("published tag does not belong to an admitted independent component")
			}
		}
		s, err := Plan(repo, *targets, *disabled, *base, *repository, *api, *offline)
		if err != nil {
			return nil, err
		}
		if *eventTag != "" {
			sha, err := requireRun(repo, "git", "rev-parse", *eventTag+"^{commit}")
			if err != nil || sha != s.SourceSHA {
				return nil, fmt.Errorf("published tag must resolve to the exact checkout")
			}
			for i, c := range s.Components {
				if c.Tag == *eventTag {
					s.Components[i].State = "BUILD"
					s.Components[i].SourceSHA = s.SourceSHA
				}
			}
		}
		path, err := repo.Path(*selection)
		if err != nil {
			return nil, err
		}
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}
		if err = writeJSON(path, s); err != nil {
			return nil, err
		}
		return s, nil
	case "build":
		return BuildComponent(repo, *selection, *id, *platform, *output)
	case "verify":
		s, err := readSelection(repo, *selection)
		if err != nil {
			return nil, err
		}
		dir, err := repo.Path(*output)
		if err != nil {
			return nil, err
		}
		receipts, err := VerifyReceipts(s, *id, dir, true)
		if err != nil {
			return nil, err
		}
		return machine.Object{"status": "PASS", "component": *id, "receipts": receipts}, nil
	case "publish":
		return PublishComponent(repo, *selection, *id, *output, *repository, *api)
	case "assemble":
		return Assemble(repo, *selection, *input, *output, *repository, *api, *publish)
	default:
		return nil, fmt.Errorf("unsupported admitted release command")
	}
}
