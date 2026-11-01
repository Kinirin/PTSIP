// Explicit read-only authoring helper. This command is not registered as a
// validator, policy resolver operation, runtime hook, or approval mechanism.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/review"
)

type repeated []string

func (r *repeated) String() string         { return fmt.Sprint([]string(*r)) }
func (r *repeated) Set(value string) error { *r = append(*r, value); return nil }

func main() {
	root := flag.String("root", ".", "repository root")
	verify := flag.String("verify", "", "repository-relative prior packet; compare captured file hashes only")
	var targets, sources repeated
	flag.Var(&targets, "target", "exact developer policy path (repeatable)")
	flag.Var(&sources, "source", "explicit historical FILE#/json/pointer (repeatable)")
	flag.Parse()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	fail := func(err error) {
		_ = encoder.Encode(map[string]any{"status": "FAIL", "problem": err.Error(), "mutation_authorized": false})
		os.Exit(2)
	}
	if flag.NArg() != 0 {
		fail(fmt.Errorf("UNEXPECTED_POSITIONAL_ARGUMENT"))
	}
	absolute, err := filepath.Abs(*root)
	if err != nil {
		fail(err)
	}
	if *verify != "" {
		if len(targets)+len(sources) != 0 {
			fail(fmt.Errorf("VERIFY_REQUIRES_SNAPSHOT_ONLY"))
		}
		packet, err := review.ReadSnapshot(absolute, *verify)
		if err != nil {
			fail(err)
		}
		if err := packet.Verify(absolute); err != nil {
			fail(err)
		}
		err = encoder.Encode(map[string]any{"status": "CAPTURED_FILES_UNCHANGED", "file_count": len(packet.Files), "mutation_authorized": false, "semantic_equivalence": "NOT_ASSERTED"})
	} else {
		packet, buildErr := review.Build(absolute, targets, sources)
		if buildErr != nil {
			fail(buildErr)
		}
		err = encoder.Encode(packet)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
