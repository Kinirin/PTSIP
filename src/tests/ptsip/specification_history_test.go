package contracts_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	specification "github.com/Kinirin/PTSIP/src/ptsip/ptsip_specification"
)

const historyFixturePlanID = "urn:ptsip:contract:plan:unit"

func publishedHistoryContract(key, id, version, constraint string) specification.PublishedContract {
	raw := []byte(fmt.Sprintf(`{"$id":%q,"version":%q,"$anchor":"stable","type":%q}`, id, version, constraint))
	return specification.PublishedContract{
		Key: key, Domain: "plan", ID: id, Version: version, SourcePath: "src/ptsip/schemas/plan/" + key + ".schema.json",
		SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Bytes: raw,
	}
}

func TestSpecificationPublicationBoundHistory(t *testing.T) {
	root := t.TempDir()
	const dir = "src/ptsip/schemas/history/plan/"
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0755); err != nil {
		t.Fatal(err)
	}
	previous := specification.Publication{
		Platform: "github", Reference: "release-40", Version: "0.3.7", SourceSHA: strings.Repeat("a", 40),
		ArtifactName: "ptsip-0.3.7.whl", ArtifactSHA256: strings.Repeat("a", 64),
	}
	current := specification.Publication{
		Platform: "github", Reference: "release-41", Version: "0.3.8", SourceSHA: strings.Repeat("b", 40),
		ArtifactName: "ptsip-0.3.8.whl", ArtifactSHA256: strings.Repeat("b", 64),
	}
	oldPlan := publishedHistoryContract("plan", historyFixturePlanID, "SCH.1.00", "string")
	newPlan := publishedHistoryContract("plan", historyFixturePlanID, "SCH.2.00", "integer")
	unchanged := publishedHistoryContract("unchanged", "urn:ptsip:plan:unchanged", "SCH.1.00", "string")
	firstPublication := publishedHistoryContract("first", "urn:ptsip:plan:first", "SCH.1.00", "string")
	after := []specification.PublishedContract{newPlan, unchanged, firstPublication}
	verify := func(_ context.Context, requested specification.Publication) ([]specification.PublishedContract, error) {
		switch requested {
		case previous:
			return []specification.PublishedContract{oldPlan, unchanged}, nil
		case current:
			return after, nil
		default:
			return nil, errors.New("fixture platform has no matching published artifact")
		}
	}
	target := specification.HistoryTarget{ArchivePath: dir + "unit.fixture.schema.json", EvidencePath: dir + "unit.fixture.evidence.json"}
	targets := map[string]specification.HistoryTarget{"plan": target}
	ctx := context.Background()
	archivePath := filepath.Join(root, filepath.FromSlash(target.ArchivePath))
	evidencePath := filepath.Join(root, filepath.FromSlash(target.EvidencePath))
	prepare := func(verifier specification.PublicationVerifier, mapping map[string]specification.HistoryTarget, refs specification.HistoryReferenceVerifier) (specification.HistoryPlan, error) {
		return specification.PrepareHistory(ctx, root, previous, current, verifier, mapping, refs)
	}
	if _, err := prepare(nil, targets, nil); err == nil {
		t.Fatal("unverified publication was accepted")
	}
	missing := current
	missing.Reference = "no-published-artifact"
	if _, err := specification.PrepareHistory(ctx, root, previous, missing, verify, targets, nil); err == nil {
		t.Fatal("absent platform publication was accepted")
	}
	if _, err := prepare(verify, nil, nil); err == nil || !strings.Contains(err.Error(), "paths unresolved") {
		t.Fatalf("missing exact resource paths: %v", err)
	}
	tampered := newPlan
	tampered.Bytes = bytes.Replace(tampered.Bytes, []byte("integer"), []byte("boolean"), 1)
	after[0] = tampered
	if _, err := prepare(verify, targets, nil); err == nil || !strings.Contains(err.Error(), "invalid published contract evidence") {
		t.Fatalf("artifact drift was admitted: %v", err)
	}
	after[0] = publishedHistoryContract("plan", historyFixturePlanID, "SCH.1.00", "boolean")
	if _, err := prepare(verify, targets, nil); err == nil || !strings.Contains(err.Error(), "without a version change") {
		t.Fatalf("same published version with different semantics was admitted: %v", err)
	}
	referenceBody := bytes.ReplaceAll(newPlan.Bytes, []byte(`"type":"integer"`), []byte(`"$ref":"../related.json#stable"`))
	after[0] = newPlan
	after[0].Bytes = referenceBody
	after[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(referenceBody))
	if _, err := prepare(verify, targets, nil); err == nil || !strings.Contains(err.Error(), "reference resolution is unverified") {
		t.Fatalf("relocated unresolved reference was admitted: %v", err)
	}
	if _, err := prepare(verify, targets, func(context.Context, specification.HistoryRecord, []byte, []byte) error {
		return errors.New("published file anchor resolves to a different target")
	}); err == nil || !strings.Contains(err.Error(), "reference resolution differs") {
		t.Fatalf("reference mismatch was admitted: %v", err)
	}
	after[0] = newPlan

	plan, err := prepare(verify, targets, nil)
	if err != nil || len(plan.Records()) != 1 || plan.Records()[0].ArchiveID != historyFixturePlanID+":2_00" {
		t.Fatalf("expected the one changed published contract: %v, %v", plan.Records(), err)
	}
	if _, err := os.Stat(archivePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("preparing history wrote files")
	}
	if err := plan.Materialize(ctx); err != nil {
		t.Fatal(err)
	}
	if err := plan.Materialize(ctx); err != nil {
		t.Fatalf("repeated verified publication was not idempotent: %v", err)
	}
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		ID      string `json:"$id"`
		Anchor  string `json:"$anchor"`
		Version string `json:"version"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(archive, &schema); err != nil || schema.ID != historyFixturePlanID+":2_00" ||
		schema.Anchor != "stable" || schema.Version != "SCH.2.00" || schema.Type != "integer" {
		t.Fatalf("archive did not preserve version, anchor and content: %+v, %v", schema, err)
	}
	evidence, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	var record specification.HistoryRecord
	if err := json.Unmarshal(evidence, &record); err != nil || record.Publication != current ||
		record.OriginalID != historyFixturePlanID || record.OriginalSHA != newPlan.SHA256 ||
		record.ArchiveSHA != fmt.Sprintf("%x", sha256.Sum256(archive)) {
		t.Fatalf("publication provenance was not preserved: %+v, %v", record, err)
	}
	duplicate := filepath.Join(root, filepath.FromSlash(dir+"duplicate.fixture.json"))
	if err := os.WriteFile(duplicate, archive, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := prepare(verify, targets, nil); err == nil || !strings.Contains(err.Error(), "duplicate existing history ID") {
		t.Fatalf("duplicate archive identity was admitted: %v", err)
	}
	if err := os.Remove(duplicate); err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(archive, []byte("integer"), []byte("boolean"), 1)
	if err := os.WriteFile(archivePath, changed, 0644); err != nil {
		t.Fatal(err)
	}
	if err := plan.Materialize(ctx); err == nil || !strings.Contains(err.Error(), "immutable history conflicts") {
		t.Fatalf("immutable history conflict was overwritten: %v", err)
	}
	got, err := os.ReadFile(archivePath)
	if err != nil || !bytes.Equal(got, changed) {
		t.Fatalf("the conflicting history was modified: %v", err)
	}
}

func TestSpecificationPublishedWheelContractSourceBinding(t *testing.T) {
	contract := publishedHistoryContract("plan", historyFixturePlanID, "SCH.2.00", "integer")
	member := "ptsip/schemas/plan/plan.schema.json"
	var raw bytes.Buffer
	zipWriter := zip.NewWriter(&raw)
	entry, err := zipWriter.Create(member)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(contract.Bytes); err != nil {
		t.Fatal(err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	publication := specification.Publication{
		Platform: "pypi", Reference: "PTSIP/0.3.8", Version: "0.3.8", SourceSHA: strings.Repeat("a", 40),
		ArtifactName: "ptsip-0.3.8.whl", ArtifactSHA256: fmt.Sprintf("%x", sha256.Sum256(raw.Bytes())),
	}
	resource := specification.PublishedResource{
		Key: contract.Key, Domain: contract.Domain, ID: contract.ID, Version: contract.Version,
		SourcePath: contract.SourcePath, WheelPath: member,
	}
	gitBytes := contract.Bytes
	verify := specification.PublishedWheelVerifier(
		func(_ context.Context, platform, reference, version, sha, name, digest string) ([]byte, error) {
			if platform != publication.Platform || reference != publication.Reference || version != publication.Version ||
				sha != publication.SourceSHA || name != publication.ArtifactName || digest != publication.ArtifactSHA256 {
				return nil, errors.New("no publication matching the coordinates")
			}
			return raw.Bytes(), nil
		},
		func(_ context.Context, sha string) ([]specification.PublishedResource, error) {
			if sha != publication.SourceSHA {
				t.Fatalf("registration read from the wrong revision: %s", sha)
			}
			return []specification.PublishedResource{resource}, nil
		},
		func(_ context.Context, sha, path string) ([]byte, error) {
			if sha != publication.SourceSHA || path != contract.SourcePath {
				t.Fatalf("source bytes read from the wrong revision or path: %s %s", sha, path)
			}
			return gitBytes, nil
		},
	)
	resources, err := verify(context.Background(), publication)
	if err != nil || len(resources) != 1 || resources[0].SHA256 != contract.SHA256 {
		t.Fatalf("published resource was not verified against source: %v, %v", resources, err)
	}
	gitBytes = bytes.Replace(contract.Bytes, []byte("integer"), []byte("boolean"), 1)
	if _, err := verify(context.Background(), publication); err == nil || !strings.Contains(err.Error(), "differs from exact source") {
		t.Fatalf("wheel resource from another source revision was admitted: %v", err)
	}
	resource.WheelPath = "ptsip/schemas/plan/missing.schema.json"
	if _, err := verify(context.Background(), publication); err == nil || !strings.Contains(err.Error(), "missing from published wheel") {
		t.Fatalf("missing published resource was admitted: %v", err)
	}
}
