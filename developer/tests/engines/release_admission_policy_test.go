package engines_test

import (
    "os"
    "path/filepath"
    "testing"
    "github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestEffectiveCTRLReleaseAdmissionSchema(t *testing.T) {
    checker:=testrepo.Open(testrepo.Root(t))
    policy:=document(t,"developer/policy/CTRL/release-admission.policy.json")
    if err:=checker.Validate(".ptsip/contracts/schemas/release/ctrl-release-admission.schema.json",policy);err!=nil{t.Fatal(err)}
    identity:=policy["policy_identity"].(object)
    if identity["status"]!="APPROVED" || identity["substatus"]!="Applying"{t.Fatal("CTRL publication policy inactive")}
    a:=policy["authorization"].(object)
    if a["mode"]!="PER_RELEASE_CONDITIONAL" || a["test_pass_grants_publication"]!=false{t.Fatal("unsafe publication permission")}
    projects:=a["pypi_projects"].([]any)
    if len(projects)!=1 || projects[0]!="PTSIP"{t.Fatal("PyPI project scope differs")}
}

func TestCTRLAdmissionSchemaDoesNotDependOnLegacyCatalog(t *testing.T) {
    const schemaRef = ".ptsip/contracts/schemas/release/ctrl-release-admission.schema.json"
    source := testrepo.Open(testrepo.Root(t))
    raw, err := os.ReadFile(filepath.Join(source.Root, filepath.FromSlash(schemaRef)))
    if err != nil { t.Fatal(err) }
    // Only the requested current schema exists in this fixture. In particular,
    // no retired developer-policy catalog or nested status validator exists.
    isolated := testrepo.Open(t.TempDir())
    target := filepath.Join(isolated.Root, filepath.FromSlash(schemaRef))
    if err := os.MkdirAll(filepath.Dir(target),0755);err!=nil {t.Fatal(err)}
    if err := os.WriteFile(target,raw,0644);err!=nil {t.Fatal(err)}
    policy := document(t,"developer/policy/CTRL/release-admission.policy.json")
    if err := isolated.Validate(schemaRef,policy);err!=nil {
        t.Fatalf("current CTRL schema must validate without legacy global resource preload: %v",err)
    }
}
