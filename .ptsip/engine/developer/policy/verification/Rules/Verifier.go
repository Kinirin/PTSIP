package rules

import authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"

func Check(root string, document authoring.Document, validate authoring.Validator) error {
	catalog, err := Load(root, validate)
	if err != nil {
		return err
	}
	return catalog.CheckRules(validate, document)
}

func Load(root string, validate authoring.Validator) (*authoring.Catalog, error) {
	catalog, err := authoring.Load(root)
	if err != nil {
		return nil, err
	}
	if err := catalog.CheckContracts(validate); err != nil {
		return nil, err
	}
	return catalog, nil
}
