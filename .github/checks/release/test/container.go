package releasetest

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

const containerCheck = `import importlib.metadata as metadata, importlib.resources as resources, json, yaml
from pathlib import Path
from ptsip.constants import TOOL_VERSION
from ptsip.engine_installation import inspect
m=json.loads(Path('/opt/ptsip-release/shipping-manifest.json').read_text())
components={c['id']:c for c in m['package']['components']}
assert metadata.version('PTSIP') == m['pypi_package']['version']
assert TOOL_VERSION == components['tool']['version']
requirements=[r.replace(' ','') for r in metadata.requires('PTSIP') if r.startswith('ptsip-engine-')]
assert len(requirements)==7
for name,c in components.items():
 if c['kind']!='engine': continue
 assert 'ptsip-engine-'+name+'=='+c['version'] in requirements
 d=metadata.distribution('ptsip-engine-'+name)
 origin=json.loads(d.read_text('ptsip-origin.json'))
 assert d.version==c['version'] and origin['engine_version']==c['version']
 assert origin['source_sha']==c['source_sha']
status=inspect()
assert len(status['entries'])==7 and status['all_installed']
assert status['execution_authorized'] is False
p=resources.files('ptsip.profiles')
assert p.joinpath('index.yaml').is_file() and not p.joinpath('history').is_dir()
catalog=yaml.safe_load(resources.files('ptsip').joinpath('specdata/project-profile-contracts.yaml').read_text())
assert catalog['current']==components['profiles']['version']
print(json.dumps({'engine_count':len(status['entries']),'all_installed':status['all_installed'],'execution_authorized':False,'profile_history_excluded':True,'tool':TOOL_VERSION,'profile':catalog['current'],'entries':status['entries']}))
`

func ExecuteContainer(root string, s release.Selection, id, platform string) (Receipt, error) {
	var result Receipt
	if id != "container" || platform != "linux-x64" {
		return result, fmt.Errorf("unregistered temporary container target")
	}
	if err := release.ValidateContainerSelection(root, s, true); err != nil {
		return result, err
	}
	c, err := release.ReadContainerContract(root)
	if err != nil {
		return result, err
	}
	context := ".artifacts/release/container/build"
	m, err := release.PrepareContainer(root, context)
	if err != nil {
		return result, err
	}
	tag := "ptsip-container-test:" + s.SourceSHA
	args := []string{"docker", "build", "--platform", c.Platform, "--provenance=false", "--build-arg", "BASE_IMAGE=" + c.Base, "--tag", tag}
	labels := map[string]string{"org.opencontainers.image.source": "https://github.com/Kinirin/PTSIP", "org.opencontainers.image.revision": release.ContainerSource, "org.opencontainers.image.version": m.PackageDistribution.Version, "org.opencontainers.image.description": "Temporary PTSIP complete Package with Tool, Profiles and all seven independently versioned engines", "io.ptsip.package.identity": m.Snapshot.Version, "io.ptsip.container.recipe.source-sha": s.SourceSHA}
	for key, value := range labels {
		args = append(args, "--label", key+"="+value)
	}
	args = append(args, filepath.Join(root, filepath.FromSlash(context)))
	if err := command(root, nil, args...); err != nil {
		return result, err
	}
	version, err := release.Run(root, "docker", "run", "--rm", tag, "--version")
	if err != nil {
		return result, err
	}
	tool := ""
	for _, component := range m.Snapshot.Components {
		if component.ID == "tool" {
			tool = component.Version
		}
	}
	if strings.TrimSpace(string(version)) != "PTSIP Tool "+tool {
		return result, fmt.Errorf("container CLI Tool version differs")
	}
	if err := command(root, nil, "docker", "run", "--rm", "--entrypoint", "python", tag, "-m", "pip", "check"); err != nil {
		return result, err
	}
	data, err := release.Run(root, "docker", "run", "--rm", "--entrypoint", "python", tag, "-c", containerCheck)
	if err != nil {
		return result, err
	}
	var runtime map[string]any
	if err := json.Unmarshal(data, &runtime); err != nil {
		return result, err
	}
	inspection, err := release.Run(root, "docker", "image", "inspect", tag)
	if err != nil {
		return result, err
	}
	var images []struct {
		ID           string
		OS           string `json:"Os"`
		Architecture string
		Config       struct{ User string }
	}
	if err := json.Unmarshal(inspection, &images); err != nil || len(images) != 1 || images[0].OS != "linux" || images[0].Architecture != "amd64" || images[0].Config.User != "10001:10001" {
		return result, fmt.Errorf("container host or non-root configuration differs")
	}
	out := filepath.Join(root, ".artifacts/release/components/container/linux-x64")
	if err := os.MkdirAll(out, 0755); err != nil {
		return result, err
	}
	archive := filepath.Join(out, "image.tar.gz")
	if err := saveContainer(root, tag, archive); err != nil {
		return result, err
	}
	b, err := os.ReadFile(archive)
	if err != nil {
		return result, err
	}
	if len(b) > 128*1024*1024 {
		return result, fmt.Errorf("compressed container exceeds admitted Actions file limit")
	}
	r := release.ContainerReceipt{Schema: "ptsip-temporary-container-test/v1", Source: s.SourceSHA, OriginalSource: release.ContainerSource, Selection: s.SelectionSHA, ImageID: images[0].ID, ImageTag: tag, Archive: release.File{Name: "image.tar.gz", SHA256: release.Digest(b), Platform: platform}, ManifestSHA: c.ManifestSHA, Status: "PASS", Runtime: runtime}
	if err := release.ValidateContainerReceipt(root, s, r, out); err != nil {
		return result, err
	}
	if err := release.WriteJSON(filepath.Join(out, "container-receipt.json"), r); err != nil {
		return result, err
	}
	result = Receipt{Schema: "ptsip-release-test-receipt/v1", SourceSHA: s.SourceSHA, SelectionSHA: s.SelectionSHA, Component: id, Kind: "container", Version: m.PackageDistribution.Version, Platform: platform, Status: "PASS", Files: []release.File{r.Archive}}
	if err := release.WriteJSON(filepath.Join(out, "test-receipt.json"), result); err != nil {
		return result, err
	}
	return result, nil
}

func saveContainer(root, tag, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	z := gzip.NewWriter(f)
	cmd := exec.Command("docker", "save", tag)
	cmd.Dir = root
	cmd.Stdout = z
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	closeErr := z.Close()
	fileErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return fileErr
}
