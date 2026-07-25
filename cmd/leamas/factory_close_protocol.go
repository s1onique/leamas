package main

import (
	"errors"
	"strings"
)

// parseProtocolFlag parses the --protocol flag from args.
// It supports both --protocol v2 and --protocol=v2 formats.
// Scans the full argument list, removes every recognized protocol token,
// counts occurrences, and only returns after completing the scan.
// Rejects duplicates, missing values, and unsupported values.
func parseProtocolFlag(args []string) (protocol string, remaining []string, err error) {
	protocol = "v1" // default
	count := 0
	remaining = make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Handle --protocol=value
		if strings.HasPrefix(arg, "--protocol=") {
			count++
			if count > 1 {
				return "", nil, errors.New("--protocol specified multiple times")
			}
			value := strings.TrimPrefix(arg, "--protocol=")
			if value == "" {
				return "", nil, errors.New("--protocol requires a value")
			}
			if value != "v1" && value != "v2" {
				return "", nil, errors.New("unsupported protocol: " + value + " (supported: v1, v2)")
			}
			protocol = value
			// Skip this flag; do not add to remaining
			continue
		}

		// Handle --protocol value
		if arg == "--protocol" {
			count++
			if count > 1 {
				return "", nil, errors.New("--protocol specified multiple times")
			}
			if i+1 >= len(args) {
				return "", nil, errors.New("--protocol requires a value")
			}
			value := args[i+1]
			if value == "" {
				return "", nil, errors.New("--protocol requires a value")
			}
			if value != "v1" && value != "v2" {
				return "", nil, errors.New("unsupported protocol: " + value + " (supported: v1, v2)")
			}
			protocol = value
			i++ // Skip the value
			// Skip this flag and its value; do not add to remaining
			continue
		}

		// Regular argument, keep it
		remaining = append(remaining, arg)
	}

	return protocol, remaining, nil
}
