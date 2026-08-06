// SPDX-License-Identifier: Apache-2.0

// Package closure - internal_helpers.go contains the small
// unexported helpers used by the resolver and the gate capture
// subsystem.

package closure

import (
	"crypto/rand"
	"encoding/hex"
)

// randomHexRunID returns a short opaque identifier suitable for
// RunID. It deliberately avoids ambient entropy sources.
func randomHexRunID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(buf[:])
}
