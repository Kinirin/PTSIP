package update

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/authoring"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

// Run dispatches the explicit authoring operation. The surrounding trusted
// runtime supplies an authorizer through its owner-controlled integration.
func Run(ctx context.Context, root, input string, authorize Authorizer) (any, int) {
	reject := func(err error) (any, int) {
		return map[string]any{"status": "FAIL", "problem": err.Error(),
			"execution_authorized": false, "lifecycle_transition_authorized": false, "normative_approval": false}, 2
	}
	if input == "" {
		return reject(fmt.Errorf("POLICY_UPDATE_REQUIRES_EXPLICIT_JSON_INPUT"))
	}
	if authorize == nil {
		return reject(fmt.Errorf("POLICY_UPDATE_APPROVAL_PROVIDER_UNAVAILABLE"))
	}
	var raw []byte
	var err error
	if input == "-" {
		raw, err = io.ReadAll(io.LimitReader(os.Stdin, source.MaxBytes+1))
	} else {
		name, pathErr := authoring.LocalFile(root, input)
		if pathErr != nil {
			return reject(pathErr)
		}
		snapshot, fileErr := readSnapshot(name)
		raw, err = snapshot.data, fileErr
	}
	if err != nil {
		return reject(err)
	}
	if len(raw) > source.MaxBytes {
		return reject(fmt.Errorf("POLICY_UPDATE_INPUT_TOO_LARGE"))
	}
	if _, err := jsondoc.Decode(raw); err != nil {
		return reject(err)
	}
	var update Request
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&update); err != nil {
		return reject(err)
	}
	result, err := Apply(ctx, root, update, authorize)
	if err != nil {
		return reject(err)
	}
	return map[string]any{"status": result.Status, "result": result,
		"execution_authorized": false, "lifecycle_transition_authorized": false, "normative_approval": false}, 0
}
