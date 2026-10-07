package machine

import "fmt"

func init() {
	RegisterOperations("policy-plan-binding", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		query := Object{}
		for option, key := range map[string]string{"--binding-id": "binding_id", "--policy": "policy_ref", "--resolved-plan-id": "resolved_plan_id", "--plan-file-id": "plan_file_id", "--plan": "plan_ref", "--plan-ref": "plan_ref", "--version": "version", "--revision": "revision", "--from-plan-ref": "from_plan_ref", "--to-plan-ref": "to_plan_ref"} {
			if value := options[option]; value != "" {
				query[key] = value
			}
		}
		switch command {
		case "resolve":
			return r.ResolveBindings(query)
		case "create":
			return r.CreateBinding(Text(query["policy_ref"]))
		case "link":
			return r.LinkPlan(query)
		case "move":
			return r.MovePlanRef(query)
		case "reconcile":
			return r.ReconcileBindings(BoolOption(options, "apply"))
		case "track":
			if len(args) != 1 {
				return nil, fmt.Errorf("binding_id required")
			}
			return r.TrackPlanRef(args[0], BoolOption(options, "apply"))
		case "validate":
			snap, err := r.LoadBindingRegistry(true)
			if err != nil {
				return nil, err
			}
			failures := r.ValidateBindingRegistry(snap.Payload)
			status := "PASS"
			if len(failures) > 0 {
				status = "FAIL"
			}
			return Object{"status": status, "failures": failures}, nil
		}
		return nil, fmt.Errorf("unregistered policy-plan-binding command")
	})
	RegisterOperations("planning", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "validate":
			failures := r.ValidatePlanning()
			status := "PASS"
			if len(failures) > 0 {
				status = "FAIL"
			}
			return Object{"status": status, "failures": failures}, nil
		case "reconcile":
			return r.ReconcilePlanning(options["--current-branch"], options["--merged-branch"], BoolOption(options, "apply"))
		case "merge-leaf":
			if len(args) != 1 {
				return nil, fmt.Errorf("leaf branch required")
			}
			return r.MergePlanningLeaf(args[0], options["--message"])
		case "finalize-stage":
			if len(args) != 2 {
				return nil, fmt.Errorf("planning path and stage_id required")
			}
			return r.FinalizePlanningStage(args[0], args[1])
		case "finalize-extension":
			if len(args) != 1 {
				return nil, fmt.Errorf("extension path required")
			}
			return r.FinalizeExtension(args[0])
		}
		return nil, fmt.Errorf("unregistered planning command")
	})
	RegisterOperations("pp", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "delta":
			return r.ComparePP(options["--base"], options["--candidate"], BoolOption(options, "staged"))
		case "reconcile":
			return r.ReconcilePP(BoolOption(options, "apply"))
		case "pre-commit":
			return r.VerifyPPPreCommit()
		case "commit":
			result, err := r.VerifyPPCommit(options["--commit"])
			if err != nil {
				return nil, err
			}
			return Object{"status": "PASS", "verified_commit_count": 1, "commits": []any{result}}, nil
		case "range":
			return r.VerifyPPRange(options["--base"], options["--head"])
		case "release":
			return r.VerifyPPRelease(options["--sha"])
		}
		return nil, fmt.Errorf("unregistered PP command")
	})
}
