package utils

import (
	"fmt"
	"os"
	"syscall"
)

const (
	ioctlFIFREEZE = 0xc0045877
	ioctlFITHAW   = 0xc0045878
)

// FIFreeze freeze the file system mount on dirPath
func FIFreeze(dirPath string) error {
	f, err := os.Open(dirPath)
	defer f.Close()

	if err != nil {
		return fmt.Errorf("open dir %s failed %s", dirPath, err)
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), ioctlFIFREEZE, uintptr(0)); errno != 0 {
		return fmt.Errorf("Freeze dir %s failed %d", dirPath, errno)
	}

	return nil
}

// FIThaw thaw the file system mount on dirPath
func FIThaw(dirPath string) error {
	f, err := os.Open(dirPath)
	defer f.Close()

	if err != nil {
		return fmt.Errorf("open dir %s failed %s", dirPath, err)
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), ioctlFITHAW, uintptr(0)); errno != 0 {
		return fmt.Errorf("thaw dir %s failed %d", dirPath, errno)
	}

	return nil
}
