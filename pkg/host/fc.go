package host

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/storage"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

const (
	byPath = "/dev/disk/by-path/"
	byID   = "/dev/disk/by-id/"
)

//FCUtil define the actions used when host is in FC environment.
type FCUtil struct {
	hostCfg utils.HostCfg
}

//NewFCUtil create a FCUtil for use.
func NewFCUtil(cfg utils.HostCfg) *FCUtil {
	return &FCUtil{
		hostCfg: cfg,
	}
}

// rescan scsi bus
func (u *FCUtil) scsiHostRescan(io ioHandler) error {
	scsiPath := "/sys/class/scsi_host"

	if dirs, err := io.ReadDir(scsiPath); err == nil {
		var lastErr error
		for _, f := range dirs {
			// echo "- - -" > /sys/class/scsi_host/hostX/scan
			name := path.Join(scsiPath, f.Name(), "scan")
			data := []byte("- - -")
			lastErr = io.WriteFile(name, data, 0666)
		}
		if lastErr != nil {
			glog.Errorf("not all host scan success")
		}

		return lastErr
	} else {
		glog.Errorf("Read %s directory failed", scsiPath)
		return fmt.Errorf("read scsi path failed")
	}
}

// given a wwn and lun, find the device and associated devicemapper parent
func (u *FCUtil) findDisks(wwns []string, luns []string, io ioHandler) ([]string, error) {
	availableDisk := []string{}
	fcPathSuffix := []string{}
	for idx, wwn := range wwns {
		fcPathSuffix = append(fcPathSuffix, fmt.Sprintf("-fc-0x%s-lun-%s", strings.ToLower(wwn), luns[idx]))
	}

	basePath := "/dev/disk/by-path"
	dirs, err := io.ReadDir(basePath)
	if err != nil {
		return availableDisk, fmt.Errorf("read by-path dir failed")
	}

	var lastErr error
	for _, f := range dirs {
		realPath := f.Name()
		for _, suffix := range fcPathSuffix {
			if strings.HasSuffix(realPath, suffix) {
				//read path match the wanted suffix, it's a wanted path
				disk, err := io.EvalSymlinks(path.Join(basePath, realPath))
				if err == nil {
					availableDisk = append(availableDisk, disk)
				} else {
					lastErr = err
				}
			}
		}
	}

	if len(availableDisk) == 0 {
		return availableDisk, fmt.Errorf("not found any avaliable disk")
	} else {
		return availableDisk, lastErr
	}
}

//CollectHostPortName use /sys/class/fc_host/host2/port_name get the port name
func (u *FCUtil) CollectHostPortName() ([]string, error) {
	io := &osIOHandler{}
	fcHostPath := "/sys/class/fc_host"
	portNames := []string{}
	var lastErr error

	dirs, err := io.ReadDir(fcHostPath)
	if err != nil {
		glog.Errorf("Read %s failed", fcHostPath)
		return portNames, err
	}

	for _, hostDir := range dirs {
		portNameFile := path.Join(fcHostPath, hostDir.Name(), "port_name")
		if contents, err := io.ReadFile(portNameFile); err == nil {
			//original port name is like 0x21000024ff8fbcbb\n
			//we need remove the 0x, and use upper case
			portName := strings.Trim(string(contents), "\n")
			portNames = append(portNames, strings.ToUpper(portName[2:]))
		} else {
			glog.Errorf("read %s content failed", portNameFile)
			lastErr = err
		}
	}

	return portNames, lastErr
}

//SearchDiskPath search a general disk or a multipath disk if possible with the given WWIDs or WWNs and LUN.
func (u *FCUtil) SearchDiskPath(wwns []string, luns []string, doScan bool, maxTryCount int) (string, error) {
	availableDisks := []string{}

	io := &osIOHandler{}
	mpathUtil := &multiPathUtil{}

	for tryCount := 0; tryCount < maxTryCount; tryCount = tryCount + 1 {
		if doScan {
			u.scsiHostRescan(io)
			time.Sleep(time.Duration(u.hostCfg.SCSIScanWaitInterval) * time.Second)
		}

		disks, err := u.findDisks(wwns, luns, io)
		if len(disks) > 0 {
			availableDisks = disks
		}
		if err == nil {
			break
		}
	}

	if len(availableDisks) == 0 {
		glog.Error("no fc disk found")
		return "", fmt.Errorf("no fc disk found")
	}

	dm, err := mpathUtil.FindMultiPathDev(availableDisks, u.hostCfg.MultiPathSearchRetryTimes, u.hostCfg.MultiPathSearchWaitInterval)
	if err == nil {
		return dm, nil
	}

	if u.hostCfg.ForceUseMultiPath {
		return "", fmt.Errorf("no multipath device found")
	}

	glog.Warningf("can not find the multipath device, use a path instead.")
	return availableDisks[0], nil
}

//ExtendDisk extend the path device and multipath device on the host.
func (u *FCUtil) ExtendDisk(property *storage.ConnProperty) (string, error) {
	io := &osIOHandler{}
	mpathUtil := &multiPathUtil{
		resizeDelay: u.hostCfg.MultiPathResizeDelay,
	}

	disks, err := u.findDisks(property.WWPNs, property.LunIDs, io)
	if err != nil {
		return "", err
	}

	//rescan each disk
	for _, disk := range disks {
		if err := RescanBlockDevice(disk, io); err != nil {
			glog.Errorf("rescan device %s failed for %s", disk, err)
		}
	}

	//find the multipath device
	dm, err := mpathUtil.FindMultiPathDev(disks, 1, 0)
	if err == nil {
		//resize the multipath device
		return dm, mpathUtil.ResizeMultiPathDev(dm)
	}

	if u.hostCfg.ForceUseMultiPath {
		return "", fmt.Errorf("no multipath device found")
	}

	glog.Warningf("can not find the multipath device, use a path instead.")
	return disks[0], nil
}

func (u *FCUtil) AttachDisk(property *storage.ConnProperty) (string, error) {
	return u.SearchDiskPath(property.WWPNs, property.LunIDs, true, u.hostCfg.SCSIScanRetryTimes)
}

func (u *FCUtil) GetDiskAttachPath(property *storage.ConnProperty) (string, error) {
	return u.SearchDiskPath(property.WWPNs, property.LunIDs, false, 1)
}

func (u *FCUtil) DetachDisk(property *storage.ConnProperty) (string, error) {
	devicePath, err := u.GetDiskAttachPath(property)
	if err != nil {
		return devicePath, err
	}

	return devicePath, RemoveDisk(devicePath)
}

func (u *FCUtil) BuildHostInfo(hostname string) (*storage.HostInfo, error) {
	name, err := GenerateHostName("fc", hostname)
	if err != nil {
		return nil, fmt.Errorf("generate host name failed")
	}

	hostInfo := &storage.HostInfo{
		Hostname: name,
		Link:     storage.HostLinkFC,
	}

	if hostname != "" {
		return hostInfo, nil
	}

	//get local FC WWPN and iSCSI initiator
	portNames, err := u.CollectHostPortName()
	if len(portNames) == 0 {
		return nil, fmt.Errorf("port names get failed, get none port")
	}
	if err != nil {
		glog.Warningf("can not get all port names, not all path is available")
	}

	hostInfo.WWPNs = portNames

	return hostInfo, nil
}
