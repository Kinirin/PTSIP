package main

import "testing"

func TestBeta2ProfileCompatibilityOnlyAdmitsReadOnlyProfileCommands(t *testing.T) {
	for _, pair := range [][2]string{{"profile-registry", "current"}, {"profile-registry", "catalog"}, {"profile-registry", "contracts"}, {"profile-registry", "validate"}, {"pp", "release"}, {"pp", "commit"}, {"pp", "range"}} {
		if !beta2ProfileCommand(pair[0], pair[1]) {
			t.Fatalf("verification command refused: %v", pair)
		}
	}
	for _, pair := range [][2]string{{"pp", "reconcile"}, {"pp", "pre-commit"}, {"policy-resolver", "get"}, {"policy-validator", "validate"}, {"release", "draft"}, {"profile-registry", "write"}} {
		if beta2ProfileCommand(pair[0], pair[1]) {
			t.Fatalf("mutation or retired policy command admitted: %v", pair)
		}
	}
}
