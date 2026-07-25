// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"fmt"
	"io/fs"
	"os"
)

// writeExclusiveRegular creates path without replacing or following anything
// already present at its final component. It persists exact bytes before
// returning and accepts only a regular file at both descriptor and path.
func writeExclusiveRegular(path string, data []byte, mode fs.FileMode) (err error) {
	if _, statErr := os.Lstat(path); statErr == nil {
		return fmt.Errorf("reserved evidence path already exists: %s", path)
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect reserved evidence path: %w", statErr)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create reserved evidence file: %w", err)
	}
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("created evidence path is not a regular file")
	}
	written, err := file.Write(data)
	if err != nil {
		return fmt.Errorf("write reserved evidence file: %w", err)
	}
	if written != len(data) {
		return fmt.Errorf("short write to reserved evidence file: wrote %d of %d bytes", written, len(data))
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync reserved evidence file: %w", err)
	}
	if err := file.Close(); err != nil {
		file = nil
		return fmt.Errorf("close reserved evidence file: %w", err)
	}
	file = nil
	pathInfo, err := os.Lstat(path)
	if err != nil || !pathInfo.Mode().IsRegular() {
		return fmt.Errorf("reserved evidence path is not a regular file after close")
	}
	return nil
}
