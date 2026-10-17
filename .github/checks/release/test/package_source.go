package releasetest

import (
	"fmt"
	"os"
	"path/filepath"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

// The source distribution is built once on the required Linux host, then rebuilt
// and installed in a fresh environment with no local native wheel candidates.
func verifySourcePackage(root string, s release.Selection, output, check, version, tool string) (release.File, error) {
	stage, cleanup, err := toolSource(root, s.SourceSHA)
	if err != nil {
		return release.File{}, err
	}
	defer cleanup()
	if err := release.WriteJSON(filepath.Join(stage, "ptsip-build-source.json"), map[string]string{"schema_version": "ptsip-build-source/v1", "source_sha": s.SourceSHA}); err != nil {
		return release.File{}, err
	}
	if err := command(stage, nil, "python", "-m", "build", "--sdist", "--outdir", output); err != nil {
		return release.File{}, err
	}
	name := "ptsip-" + version + ".tar.gz"
	source := filepath.Join(output, name)
	verify := `import json, sys, tarfile, tomllib
with tarfile.open(sys.argv[1]) as archive:
 prefix='ptsip-'+sys.argv[2]+'/'
 names=set(archive.getnames())
 config=tomllib.loads(archive.extractfile(prefix+'pyproject.toml').read().decode())
 assert config['build-system']['build-backend']=='ptsip_build_backend'
 assert not any(x.startswith('ptsip-engine-') for x in config['project']['dependencies'])
 descriptor=json.load(archive.extractfile(prefix+'ptsip-build-source.json'))
 assert descriptor['source_sha']==sys.argv[3]
 for kind in config['tool']['ptsip']['release']['engines']:
  for ref in ('Cargo.toml','Cargo.lock','src/main.rs'):
   assert prefix+'src/ptsip/engines/'+kind+'/'+ref in names, (kind,ref)
print('Complete source distribution contains the exact source, Rust crates and lockfiles')
`
	if err := command(root, nil, "python", "-c", verify, source, version, s.SourceSHA); err != nil {
		return release.File{}, err
	}
	dir, err := os.MkdirTemp("", "ptsip-source-consumer-")
	if err != nil {
		return release.File{}, err
	}
	defer os.RemoveAll(dir)
	if err := command(root, nil, "python", "-m", "venv", dir); err != nil {
		return release.File{}, err
	}
	python := filepath.Join(dir, "bin/python")
	if err := command(dir, nil, python, "-m", "pip", "install", "--disable-pip-version-check", source); err != nil {
		return release.File{}, fmt.Errorf("source Package installation failed: %w", err)
	}
	if err := command(dir, nil, python, "-c", check, version, tool); err != nil {
		return release.File{}, err
	}
	b, err := os.ReadFile(source)
	if err != nil {
		return release.File{}, err
	}
	return release.File{Name: name, SHA256: release.Digest(b), Platform: "any"}, nil
}
