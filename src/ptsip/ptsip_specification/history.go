package specification

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

const historyRoot = "src/ptsip/schemas/history"

var (
	publicationSHA = regexp.MustCompile(`^[a-f0-9]{40}$`)
	contractSHA    = regexp.MustCompile(`^[a-f0-9]{64}$`)
	contractSuffix = regexp.MustCompile(`^(SCH\.|pp\.)?([0-9]+(\.[0-9]+)+)$`)
)

// Publication identifies an observed immutable platform artifact. A selector
// is not proof that its publication completed.
type Publication struct {
	Platform       string `json:"platform"`
	Reference      string `json:"reference"`
	Version        string `json:"version"`
	SourceSHA      string `json:"source_sha"`
	ArtifactName   string `json:"artifact_name"`
	ArtifactSHA256 string `json:"artifact_sha256"`
}

// PublishedContract binds a registered contract to bytes read from the
// published artifact and verified against its exact source revision.
type PublishedContract struct {
	Key        string
	Domain     string
	ID         string
	Version    string
	SourcePath string
	SHA256     string
	Bytes      []byte
}

// PublicationVerifier is supplied by the existing release evidence owner. It
// must check actual platform publication and join creation evidence, artifact
// digest, exact source revision, and the complete registered inventory.
type PublicationVerifier func(context.Context, Publication) ([]PublishedContract, error)

// PublishedResource is one explicit member of the complete exact-revision
// contract registration. SourcePath and WheelPath identify independent copies.
type PublishedResource struct {
	Key, Domain, ID, Version, SourcePath, WheelPath string
}

// PublishedWheelVerifier joins an actual platform observation with the owner
// registry and exact Git source. In particular a reused wheel component only
// qualifies when its contract bytes match the declared source revision.
func PublishedWheelVerifier(
	observe func(context.Context, string, string, string, string, string, string) ([]byte, error),
	registration func(context.Context, string) ([]PublishedResource, error),
	readSource func(context.Context, string, string) ([]byte, error),
) PublicationVerifier {
	return func(ctx context.Context, p Publication) ([]PublishedContract, error) {
		if observe == nil || registration == nil || readSource == nil || !validPublication(p) {
			return nil, errors.New("published wheel observation and exact-source registration required")
		}
		raw, err := observe(ctx, p.Platform, p.Reference, p.Version, p.SourceSHA, p.ArtifactName, p.ArtifactSHA256)
		if err != nil {
			return nil, err
		}
		if digestHistory(raw) != p.ArtifactSHA256 {
			return nil, errors.New("published wheel digest differs")
		}
		resources, err := registration(ctx, p.SourceSHA)
		if err != nil {
			return nil, err
		}
		archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		if err != nil {
			return nil, err
		}
		files := make(map[string]*zip.File, len(archive.File))
		for _, f := range archive.File {
			if files[f.Name] != nil {
				return nil, fmt.Errorf("duplicate published wheel member: %s", f.Name)
			}
			files[f.Name] = f
		}
		result := make([]PublishedContract, 0, len(resources))
		wheelPaths := map[string]bool{}
		for _, resource := range resources {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if !source.Relative(resource.SourcePath) || !source.Relative(resource.WheelPath) || wheelPaths[resource.WheelPath] {
				return nil, fmt.Errorf("invalid published wheel resource mapping for %s", resource.Key)
			}
			wheelPaths[resource.WheelPath] = true
			member, err := readWheelContract(files[resource.WheelPath])
			if err != nil {
				return nil, fmt.Errorf("read published %s: %w", resource.Key, err)
			}
			expected, err := readSource(ctx, p.SourceSHA, resource.SourcePath)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(member, expected) {
				return nil, fmt.Errorf("published wheel member differs from exact source: %s", resource.SourcePath)
			}
			result = append(result, PublishedContract{
				Key: resource.Key, Domain: resource.Domain, ID: resource.ID, Version: resource.Version,
				SourcePath: resource.SourcePath, SHA256: digestHistory(member), Bytes: member,
			})
		}
		return result, nil
	}
}

func readWheelContract(file *zip.File) ([]byte, error) {
	if file == nil || !file.Mode().IsRegular() || file.UncompressedSize64 > source.MaxBytes {
		return nil, errors.New("verified regular contract file missing from published wheel")
	}
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	raw, err := io.ReadAll(io.LimitReader(stream, source.MaxBytes+1))
	if err != nil || len(raw) > source.MaxBytes {
		return nil, errors.New("published contract exceeds readable size limit or has corrupt bytes")
	}
	return raw, nil
}

// HistoryTarget binds both leaves explicitly. The evidence sidecar preserves
// original provenance without adding fields to a schema or engine contract.
type HistoryTarget struct {
	ArchivePath  string
	EvidencePath string
}

type HistoryRecord struct {
	Publication  Publication `json:"publication"`
	Key          string      `json:"key"`
	SourcePath   string      `json:"published_source_path"`
	OriginalID   string      `json:"published_id"`
	Version      string      `json:"published_version"`
	OriginalSHA  string      `json:"published_sha256"`
	ArchiveID    string      `json:"archive_id"`
	ArchivePath  string      `json:"archive_path"`
	ArchiveSHA   string      `json:"archive_sha256"`
	EvidencePath string      `json:"evidence_path"`
}

// HistoryReferenceVerifier verifies that published and archived references
// resolve to the same exact target and anchor after the resource is moved.
// The release owner must supply it whenever there are outbound references.
type HistoryReferenceVerifier func(context.Context, HistoryRecord, []byte, []byte) error

type archiveEntry struct {
	record   HistoryRecord
	archive  []byte
	evidence []byte
}

// HistoryPlan holds checked archive bytes. Its zero value cannot write.
type HistoryPlan struct {
	root    string
	entries []archiveEntry
}

func (p HistoryPlan) Records() []HistoryRecord {
	records := make([]HistoryRecord, 0, len(p.entries))
	for _, entry := range p.entries {
		records = append(records, entry.record)
	}
	return records
}

// PrepareHistory compares two observed publication inventories. New keys
// without a prior published version establish a baseline without an archive.
// Exact paths and provenance are validated before any files can be written.
func PrepareHistory(ctx context.Context, root string, previous, current Publication, verify PublicationVerifier, targets map[string]HistoryTarget, verifyRefs HistoryReferenceVerifier) (HistoryPlan, error) {
	if verify == nil || !validPublication(previous) || !validPublication(current) || previous.Platform != current.Platform {
		return HistoryPlan{}, errors.New("verified prior and current platform publications required")
	}
	prior, err := readPublishedContracts(ctx, previous, verify)
	if err != nil {
		return HistoryPlan{}, err
	}
	observed, err := readPublishedContracts(ctx, current, verify)
	if err != nil {
		return HistoryPlan{}, err
	}
	root, err = canonicalRoot(root)
	if err != nil {
		return HistoryPlan{}, err
	}
	plan := HistoryPlan{root: root}
	ids, paths := map[string]bool{}, map[string]bool{}
	keys := make([]string, 0, len(observed))
	for key := range observed {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return HistoryPlan{}, err
		}
		contract := observed[key]
		old, exists := prior[contract.Key]
		if !exists || old.SHA256 == contract.SHA256 {
			continue
		}
		before, _ := jsondoc.Decode(old.Bytes)
		after, _ := jsondoc.Decode(contract.Bytes)
		if reflect.DeepEqual(before, after) {
			continue
		}
		if old.Version == contract.Version {
			return HistoryPlan{}, fmt.Errorf("published contract %s changed without a version change", contract.Key)
		}
		target, ok := targets[contract.Key]
		if !ok || !validHistoryTarget(contract.Domain, target) {
			return HistoryPlan{}, fmt.Errorf("approved exact history and evidence paths unresolved for %s", contract.Key)
		}
		entry, err := prepareArchive(ctx, current, contract, target, source.Object(after), verifyRefs)
		if err != nil {
			return HistoryPlan{}, err
		}
		archivePath, evidencePath := strings.ToLower(target.ArchivePath), strings.ToLower(target.EvidencePath)
		if ids[entry.record.ArchiveID] || paths[archivePath] || paths[evidencePath] || archivePath == evidencePath {
			return HistoryPlan{}, fmt.Errorf("duplicate archive ID or output path for %s", contract.Key)
		}
		ids[entry.record.ArchiveID], paths[archivePath], paths[evidencePath] = true, true, true
		plan.entries = append(plan.entries, entry)
	}
	if _, err := plan.preflight(); err != nil {
		return HistoryPlan{}, err
	}
	return plan, nil
}

func validPublication(p Publication) bool {
	return (p.Platform == "github" || p.Platform == "pypi") && p.Reference != "" && p.Version != "" &&
		p.ArtifactName != "" && path.Base(p.ArtifactName) == p.ArtifactName && source.Relative(p.ArtifactName) &&
		publicationSHA.MatchString(p.SourceSHA) && contractSHA.MatchString(p.ArtifactSHA256)
}

func readPublishedContracts(ctx context.Context, p Publication, verify PublicationVerifier) (map[string]PublishedContract, error) {
	contracts, err := verify(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(contracts) == 0 {
		return nil, errors.New("published contract inventory is empty")
	}
	byKey, ids := map[string]PublishedContract{}, map[string]bool{}
	for _, c := range contracts {
		value, err := jsondoc.Decode(c.Bytes)
		document := source.Object(value)
		identity, parseErr := url.Parse(c.ID)
		if err != nil || parseErr != nil || !identity.IsAbs() || identity.Fragment != "" ||
			c.Key == "" || c.Domain == "" || byKey[c.Key].Key != "" || ids[c.ID] ||
			!source.Relative(c.SourcePath) || !strings.HasSuffix(c.SourcePath, ".json") ||
			digestHistory(c.Bytes) != c.SHA256 || !publishedIdentity(c, document) {
			return nil, fmt.Errorf("invalid published contract evidence for %s", c.Key)
		}
		c.Bytes = bytes.Clone(c.Bytes)
		byKey[c.Key], ids[c.ID] = c, true
	}
	return byKey, nil
}

func publishedIdentity(c PublishedContract, document map[string]any) bool {
	if strings.HasPrefix(c.Domain, "engines/") {
		return document["contract_id"] == c.ID && document["version"] == c.Version
	}
	version := document["version"]
	if c.Domain == "profile" {
		identity := source.Object(source.Object(document["properties"])["ptsip"])
		version = source.Object(source.Object(identity["properties"])["version"])["const"]
	}
	return document["$id"] == c.ID && version == c.Version && c.Version != ""
}

func validHistoryTarget(domain string, t HistoryTarget) bool {
	switch domain {
	case "profile", "policy", "plan", "engines/validation", "engines/execution", "engines/projection":
	default:
		return false
	}
	for _, name := range []string{t.ArchivePath, t.EvidencePath} {
		if !source.Relative(name) || path.Dir(name) != historyRoot+"/"+domain || !strings.HasSuffix(name, ".json") {
			return false
		}
	}
	return true
}

func prepareArchive(ctx context.Context, p Publication, c PublishedContract, t HistoryTarget, doc map[string]any, verify HistoryReferenceVerifier) (archiveEntry, error) {
	suffix := contractSuffix.FindStringSubmatch(c.Version)
	if suffix == nil {
		return archiveEntry{}, fmt.Errorf("history ID version form unresolved for %s", c.Key)
	}
	id := c.ID + ":" + strings.ReplaceAll(suffix[2], ".", "_")
	record := HistoryRecord{
		Publication: p, Key: c.Key, SourcePath: c.SourcePath, OriginalID: c.ID, Version: c.Version,
		OriginalSHA: c.SHA256, ArchiveID: id, ArchivePath: t.ArchivePath, EvidencePath: t.EvidencePath,
	}
	field := "$id"
	if strings.HasPrefix(c.Domain, "engines/") {
		field = "contract_id"
	}
	doc[field] = id
	archive, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return archiveEntry{}, err
	}
	archive = append(archive, '\n')
	record.ArchiveSHA = digestHistory(archive)
	if verify != nil {
		if err := verify(ctx, record, bytes.Clone(c.Bytes), bytes.Clone(archive)); err != nil {
			return archiveEntry{}, fmt.Errorf("archive reference resolution differs for %s: %w", c.Key, err)
		}
	} else if hasOutboundReference(doc) {
		return archiveEntry{}, fmt.Errorf("archive reference resolution is unverified for %s", c.Key)
	}
	evidence, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return archiveEntry{}, err
	}
	return archiveEntry{record: record, archive: archive, evidence: append(evidence, '\n')}, nil
}

func hasOutboundReference(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			switch key {
			case "$ref", "$dynamicRef", "$recursiveRef":
				ref, _ := child.(string)
				if !strings.HasPrefix(ref, "#") {
					return true
				}
			}
			if hasOutboundReference(child) {
				return true
			}
		}
	case []any:
		return slices.ContainsFunc(v, hasOutboundReference)
	}
	return false
}

type historyFile struct {
	name     string
	contents []byte
}

func (p HistoryPlan) preflight() ([]historyFile, error) {
	if p.root == "" {
		return nil, errors.New("history plan is unverified")
	}
	if len(p.entries) == 0 {
		return nil, nil
	}
	root := filepath.Join(p.root, filepath.FromSlash(historyRoot))
	seen := map[string]string{}
	if err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("history contains a path alias: %s", name)
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			return nil
		}
		relative, err := filepath.Rel(p.root, name)
		if err != nil {
			return err
		}
		raw, err := asset(p.root, filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		value, err := jsondoc.Decode(raw)
		if err != nil {
			return err
		}
		document := source.Object(value)
		for _, field := range []string{"$id", "contract_id"} {
			if id := source.String(document[field]); id != "" {
				if seen[id] != "" {
					return fmt.Errorf("duplicate existing history ID %s", id)
				}
				seen[id] = name
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	files := make([]historyFile, 0, len(p.entries)*2)
	for _, entry := range p.entries {
		archive := filepath.Join(p.root, filepath.FromSlash(entry.record.ArchivePath))
		if found := seen[entry.record.ArchiveID]; found != "" && found != archive {
			return nil, fmt.Errorf("archive ID already exists at %s", found)
		}
		// Provenance is installed before its archive. Interrupted creation can
		// resume without an archive ever missing its corresponding evidence.
		for _, file := range []historyFile{
			{filepath.Join(p.root, filepath.FromSlash(entry.record.EvidencePath)), entry.evidence},
			{archive, entry.archive},
		} {
			parent := filepath.Dir(file.name)
			resolved, err := filepath.EvalSymlinks(parent)
			if err != nil || resolved != parent {
				return nil, fmt.Errorf("approved history directory unavailable: %s", parent)
			}
			info, err := os.Lstat(file.name)
			if errors.Is(err, os.ErrNotExist) {
				files = append(files, file)
				continue
			}
			if err != nil || !info.Mode().IsRegular() {
				return nil, fmt.Errorf("unsafe history leaf: %s", file.name)
			}
			data, err := os.ReadFile(file.name)
			if err != nil || !bytes.Equal(data, file.contents) {
				return nil, fmt.Errorf("immutable history conflicts: %s", file.name)
			}
		}
	}
	return files, nil
}

// Materialize exclusively installs the prepared archive and its evidence.
// Existing matching bytes are idempotent. Conflicting history is immutable.
func (p HistoryPlan) Materialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.root == "" {
		return errors.New("history plan is unverified")
	}
	if len(p.entries) == 0 {
		return nil
	}
	historyDirectory := filepath.Join(p.root, filepath.FromSlash(historyRoot))
	if resolved, err := filepath.EvalSymlinks(historyDirectory); err != nil || resolved != historyDirectory {
		return fmt.Errorf("approved history directory unavailable: %s", historyDirectory)
	}
	lockName := filepath.Join(historyDirectory, ".ptsip-history.lock")
	lock, err := os.OpenFile(lockName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("history installation lock unavailable: %w", err)
	}
	defer os.Remove(lockName)
	if err := lock.Close(); err != nil {
		return err
	}
	files, err := p.preflight()
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := installHistoryFile(file); err != nil {
			return err
		}
	}
	return nil
}

func installHistoryFile(file historyFile) error {
	temp, err := os.CreateTemp(filepath.Dir(file.name), ".ptsip-history-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	_, writeErr := temp.Write(file.contents)
	if err := errors.Join(writeErr, temp.Chmod(0644), temp.Sync(), temp.Close()); err != nil {
		return err
	}
	if err := os.Link(temp.Name(), file.name); err != nil {
		return fmt.Errorf("cannot install immutable history %s: %w", file.name, err)
	}
	return nil
}

func digestHistory(raw []byte) string { return fmt.Sprintf("%x", sha256.Sum256(raw)) }
