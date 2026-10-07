package repository_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestRepositoryStateResolvesExactMachineOwners(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	for domain, ref := range map[string]string{"developer_policy": "developer/policy/index.yaml", "developer_planning": "developer/planning/index.yaml", "project_profile": ".ptsip/profiles/main.ptsip.yaml", "policy_plan_binding": "developer/bindings/policy-plan-bindings.yaml", "agent_contract": "src/agent_contracts/bindings/current.yaml", "governance_source": "developer/policy/registries/governance-source-registry.yaml", "context_migration": "developer/planning/migrations/MPD-0012-agent-context-machine-migration.yaml"} {
		t.Run(domain, func(t *testing.T) {
			result, err, output := testrepo.CLI(t, binary, testrepo.Root(t), "repository-state", "resolve", "--domain", domain)
			if err != nil || result["ref"] != ref || result["authority"] != false {
				t.Fatalf("%#v %v\n%s", result, err, output)
			}
		})
	}
}

func TestRepositoryStateExcludesDefaultMarkdownContext(t *testing.T) {
	payload, err := testrepo.Open(testrepo.Root(t)).Read("developer/state/index.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := testrepo.Object{"prose_history_required": false, "status_markdown_required": false, "memory_markdown_required": false, "reference_markdown_required": false}
	if payload["authority"] != false || !reflect.DeepEqual(payload["default_agent_context"], want) {
		t.Fatal(payload)
	}
}

func TestUnknownRepositoryStateDomainFailsClosed(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	_, err, output := testrepo.CLI(t, binary, testrepo.Root(t), "repository-state", "resolve", "--domain", "unknown")
	if err == nil || !strings.Contains(output, "unknown state domain") {
		t.Fatalf("%v\n%s", err, output)
	}
}
