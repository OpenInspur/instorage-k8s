package host

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

//MultiPathUtil define the actions used to deal with multipath device.
type MultiPathUtil interface {
	FindMultiPathDevForDev(disk string) (string, error)
	FindSlaveDevicesOnMultiPath(disk string) ([]string, error)
	FindMultiPathDev(devs []string, maxRetry int, waitInterval int) (string, error)
	FlushMultiPathDev(dm string) error
	ResizeMultiPathDev(dm string) error
}

type multiPathUtil struct {
	resizeDelay int
}

// findDeviceForPath Find the underlaying disk for a linked path such as /dev/disk/by-path/XXXX or /dev/mapper/XXXX
// will return sdX or hdX etc, if /dev/sdX is passed in then sdX will be returned
func findDeviceForPath(path string, io ioHandler) (string, error) {
	devicePath, err := io.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	// if path /dev/hdX split into "", "dev", "hdX" then we will
	// return just the last part
	parts := strings.Split(devicePath, "/")
	if len(parts) == 3 && strings.HasPrefix(parts[1], "dev") {
		return parts[2], nil
	}

	return "", fmt.Errorf("Illegal path for device " + devicePath)
}

func (util *multiPathUtil) FindMultiPathDev(devs []string, maxRetry int, waitInterval int) (string, error) {
	for tryCount := 0; tryCount < maxRetry; tryCount++ {
		if tryCount != 0 {
			glog.Warningf("Find multipath device failed, just wait and then try again.")
			time.Sleep(time.Duration(waitInterval) * time.Second)
		}

		for _, dev := range devs {
			dm, err := util.FindMultiPathDevForDev(dev)
			if err != nil {
				glog.Warningf("Find multipath through %s failed: %s.", dev, err)
			} else if dm != "" {
				glog.Infof("Find multipath: %+v", dm)
				return dm, nil
			} else {
				glog.Warningf("Can not find multipath device through %s.", dev)
			}
		}
	}
	glog.Warningf("can not find multipath through dev path %s", devs)
	return "", fmt.Errorf("can not find multipath through dev path %s", devs)
}

// FindMultiPathDevForDev given a device name like /dev/sdx, find the devicemapper parent
func (util *multiPathUtil) FindMultiPathDevForDev(device string) (string, error) {
	io := &osIOHandler{}

	disk, err := findDeviceForPath(device, io)
	if err != nil {
		return "", err
	}

	sysPath := "/sys/block/"
	dirs, err := io.ReadDir(sysPath)
	if err != nil {
		return "", fmt.Errorf("read directory %s failed", sysPath)
	}

	for _, f := range dirs {
		name := f.Name()
		if strings.HasPrefix(name, "dm-") {
			if _, err1 := io.Lstat(sysPath + name + "/slaves/" + disk); err1 == nil {
				return "/dev/" + name, nil
			}
		}
	}

	return "", fmt.Errorf("no multipath device find for %s", device)
}

// FindSlaveDevicesOnMultiPath given a dm name like /dev/dm-1, find all devices
// which are managed by the devicemapper dm-1.
func (util *multiPathUtil) FindSlaveDevicesOnMultiPath(dm string) ([]string, error) {
	var devices []string
	io := &osIOHandler{}

	// Split path /dev/dm-1 into "", "dev", "dm-1"
	parts := strings.Split(dm, "/")
	if len(parts) != 3 || !strings.HasPrefix(parts[1], "dev") {
		return []string{}, fmt.Errorf("dm path invalid %s", dm)
	}

	disk := parts[2]
	slavesPath := path.Join("/sys/block/", disk, "/slaves/")
	files, err := io.ReadDir(slavesPath)
	if err != nil {
		return []string{}, fmt.Errorf("read directory failed %s", slavesPath)
	}

	for _, f := range files {
		devices = append(devices, path.Join("/dev/", f.Name()))
	}

	return devices, nil
}

// FlushMultiPathDev flush the given dm device.
// dm is the absolute path like "/dev/dm-2"
func (util *multiPathUtil) FlushMultiPathDev(dm string) error {
	_, err := utils.Execute("multipath", "-f", dm)
	return err
}

// ResizeMultiPathDev do a resize for the given dm device.
// dm is the absolute path like "/dev/dm-2"
func (util *multiPathUtil) ResizeMultiPathDev(dm string) error {
	//first do a reconfigure
	if _, err := utils.Execute("multipathd", "reconfigure"); err != nil {
		return err
	}

	//do a wait so that resize will not report timeout
	time.Sleep(time.Duration(util.resizeDelay) * time.Second)

	//then do a resize operation
	if _, err := utils.Execute("multipathd", "resize", "map", dm); err != nil {
		return err
	}

	return nil
}
