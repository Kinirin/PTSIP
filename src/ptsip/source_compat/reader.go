package sourcecompat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/resolver"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/validator"
	"github.com/Kinirin/PTSIP/src/ptsip/internal/enginerouter"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	SpecificationSource = "https://github.com/Kinirin/PTSIP"
	V034Revision        = "b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e"
	V036Revision        = "d6995ed232e845b88d8235b851e80ab54b7804ea"
)

type SourceFamily struct {
	Version  string
	Revision string
	Family   Family
}

type frozenSource struct {
	SourceFamily
	file   string
	digest string
}

// The plan records these original, unregistered sources as frozen migration
// inputs. The hashes are of those Git blob bytes, with no checkout EOL filter.
var historicalSources = []frozenSource{
	{SourceFamily{"0.3.4-draft", V034Revision, Tool035Profile}, "specdata/ptsip-source-profile-0.3.4.schema.json", "fa1c47055dabc99d6e39e23b3296b1e85e62e9cf1bb31b1d2f94b322c484ba42"},
	{SourceFamily{"0.3.6-draft", V036Revision, Tool036Profile}, "specdata/ptsip-source-profile-0.3.6.schema.json", "056c1009f5c8ababf4100271e9ff3902c191203b02b71baed2338aeb47323336"},
}

func SupportedSourceFamilies() []SourceFamily {
	families := make([]SourceFamily, 0, len(historicalSources))
	for _, historical := range historicalSources {
		families = append(families, historical.SourceFamily)
	}
	return families
}

type Reader struct {
	caller enginerouter.Caller
	assets fs.FS
}

func NewReader(caller enginerouter.Caller, assets fs.FS) Reader {
	return Reader{caller: caller, assets: assets}
}

func OpenInstalled(cli, packageRoot, scriptsDir string) Reader {
	return NewReader(enginerouter.Client{CLI: cli, PackageRoot: packageRoot, ScriptsDir: scriptsDir}, os.DirFS(packageRoot))
}

func issue(binding ObservedBinding, code, message string, pointer *string) Issue {
	return Issue{Code: code, Message: message, ProfilePath: binding.Path, Pointer: pointer}
}

func at(pointer string) *string { return &pointer }

func pointerToken(part string) string {
	return strings.ReplaceAll(strings.ReplaceAll(part, "~", "~0"), "/", "~1")
}

func (r Reader) ReadSourceProfile(ctx context.Context, root string, generation ObservedBinding) ReadResult {
	doc, failed := ReadObservedProfile(ctx, r.caller, root, generation)
	if failed != nil {
		return ReadResult{Issues: []Issue{*failed}}
	}
	identity := source.Object(doc["ptsip"])
	specification := source.Object(identity["specification"])
	var issues []Issue
	if version, ok := identity["version"].(string); !ok || version != generation.DeclaredVersion {
		issues = append(issues, issue(generation, "SOURCE_VERSION_MISMATCH", "ptsip.version differs from the observed binding", at("/ptsip/version")))
	}
	if revision, ok := specification["revision"].(string); !ok || revision != generation.SpecificationRevision {
		issues = append(issues, issue(generation, "SOURCE_REVISION_MISMATCH", "specification revision differs from the observed binding", at("/ptsip/specification/revision")))
	}
	value := stringPointer(specification["source"])
	if value == nil && generation.SpecificationSource != nil || value != nil && (generation.SpecificationSource == nil || *value != *generation.SpecificationSource) {
		issues = append(issues, issue(generation, "SOURCE_SPECIFICATION_SOURCE_MISMATCH", "specification source differs from the observed binding", at("/ptsip/specification/source")))
	}
	if len(issues) != 0 {
		return ReadResult{Issues: issues}
	}
	index := slices.IndexFunc(historicalSources, func(item frozenSource) bool {
		return item.Version == generation.DeclaredVersion
	})
	if index < 0 {
		return ReadResult{Issues: []Issue{issue(generation, "UNSUPPORTED_SOURCE_FAMILY", "unsupported historical Profile family", at("/ptsip"))}}
	}
	historical := historicalSources[index]
	if historical.Revision != generation.SpecificationRevision {
		return ReadResult{Issues: []Issue{issue(generation, "UNSUPPORTED_SOURCE_REVISION", "historical Profile revision is not frozen", at("/ptsip"))}}
	}
	if value == nil || *value != SpecificationSource {
		return ReadResult{Issues: []Issue{issue(generation, "UNSUPPORTED_SPECIFICATION_SOURCE", "unsupported historical specification source", at("/ptsip/specification/source"))}}
	}
	if issues = r.validateSchema(historical, generation, doc); len(issues) != 0 {
		return ReadResult{Issues: issues}
	}
	profile := project(historical.Family, generation, doc)
	if issues = sourceIntegrityIssues(profile, generation); len(issues) != 0 {
		return ReadResult{Issues: issues}
	}
	return ReadResult{Complete: true, Profile: &profile, Issues: []Issue{}}
}

// ReadObservedProfile validates the owner-relative path and content binding of
// a native Discovery result before decoding its Profile document.
func ReadObservedProfile(ctx context.Context, caller enginerouter.Caller, root string, generation ObservedBinding) (source.Document, *Issue) {
	raw, failed := readObservedBytes(ctx, caller, root, generation)
	if failed != nil {
		return nil, failed
	}
	document, err := source.Decode(raw)
	if err != nil {
		failure := issue(generation, "SOURCE_PARSE_ERROR", err.Error(), nil)
		if err.Error() == "POLICY_DOCUMENT_MUST_BE_OBJECT" {
			failure = issue(generation, "SOURCE_SHAPE_ERROR", "source Profile root must be a mapping", at(""))
		}
		return nil, &failure
	}
	return document, nil
}

func readObservedBytes(ctx context.Context, caller enginerouter.Caller, root string, generation ObservedBinding) ([]byte, *Issue) {
	if caller == nil || root == "" || generation.Path == "" {
		failure := issue(generation, "SOURCE_PATH_ERROR", "native Profile path validator, owner root and observed path are required", nil)
		return nil, &failure
	}
	var resolved struct {
		Relative string `json:"normalized_path"`
		File     string `json:"path_on_disk"`
	}
	absolute, err := filepath.Abs(root)
	if err == nil {
		err = caller.Invoke(ctx, "validation.profile.path", map[string]string{"root": absolute, "path": generation.Path}, &resolved)
	}
	if err != nil {
		failure := issue(generation, "SOURCE_PATH_ERROR", err.Error(), nil)
		return nil, &failure
	}
	if !fs.ValidPath(resolved.Relative) || resolved.Relative == "." || strings.ContainsAny(resolved.Relative, "\\:") ||
		filepath.Clean(resolved.File) != filepath.Join(absolute, filepath.FromSlash(resolved.Relative)) {
		failure := issue(generation, "SOURCE_PATH_ERROR", "native Profile path does not resolve under the owner root", nil)
		return nil, &failure
	}
	raw, err := readFile(os.DirFS(absolute), resolved.Relative)
	if err != nil {
		failure := issue(generation, "SOURCE_READ_ERROR", err.Error(), nil)
		return nil, &failure
	}
	if digest(raw) != generation.ContentSHA256 {
		failure := issue(generation, "SOURCE_CONTENT_STALE", "source Profile bytes differ from the observed binding", nil)
		return nil, &failure
	}
	return raw, nil
}

func (r Reader) ValidateSourceReadBinding(ctx context.Context, root string, result ReadResult) []Issue {
	if result.Profile == nil {
		if len(result.Issues) != 0 {
			return result.Issues
		}
		return []Issue{{Code: "NO_SOURCE_PROFILE", Message: "no compatibility source Profile is available", ProfilePath: "<unknown>"}}
	}
	bound := result.Profile.Generation
	generation := ObservedBinding{Path: bound.ProfilePath, ContentSHA256: bound.ContentSHA256}
	_, failure := readObservedBytes(ctx, r.caller, root, generation)
	if failure == nil {
		return []Issue{}
	}
	if failure.Code != "SOURCE_CONTENT_STALE" {
		failure.Code = "SOURCE_BINDING_INVALID"
	}
	return []Issue{*failure}
}

func (r Reader) validateSchema(historical frozenSource, generation ObservedBinding, doc source.Document) []Issue {
	body, err := readFile(r.assets, historical.file)
	if err != nil {
		return []Issue{issue(generation, "SOURCE_SCHEMA_ERROR", err.Error(), nil)}
	}
	if digest(bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))) != historical.digest {
		return []Issue{issue(generation, "SOURCE_SCHEMA_ERROR", "frozen source schema digest mismatch: "+historical.file, nil)}
	}
	value, err := jsondoc.Decode(body)
	if err != nil {
		return []Issue{issue(generation, "SOURCE_SCHEMA_ERROR", err.Error(), nil)}
	}
	location := "file:///" + historical.file
	schema, err := validator.Compile(resolver.Graph{Root: location, Docs: map[string]map[string]any{location: source.Object(value)}})
	if err == nil {
		err = schema.Validate(doc)
	}
	if err == nil {
		return nil
	}
	var invalid *jsonschema.ValidationError
	if !errors.As(err, &invalid) {
		return []Issue{issue(generation, "SOURCE_SCHEMA_ERROR", err.Error(), at(""))}
	}
	var issues []Issue
	var appendLeaf func(jsonschema.OutputUnit)
	appendLeaf = func(output jsonschema.OutputUnit) {
		if output.Error != nil {
			issues = append(issues, issue(generation, "SOURCE_SCHEMA_ERROR", output.Error.String(), at(output.InstanceLocation)))
		}
		for _, child := range output.Errors {
			appendLeaf(child)
		}
	}
	appendLeaf(*invalid.DetailedOutput())
	slices.SortStableFunc(issues, func(a, b Issue) int { return strings.Compare(*a.Pointer, *b.Pointer) })
	return issues
}

func readFile(files fs.FS, path string) ([]byte, error) {
	if files == nil {
		return nil, fmt.Errorf("source resource root is unavailable")
	}
	info, err := fs.Stat(files, path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("cannot read regular source file %s: %v", path, err)
	}
	file, err := files.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, source.MaxBytes+1))
	if len(data) > source.MaxBytes {
		return nil, fmt.Errorf("source size exceeds the decoder limit: %s", path)
	}
	return data, err
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func stringPointer(value any) *string {
	if value, valid := value.(string); valid {
		return &value
	}
	return nil
}
