package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller"
	test "github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/test"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "mode runner command required")
		os.Exit(2)
	}
	modeCommand := os.Args[1]
	f := flag.NewFlagSet(modeCommand, flag.ExitOnError)
	root := f.String("root", ".", "repository root")
	mode := f.String("mode", "", "registered mode")
	targets := f.String("targets", "", "registered selection")
	optional := f.String("optional-platforms", "", "opt-in hosts")
	component := f.String("component", "", "selected component")
	platform := f.String("platform", "", "selected host")
	selection := f.String("selection", ".artifacts/release/selection.json", "derived selection")
	input := f.String("input", ".artifacts/release/verified", "verified original component artifacts")
	f.Parse(os.Args[2:])
	abs, err := filepath.Abs(*root)
	if err != nil {
		fail(err)
	}
	var result any
	switch modeCommand {
	case "plan":
		p, e := test.Plan(abs, *mode, *targets, *optional)
		if e != nil {
			fail(e)
		}
		if err := release.WriteJSON(filepath.Join(abs, *selection), p.Selection); err != nil {
			fail(err)
		}
		if os.Getenv("GITHUB_OUTPUT") != "" {
			file, e := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_APPEND|os.O_WRONLY, 0600)
			if e != nil {
				fail(e)
			}
			fmt.Fprintf(file, "matrix=%s\njoin_matrix=%s\njoin_count=%d\nlightweight_matrix=%s\nlightweight_count=%d\npython=%s\nfetch_depth=%d\n", test.Encode(p.Matrix), test.Encode(p.Join), len(p.Join.Include), test.Encode(p.Lightweight), len(p.Lightweight.Include), p.Python, p.FetchDepth)
			file.Close()
		}
		result = p
	case "validate":
		if err := test.Validate(abs); err != nil {
			fail(err)
		}
		result = map[string]string{"status": "PASS"}
	case "environment":
		key, paths, e := test.Environment(abs, *component, *platform)
		if e != nil {
			fail(e)
		}
		if os.Getenv("GITHUB_OUTPUT") != "" {
			file, e := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_APPEND|os.O_WRONLY, 0600)
			if e != nil {
				fail(e)
			}
			prefix := key
			if strings.HasPrefix(key, "release-test-v2-") {
				prefix = key[:len(key)-40]
			}
			fmt.Fprintf(file, "cache_key=%s\ncache_restore_prefix=%s\ncache_paths<<PTSIP_PATHS\n%s\nPTSIP_PATHS\n", key, prefix, paths)
			file.Close()
		}
		result = map[string]string{"cache_key": key, "cache_paths": paths}
	case "execute":
		s, e := test.ReadSelection(abs, *selection)
		if e != nil {
			fail(e)
		}
		v, e := test.Execute(abs, s, *component, *platform)
		if e != nil {
			fail(e)
		}
		result = v
	case "join":
		s, e := test.ReadSelection(abs, *selection)
		if e != nil {
			fail(e)
		}
		v, e := test.VerifyPackageJoin(abs, s, *input, *platform)
		if e != nil {
			fail(e)
		}
		result = v
    case "lightweight":
        s,e:=test.ReadSelection(abs,*selection)
        if e!=nil{fail(e)}
        v,e:=test.VerifyLightweightPackageInstall(abs,s,*input,*platform)
        if e!=nil{fail(e)}
        result=v
	default:
		fail(fmt.Errorf("unregistered mode runner command"))
	}
	json.NewEncoder(os.Stdout).Encode(result)
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
