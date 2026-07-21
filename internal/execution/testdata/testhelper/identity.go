//go:build unix || darwin || linux

package main

import (
	"fmt"
	"runtime"
)

// helperSourceDigest is replaced at link time by the harness after it hashes
// every buildable Go source discovered from this package.
var helperSourceDigest = "unverified"

func runIdentity() {
	fmt.Printf("helper_source_digest=%s\n", helperSourceDigest)
	fmt.Printf("helper_build_go_version=%s\n", runtime.Version())
}
