package releasecontroller

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

// PackageWheel composes the Tool and Profile payload with mandatory native-wheel
// dependencies. No Python namespace discovery is treated as a Rust build.
func PackageWheel(toolWheel, profileArchive []byte, s Snapshot, registered map[string]string, pypiVersion string) ([]byte, error) {
	return packageWheel(toolWheel, profileArchive, s, registered, pypiVersion, nil)
}

// PackageNativePayload contains only the admitted original engine wheels for one
// host. Building a bundle does not admit installation, publication or activation.
// Existing callers retain their independent-package contract until the Package
// receipt and consumer installation owners explicitly adopt this representation.
type PackageNativePayload struct {
	Platform string
	Wheels   map[string][]byte
}

// BundledPackageWheel assembles one complete platform distribution without
// Requires-Dist edges to separate engine projects. Engine versions and original
// source/binary identities remain independent inside ptsip-engines.json.
func BundledPackageWheel(toolWheel, profileArchive []byte, s Snapshot, registered map[string]string, pypiVersion string, native PackageNativePayload) ([]byte, error) {
	return packageWheel(toolWheel, profileArchive, s, registered, pypiVersion, &native)
}

func packageWheel(toolWheel, profileArchive []byte, s Snapshot, registered map[string]string, pypiVersion string, native *PackageNativePayload) ([]byte, error) {
	if err := CompletePackage(s, registered); err != nil {
		return nil, err
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:(?:a|b|rc)[1-9][0-9]*)?$`).MatchString(pypiVersion) {
		return nil, fmt.Errorf("explicit valid Package PyPI version required")
	}
	archive, err := zip.NewReader(bytes.NewReader(toolWheel), int64(len(toolWheel)))
	if err != nil {
		return nil, err
	}
	members := map[string][]byte{}
	executables := map[string]bool{}
	var metadata, entrypoints []byte
	di := "ptsip-" + pypiVersion + ".dist-info/"
	for _, f := range archive.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 || f.UncompressedSize64 > 128*1024*1024 || path.Clean(f.Name) != f.Name || strings.HasPrefix(f.Name, "../") || strings.HasPrefix(f.Name, "/") || strings.ContainsAny(f.Name, "\\:") {
			return nil, fmt.Errorf("unsafe Tool wheel member")
		}
		stream, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(stream, 128*1024*1024+1))
		stream.Close()
		if err != nil {
			return nil, err
		}
		if strings.Contains(f.Name, ".dist-info/") {
			if strings.HasSuffix(f.Name, "/METADATA") {
				if metadata != nil {
					return nil, fmt.Errorf("ambiguous Tool metadata")
				}
				metadata = b
			}
			if strings.HasSuffix(f.Name, "/entry_points.txt") {
				entrypoints = b
			}
			if p := strings.Index(f.Name, ".dist-info/licenses/"); p >= 0 {
				members[di+"licenses/"+f.Name[p+len(".dist-info/licenses/"):]] = b
			}
			continue
		}
		if strings.Contains(f.Name, ".data/") || strings.Contains(f.Name, "/profiles/history/") {
			return nil, fmt.Errorf("non-Tool payload in Tool wheel")
		}
		members[f.Name] = b
	}
	message, err := mail.ReadMessage(bytes.NewReader(metadata))
	if err != nil {
		return nil, fmt.Errorf("Tool metadata missing")
	}
	toolVersion := ""
	engineDeps := []string{}
	for _, c := range s.Components {
		if c.Kind == "tool" {
			toolVersion = c.Version
		}
		if c.Kind == "engine" {
			engineDeps = append(engineDeps, "ptsip-engine-"+c.ID+"=="+c.Version)
		}
	}
	toolPEP, err := ToolDistributionVersion(toolVersion)
	if err != nil {
		return nil, err
	}
	if (message.Header.Get("Name") != "PTSIP" && message.Header.Get("Name") != "ptsip-tool") || message.Header.Get("Version") != toolPEP {
		return nil, fmt.Errorf("Tool payload identity differs from complete composition")
	}
	resources, err := ProfileResources(profileArchive)
	if err != nil {
		return nil, err
	}
	for name, data := range resources {
		if name == "@project-profile-contracts.yaml" {
			members["ptsip/specdata/project-profile-contracts.yaml"] = data
		} else {
			members["ptsip/profiles/"+name] = data
		}
	}
	// The legacy projection keeps its declared independent dependencies. The
	// explicit bundle instead carries all native artifacts for the selected host.
	text := strings.ReplaceAll(string(metadata), "\r\n", "\n")
	parts := strings.SplitN(text, "\n\n", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid Tool metadata body")
	}
	lines := strings.Split(parts[0], "\n")
	headers := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			line = "Name: PTSIP"
		}
		if strings.HasPrefix(line, "Version:") {
			line = "Version: " + pypiVersion
		}
		if strings.HasPrefix(line, "Requires-Dist: ptsip-engine-") {
			continue
		}
		headers = append(headers, line)
	}
	sort.Strings(engineDeps)
	if native == nil {
		for _, dependency := range engineDeps {
			headers = append(headers, "Requires-Dist: "+dependency)
		}
	}
	composition, _ := json.MarshalIndent(s, "", "  ")
	members["ptsip/package-composition.json"] = append(composition, '\n')
	members[di+"METADATA"] = []byte(strings.Join(headers, "\n") + "\n\n" + parts[1])
	members[di+"WHEEL"] = []byte("Wheel-Version: 1.0\nGenerator: ptsip-complete-package-go/1\nRoot-Is-Purelib: true\nTag: py3-none-any\n")
	if native != nil {
		metadata, err := mail.ReadMessage(bytes.NewReader(members[di+"METADATA"]))
		if err != nil {
			return nil, fmt.Errorf("invalid bundled Package metadata: %w", err)
		}
		engineRequirement := regexp.MustCompile(`(?i)^ptsip[-_.]+engine[-_.]+`)
		for _, requirement := range metadata.Header["Requires-Dist"] {
			if engineRequirement.MatchString(strings.TrimSpace(requirement)) {
				return nil, fmt.Errorf("bundled Package retains an independent engine requirement")
			}
		}
		tag, _, err := packageNativeHost(native.Platform)
		if err != nil {
			return nil, err
		}
		if err := bundlePackageEngines(members, executables, s, registered, pypiVersion, *native); err != nil {
			return nil, err
		}
		members[di+"WHEEL"] = []byte("Wheel-Version: 1.0\nGenerator: ptsip-complete-package-go/1\nRoot-Is-Purelib: false\nTag: py3-none-" + tag + "\n")
	}
	if entrypoints != nil {
		members[di+"entry_points.txt"] = entrypoints
	}
	names := []string{}
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	var record bytes.Buffer
	writer := csv.NewWriter(&record)
	for _, name := range names {
		sum := sha256.Sum256(members[name])
		if err := writer.Write([]string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]), fmt.Sprint(len(members[name]))}); err != nil {
			return nil, err
		}
	}
	_ = writer.Write([]string{di + "RECORD", "", ""})
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	members[di+"RECORD"] = record.Bytes()
	names = append(names, di+"RECORD")
	var output bytes.Buffer
	zipWriter := zip.NewWriter(&output)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(0644)
		if executables[name] {
			header.SetMode(0755)
		}
		w, err := zipWriter.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(members[name]); err != nil {
			return nil, err
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return PyPIArchive(output.Bytes())
}

func packageNativeHost(platform string) (tag, target string, err error) {
	switch platform {
	case "windows-x64":
		return "win_amd64", "x86_64-pc-windows-msvc", nil
	case "linux-x64":
		return "manylinux_2_39_x86_64", "x86_64-unknown-linux-gnu", nil
	case "macos-arm64":
		return "macosx_11_0_arm64", "aarch64-apple-darwin", nil
	default:
		return "", "", fmt.Errorf("unregistered Package native host: %s", platform)
	}
}

// BundledPackageWheelName is separate from the opaque composition identity.
func BundledPackageWheelName(version, platform string) (string, error) {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:(?:a|b|rc)[1-9][0-9]*)?$`).MatchString(version) {
		return "", fmt.Errorf("explicit valid Package PyPI version required")
	}
	tag, _, err := packageNativeHost(platform)
	if err != nil {
		return "", err
	}
	return "ptsip-" + version + "-py3-none-" + tag + ".whl", nil
}

func bundlePackageEngines(members map[string][]byte, executables map[string]bool, s Snapshot, registered map[string]string, version string, native PackageNativePayload) error {
	tag, target, err := packageNativeHost(native.Platform)
	if err != nil {
		return err
	}
	if len(native.Wheels) != len(registered) {
		return fmt.Errorf("Package native payload must contain every registered engine exactly once")
	}
	var index struct {
		Entries []struct {
			Kind, Version string
			ID            string `json:"engine_id"`
		} `json:"official_engine_entries"`
	}
	if err := json.Unmarshal(members["ptsip/engines/index.json"], &index); err != nil {
		return fmt.Errorf("bundled Package requires the Tool's engine identity registry: %w", err)
	}
	identities := map[string]string{}
	for _, entry := range index.Entries {
		id := strings.ToLower(entry.Kind)
		if registered[id] != entry.Version || entry.ID == "" || identities[id] != "" {
			return fmt.Errorf("bundled Package engine registry differs from composition")
		}
		identities[id] = entry.ID
	}
	if len(identities) != len(registered) {
		return fmt.Errorf("bundled Package engine identity registry incomplete")
	}
	origins := map[string]json.RawMessage{}
	for _, c := range s.Components {
		if c.Kind != "engine" {
			continue
		}
		if registered[c.ID] != c.Version {
			return fmt.Errorf("bundled engine version differs from registered composition: %s", c.ID)
		}
		data, ok := native.Wheels[c.ID]
		if !ok {
			return fmt.Errorf("bundled Package missing engine: %s", c.ID)
		}
		distribution := "ptsip-engine-" + c.ID
		wheelBase := strings.ReplaceAll(distribution, "-", "_") + "-" + c.Version
		name := wheelBase + "-py3-none-" + tag + ".whl"
		matches := 0
		for _, file := range c.Files {
			if file.Platform == native.Platform && file.Name == name && file.SHA256 == Digest(data) {
				matches++
			}
		}
		if matches != 1 {
			return fmt.Errorf("bundled engine original host wheel identity differs: %s", c.ID)
		}
		archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return err
		}
		contents := map[string][]byte{}
		for _, file := range archive.File {
			if file.FileInfo().IsDir() {
				continue
			}
			if _, exists := contents[file.Name]; exists || file.Mode()&os.ModeSymlink != 0 || file.UncompressedSize64 > 128*1024*1024 || path.Clean(file.Name) != file.Name || strings.HasPrefix(file.Name, "../") || strings.HasPrefix(file.Name, "/") || strings.ContainsAny(file.Name, "\\:") {
				return fmt.Errorf("unsafe bundled engine wheel member")
			}
			r, err := file.Open()
			if err != nil {
				return err
			}
			b, readErr := io.ReadAll(io.LimitReader(r, 128*1024*1024+1))
			r.Close()
			if readErr != nil || len(b) > 128*1024*1024 {
				return fmt.Errorf("bundled engine member read failed")
			}
			contents[file.Name] = b
		}
		di := wheelBase + ".dist-info/"
		metadata, err := mail.ReadMessage(bytes.NewReader(contents[di+"METADATA"]))
		if err != nil || metadata.Header.Get("Name") != distribution || metadata.Header.Get("Version") != c.Version {
			return fmt.Errorf("bundled engine metadata differs: %s", c.ID)
		}
		wheel, err := mail.ReadMessage(bytes.NewReader(append(contents[di+"WHEEL"], '\n')))
		if err != nil || wheel.Header.Get("Root-Is-Purelib") != "false" || len(wheel.Header["Tag"]) != 1 || wheel.Header.Get("Tag") != "py3-none-"+tag {
			return fmt.Errorf("bundled engine wheel platform differs: %s", c.ID)
		}
		entrypoint := distribution
		if native.Platform == "windows-x64" {
			entrypoint += ".exe"
		}
		binary := contents[wheelBase+".data/scripts/"+entrypoint]
		magicOK := native.Platform == "windows-x64" && bytes.HasPrefix(binary, []byte("MZ")) || native.Platform == "linux-x64" && bytes.HasPrefix(binary, []byte("\x7fELF")) || native.Platform == "macos-arm64" && (bytes.HasPrefix(binary, []byte{0xcf, 0xfa, 0xed, 0xfe}) || bytes.HasPrefix(binary, []byte{0xfe, 0xed, 0xfa, 0xcf}))
		var origin struct {
			Schema     string `json:"schema_version"`
			Source     string `json:"source_sha"`
			EngineID   string `json:"engine_id"`
			Version    string `json:"engine_version"`
			Host       string `json:"host_target"`
			Entrypoint string `json:"entrypoint"`
			SHA256     string `json:"binary_sha256"`
			Authorized *bool  `json:"execution_authorized"`
		}
		originData := contents[di+"ptsip-origin.json"]
		if json.Unmarshal(originData, &origin) != nil || origin.Schema != "ptsip-independent-wheel-origin/v1" || origin.Source != c.SourceSHA || origin.EngineID != identities[c.ID] || origin.Version != c.Version || origin.Host != target || origin.Entrypoint != entrypoint || origin.SHA256 != Digest(binary) || !magicOK || origin.Authorized == nil || *origin.Authorized {
			return fmt.Errorf("bundled engine native origin differs: %s", c.ID)
		}
		name = "ptsip-" + version + ".data/scripts/" + entrypoint
		if _, exists := members[name]; exists {
			return fmt.Errorf("duplicate bundled engine executable")
		}
		members[name], executables[name] = binary, true
		origins[c.ID] = append(json.RawMessage(nil), originData...)
	}
	if len(origins) != len(registered) {
		return fmt.Errorf("bundled Package engine inventory incomplete")
	}
	bundle, err := json.MarshalIndent(map[string]any{"schema_version": "ptsip-bundled-engine-origins/v1", "package_version": version, "platform": native.Platform, "engines": origins, "execution_authorized": false}, "", "  ")
	if err != nil {
		return err
	}
	members["ptsip-"+version+".dist-info/ptsip-engines.json"] = append(bundle, '\n')
	return nil
}
