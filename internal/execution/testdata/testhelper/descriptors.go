//go:build unix || darwin || linux

package main

import "fmt"

type descriptorIdentity struct {
	Target string `json:"target"`
	Dev    uint64 `json:"dev"`
	Ino    uint64 `json:"ino"`
}

type descriptorSet struct {
	FD1 descriptorIdentity `json:"fd1"`
	FD2 descriptorIdentity `json:"fd2"`
}

func formatDescriptorEvidence(descriptors descriptorSet) string {
	return fmt.Sprintf(
		"fd1_target=%s\nfd1_dev=%d\nfd1_ino=%d\n"+
			"fd2_target=%s\nfd2_dev=%d\nfd2_ino=%d\n",
		descriptors.FD1.Target, descriptors.FD1.Dev, descriptors.FD1.Ino,
		descriptors.FD2.Target, descriptors.FD2.Dev, descriptors.FD2.Ino,
	)
}
