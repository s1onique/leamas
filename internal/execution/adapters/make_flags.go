// Package adapters provides typed adapters for the execution gateway.
package adapters

import (
	"os"
	"strconv"
	"strings"
)

// clampJobs ensures jobs flags are bounded and not duplicated.
func (a *MakeAdapter) clampJobs(args []string) []string {
	result := make([]string, 0, len(args)+1)
	hasJobs := false
	hasJobsFlag := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Handle -j N (spaced form) FIRST so it's not caught by HasPrefix below
		if arg == "-j" {
			if i+1 < len(args) {
				val := args[i+1]
				if hasJobsFlag {
					// Skip both -j and its value
					i++
					continue
				}
				hasJobsFlag = true
				if n, err := strconv.ParseInt(val, 10, 64); err == nil && (n == 0 || n > a.limit) {
					result = append(result, "-j"+strconv.FormatInt(a.limit, 10))
				} else {
					result = append(result, "-j"+val)
				}
				i++ // skip the value
			} else {
				// Bare -j with no following arg
				if !hasJobsFlag {
					hasJobsFlag = true
					result = append(result, "-j"+strconv.FormatInt(a.limit, 10))
				}
			}
			continue
		}

		// Handle --jobs N (spaced form)
		if arg == "--jobs" {
			if i+1 < len(args) {
				val := args[i+1]
				if hasJobs {
					// Skip both --jobs and its value
					i++
					continue
				}
				hasJobs = true
				if n, err := strconv.ParseInt(val, 10, 64); err == nil && (n == 0 || n > a.limit) {
					result = append(result, "--jobs="+strconv.FormatInt(a.limit, 10))
				} else {
					result = append(result, "--jobs="+val)
				}
			} else {
				// Bare --jobs with no following arg
				if !hasJobs {
					hasJobs = true
					result = append(result, "--jobs="+strconv.FormatInt(a.limit, 10))
				}
			}
			i++
			continue
		}

		// Handle -jN or -j=N (must be exact token, not --jobserver-auth)
		if strings.HasPrefix(arg, "-j") && arg != "-j" {
			val := strings.TrimPrefix(arg, "-j")
			if strings.HasPrefix(val, "=") {
				numVal := strings.TrimPrefix(val, "=")
				if hasJobsFlag {
					continue // Skip duplicate
				}
				hasJobsFlag = true
				if numVal == "" {
					result = append(result, "-j"+strconv.FormatInt(a.limit, 10))
				} else if n, err := strconv.ParseInt(numVal, 10, 64); err == nil && (n == 0 || n > a.limit) {
					result = append(result, "-j"+strconv.FormatInt(a.limit, 10))
				} else {
					result = append(result, arg)
				}
			} else if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				if hasJobsFlag {
					continue // Skip duplicate
				}
				hasJobsFlag = true
				if n == 0 || n > a.limit {
					result = append(result, "-j"+strconv.FormatInt(a.limit, 10))
				} else {
					result = append(result, arg)
				}
			} else {
				// Not a number, skip
			}
			continue
		}

		// Handle --jobs=N (must be exact token, not --jobserver-auth=)
		if strings.HasPrefix(arg, "--jobs=") {
			val := strings.TrimPrefix(arg, "--jobs=")
			if hasJobs {
				continue // Skip duplicate
			}
			hasJobs = true
			if val == "" {
				result = append(result, "--jobs="+strconv.FormatInt(a.limit, 10))
			} else if n, err := strconv.ParseInt(val, 10, 64); err == nil && (n == 0 || n > a.limit) {
				result = append(result, "--jobs="+strconv.FormatInt(a.limit, 10))
			} else {
				result = append(result, arg)
			}
			continue
		}

		// Not a job flag - copy verbatim
		result = append(result, arg)
	}

	if !hasJobs && !hasJobsFlag {
		result = append(result, "-j"+strconv.FormatInt(a.limit, 10))
	}
	return result
}

// sanitizeMakeFlags removes or overrides unbounded parallelism from MAKEFLAGS and MFLAGS.
func (a *MakeAdapter) sanitizeMakeFlags(env []string) []string {
	if env == nil {
		env = []string{}
	}

	result := make([]string, 0, len(env)+2)
	hasMakeflags, hasMflags := false, false

	for _, e := range env {
		if strings.HasPrefix(e, "MAKEFLAGS=") {
			hasMakeflags = true
			result = append(result, a.clampMakeflagsVar(e))
		} else if strings.HasPrefix(e, "MFLAGS=") {
			hasMflags = true
			result = append(result, a.clampMakeflagsVar(e))
		} else {
			result = append(result, e)
		}
	}

	if !hasMakeflags {
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "MAKEFLAGS=") {
				result = append(result, a.clampMakeflagsVar(e))
				break
			}
		}
	}
	if !hasMflags {
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "MFLAGS=") {
				result = append(result, a.clampMakeflagsVar(e))
				break
			}
		}
	}
	return result
}

// clampMakeflagsVar clamps job parallelism in MAKEFLAGS or MFLAGS.
func (a *MakeAdapter) clampMakeflagsVar(e string) string {
	prefix := "MAKEFLAGS="
	if strings.HasPrefix(e, "MFLAGS=") {
		prefix = "MFLAGS="
	}
	return prefix + a.clampJobsInString(strings.TrimPrefix(e, prefix))
}

// clampJobsInString clamps job flags in a string like MAKEFLAGS value.
// Uses token-based parsing to recognize only job-related options.
// All other tokens (including long options like --no-print-directory,
// --output-sync, --jobserver-auth) are copied byte-for-byte.
func (a *MakeAdapter) clampJobsInString(s string) string {
	result := make([]byte, 0, len(s))
	i := 0

	for i < len(s) {
		// Check for job flag tokens (-j, --jobs)
		if s[i] == '-' {
			// Try short -j form (must be exact, not --jobserver-auth)
			if i+1 < len(s) && s[i+1] == 'j' {
				// Check what follows -j: must be end, =, space, or digit
				rest := s[i+2:]
				if len(rest) == 0 || rest[0] == '=' || rest[0] == ' ' || (rest[0] >= '0' && rest[0] <= '9') {
					result = append(result, "-j"...)
					i += 2
					if i < len(s) && s[i] == '=' {
						i++
					}
					// Parse number and clamp (handles digit case)
					if i < len(s) && s[i] >= '0' && s[i] <= '9' {
						result, i = a.parseAndClampNumber(s, i, result)
					} else if i < len(s) && s[i] == ' ' {
						// Spaced form -j N, skip the space
						i++
						if i < len(s) && s[i] >= '0' && s[i] <= '9' {
							result, i = a.parseAndClampNumber(s, i, result)
						} else {
							// Bare -j with trailing space and no number
							result = strconv.AppendInt(result, a.limit, 10)
						}
					} else {
						// Bare -j with no number, clamp to limit
						result = strconv.AppendInt(result, a.limit, 10)
					}
					continue
				}
			}

			// Try long --jobs form (must be exact token with = or space, not --jobserver-auth)
			if i+len("--jobs") <= len(s) {
				token := s[i : i+len("--jobs")]
				if token == "--jobs" {
					// Check what follows: =, space, or end
					rest := ""
					if i+len("--jobs") < len(s) {
						rest = s[i+len("--jobs"):]
					}
					if len(rest) == 0 || rest[0] == '=' || rest[0] == ' ' {
						result = append(result, "--jobs="...)
						i += len("--jobs")
						if i < len(s) && s[i] == '=' {
							i++
						} else if i < len(s) && s[i] == ' ' {
							i++
						}
						// Parse number and clamp
						if i >= len(s) || s[i] < '0' || s[i] > '9' {
							// Bare --jobs, clamp to limit
							result = strconv.AppendInt(result, a.limit, 10)
						} else {
							result, i = a.parseAndClampNumber(s, i, result)
						}
						continue
					}
				}
			}
		}

		// Not a job flag - copy byte verbatim
		result = append(result, s[i])
		i++
	}
	return string(result)
}

// parseAndClampNumber parses a number from s starting at i, clamps it to limit,
// and appends the result to result. Returns updated result and i.
func (a *MakeAdapter) parseAndClampNumber(s string, i int, result []byte) ([]byte, int) {
	numStart := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	numStr := s[numStart:i]
	if n, err := strconv.ParseInt(numStr, 10, 64); err == nil && (n == 0 || n > a.limit) {
		result = strconv.AppendInt(result, a.limit, 10)
	} else if numStr != "" {
		result = append(result, s[numStart:i]...)
	}
	return result, i
}

// HasUnboundedJobs checks if args contain unbounded -j flag.
func (a *MakeAdapter) HasUnboundedJobs(args []string) bool {
	for _, arg := range args {
		if arg == "-j" || arg == "--jobs" {
			return true
		}
		if strings.HasPrefix(arg, "-j0") {
			return true
		}
		if strings.HasPrefix(arg, "--jobs=0") || strings.HasPrefix(arg, "--jobs") && len(arg) == 7 {
			return true
		}
	}
	return false
}
