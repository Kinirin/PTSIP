// Package enginerouter invokes the existing PTSIP native dispatcher from Go.
package enginerouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Caller interface {
	Invoke(ctx context.Context, operation string, input, output any) error
}

type Client struct {
	CLI         string
	PackageRoot string
	ScriptsDir  string
}

type engineIdentity struct {
	Engine          string `json:"engine_id"`
	EngineVersion   string `json:"engine_version"`
	Contract        string `json:"contract_id"`
	ContractVersion string `json:"contract_version"`
}

// Invoke resolves the current identity for a registered operation, then sends
// its request through the CLI. The CLI remains responsible for routing and
// installed artifact verification. A PASS response conveys no authority.
func (c Client) Invoke(ctx context.Context, operation string, input, output any) error {
	identity, err := c.resolve(operation)
	if err != nil {
		return err
	}
	request, err := json.Marshal(struct {
		engineIdentity
		Operation string `json:"operation_id"`
		Input     any    `json:"input"`
	}{identity, operation, input})
	if err != nil {
		return fmt.Errorf("encode %s request: %w", operation, err)
	}
	if len(request) > 1<<20 {
		return fmt.Errorf("%s request exceeds the native transport limit", operation)
	}
	cmd := exec.CommandContext(ctx, c.CLI, "engine", c.PackageRoot, c.ScriptsDir, operation)
	cmd.WaitDelay = time.Second
	cmd.Stdin = bytes.NewReader(request)
	stdout, stderr := limitedBuffer{limit: 64 << 20}, limitedBuffer{limit: 64 << 10}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var response struct {
		engineIdentity
		Schema string          `json:"schema_version"`
		Status string          `json:"status"`
		Output json.RawMessage `json:"output"`
		Error  *string         `json:"error"`
	}
	if stdout.exceeded || stderr.exceeded {
		return fmt.Errorf("%s response exceeds the client limit", operation)
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &response); decodeErr != nil {
		if err != nil {
			return fmt.Errorf("%s invocation: %w: %s", operation, err, strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("decode %s response: %w", operation, decodeErr)
	}
	if response.Schema != "ptsip-engine-operation-result/v1" || response.engineIdentity != identity {
		return fmt.Errorf("%s response does not match its registered engine and contract", operation)
	}
	if response.Status != "PASS" || response.Error != nil || err != nil {
		if response.Error != nil {
			return fmt.Errorf("%s: %s", operation, *response.Error)
		}
		return fmt.Errorf("%s did not return PASS: %s (%v)", operation, response.Status, err)
	}
	if len(response.Output) == 0 {
		return fmt.Errorf("%s response is missing its output", operation)
	}
	if err := json.Unmarshal(response.Output, output); err != nil {
		return fmt.Errorf("decode %s output: %w", operation, err)
	}
	return nil
}

func (c Client) ToolVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, c.CLI, "--version")
	cmd.WaitDelay = time.Second
	stdout, stderr := limitedBuffer{limit: 1024}, limitedBuffer{limit: 4096}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("read installed Tool version: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	version, ok := strings.CutPrefix(strings.TrimSpace(stdout.String()), "PTSIP Tool ")
	if !ok || version == "" || strings.ContainsAny(version, " \t\r\n") {
		return "", fmt.Errorf("installed Tool returned an invalid version")
	}
	return version, nil
}

func (c Client) resolve(operation string) (engineIdentity, error) {
	var identity engineIdentity
	if c.CLI == "" || c.PackageRoot == "" || c.ScriptsDir == "" || operation == "" {
		return identity, fmt.Errorf("native CLI, package root, scripts directory and operation are required")
	}
	root, err := filepath.Abs(c.PackageRoot)
	if err != nil {
		return identity, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return identity, err
	}
	var index struct {
		Schema  string `json:"schema_version"`
		Entries []struct {
			Engine     string   `json:"engine_id"`
			Version    string   `json:"version"`
			Capability string   `json:"capability_id"`
			Resource   string   `json:"contract_ref"`
			Operations []string `json:"operations"`
		} `json:"official_engine_entries"`
	}
	if err := readJSON(root, "engines/index.json", &index); err != nil {
		return identity, err
	}
	if index.Schema != "ptsip-official-engine-index/v1" {
		return identity, fmt.Errorf("unsupported native engine index")
	}
	var contractRef string
	for _, entry := range index.Entries {
		if !slices.Contains(entry.Operations, operation) {
			continue
		}
		if contractRef != "" || entry.Engine == "" || entry.Version == "" || entry.Capability == "" {
			return identity, fmt.Errorf("missing or ambiguous engine identity for %s", operation)
		}
		relative, ok := strings.CutPrefix(entry.Resource, "src/ptsip/")
		if !ok || !strings.HasPrefix(relative, "engines/") {
			return identity, fmt.Errorf("invalid engine contract reference for %s", operation)
		}
		contractRef = relative
		identity.Engine, identity.EngineVersion, identity.Contract = entry.Engine, entry.Version, entry.Capability
	}
	if contractRef == "" {
		return identity, fmt.Errorf("unregistered native operation %s", operation)
	}
	var contract struct {
		Engine  string `json:"engine_id"`
		ID      string `json:"contract_id"`
		Version string `json:"version"`
	}
	if err := readJSON(root, contractRef, &contract); err != nil {
		return identity, err
	}
	if contract.Engine != identity.Engine || contract.ID != identity.Contract || contract.Version == "" {
		return identity, fmt.Errorf("registry and contract identity differ for %s", operation)
	}
	identity.ContractVersion = contract.Version
	return identity, nil
}

func readJSON(root, relative string, dst any) error {
	if !fs.ValidPath(relative) || relative == "." || strings.ContainsAny(relative, "\\:") {
		return fmt.Errorf("invalid engine resource path %q", relative)
	}
	file, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return err
	}
	inside, err := filepath.Rel(root, file)
	if err != nil || filepath.ToSlash(inside) != relative {
		return fmt.Errorf("engine resource uses a filesystem alias: %s", relative)
	}
	info, err := os.Stat(file)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular engine resource: %s", relative)
	}
	handle, err := os.Open(file)
	if err != nil {
		return err
	}
	defer handle.Close()
	const maximum = 4 << 20
	raw, err := io.ReadAll(io.LimitReader(handle, maximum+1))
	if err != nil {
		return err
	}
	if len(raw) > maximum {
		return fmt.Errorf("engine resource too large: %s", relative)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("decode %s: %w", relative, err)
	}
	return nil
}

type limitedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.buffer.Len() {
		b.exceeded = true
		return 0, fmt.Errorf("native output limit exceeded")
	}
	return b.buffer.Write(data)
}

func (b *limitedBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *limitedBuffer) String() string { return b.buffer.String() }
