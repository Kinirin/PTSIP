package rootfamily

import "testing"

func TestRegisteredCIPolicyConsumers(t *testing.T) {
	for _, entry := range contract(t).SmokeCases {
		t.Run(entry.ID, func(t *testing.T) {
			result := python(t, entry.Command...)
			if result["mutation_performed"] != entry.ExpectedMutation {
				t.Fatal("CI policy consumer changed its mutation boundary", result)
			}
			if result["status"] != "READY" && result["status"] != "BLOCKED" {
				t.Fatal("CI policy consumer did not produce a resolved simulation", result)
			}
		})
	}
}
