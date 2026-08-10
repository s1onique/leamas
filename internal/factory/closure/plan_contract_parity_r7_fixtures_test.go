// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_contract_parity_r7_fixtures_test.go
// is the B2-R7 fixture data for the parity matrix. The
// matrix lives in plan_contract_parity_r7_test.go; this
// file owns only the fixture builders so each file stays
// under the LLM-friendly 400-line threshold.
package closure

import ()

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
