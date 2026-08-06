// SPDX-License-Identifier: Apache-2.0

package closure

// r2cr4_test_helpers.go provides tiny wrappers around os
// helpers so the r2cr4 test file can avoid the import and
// keep its own dependency surface focused on the r2cr4
// subject.

import "os"

// osStatImpl wraps os.Stat so the r2cr4 test file can
// delegate stat checks without an explicit "os" import.
func osStatImpl(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// osIsNotExistImpl wraps os.IsNotExist so the r2cr4 test
// file can distinguish absent-path errors without an explicit
// "os" import.
func osIsNotExistImpl(err error) bool {
	return os.IsNotExist(err)
}
