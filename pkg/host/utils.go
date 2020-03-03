package host

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

type ioHandler interface {
	ReadDir(dirname string) ([]os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
	EvalSymlinks(path string) (string, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadFile(filename string) ([]byte, error)
}

type osIOHandler struct{}

func (h *osIOHandler) ReadDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(dirname)
}

func (h *osIOHandler) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (h *osIOHandler) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (h *osIOHandler) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

func (h *osIOHandler) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

func GetDiskUID(devicePath string) (string, error) {
	out, err := utils.Execute("/lib/udev/scsi_id", "--whitelisted", "--page=0x83", devicePath)
	if err != nil {
		return "", err
	}
	if len(out) == 0 {
		return "", fmt.Errorf("can not get scsi id for %s", devicePath)
	}

	return out, nil
}

func RemoveDisk(devicePath string) error {
	var devices []string
	io := &osIOHandler{}
	mpathUtil := &multiPathUtil{}

	// devicePath might be like /dev/mapper/mpathX. Find destination.
	dstPath, err := io.EvalSymlinks(devicePath)
	if err != nil {
		return err
	}

	if strings.HasPrefix(dstPath, "/dev/dm-") {
		// if multipath device then find the slaves
		if slaves, err := mpathUtil.FindSlaveDevicesOnMultiPath(dstPath); err == nil {
			devices = slaves
		} else {
			//nothing we can do if we can not find any slaves, system is in wrong status
			glog.Warningf("find slave device of %s failed", dstPath)
		}
	} else {
		// Add single device path to devices
		devices = append(devices, dstPath)
	}

	glog.Infof("DetachDisk devicePath: %v, dstPath: %v, devices: %v", devicePath, dstPath, devices)

	if strings.HasPrefix(dstPath, "/dev/dm-") {
		if err := mpathUtil.FlushMultiPathDev(dstPath); err != nil {
			glog.Errorf("flush multipath device %s failed %s", dstPath, err)
		}
	}

	var lastErr error
	for _, device := range devices {
		err := removeFromScsiSubsystem(device, io)
		if err != nil {
			glog.Errorf("fc: detachFCDisk failed. device: %v err: %v", device, err)
			lastErr = fmt.Errorf("fc: detachFCDisk failed. device: %v err: %v", device, err)
		}
	}

	if lastErr != nil {
		glog.Errorf("fc: last error occurred during detach disk:\n%v", lastErr)
		return lastErr
	}
	return nil
}

// Removes a scsi device based upon /dev/sdX name
func removeFromScsiSubsystem(devicePath string, io ioHandler) error {
	if !strings.HasPrefix(devicePath, "/dev/") {
		return fmt.Errorf("invalid device path: %s, path should like /dev/sdX", devicePath)
	}

	pathParts := strings.Split(devicePath, "/")
	deviceName := pathParts[len(pathParts)-1]

	fileName := "/sys/block/" + deviceName + "/device/delete"
	glog.Infof("remove device from scsi-subsystem: path: %s", fileName)

	data := []byte("1")
	return io.WriteFile(fileName, data, 0666)
}

func RescanBlockDevice(devicePath string, io ioHandler) error {
	if !strings.HasPrefix(devicePath, "/dev/") {
		return fmt.Errorf("invalid device path: %s, path should like /dev/sdX", devicePath)
	}

	pathParts := strings.Split(devicePath, "/")
	deviceName := pathParts[len(pathParts)-1]

	fileName := "/sys/block/" + deviceName + "/device/rescan"
	glog.Debugf("rescan the block device path: %s", fileName)

	data := []byte("1")
	return io.WriteFile(fileName, data, 0666)
}
