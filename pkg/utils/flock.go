package utils

import (
	"fmt"
	"os"
	"syscall"

	"github.com/golang/glog"
)

//FileLock used for multi-process paralism controll.
type FileLock struct {
	filePath string
	f        *os.File
}

//NewFileLock generate an file lock on given path.
func NewFileLock(filePath string) *FileLock {
	return &FileLock{
		filePath: filePath,
	}
}

//WaitLockEx waiting to get an exclusive lock on the file lock.
func (l *FileLock) WaitLockEx() error {
	glog.Debugf("flock try to wait lock exclusive %s", l.filePath)
	f, err := os.Open(l.filePath)
	if err != nil {
		return fmt.Errorf("flock try to wait lock %s failed for %s", l.filePath, err)
	}
	l.f = f
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock try to wait lock %s failed for %s", l.filePath, err)
	}
	glog.Debugf("flock try to wait lock exclusive %s success", l.filePath)
	return nil
}

//Unlock just unlock the file lock.
func (l *FileLock) Unlock() error {
	glog.Debugf("flock will unlock %s", l.filePath)

	defer l.f.Close()

	if err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN); err != nil {
		glog.Fatal("flock unlock %s failed for %s", l.filePath, err)
		return fmt.Errorf("flock unlock %s failed for %s", l.filePath, err)
	}
	glog.Debugf("flock unlock %s success", l.filePath)
	return nil
}
