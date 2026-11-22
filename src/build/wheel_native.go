package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/pelletier/go-toml/v2"
)

var macOSArm64Tag = regexp.MustCompile("^macosx_[0-9]+_[0-9]+_arm64$")
var engineKindPattern = regexp.MustCompile("^[a-z]+$")
var toolVersionPattern = regexp.MustCompile(`^T\.[0-9]+\.[0-9]+\.[0-9]+(?:\.[A-Za-z0-9]+)*$`)

type wheelHost struct {
	tag, platform, target, extension string
}

func currentWheelHost(tag string) (wheelHost, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64":
		if tag == "win_amd64" {
			return wheelHost{tag, "windows-x64", "x86_64-pc-windows-msvc", ".exe"}, nil
		}
	case "linux/amd64":
		if tag == "linux_x86_64" {
			return wheelHost{tag, "linux-x64", "x86_64-unknown-linux-gnu", ""}, nil
		}
	case "darwin/arm64":
		if macOSArm64Tag.MatchString(tag) {
			return wheelHost{tag, "macos-arm64", "aarch64-apple-darwin", ""}, nil
		}
	}
	return wheelHost{}, fmt.Errorf("unsupported wheel host %s/%s for tag %q", runtime.GOOS, runtime.GOARCH, tag)
}

type nativeWheelBuilder struct {
	root, revision, stage string
	host                  wheelHost
	progress              io.Writer
}

func (b nativeWheelBuilder) goBinary(module, output, target string, flags ...string) error {
	args := []string{"build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-o", output}
	args = append(args, flags...)
	command := exec.Command("go", append(args, target)...)
	command.Dir = filepath.Join(b.root, filepath.FromSlash(module))
	command.Env = append(command.Environ(), "GOWORK=off", "CGO_ENABLED=0", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	command.Stdout, command.Stderr = b.progress, b.progress
	return command.Run()
}

func jsonAsset(files wheelContents, name string, document any) error {
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	return files.add(name, append(raw, '\n'), false)
}

func (b nativeWheelBuilder) addGo(files wheelContents, distribution string) error {
	const manifest = "common_contract/registry/manifest.json"
	pathOnDisk := filepath.Join(b.root, "src", "ptsip", filepath.FromSlash(manifest))
	raw, err := os.ReadFile(pathOnDisk)
	if err != nil {
		return err
	}
	if !bytes.Equal(files["ptsip/"+manifest].data, raw) {
		return fmt.Errorf("staged Common Contract registry differs from source")
	}
	if _, err := jsondoc.Decode(raw); err != nil {
		return fmt.Errorf("read Common Contract registry: %w", err)
	}
	type asset struct {
		Resource string `json:"resource"`
		SHA256   string `json:"sha256"`
	}
	var registry struct {
		Revision    string  `json:"revision"`
		Contracts   []asset `json:"contracts"`
		Diagnostics []asset `json:"diagnostics"`
	}
	if err := json.Unmarshal(raw, &registry); err != nil {
		return err
	}
	if registry.Revision == "" || len(registry.Contracts) == 0 || len(registry.Diagnostics) == 0 {
		return fmt.Errorf("incomplete Common Contract registry")
	}
	for _, entry := range append(registry.Contracts, registry.Diagnostics...) {
		if path.Clean(entry.Resource) != entry.Resource ||
			!(strings.HasPrefix(entry.Resource, "schemas/policy/") || strings.HasPrefix(entry.Resource, "schemas/plan/")) ||
			!strings.HasSuffix(entry.Resource, ".schema.json") || strings.ContainsAny(entry.Resource, "\\:") {
			return fmt.Errorf("unsupported Common Contract resource %q", entry.Resource)
		}
		member := "ptsip/" + entry.Resource
		if data, ok := files[member]; !ok || wheelDigest(data.data) != entry.SHA256 {
			return fmt.Errorf("staged Common Contract resource absent or different: %s", member)
		}
	}
	digest := wheelDigest(raw)
	common := "ptsip-common-contract" + b.host.extension
	command := filepath.Join(b.stage, common)
	flags := "-X main.sourceRevision=" + b.revision + " -X main.registryDigest=" + digest
	if err := b.goBinary("src/ptsip/common_contract", command, "./cmd/ptsip-common-contract", "-ldflags", flags); err != nil {
		return fmt.Errorf("build Common Contract executable: %w", err)
	}
	binary, err := os.ReadFile(command)
	if err != nil {
		return err
	}
	if err := files.add("ptsip/common_contract/"+common, binary, true); err != nil {
		return err
	}
	err = jsonAsset(files, "ptsip/common_contract/origin.json", map[string]string{
		"format": "ptsip-common-contract-wheel-origin/v1", "source_sha": b.revision, "goos": runtime.GOOS, "goarch": runtime.GOARCH,
		"registry_revision": registry.Revision, "registry_sha256": digest, "binary": common, "binary_sha256": wheelDigest(binary),
	})
	if err != nil {
		return err
	}
	versionSource := files["ptsip/VERSION"].data
	var versions struct {
		Tool string `toml:"tool_version"`
	}
	if err := toml.Unmarshal(versionSource, &versions); err != nil || !toolVersionPattern.MatchString(versions.Tool) {
		return fmt.Errorf("invalid Tool version in src/VERSION")
	}
	command = filepath.Join(b.stage, "ptsip"+b.host.extension)
	if err := b.goBinary("src/ptsip", command, ".", "-ldflags", "-X main.toolVersion="+versions.Tool); err != nil {
		return fmt.Errorf("build PTSIP Go router: %w", err)
	}
	binary, err = os.ReadFile(command)
	if err != nil {
		return err
	}
	scripts := distribution + ".data/scripts/"
	if err := files.add(scripts+"ptsip"+b.host.extension, binary, true); err != nil {
		return err
	}
	if err := files.add(scripts+"ptsip-go"+b.host.extension, binary, true); err != nil {
		return err
	}
	app := "ptsip-app" + b.host.extension
	command = filepath.Join(b.stage, app)
	if err := b.goBinary("src/ptsip", command, "./app/cmd/ptsip-app", "-ldflags", "-X main.toolVersion="+versions.Tool); err != nil {
		return fmt.Errorf("build PTSIP App: %w", err)
	}
	binary, err = os.ReadFile(command)
	if err != nil {
		return err
	}
	return files.add(scripts+app, binary, true)
}

type engineOrigin struct {
	Format              string `json:"schema_version"`
	Source              string `json:"source_sha"`
	EngineID            string `json:"engine_id"`
	Version             string `json:"engine_version"`
	Target              string `json:"host_target"`
	Entrypoint          string `json:"entrypoint"`
	Digest              string `json:"binary_sha256"`
	ExecutionAuthorized bool   `json:"execution_authorized"`
}

func (b nativeWheelBuilder) addEngines(files wheelContents, distribution string) error {
	var index struct {
		Entries []struct {
			engineEntry
			ID      string `json:"engine_id"`
			Version string `json:"version"`
		} `json:"official_engine_entries"`
	}
	if err := readJSON(filepath.Join(b.root, filepath.FromSlash(engineIndex)), &index); err != nil {
		return err
	}
	if len(index.Entries) == 0 {
		return fmt.Errorf("no official Rust engines registered")
	}
	origins := make(map[string]engineOrigin, len(index.Entries))
	stage := filepath.Join(b.stage, "cargo")
	for _, engine := range index.Entries {
		kind := strings.ToLower(engine.Kind)
		if !engineKindPattern.MatchString(kind) || engine.CargoFile != engineRoot+kind+"/Cargo.toml" ||
			engine.ID == "" || engine.Version == "" {
			return fmt.Errorf("invalid Rust engine registration %q", engine.Kind)
		}
		if _, exists := origins[kind]; exists {
			return fmt.Errorf("duplicate Rust engine registration %q", engine.Kind)
		}
		manifest := filepath.Join(b.root, filepath.FromSlash(engine.CargoFile))
		binName := "ptsip-engine-" + kind
		if err := b.checkCrate(manifest, binName, engine.Version); err != nil {
			return err
		}
		command := exec.Command("cargo", "build", "--locked", "--release", "--target", b.host.target,
			"--manifest-path", manifest, "--target-dir", stage, "--bin", binName)
		command.Dir = b.root
		command.Stdout, command.Stderr = b.progress, b.progress
		if err := command.Run(); err != nil {
			return fmt.Errorf("build Rust engine %s: %w", kind, err)
		}
		entrypoint := binName + b.host.extension
		binary, err := os.ReadFile(filepath.Join(stage, b.host.target, "release", entrypoint))
		if err != nil {
			return err
		}
		if err := files.add(distribution+".data/scripts/"+entrypoint, binary, true); err != nil {
			return err
		}
		origins[kind] = engineOrigin{
			Format: "ptsip-independent-wheel-origin/v1", Source: b.revision, EngineID: engine.ID, Version: engine.Version,
			Target: b.host.target, Entrypoint: entrypoint, Digest: wheelDigest(binary),
		}
	}
	_, version, _ := strings.Cut(distribution, "-")
	return jsonAsset(files, distribution+".dist-info/ptsip-engines.json", map[string]any{
		"schema_version": "ptsip-bundled-engine-origins/v1", "package_version": version, "platform": b.host.platform,
		"engines": origins, "execution_authorized": false,
	})
}

func (b nativeWheelBuilder) checkCrate(manifest, binName, version string) error {
	command := exec.Command("cargo", "metadata", "--locked", "--no-deps", "--format-version", "1", "--manifest-path", manifest)
	command.Dir = b.root
	command.Stderr = b.progress
	raw, err := command.Output()
	if err != nil {
		return fmt.Errorf("read Rust crate %s: %w", binName, err)
	}
	var workspace struct {
		Packages []struct {
			Version string
			Targets []struct {
				Name string
				Kind []string
			}
		}
	}
	if err := json.Unmarshal(raw, &workspace); err != nil {
		return err
	}
	for _, crate := range workspace.Packages {
		for _, target := range crate.Targets {
			if target.Name == binName && len(target.Kind) == 1 && target.Kind[0] == "bin" && crate.Version == version {
				return nil
			}
		}
	}
	return fmt.Errorf("Rust crate %s does not match registered version %s", binName, version)
}
