package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
)

const buildDescriptor = "ptsip-build-source.json"

var revisionPattern = regexp.MustCompile("^[a-f0-9]{40}$")
var distributionPattern = regexp.MustCompile("(?i)^ptsip-[a-z0-9_.+!]+$")

func buildRevision(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	var revision string
	if _, err := os.Lstat(filepath.Join(root, ".git")); err == nil {
		status, err := runGit(root, "status", "--porcelain")
		if err != nil {
			return "", err
		}
		if len(status) != 0 {
			return "", errors.New("wheel and sdist builds require a clean source checkout")
		}
		head, err := runGit(root, "rev-parse", "--verify", "HEAD^{commit}")
		if err != nil {
			return "", err
		}
		revision = strings.TrimSpace(string(head))
	} else if os.IsNotExist(err) {
		var source struct {
			Format string `json:"schema_version"`
			SHA    string `json:"source_sha"`
		}
		if err := readJSON(filepath.Join(root, buildDescriptor), &source); err != nil {
			return "", fmt.Errorf("read sdist source binding: %w", err)
		}
		if source.Format != "ptsip-build-source/v1" {
			return "", errors.New("unsupported sdist source binding")
		}
		revision = source.SHA
	} else {
		return "", err
	}
	if !revisionPattern.MatchString(revision) {
		return "", errors.New("invalid PTSIP source revision")
	}
	return revision, nil
}

func readJSON(name string, dst any) error {
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if _, err := jsondoc.Decode(raw); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return json.Unmarshal(raw, dst)
}

type wheelFile struct {
	data []byte
	mode os.FileMode
}

type wheelContents map[string]wheelFile

func (files wheelContents) add(name string, data []byte, executable bool) error {
	if _, exists := files[name]; exists {
		return fmt.Errorf("duplicate wheel member %s", name)
	}
	mode := os.FileMode(0o644)
	if executable {
		mode = 0o755
	}
	files[name] = wheelFile{data: data, mode: mode}
	return nil
}

func readBaseWheel(base string) (wheelContents, string, error) {
	distribution, ok := strings.CutSuffix(filepath.Base(base), "-py3-none-any.whl")
	if !ok || !distributionPattern.MatchString(distribution) {
		return nil, "", errors.New("base wheel must contain the staged PTSIP Python distribution")
	}
	wheel, err := zip.OpenReader(base)
	if err != nil {
		return nil, "", err
	}
	defer wheel.Close()
	files := wheelContents{}
	seen := make(map[string]bool, len(wheel.File))
	var size uint64
	for _, item := range wheel.File {
		name := item.Name
		if name == "." || name == ".." || path.Clean(name) != name || path.IsAbs(name) || strings.HasPrefix(name, "../") ||
			strings.ContainsAny(name, "\\:\r\n") || !item.Mode().IsRegular() {
			return nil, "", fmt.Errorf("unsupported base wheel member %q", name)
		}
		if retiredProductMember(name) {
			continue
		}
		folded := strings.ToLower(name)
		if seen[folded] {
			return nil, "", fmt.Errorf("ambiguous base wheel member %q", name)
		}
		seen[folded] = true
		size += item.UncompressedSize64
		if item.UncompressedSize64 > 128<<20 || size > 512<<20 {
			return nil, "", fmt.Errorf("base wheel member too large: %s", name)
		}
		reader, err := item.Open()
		if err != nil {
			return nil, "", err
		}
		data, err := io.ReadAll(io.LimitReader(reader, int64(item.UncompressedSize64)+1))
		if err := errors.Join(err, reader.Close()); err != nil {
			return nil, "", err
		}
		if uint64(len(data)) != item.UncompressedSize64 {
			return nil, "", fmt.Errorf("base wheel member size differs: %s", name)
		}
		if err := files.add(name, data, false); err != nil {
			return nil, "", err
		}
	}
	info := distribution + ".dist-info/"
	if len(files[info+"METADATA"].data) == 0 || !bytes.Contains(files[info+"WHEEL"].data, []byte("Root-Is-Purelib: true")) {
		return nil, "", errors.New("base wheel lacks pure Python distribution metadata")
	}
	delete(files, info+"RECORD")
	delete(files, info+"WHEEL")
	return files, distribution, nil
}

func buildWheel(root, base, output, platformTag string, progress io.Writer) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	revision, err := buildRevision(root)
	if err != nil {
		return "", err
	}
	if err := verifyEngineWheelSource(root, revision); err != nil {
		return "", err
	}
	versions, err := versionsource.Read(root)
	if err != nil {
		return "", err
	}
	host, err := currentWheelHost(platformTag)
	if err != nil {
		return "", err
	}
	files, distribution, err := readBaseWheel(base)
	if err != nil {
		return "", err
	}
	if err := checkWheelVersion(files, distribution, versions); err != nil {
		return "", err
	}
	source, err := loadGitSource(root)
	if err != nil {
		return "", err
	}
	if source.Revision != revision {
		return "", errors.New("source HEAD changed before Wheel staging")
	}
	if err := stageRegisteredProductAssets(source, files); err != nil {
		return "", err
	}
	if err := stageWheelAssets(root, files); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp("", "ptsip-wheel-native-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)

	builder := nativeWheelBuilder{root: root, revision: revision, host: host, stage: stage, progress: progress}
	if err := builder.addGo(files, distribution, versions.ToolVersion); err != nil {
		return "", err
	}
	if err := builder.addEngines(files, distribution, versions.Engines); err != nil {
		return "", err
	}
	if current, err := buildRevision(root); err != nil || current != revision {
		return "", errors.Join(errors.New("source changed during Wheel assembly"), err)
	}
	info := distribution + ".dist-info/"
	metadata := fmt.Sprintf("Wheel-Version: 1.0\nGenerator: ptsip-source-build/2\nRoot-Is-Purelib: false\nTag: py3-none-%s\n", host.tag)
	if err := files.add(info+"WHEEL", []byte(metadata), false); err != nil {
		return "", err
	}
	name := distribution + "-py3-none-" + host.tag + ".whl"
	if err := writeWheel(filepath.Join(output, name), info+"RECORD", files); err != nil {
		return "", err
	}
	return name, nil
}

func checkWheelVersion(files wheelContents, distribution string, versions versionsource.Versions) error {
	want := versions.DistributionVersion()
	if distribution != "ptsip-"+want {
		return fmt.Errorf("base wheel distribution %q differs from canonical Package version %s", distribution, want)
	}
	info := distribution + ".dist-info/"
	metadata, err := mail.ReadMessage(bytes.NewReader(files[info+"METADATA"].data))
	if err != nil {
		return fmt.Errorf("read base wheel metadata: %w", err)
	}
	if actual := metadata.Header["Version"]; len(actual) != 1 || actual[0] != want {
		return fmt.Errorf("base wheel METADATA Version differs from canonical Package version %s", want)
	}
	if names := metadata.Header["Name"]; len(names) != 1 || !strings.EqualFold(names[0], "PTSIP") {
		return errors.New("base wheel METADATA Name does not identify PTSIP")
	}
	for member := range files {
		top, _, _ := strings.Cut(member, "/")
		if strings.HasSuffix(top, ".dist-info") && top+"/" != info {
			return fmt.Errorf("base wheel contains foreign dist-info %s", top)
		}
	}
	return nil
}

func wheelDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func writeWheel(destination, recordPath string, files wheelContents) error {
	names := make([]string, 0, len(files)+1)
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	var record bytes.Buffer
	rows := csv.NewWriter(&record)
	for _, name := range names {
		data := files[name].data
		digest := sha256.Sum256(data)
		if err := rows.Write([]string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(digest[:]), strconv.Itoa(len(data))}); err != nil {
			return err
		}
	}
	if err := rows.Write([]string{recordPath, "", ""}); err != nil {
		return err
	}
	rows.Flush()
	if err := rows.Error(); err != nil {
		return err
	}
	if err := files.add(recordPath, record.Bytes(), false); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(destination), ".ptsip-wheel-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	archive := zip.NewWriter(file)
	// Keep dist-info at the end so consumers can update metadata without
	// rewriting the package payload.
	slices.SortFunc(names, func(a, b string) int {
		info := path.Dir(recordPath) + "/"
		x, y := strings.HasPrefix(a, info), strings.HasPrefix(b, info)
		if x != y {
			if x {
				return 1
			}
			return -1
		}
		return strings.Compare(a, b)
	})
	for _, name := range append(names, recordPath) {
		member := files[name]
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(member.mode)
		writer, err := archive.CreateHeader(header)
		if err == nil {
			_, err = writer.Write(member.data)
		}
		if err != nil {
			return errors.Join(err, archive.Close(), file.Close())
		}
	}
	if err := errors.Join(archive.Close(), file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), destination)
}
