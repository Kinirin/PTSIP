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

    // Build the sdist exactly once and check its declared source. This is
    // not the full independent sdist-install/compatibility integration Join.
    stage,cleanup,err:=toolSource(root,s.SourceSHA)
    if err!=nil{return r,err}
    defer cleanup()
    if err=release.WriteJSON(filepath.Join(stage,"ptsip-build-source.json"),map[string]string{"schema_version":"ptsip-build-source/v1","source_sha":s.SourceSHA});err!=nil{return r,err}
    dist:=filepath.Join(root,filepath.FromSlash(output))
    if err=command(stage,nil,"python","-m","build","--sdist","--outdir",dist);err!=nil{return r,err}
    sourceName:="ptsip-"+c.PackagePyPI+".tar.gz"
    sourcePath:=filepath.Join(dist,sourceName)
    verifier:=`import tarfile,sys,json,tomllib
with tarfile.open(sys.argv[1]) as f:
 prefix='ptsip-'+sys.argv[2]+'/'
 names=set(f.getnames())
 d=json.load(f.extractfile(prefix+'ptsip-build-source.json'))
 assert d['source_sha']==sys.argv[3]
 project=tomllib.loads(f.extractfile(prefix+'pyproject.toml').read().decode())
 assert project['project']['version']==sys.argv[2]
 assert project['tool']['ptsip']['release']['profile_version']=='pp.2.00'
 assert prefix+'schemas/ptsip-profile-pp-2.00.schema.json' in names
 for engine in ('context','storage','validation','discovery','planning','execution','projection'):
  for ext in ('Cargo.toml','Cargo.lock','src/main.rs'):
   assert prefix+'src/ptsip/engines/'+engine+'/'+ext in names
print('LIGHTWEIGHT_SOURCE_DISTRIBUTION_CONTENTS_OK')`
    if err=command(root,nil,"python","-c",verifier,sourcePath,c.PackagePyPI,s.SourceSHA);err!=nil{return r,err}
    b,err:=os.ReadFile(sourcePath)
    if err!=nil{return r,err}
    r.SourceDist=&release.File{Name:sourceName,SHA256:release.Digest(b),Platform:"any"}
    if len(r.SourceDist.SHA256)!=64{return r,fmt.Errorf("sdist digest missing")}
    r.Status="PASS"
    if err=release.WriteJSON(filepath.Join(dist,"package-receipt.json"),r);err!=nil{return r,err}
    return r,nil
}
