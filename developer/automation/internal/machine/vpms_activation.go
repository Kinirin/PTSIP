package machine

import (
	"os"
)

func (r *Repository) ActivationRecord() (Object, error) {
	path, err := r.Path(activationScopeRecord)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	record, err := r.Read(activationScopeRecord)
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/vpms-runtime-activation.schema.json", record); err != nil {
		return nil, err
	}
	for relative, expected := range Map(record["prior_records"]) {
		content, err := r.ReadSource(relative)
		if err != nil {
			return nil, err
		}
		if LFDigest(content) != Text(expected) {
			return nil, Fail("ACTIVATION_PRIOR_PROVENANCE_CHANGED", relative)
		}
	}
	return record, nil
}
