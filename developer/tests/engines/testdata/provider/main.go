package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// This independently owned consumer provider exists only in isolated native tests.
func main() {
	var request struct {
		Operation string `json:"operation_id"`
		Input     struct {
			Target     string `json:"target"`
			Value      any    `json:"value"`
			Delay      int    `json:"delay_ms"`
			OutputSize int    `json:"output_size"`
			Fail       bool   `json:"fail"`
			Invalid    bool   `json:"invalid_response"`
		} `json:"input"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		os.Exit(2)
	}
	if request.Operation != "put" || request.Input.Target == "" || filepath.IsAbs(request.Input.Target) || strings.ContainsAny(request.Input.Target, `/\`) || request.Input.Target == ".." {
		os.Exit(2)
	}
	data, err := json.Marshal(request.Input.Value)
	if err != nil || os.WriteFile(request.Input.Target, data, 0600) != nil {
		os.Exit(2)
	}
	time.Sleep(time.Duration(request.Input.Delay) * time.Millisecond)
	if request.Input.Fail {
		fmt.Print(`{"status":"FAIL","output":null,"error":"fixture failure after observed effect"}`)
		os.Exit(3)
	}
	value := request.Input.Value
	if request.Input.Invalid {
		value = "invalid-typed-output"
	}
	output := map[string]any{"stored": value}
	if request.Input.OutputSize > 0 {
		output["extra"] = strings.Repeat("x", request.Input.OutputSize)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "output": output, "error": nil}); err != nil {
		os.Exit(4)
	}
}
