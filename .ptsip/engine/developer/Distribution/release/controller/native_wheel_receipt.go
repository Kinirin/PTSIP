package releasecontroller

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
)

const wheelReceiptMetadataLimit = 8 << 20
const wheelReceiptArtifactLimit = 1 << 30

var wheelReceiptCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)
var wheelReceiptMacTag = regexp.MustCompile(`^macosx_[0-9]+_[0-9]+_arm64$`)

type nativeWheelComponent struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type NativeWheelReceipt struct {
	Source    string                          `json:"source_sha"`
	Snapshot  string                          `json:"verified_snapshot_sha256"`
	Inventory map[string]nativeWheelComponent `json:"components"`
	Filename  string                          `json:"filename"`
	Platform  string                          `json:"platform"`
	SHA256    string                          `json:"wheel_sha256"`
}

type engineVerificationProof struct {
	Source   string `json:"source_sha"`
	Sync     string `json:"sync_sha"`
	Snapshot string `json:"snapshot_sha256"`
	Graph    string `json:"dependency_graph_sha256"`
}

// VerifyNativeWheelReceipt binds existing installed verification evidence to an
// independently verified source and actual artifact. Publication has its own
// admission path.
func VerifyNativeWheelReceipt(root, wheelFile, receiptFile, engineProofFile string) (NativeWheelReceipt, error) {
	var receipt NativeWheelReceipt
	if wheelFile == "" || receiptFile == "" || engineProofFile == "" {
		return receipt, fmt.Errorf("wheel, installed receipt and engine proof are required")
	}
	head, err := cleanWheelReceiptHead(root)
	if err != nil {
		return receipt, err
	}
	if err := boundary.CheckOperation(root, "SOURCE_BOUND_TEST_EVIDENCE"); err != nil {
		return receipt, err
	}
	contract, err := ReadContract(root)
	if err != nil {
		return receipt, err
	}
	raw, err := readWheelReceiptInput(receiptFile)
	if err != nil {
		return receipt, fmt.Errorf("read native wheel receipt: %w", err)
	}
	if err := boundary.ValidateNativeWheelReceipt(root, raw); err != nil {
		return receipt, err
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return receipt, err
	}
	if receipt.Source != head || receipt.Filename != filepath.Base(wheelFile) {
		return receipt, fmt.Errorf("native wheel receipt source SHA or filename differs")
	}
	if !slices.Contains(contract.Required, receipt.Platform) && !slices.Contains(contract.Optional, receipt.Platform) {
		return receipt, fmt.Errorf("native wheel receipt platform is unregistered")
	}
	if err := verifyWheelReceiptEngineProof(root, engineProofFile, contract, receipt); err != nil {
		return receipt, err
	}
	versions, err := ReadSourceVersions(root)
	if err != nil {
		return receipt, err
	}
	if _, err := SyncPackageVersion(root, true); err != nil {
		return receipt, err
	}
	if err := verifyProfileProjection(root, versions.ProfileVersion); err != nil {
		return receipt, err
	}
	profiles, _, err := ReadProfileRegistrations(root)
	if err != nil {
		return receipt, err
	}
	profile := profiles[versionsource.ProfileKey]
	var engines struct {
		Entries []struct {
			Kind string `json:"kind"`
			ID   string `json:"engine_id"`
		} `json:"official_engine_entries"`
	}
	if err := ReadJSON(root, contract.Sources.EngineRegistry, &engines); err != nil {
		return receipt, err
	}
	ids := make(map[string]string, len(engines.Entries))
	for _, engine := range engines.Entries {
		ids[strings.ToLower(engine.Kind)] = engine.ID
	}
	if err := verifyReceiptArchive(wheelFile, receipt, versions, profile, ids); err != nil {
		return receipt, err
	}
	finalHead, err := cleanWheelReceiptHead(root)
	if err != nil {
		return receipt, err
	}
	if finalHead != head {
		return receipt, fmt.Errorf("source HEAD changed during native wheel receipt verification")
	}
	return receipt, nil
}

func cleanWheelReceiptHead(root string) (string, error) {
	head, err := Run(root, "git", "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	commit := strings.TrimSpace(string(head))
	if !wheelReceiptCommit.MatchString(commit) {
		return "", fmt.Errorf("native wheel receipt requires an exact committed source SHA")
	}
	status, err := Run(root, "git", "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	if len(status) != 0 {
		return "", fmt.Errorf("native wheel receipt requires a clean checkout")
	}
	return commit, nil
}

func readWheelReceiptInput(name string) ([]byte, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > wheelReceiptMetadataLimit {
		return nil, fmt.Errorf("receipt input must be a bounded regular file: %s", name)
	}
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readReceiptMetadata(file)
}

func verifyWheelReceiptEngineProof(root, file string, contract Contract, receipt NativeWheelReceipt) error {
	raw, err := readWheelReceiptInput(file)
	if err != nil {
		return fmt.Errorf("read independent engine source proof: %w", err)
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		return fmt.Errorf("decode independent engine source proof: %w", err)
	}
	object, ok := value.(map[string]any)
	if !ok || len(object) != 4 {
		return fmt.Errorf("independent engine source proof has an unexpected shape")
	}
	var proof engineVerificationProof
	if err := json.Unmarshal(raw, &proof); err != nil {
		return err
	}
	if proof.Source != receipt.Source || !wheelReceiptCommit.MatchString(proof.Sync) || proof.Snapshot != receipt.Snapshot {
		return fmt.Errorf("independent engine source proof does not bind the receipt source and snapshot")
	}
	if _, err := Run(root, "git", "merge-base", "--is-ancestor", proof.Sync, proof.Source); err != nil {
		return fmt.Errorf("native wheel source must descend from the engine synchronization commit: %w", err)
	}
	if _, err := Run(root, "git", "diff", "--quiet", proof.Sync, proof.Source, "--", versionsource.Path); err != nil {
		return fmt.Errorf("native wheel version source differs from the engine synchronization commit: %w", err)
	}
	for _, observed := range []struct{ path, digest string }{
		{contract.Verification.Snapshot, proof.Snapshot},
		{contract.Verification.Graph, proof.Graph},
	} {
		data, err := Run(root, "git", "show", proof.Source+":"+observed.path)
		if err != nil {
			return err
		}
		if Digest(data) != observed.digest {
			return fmt.Errorf("independent engine proof digest differs from committed %s", observed.path)
		}
	}
	return nil
}

func verifyReceiptArchive(file string, receipt NativeWheelReceipt, versions VersionSource, profile ProfileRegistration, engineIDs map[string]string) error {
	distribution := "ptsip-" + versions.DistributionVersion()
	tag, ok := strings.CutPrefix(receipt.Filename, distribution+"-py3-none-")
	if !ok || !strings.HasSuffix(tag, ".whl") {
		return fmt.Errorf("native wheel filename differs from source distribution version")
	}
	tag = strings.TrimSuffix(tag, ".whl")
	extension, target, err := receiptPlatform(receipt.Platform, tag)
	if err != nil {
		return err
	}

	info, err := os.Lstat(file)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > wheelReceiptArtifactLimit {
		return fmt.Errorf("native wheel must be a bounded regular file")
	}
	input, err := os.Open(file)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := verifyReceiptFileDigest(input, receipt.SHA256); err != nil {
		return err
	}
	archive, err := zip.NewReader(input, info.Size())
	if err != nil {
		return err
	}
	members := make(map[string]*zip.File, len(archive.File))
	folded := make(map[string]bool, len(archive.File))
	for _, member := range archive.File {
		name := member.Name
		key := strings.ToLower(name)
		if folded[key] || !member.Mode().IsRegular() || member.UncompressedSize64 > 128<<20 ||
			name == "." || name == ".." || path.Clean(name) != name || path.IsAbs(name) ||
			strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\:\x00\r\n") {
			return fmt.Errorf("invalid or ambiguous native wheel member: %s", name)
		}
		folded[key], members[name] = true, member
	}
	prefix := distribution + ".dist-info/"
	if err := verifyReceiptWheelHeaders(members, prefix, versions.DistributionVersion(), tag); err != nil {
		return err
	}
	if err := verifyReceiptComponents(members, distribution, receipt, versions, profile, extension, target, engineIDs); err != nil {
		return err
	}
	if err := verifyReceiptRecord(members, prefix+"RECORD"); err != nil {
		return err
	}
	_, err = input.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	return verifyReceiptFileDigest(input, receipt.SHA256)
}

func verifyReceiptFileDigest(input io.Reader, expected string) error {
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(input, wheelReceiptArtifactLimit+1))
	if err != nil {
		return err
	}
	if size > wheelReceiptArtifactLimit || hex.EncodeToString(hash.Sum(nil)) != expected {
		return fmt.Errorf("native wheel artifact SHA-256 differs from receipt")
	}
	return nil
}

func receiptPlatform(platform, tag string) (extension, target string, err error) {
	switch platform {
	case "windows-x64":
		if tag == "win_amd64" {
			return ".exe", "x86_64-pc-windows-msvc", nil
		}
	case "linux-x64":
		if tag == "linux_x86_64" {
			return "", "x86_64-unknown-linux-gnu", nil
		}
	case "macos-arm64":
		if wheelReceiptMacTag.MatchString(tag) {
			return "", "aarch64-apple-darwin", nil
		}
	}
	return "", "", fmt.Errorf("native wheel filename tag differs from receipt platform")
}

func readReceiptMetadata(input io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(input, wheelReceiptMetadataLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > wheelReceiptMetadataLimit {
		return nil, fmt.Errorf("native wheel receipt metadata exceeds limit")
	}
	return data, nil
}

func receiptWheelMetadata(members map[string]*zip.File, name string) ([]byte, error) {
	member := members[name]
	if member == nil {
		return nil, fmt.Errorf("native wheel metadata missing: %s", name)
	}
	input, err := member.Open()
	if err != nil {
		return nil, err
	}
	defer input.Close()
	return readReceiptMetadata(input)
}

func verifyReceiptWheelHeaders(members map[string]*zip.File, prefix, version, tag string) error {
	for name, headers := range map[string]map[string]string{
		"METADATA": {"Name": "PTSIP", "Version": version},
		"WHEEL":    {"Root-Is-Purelib": "false", "Tag": "py3-none-" + tag},
	} {
		raw, err := receiptWheelMetadata(members, prefix+name)
		if err != nil {
			return err
		}
		message, err := mail.ReadMessage(bytes.NewReader(append(raw, '\n')))
		if err != nil {
			return fmt.Errorf("invalid native wheel %s: %w", name, err)
		}
		for key, expected := range headers {
			values := message.Header[textproto.CanonicalMIMEHeaderKey(key)]
			if len(values) != 1 || (key == "Name" && !strings.EqualFold(values[0], expected)) ||
				(key != "Name" && values[0] != expected) {
				return fmt.Errorf("native wheel %s %s differs from source", name, key)
			}
		}
	}
	return nil
}

func verifyReceiptRecord(members map[string]*zip.File, name string) error {
	raw, err := receiptWheelMetadata(members, name)
	if err != nil {
		return err
	}
	rows := csv.NewReader(bytes.NewReader(raw))
	rows.FieldsPerRecord = 3
	seen := make(map[string]bool, len(members))
	var total uint64
	for {
		row, err := rows.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read native wheel RECORD: %w", err)
		}
		member := members[row[0]]
		if member == nil || seen[row[0]] {
			return fmt.Errorf("native wheel RECORD member is unknown or duplicated: %s", row[0])
		}
		seen[row[0]] = true
		if row[0] == name && row[1] == "" && row[2] == "" {
			continue
		}
		total += member.UncompressedSize64
		if total > wheelReceiptArtifactLimit {
			return fmt.Errorf("native wheel expanded payload exceeds receipt verification limit")
		}
		encodedDigest, ok := strings.CutPrefix(row[1], "sha256=")
		decoded, err := base64.RawURLEncoding.DecodeString(encodedDigest)
		if row[0] == name || !ok || err != nil || len(decoded) != sha256.Size || row[2] != strconv.FormatUint(member.UncompressedSize64, 10) {
			return fmt.Errorf("native wheel RECORD lacks exact SHA-256 and size: %s", row[0])
		}
		input, err := member.Open()
		if err != nil {
			return err
		}
		hash := sha256.New()
		size, readErr := io.Copy(hash, io.LimitReader(input, int64(member.UncompressedSize64)+1))
		closeErr := input.Close()
		if readErr != nil || closeErr != nil || size != int64(member.UncompressedSize64) || !bytes.Equal(hash.Sum(nil), decoded) {
			return fmt.Errorf("native wheel RECORD SHA-256 differs: %s", row[0])
		}
	}
	if len(seen) != len(members) {
		return fmt.Errorf("native wheel RECORD does not cover every member")
	}
	return nil
}

func verifyReceiptComponents(members map[string]*zip.File, distribution string, receipt NativeWheelReceipt, versions VersionSource, profile ProfileRegistration, extension, target string, engineIDs map[string]string) error {
	if len(receipt.Inventory) != len(versions.Engines)+2 {
		return fmt.Errorf("native wheel receipt component inventory differs from source")
	}
	verify := func(id, version, digest string) error {
		if (nativeWheelComponent{version, digest}) != receipt.Inventory[id] {
			return fmt.Errorf("native wheel receipt version or member digest differs for %s", id)
		}
		return nil
	}
	scripts := distribution + ".data/scripts/"
	tool, err := nativeWheelMemberDigest(members[scripts+"ptsip"+extension], receipt.Platform)
	if err != nil {
		return fmt.Errorf("native wheel Tool member: %w", err)
	}
	if err := verify("tool", versions.ToolVersion, tool); err != nil {
		return err
	}
	profileBytes, err := receiptWheelMetadata(members, "ptsip/schemas/"+profile.Path)
	if err != nil {
		return err
	}
	if Digest(bytes.ReplaceAll(profileBytes, []byte("\r\n"), []byte("\n"))) != profile.Digest {
		return fmt.Errorf("native wheel Profile schema differs from registered source")
	}
	if err := verify("profiles", versions.ProfileVersion, Digest(profileBytes)); err != nil {
		return err
	}
	bundleBytes, err := receiptWheelMetadata(members, distribution+".dist-info/ptsip-engines.json")
	if err != nil {
		return err
	}
	var bundle struct {
		Schema     string `json:"schema_version"`
		Package    string `json:"package_version"`
		Platform   string `json:"platform"`
		Authorized *bool  `json:"execution_authorized"`
		Engines    map[string]struct {
			Schema     string `json:"schema_version"`
			Source     string `json:"source_sha"`
			ID         string `json:"engine_id"`
			Version    string `json:"engine_version"`
			Target     string `json:"host_target"`
			Entrypoint string `json:"entrypoint"`
			Digest     string `json:"binary_sha256"`
			Authorized *bool  `json:"execution_authorized"`
		} `json:"engines"`
	}
	if _, err := jsondoc.Decode(bundleBytes); err != nil {
		return err
	}
	if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
		return err
	}
	if bundle.Schema != "ptsip-bundled-engine-origins/v1" || bundle.Package != versions.DistributionVersion() ||
		bundle.Platform != receipt.Platform || bundle.Authorized == nil || *bundle.Authorized ||
		len(bundle.Engines) != len(versions.Engines) {
		return fmt.Errorf("native wheel engine origin inventory differs from receipt")
	}
	for _, id := range versionsource.EngineIDs() {
		entrypoint := "ptsip-engine-" + id + extension
		digest, err := nativeWheelMemberDigest(members[scripts+entrypoint], receipt.Platform)
		if err != nil {
			return fmt.Errorf("native wheel engine %s member: %w", id, err)
		}
		version := versions.Engines[id]
		if err := verify(id, version, digest); err != nil {
			return err
		}
		origin := bundle.Engines[id]
		if origin.Schema != "ptsip-independent-wheel-origin/v1" || origin.Source != receipt.Source ||
			origin.ID == "" || origin.ID != engineIDs[id] || origin.Version != version ||
			origin.Target != target || origin.Entrypoint != entrypoint || origin.Digest != digest ||
			origin.Authorized == nil || *origin.Authorized {
			return fmt.Errorf("native wheel engine origin differs for %s", id)
		}
	}
	return nil
}

func nativeWheelMemberDigest(member *zip.File, platform string) (string, error) {
	if member == nil {
		return "", fmt.Errorf("binary is missing")
	}
	input, err := member.Open()
	if err != nil {
		return "", err
	}
	defer input.Close()
	var signature [4]byte
	if _, err := io.ReadFull(input, signature[:]); err != nil {
		return "", err
	}
	if !(platform == "windows-x64" && string(signature[:2]) == "MZ" ||
		platform == "linux-x64" && string(signature[:]) == "\x7fELF" ||
		platform == "macos-arm64" && (signature == [4]byte{0xcf, 0xfa, 0xed, 0xfe} ||
			signature == [4]byte{0xfe, 0xed, 0xfa, 0xcf})) {
		return "", fmt.Errorf("binary does not match platform %s", platform)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.MultiReader(bytes.NewReader(signature[:]), input)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
