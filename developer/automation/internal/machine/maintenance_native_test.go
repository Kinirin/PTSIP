package machine

import (
	"reflect"
	"testing"
)

func TestCurrentDependencyGateDispatchesTheRegisteredNativeImplementation(t *testing.T) {
	repo, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AdmitCommand([]string{"current-dependency-gate", "validate"}, map[string]string{}); err != nil {
		t.Fatal(err)
	}
	want, err := repo.DispatchOperation("dependency-gate", "verify", map[string]string{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.DispatchOperation("current-dependency-gate", "validate", map[string]string{}, nil)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("native dependency result differs: got=%#v want=%#v error=%v", got, want, err)
	}
}

func TestWU02ControlContextSeparatesQueryFromExecutionValidation(t *testing.T) {
	repo, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		branch, status string
		validated      bool
	}{{"", "PASS", false}, {"dev/0.3.8", "PASS", true}, {"unregistered-branch", "FAIL", true}} {
		options := map[string]string{}
		if test.branch != "" {
			options["--execution-branch"] = test.branch
		}
		if err := repo.AdmitCommand([]string{"wu02", "control-context"}, options); err != nil {
			t.Fatal(err)
		}
		result, err := repo.DispatchOperation("wu02", "control-context", options, nil)
		context := Map(result)
		if err != nil || context["status"] != test.status || context["validation_performed"] != test.validated || context["control_branch"] != "dev/0.3.8" {
			t.Fatalf("branch=%q, result=%#v, error=%v", test.branch, result, err)
		}
	}
}

func TestCleanupSimulationTransportDoesNotWeakenMutationGates(t *testing.T) {
	repo, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		command string
		want    int
	}{{"simulate", 0}, {"apply", 2}} {
		code, err := repo.CommandStatusExitCode([]string{"markdown-cleanup", test.command}, "BLOCKED")
		if err != nil || code != test.want {
			t.Fatalf("%s exit=%d want=%d error=%v", test.command, code, test.want, err)
		}
	}
}
