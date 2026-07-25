// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
	"strings"
)

// hashBlob writes exact bytes to the object database without touching an index.
func hashBlob(ctx context.Context, git gitClient, repoRoot string, data []byte) (string, error) {
	result := git.RunWithStdin(ctx, repoRoot, string(data), "hash-object", "-w", "--stdin")
	if result.ExitCode != 0 || result.Err != nil {
		return "", fmt.Errorf("git hash-object failed: %s", gitFailureDetail(result))
	}
	return strings.TrimSpace(string(result.Stdout)), nil
}
