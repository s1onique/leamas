package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/version"
)

func handleVersion() {
	info := version.Get()

	// Check for --json flag
	if len(os.Args) >= 3 && os.Args[2] == "--json" {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to marshal version info: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Default: line-oriented output
	fmt.Printf("version: %s\n", info.Version)
	fmt.Printf("commit: %s\n", info.Commit)
	fmt.Printf("build_time: %s\n", info.BuildTime)
}
