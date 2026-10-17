package schema_check

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ValidateInstance asks the installed PTSIP VALIDATION engine to apply an
// inline JSON Schema. A valid/invalid instance is separate from invocation
// failure and from PTSIP policy conformance.
func ValidateInstance(schema any, instance any) (bool, error) {
	engine, err := installed()
	if err != nil {
		return false, err
	}
	request := map[string]any{
		"engine_id": engine.EngineID, "engine_version": engine.EngineVersion,
		"contract_id": engine.ContractID, "contract_version": engine.ContractVersion,
		"operation_id": "validation.schema",
		"input":        map[string]any{"schema": schema, "instance": instance},
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, engine.Executable)
	cmd.Stdin = bytes.NewReader(raw)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("engine execution: %w: %s %s", err, stdout.String(), stderr.String())
	}
	var response struct {
		Status              string `json:"status"`
		Error               any    `json:"error"`
		NormativeAuthority  bool   `json:"normative_authority"`
		ExecutionAuthorized bool   `json:"execution_authorized"`
		Output              struct {
			Valid   bool   `json:"valid"`
			Verdict string `json:"ptsip_conformance_verdict"`
		} `json:"output"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return false, fmt.Errorf("invalid engine JSON response: %w", err)
	}
	if response.Status != "PASS" || response.Error != nil || response.NormativeAuthority ||
		response.ExecutionAuthorized || response.Output.Verdict != "NOT_ASSERTED" {
		return false, fmt.Errorf("untrusted engine result: %s", strings.TrimSpace(stdout.String()))
	}
	return response.Output.Valid, nil
}
