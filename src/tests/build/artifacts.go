package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

const metadataLimit = 16 << 20

func readMetadata(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, metadataLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > metadataLimit {
		return nil, errors.New("artifact metadata exceeds verification limit")
	}
	return data, nil
}

func excluded(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) || name == strings.TrimSuffix(prefix, "/") {
			return true
		}
	}
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "engines" && len(parts) > i+2 && parts[i+2] == "target" {
			return true
		}
	}
	return false
}

func canonicalJSONDigest(data []byte) string {
	return sha(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
}

func (assets registeredAssets) digestFor(name string, data []byte) string {
	if assets.exactDigestPaths[name] {
		return sha(data)
	}
	return canonicalJSONDigest(data)
}

func verifySDist(file io.Reader, revision string, rules sdistRules, assets registeredAssets) error {
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	archive := tar.NewReader(compressed)
	seen := map[string]bool{}
	root := ""
	for {
		entry, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		name := entry.Name
		if entry.Typeflag == tar.TypeDir {
			name = strings.TrimSuffix(name, "/")
		}
		if !safePath(name) {
			return fmt.Errorf("unsafe sdist entry: %s", entry.Name)
		}
		prefix, member, hasMember := strings.Cut(name, "/")
		if root != "" && prefix != root {
			return errors.New("sdist has more than one source root")
		}
		root = prefix
		if excluded(member, rules.ExcludedPrefixes) {
			return fmt.Errorf("excluded entry in sdist: %s", name)
		}
		if member == "setup.py" || retiredPythonMember(strings.TrimPrefix(member, "src/")) {
			return fmt.Errorf("excluded Python implementation in sdist: %s", name)
		}
		if entry.Typeflag == tar.TypeDir {
			continue
		}
		if entry.Typeflag != tar.TypeReg || !hasMember || seen[member] {
			return fmt.Errorf("invalid or duplicate sdist member: %s", name)
		}
		seen[member] = true
		if member == rules.SourceDescriptor {
			var descriptor struct {
				Version string `json:"schema_version"`
				Source  string `json:"source_sha"`
			}
			raw, err := readMetadata(archive)
			if err != nil {
				return err
			}
			if err := decode(raw, &descriptor); err != nil || descriptor.Version != "ptsip-build-source/v1" || descriptor.Source != revision {
				return errors.New("sdist is not bound to the exact Git HEAD")
			}
		}
		if expected, isAsset := assets.digests[member]; isAsset {
			raw, err := readMetadata(archive)
			if err != nil || assets.digestFor(member, raw) != expected {
				return fmt.Errorf("sdist asset digest mismatch: %s", member)
			}
		}
	}
	// Read through the gzip checksum after the end-of-archive marker.
	if _, err := io.Copy(io.Discard, compressed); err != nil {
		return err
	}
	if !seen[rules.SourceDescriptor] {
		return errors.New("sdist source descriptor is missing")
	}
	for name := range assets.digests {
		if !seen[name] {
			return fmt.Errorf("required registered sdist asset missing: %s", name)
		}
	}
	return nil
}

func zipMetadata(member *zip.File) ([]byte, error) {
	reader, err := member.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return readMetadata(reader)
}

func verifyWheel(file io.ReaderAt, size int64, revision string, contract buildContract, assets registeredAssets) error {
	archive, err := zip.NewReader(file, size)
	if err != nil {
		return err
	}
	members := map[string]*zip.File{}
	originPath := ""
	for _, entry := range archive.File {
		name := strings.TrimSuffix(entry.Name, "/")
		if !safePath(name) || excluded(name, contract.Wheel.ExcludedPrefixes) || strings.Contains("/"+name+"/", "/.ptsip/") ||
			retiredPythonMember(name) {
			return fmt.Errorf("unsafe or excluded wheel member: %s", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if members[name] != nil || !entry.Mode().IsRegular() {
			return fmt.Errorf("invalid or duplicate wheel member: %s", entry.Name)
		}
		members[name] = entry
		if strings.HasSuffix(name, contract.Wheel.EngineOriginSuffix) {
			if originPath != "" || strings.Contains(strings.TrimSuffix(name, contract.Wheel.EngineOriginSuffix), "/") {
				return errors.New("wheel engine origin location is ambiguous")
			}
			originPath = name
		}
	}
	if originPath == "" {
		return errors.New("wheel bundled-engine origins are missing")
	}
	metadataPrefix := path.Dir(originPath) + "/"
	for _, required := range []string{"METADATA", "WHEEL", "RECORD"} {
		if members[metadataPrefix+required] == nil {
			return fmt.Errorf("wheel metadata is missing: %s", required)
		}
	}
	digests, err := verifyRecord(members, metadataPrefix+"RECORD")
	if err != nil {
		return err
	}
	for reference, expected := range assets.digests {
		name := path.Base(strings.TrimSuffix(assets.productPrefix, "/")) + "/" + strings.TrimPrefix(reference, assets.productPrefix)
		member := members[name]
		if member == nil {
			return fmt.Errorf("required registered wheel asset missing: %s", name)
		}
		raw, err := zipMetadata(member)
		if err != nil || assets.digestFor(reference, raw) != expected {
			return fmt.Errorf("wheel registered asset digest mismatch: %s", name)
		}
	}
	return verifyWheelOrigins(members, digests, originPath, revision, contract.Wheel, assets)
}

func retiredPythonMember(name string) bool {
	return (strings.HasPrefix(name, "ptsip/") && strings.HasSuffix(name, ".py")) ||
		strings.HasPrefix(name, "ptsip/migration/") || name == "ptsip/migration"
}

func verifyRecord(members map[string]*zip.File, recordPath string) (map[string]string, error) {
	raw, err := zipMetadata(members[recordPath])
	if err != nil {
		return nil, err
	}
	record := csv.NewReader(bytes.NewReader(raw))
	record.FieldsPerRecord = 3
	digests := map[string]string{}
	for {
		entry, err := record.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		name := entry[0]
		member := members[name]
		if _, seen := digests[name]; seen || member == nil {
			return nil, fmt.Errorf("duplicate or unknown wheel RECORD entry: %s", name)
		}
		if name == recordPath && entry[1] == "" && entry[2] == "" {
			digests[name] = ""
			continue
		}
		encoded, present := strings.CutPrefix(entry[1], "sha256=")
		expected, decodeErr := base64.RawURLEncoding.DecodeString(encoded)
		if !present || decodeErr != nil || len(expected) != 32 || entry[2] != strconv.FormatUint(member.UncompressedSize64, 10) {
			return nil, fmt.Errorf("wheel RECORD requires SHA-256 and exact size: %s", name)
		}
		reader, err := member.Open()
		if err != nil {
			return nil, err
		}
		observed, err := digestReader(reader)
		closeErr := reader.Close()
		if err != nil || closeErr != nil || observed != hex.EncodeToString(expected) {
			return nil, fmt.Errorf("wheel RECORD digest mismatch: %s", name)
		}
		digests[name] = observed
	}
	if len(digests) != len(members) {
		return nil, errors.New("wheel RECORD does not cover every regular member")
	}
	return digests, nil
}

func verifyWheelOrigins(members map[string]*zip.File, digests map[string]string, originPath, revision string, rules wheelRules, assets registeredAssets) error {
	var bundle struct {
		Version             string `json:"schema_version"`
		Platform            string `json:"platform"`
		ExecutionAuthorized *bool  `json:"execution_authorized"`
		Engines             map[string]struct {
			Version             string `json:"schema_version"`
			Source              string `json:"source_sha"`
			EngineID            string `json:"engine_id"`
			EngineVersion       string `json:"engine_version"`
			Entrypoint          string `json:"entrypoint"`
			Digest              string `json:"binary_sha256"`
			ExecutionAuthorized *bool  `json:"execution_authorized"`
		} `json:"engines"`
	}
	raw, err := zipMetadata(members[originPath])
	if err != nil {
		return err
	}
	if err := decode(raw, &bundle); err != nil || bundle.Version != "ptsip-bundled-engine-origins/v1" || bundle.Platform == "" ||
		bundle.ExecutionAuthorized == nil || *bundle.ExecutionAuthorized || len(bundle.Engines) != len(assets.engines) {
		return errors.New("wheel engine inventory or origin metadata is invalid")
	}
	extension := ""
	if strings.HasPrefix(bundle.Platform, "windows-") {
		extension = ".exe"
	}
	scripts := strings.TrimSuffix(originPath, rules.EngineOriginSuffix) + ".data/scripts/"
	for kind, engine := range assets.engines {
		origin := bundle.Engines[kind]
		if origin.Version != "ptsip-independent-wheel-origin/v1" || origin.Source != revision || origin.EngineID != engine.ID ||
			origin.EngineVersion != engine.Version || origin.Entrypoint != "ptsip-engine-"+kind+extension ||
			origin.Digest == "" || origin.Digest != digests[scripts+origin.Entrypoint] || origin.ExecutionAuthorized == nil || *origin.ExecutionAuthorized {
			return fmt.Errorf("wheel engine origin does not bind the registered source and binary: %s", kind)
		}
	}
	var common struct {
		Format         string `json:"format"`
		Source         string `json:"source_sha"`
		Binary         string `json:"binary"`
		Digest         string `json:"binary_sha256"`
		RegistryDigest string `json:"registry_sha256"`
	}
	member := members[rules.CommonContractOrigin]
	if member == nil {
		return errors.New("wheel common-contract binary origin is missing")
	}
	raw, err = zipMetadata(member)
	if err != nil {
		return err
	}
	if err := decode(raw, &common); err != nil || common.Format != "ptsip-common-contract-wheel-origin/v1" || common.Source != revision || common.RegistryDigest != assets.commonRegistryDigest ||
		common.Binary != "ptsip-common-contract"+extension || common.Digest == "" || common.Digest != digests[path.Join(path.Dir(rules.CommonContractOrigin), common.Binary)] {
		return errors.New("wheel common-contract binary origin does not bind the registered source and binary")
	}
	router := digests[scripts+rules.GoEntrypoint+extension]
	if router == "" {
		return fmt.Errorf("wheel Go router binary is missing: %s", rules.GoEntrypoint+extension)
	}
	if digests[scripts+"ptsip"+extension] != router {
		return errors.New("wheel native ptsip command does not match the Go router")
	}
	return nil
}
