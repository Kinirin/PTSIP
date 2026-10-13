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
	if err := CompletePackage(s, registered); err != nil {
		return nil, err
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:(?:a|b|rc)[1-9][0-9]*|\.post[1-9][0-9]*)?$`).MatchString(pypiVersion) {
		return nil, fmt.Errorf("explicit valid Package PyPI version required")
	}
	archive, err := zip.NewReader(bytes.NewReader(toolWheel), int64(len(toolWheel)))
	if err != nil {
		return nil, err
	}
	members := map[string][]byte{}
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
	if (message.Header.Get("Name") != "PTSIP" && message.Header.Get("Name") != "ptsip-tool") || message.Header.Get("Version") != toolVersion {
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
	// Preserve the existing Tool requirements while adding every engine as a
	// required dependency. There is no optional extra or platform-omission marker.
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
	for _, dependency := range engineDeps {
		headers = append(headers, "Requires-Dist: "+dependency)
	}
	composition, _ := json.MarshalIndent(s, "", "  ")
	members["ptsip/package-composition.json"] = append(composition, '\n')
	members[di+"METADATA"] = []byte(strings.Join(headers, "\n") + "\n\n" + parts[1])
	members[di+"WHEEL"] = []byte("Wheel-Version: 1.0\nGenerator: ptsip-complete-package-go/1\nRoot-Is-Purelib: true\nTag: py3-none-any\n")
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
	return output.Bytes(), nil
}
