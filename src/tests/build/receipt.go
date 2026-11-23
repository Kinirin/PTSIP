package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
)

type receiptComponent struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type nativeReceipt struct {
	SourceSHA              string                      `json:"source_sha"`
	VerifiedSnapshotSHA256 string                      `json:"verified_snapshot_sha256"`
	Components             map[string]receiptComponent `json:"components"`
	Filename               string                      `json:"filename"`
	Platform               string                      `json:"platform"`
	WheelSHA256            string                      `json:"wheel_sha256"`
}

func writeNativeReceipt(filename, artifact string, source gitSource, assets registeredAssets, wheel verifiedWheel, digest string) error {
	head, err := gitOutput(source.root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || strings.TrimSpace(string(head)) != source.revision ||
		os.Getenv("PTSIP_VERIFIED_ENGINE_SOURCE_SHA") != source.revision {
		return errors.New("native Wheel receipt requires the verified source revision")
	}
	status, err := gitOutput(source.root, "status", "--porcelain")
	if err != nil || len(status) != 0 {
		return errors.New("native Wheel receipt requires a clean source checkout")
	}
	version, err := source.read(versionsource.Path)
	if err != nil {
		return err
	}
	versions, err := versionsource.Parse(version)
	if err != nil {
		return err
	}
	snapshot, err := source.read("src/ptsip/engines/contracts/engines_registries/derived-snapshot.json")
	if err != nil {
		return err
	}
	ext := ""
	if wheel.platform == "windows-x64" {
		ext = ".exe"
	}
	scripts := "ptsip-" + versions.DistributionVersion() + ".data/scripts/"
	receipt := nativeReceipt{
		SourceSHA: source.revision, VerifiedSnapshotSHA256: sha(snapshot),
		Filename: filepath.Base(artifact), Platform: wheel.platform, WheelSHA256: digest,
		Components: map[string]receiptComponent{
			"tool":     {versions.ToolVersion, wheel.digests[scripts+"ptsip"+ext]},
			"profiles": {assets.activeProfile, wheel.digests[path.Join("ptsip", assets.activeProfilePath)]},
		},
	}
	if assets.activeProfile != versions.ProfileVersion || len(assets.engines) != len(versions.Engines) {
		return errors.New("native Wheel receipt inventory differs from src/VERSION")
	}
	for id, expected := range versions.Engines {
		if assets.engines[id].Version != expected {
			return fmt.Errorf("native Wheel receipt version differs from src/VERSION for %s", id)
		}
		receipt.Components[id] = receiptComponent{expected, wheel.digests[scripts+"ptsip-engine-"+id+ext]}
	}
	for id, component := range receipt.Components {
		if component.SHA256 == "" {
			return fmt.Errorf("native Wheel receipt is missing the verified digest for %s", id)
		}
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	output, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := output.Write(append(raw, '\n'))
	if err := errors.Join(writeErr, output.Close()); err != nil {
		return errors.Join(err, os.Remove(filename))
	}
	return nil
}
