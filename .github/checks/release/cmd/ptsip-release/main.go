package main

import (
	"encoding/json"
	"fmt"
	"os"

	release "github.com/Kinirin/PTSIP/developer/automation/release"
)

func main() {
	result, err := release.CLI(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	e.SetEscapeHTML(false)
	if err = e.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
