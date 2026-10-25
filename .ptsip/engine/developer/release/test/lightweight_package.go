package releasetest

import (
    "fmt"
    "os"
    "path/filepath"
    "runtime"

    release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller"
)

// VerifyLightweightPackageInstall keeps source-bound wheel, version and native
// artifact verification without running the repeated full Package Join's
// second sdist installation, integration suite and twine pipeline.
func VerifyLightweightPackageInstall(root string, s release.Selection, input, platform string) (release.PackageReceipt,error) {
    waived,err:=release.LightweightPackageWaiver(root,s)
    if err!=nil{return release.PackageReceipt{},err}
    if !waived{return release.PackageReceipt{},fmt.Errorf("lightweight install requires active exact 0.3.8b6 waiver")}
    output:=".artifacts/release/package/"+platform
    dist:=filepath.Join(root,filepath.FromSlash(output))
    r,err:=release.BuildSourceVerificationPackageReceipt(root,s,input,output,platform)
    if err!=nil{return r,err}
    wheel:=filepath.Join(root,filepath.FromSlash(output),r.Wheel.Name)
    dir,err:=os.MkdirTemp("","ptsip-light-pip-")
    if err!=nil{return r,err}
    defer os.RemoveAll(dir)
    if err=command(root,nil,"python","-m","venv",dir);err!=nil{return r,err}
    python:=filepath.Join(dir,"bin","python")
    if runtime.GOOS=="windows"{python=filepath.Join(dir,"Scripts","python.exe")}
    if err=command(root,nil,python,"-m","pip","install","--disable-pip-version-check",wheel);err!=nil{return r,err}
    verify:=`import importlib.metadata as m,sys
from ptsip.engine_installation import inspect
from ptsip.constants import TOOL_VERSION
assert m.version('PTSIP')==sys.argv[1]
assert TOOL_VERSION==sys.argv[2]
state=inspect()
assert state['all_installed'] is True
assert state['execution_authorized'] is False
assert state['release_attested'] is False
assert state['distribution']=='SINGLE_PROJECT_PLATFORM_BUNDLES_ONLY'
assert len(state['entries'])==7
print('LIGHTWEIGHT_COMPLETE_PACKAGE_VERSION_INSTALL_OK')`
    c,err:=release.ReadComposition(root)
    if err!=nil{return r,err}
    if c.PackagePyPI!="0.3.8b6"||c.Profile!="pp.2.00"{return r,fmt.Errorf("waived package version/profile differs")}
    if err=command(root,nil,python,"-c",verify,c.PackagePyPI,c.Tool);err!=nil{return r,err}
    if err=release.VerifyInstalledComponents(r.Snapshot,python);err!=nil{return r,err}

    // The publication admission contract requires exactly one Linux sdist.
    // Windows passes the complete wheel + original component hash evidence.
    // Source metadata, Profile schema and all native Cargo sources are checked
    // in the Linux archive without the full second sdist-install Join.
    if platform=="linux-x64" {
        stage,cleanup,err:=toolSource(root,s.SourceSHA)
        if err!=nil{return r,err}
        defer cleanup()
        if err=release.WriteJSON(filepath.Join(stage,"ptsip-build-source.json"),map[string]string{
            "schema_version":"ptsip-build-source/v1",
            "source_sha":s.SourceSHA,
        });err!=nil{return r,err}
        // Use the same per-platform output directory for the Linux source artifact.
        if err=command(stage,nil,"python","-m","build","--sdist","--outdir",dist);err!=nil{return r,err}
        sourceName:="ptsip-"+c.PackagePyPI+".tar.gz"
        sourcePath:=filepath.Join(dist,sourceName)
        verifier:=`import json,sys,tarfile,tomllib
with tarfile.open(sys.argv[1]) as archive:
 prefix='ptsip-'+sys.argv[2]+'/'
 names=set(archive.getnames())
 descriptor=json.load(archive.extractfile(prefix+'ptsip-build-source.json'))
 assert descriptor['schema_version']=='ptsip-build-source/v1'
 assert descriptor['source_sha']==sys.argv[3], 'source SHA mismatch'
 config=tomllib.loads(archive.extractfile(prefix+'pyproject.toml').read().decode())
 assert config['project']['version']==sys.argv[2], 'package version mismatch'
 assert config['tool']['ptsip']['release']['profile_version']=='pp.2.00', 'Profile version mismatch'
 canonical=prefix+'schemas/ptsip-profile-pp-2.00.schema.json'
 embedded=prefix+'src/ptsip/specdata/ptsip-profile-pp-2.00.schema.json'
 assert canonical in names, 'current canonical pp.2.00 schema absent from sdist'
 assert embedded in names, 'embedded pp.2.00 schema absent from sdist'
 assert archive.extractfile(canonical).read()==archive.extractfile(embedded).read(), 'schema projections differ'
 for engine in ('context','storage','validation','discovery','planning','execution','projection'):
  for ref in ('Cargo.toml','Cargo.lock','src/main.rs'):
   assert prefix+'src/ptsip/engines/'+engine+'/'+ref in names, (engine,ref)
print('LIGHTWEIGHT_SOURCE_DISTRIBUTION_CONTENTS_OK')`
        if err=command(root,nil,"python","-c",verifier,sourcePath,c.PackagePyPI,s.SourceSHA);err!=nil{return r,err}
        data,readErr:=os.ReadFile(sourcePath)
        if readErr!=nil{return r,readErr}
        r.SourceDist=&release.File{Name:sourceName,SHA256:release.Digest(data),Platform:"any"}
        if len(r.SourceDist.SHA256)!=64{return r,fmt.Errorf("sdist digest missing")}
    }
    r.Status="PASS"
    if err=release.WriteJSON(filepath.Join(dist,"package-receipt.json"),r);err!=nil{return r,err}
    return r,nil
}
