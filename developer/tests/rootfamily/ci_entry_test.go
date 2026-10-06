package rootfamily

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func TestRegisteredCIPolicyConsumers(t *testing.T) {
	for _, entry := range contract(t).SmokeCases {
		t.Run(entry.ID, func(t *testing.T) {
			command := exec.Command(entry.Command[0], entry.Command[1:]...)
			command.Dir = repository(t)
			command.Env = os.Environ()
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("registered consumer: %v\n%s", err, output)
			}
			var result map[string]any
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("consumer output: %v\n%s", err, output)
			}
			if result["mutation_performed"] != entry.ExpectedMutation {
				t.Fatal("CI policy consumer changed its mutation boundary", result)
			}
			if result["status"] != "READY" && result["status"] != "BLOCKED" {
				t.Fatal("CI policy consumer did not produce a resolved simulation", result)
			}
		})
	}
}
