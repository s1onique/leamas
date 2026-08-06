// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_cli_errorsas.go exposes a tiny
// adapter around errors.As so the helper file does not have
// to import the standard errors package directly. The
// indirection keeps the helper file under the LLM-
// friendliness 400-line threshold while preserving a single
// errors.As call site.

import "errors"

func isErrorsAs(err error, target any) bool {
	return errors.As(err, target)
}
