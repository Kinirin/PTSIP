package machine

import "fmt"

func init() {
	RegisterOperations("pp-seed", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		if command != "pp-102" {
			return nil, fmt.Errorf("unregistered PP seed command: %s", command)
		}
		return r.SeedPP102Transition(BoolOption(options, "apply"))
	})
	RegisterOperations("policy-lifecycle", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "inspect":
			return r.InspectPolicy(args[0])
		case "preflight":
			return r.PreflightNewPolicy(options["--approval-ref"], options["--analysis-ref"], options["--group-id"])
		case "register":
			return r.RegisterPolicy(options["--approval-ref"], options["--analysis-ref"], options["--group-id"], options["--policy-file"])
		case "family-preflight":
			return r.PreflightFamilyPolicy(options["--policy-class"], options["--family"], options["--approval-ref"], options["--analysis-ref"], options["--group-id"])
		case "family-register":
			return r.RegisterFamilyPolicy(options["--policy-class"], options["--family"], options["--approval-ref"], options["--analysis-ref"], options["--group-id"], options["--policy-file"])
		case "status-preflight":
			return r.StatusPreflight(args[0], options["--approval-ref"])
		case "version-initial":
			return Object{"status": "READY", "version": InitialPolicyVersion()}, nil
		case "version-transition":
			return ResolvePolicyVersionTransition(options["--current-version"], options["--current-status"], options["--change-class"], options["--target-status"])
		}
		return nil, fmt.Errorf("unregistered policy lifecycle command: %s", command)
	})
	RegisterOperations("policy-responsibility", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		if command != "validate" {
			return nil, fmt.Errorf("unregistered responsibility command: %s", command)
		}
		return r.ValidateResponsibilityAnalysis(options["--analysis-ref"], BoolOption(options, "current-authority-lookup"))
	})
	RegisterOperations("policy-validator", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "validate":
			errors := r.ValidateDeveloperPolicy()
			status := "PASS"
			if len(errors) > 0 {
				status = "FAIL"
			}
			return Object{"status": status, "errors": errors}, nil
		case "contract":
			return r.ResolveNeutralCatalogContract(args[0])
		case "catalog-contracts":
			errors := r.ValidateNeutralCatalogContractRegistration()
			status := "PASS"
			if len(errors) > 0 {
				status = "FAIL"
			}
			return Object{"status": status, "errors": errors}, nil
		}
		return nil, fmt.Errorf("unregistered policy validation command: %s", command)
	})
	RegisterOperations("authority-family-migration", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		if command != "migrate" {
			return nil, fmt.Errorf("unregistered authority migration command: %s", command)
		}
		return r.MigrateAuthorityFamilyCatalog(BoolOption(options, "apply"))
	})
	RegisterOperations("policy-plan-consistency", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		if command != "verify" {
			return nil, fmt.Errorf("unregistered policy plan verification command: %s", command)
		}
		return r.VerifyPolicyPlanConsistency()
	})
}
