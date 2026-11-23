package inspection

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/evidence"
)

type DependencyEdge struct {
	EvidenceID       string  `json:"evidence_id"`
	Source           string  `json:"source"`
	Target           string  `json:"target"`
	EdgeType         string  `json:"edge_type"`
	Phase            string  `json:"phase"`
	Resolution       string  `json:"resolution"`
	TargetScope      string  `json:"target_scope"`
	Provenance       string  `json:"provenance"`
	Line             *int    `json:"line"`
	ResolvedPath     *string `json:"resolved_path"`
	Adapter          string  `json:"adapter"`
	WorkingDirectory *string `json:"working_directory"`
	Note             *string `json:"note"`
}

type DependencyIssue struct {
	Adapter string `json:"adapter"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type DependencyScan struct {
	Edges            []DependencyEdge  `json:"edges"`
	Issues           []DependencyIssue `json:"issues"`
	Adapters         []string          `json:"adapters"`
	EdgeCount        int               `json:"edge_count"`
	CoverageComplete bool              `json:"coverage_complete"`
	Cache            any               `json:"cache"`
}

type dependencyContext struct {
	root  string
	paths []string
	files map[string]bool
}

// ObserveDependencies inspects only files reported by registered Discovery.
// Unavailable adapters and unreadable inputs remain visible in coverage.
func ObserveDependencies(root string, discovered json.RawMessage) (DependencyScan, error) {
	var observed struct {
		Paths  []string `json:"files"`
		Errors []string `json:"observation_errors"`
		Root   string   `json:"root"`
	}
	if err := json.Unmarshal(discovered, &observed); err != nil {
		return DependencyScan{}, fmt.Errorf("decode Discovery dependency inventory: %w", err)
	}
	if !filepath.IsAbs(root) || observed.Root != root || observed.Paths == nil || observed.Errors == nil {
		return DependencyScan{}, fmt.Errorf("dependency observation requires the files of the selected Discovery root")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return DependencyScan{}, err
	}
	c := dependencyContext{root: resolved, paths: observed.Paths, files: make(map[string]bool, len(observed.Paths))}
	for _, name := range c.paths {
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\:\x00\r\n") || c.files[name] {
			return DependencyScan{}, fmt.Errorf("invalid or repeated Discovery dependency path %q", name)
		}
		c.files[name] = true
	}
	slices.Sort(c.paths)
	result := DependencyScan{Edges: []DependencyEdge{}, Issues: []DependencyIssue{}, Adapters: []string{}}
	for _, message := range observed.Errors {
		result.Issues = append(result.Issues, DependencyIssue{"repository", "<repository>", message})
	}
	for _, name := range c.paths {
		switch suffix := strings.ToLower(path.Ext(name)); suffix {
		case ".c", ".cc", ".cpp", ".cxx", ".h", ".hh", ".hpp", ".hxx", ".java",
			".kt", ".kts", ".php", ".rb", ".rs", ".scala", ".swift":
			result.Issues = append(result.Issues, DependencyIssue{"source-coverage", name, "dependency adapter unavailable for " + suffix})
		}
	}
	seenEdges := map[string]bool{}
	for _, scanner := range []func(dependencyContext) ([]DependencyEdge, []DependencyIssue, []string){
		scanGo, scanDotNet, scanJavaScript, scanPython, scanWorkflows,
	} {
		edges, issues, adapters := scanner(c)
		result.Issues = append(result.Issues, issues...)
		result.Adapters = append(result.Adapters, adapters...)
		for _, edge := range edges {
			if edge.EvidenceID == "" {
				digest, err := evidence.StableDigest(edge)
				if err != nil {
					return DependencyScan{}, err
				}
				edge.EvidenceID = edge.Adapter + ":" + digest
			}
			if !seenEdges[edge.EvidenceID] {
				result.Edges = append(result.Edges, edge)
				seenEdges[edge.EvidenceID] = true
			}
		}
	}
	slices.SortFunc(result.Edges, func(a, b DependencyEdge) int {
		return cmp.Or(cmp.Compare(a.Source, b.Source), cmp.Compare(edgeLine(a.Line), edgeLine(b.Line)),
			cmp.Compare(a.Target, b.Target), cmp.Compare(a.EvidenceID, b.EvidenceID))
	})
	slices.SortFunc(result.Issues, func(a, b DependencyIssue) int {
		return cmp.Or(cmp.Compare(a.Adapter, b.Adapter), cmp.Compare(a.Path, b.Path), cmp.Compare(a.Message, b.Message))
	})
	slices.Sort(result.Adapters)
	result.Issues, result.Adapters = slices.Compact(result.Issues), slices.Compact(result.Adapters)
	result.EdgeCount, result.CoverageComplete = len(result.Edges), len(result.Issues) == 0
	return result, nil
}

func edgeLine(line *int) int {
	if line == nil {
		return 0
	}
	return *line
}

func (c dependencyContext) filePath(name string) (string, error) {
	if !c.files[name] {
		return "", fmt.Errorf("file was not observed by Discovery")
	}
	filename := filepath.Join(c.root, filepath.FromSlash(name))
	resolved, err := filepath.EvalSymlinks(filename)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(c.root, resolved)
	if err != nil || filepath.ToSlash(relative) != name {
		return "", fmt.Errorf("observed dependency source is an alias")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("observed dependency source is not a regular file")
	}
	return resolved, nil
}

func (c dependencyContext) exists(name string) bool {
	_, err := c.filePath(name)
	return err == nil
}

func (c dependencyContext) read(name string) ([]byte, error) {
	resolved, err := c.filePath(name)
	if err != nil {
		return nil, err
	}
	input, err := os.Open(resolved)
	if err != nil {
		return nil, err
	}
	defer input.Close()
	const maxDependencyFile = 4 << 20
	data, err := io.ReadAll(io.LimitReader(input, maxDependencyFile+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDependencyFile {
		return nil, fmt.Errorf("observed dependency source exceeds the input limit")
	}
	return bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), nil
}

func (scan DependencyScan) CandidateAnchors() []DependencyAnchor {
	var anchors []DependencyAnchor
	for _, edge := range scan.Edges {
		if edge.Adapter == "github-actions" && edge.Resolution == "RESOLVED" && edge.ResolvedPath != nil {
			anchors = append(anchors, DependencyAnchor{edge.Adapter, *edge.ResolvedPath, edge.EvidenceID})
		}
	}
	return anchors
}
