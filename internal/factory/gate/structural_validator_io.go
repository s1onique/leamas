// SPDX-License-Identifier: Apache-2.0

package gate

import "os"

// osReadDir and osReadFile are thin wrappers that exist solely to give
// the structural validator an io-equivalent surface that is testable
// in isolation. They are package-private and call directly into the
// standard library.
func osReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func osReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
