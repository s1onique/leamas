//go:build unix || darwin || linux

package execution

import (
	"errors"
	"syscall"
)

// updateEnv updates or adds an environment variable.
func updateEnv(env []string, key, value string) []string {
	for i, e := range env {
		if len(e) >= len(key) && e[:len(key)] == key && e[len(key)] == '=' {
			env[i] = key + "=" + value
			return env
		}
	}
	return append(env, key+"="+value)
}

// isESRCH returns true if the error is "no such process".
func isESRCH(err error) bool {
	return err == nil || errors.Is(err, syscall.ESRCH)
}
