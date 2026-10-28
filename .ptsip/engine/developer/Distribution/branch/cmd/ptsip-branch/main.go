package main

import (
	branch "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/branch"
	"os"
)

func main() { os.Exit(branch.CLI(os.Args[1:], os.Stdout)) }
