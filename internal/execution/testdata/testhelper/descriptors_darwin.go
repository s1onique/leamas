//go:build darwin

package main

import (
	"fmt"
	"os"
	"syscall"
)

func captureDescriptorSet() (descriptorSet, error) {
	fd1, err := captureDescriptor(1)
	if err != nil {
		return descriptorSet{}, err
	}
	fd2, err := captureDescriptor(2)
	if err != nil {
		return descriptorSet{}, err
	}
	return descriptorSet{FD1: fd1, FD2: fd2}, nil
}

func captureDescriptor(fd int) (descriptorIdentity, error) {
	target, err := os.Readlink(fmt.Sprintf("/dev/fd/%d", fd))
	if err != nil {
		target = fmt.Sprintf("fd:%d", fd)
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		return descriptorIdentity{}, fmt.Errorf("fstat fd %d: %w", fd, err)
	}
	return descriptorIdentity{
		Target: target,
		Dev:    uint64(stat.Dev),
		Ino:    stat.Ino,
	}, nil
}
