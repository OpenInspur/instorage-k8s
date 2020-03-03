package host

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/storage"
	"inspur.com/storage/instorage-k8s/pkg/utils"
	utilexec "k8s.io/utils/exec"
)

type HostUtil interface {
	AttachDisk(property *storage.ConnProperty) (string, error)
	DetachDisk(property *storage.ConnProperty) (string, error)
	GetDiskAttachPath(property *storage.ConnProperty) (string, error)
	BuildHostInfo(hostname string) (*storage.HostInfo, error)
	ExtendDisk(property *storage.ConnProperty) (string, error)
}

func DF(path string) (int64, int64, int64, error) {
	if path == "" {
		return 0, 0, 0, fmt.Errorf("failed to df -k, path is emtpy")
	}
	output, err := utils.Execute("df", "-k", path)
	if err == nil {
		lines := strings.Split(output, "\n")
		if len(lines) < 2 {
			return 0, 0, 0, fmt.Errorf("failed to df -k %s, output: %s, size of lines is less than 2", path, string(output))
		}
		fields := strings.Fields(lines[1])
		if len(fields) < 4 {
			return 0, 0, 0, fmt.Errorf("failed to df -k %s, output: %s, size of fields is less than 4", path, string(output))
		}
		total, errTotal := strconv.ParseInt(fields[1], 10, 64)
		used, errUsed := strconv.ParseInt(fields[2], 10, 64)
		available, errAvailable := strconv.ParseInt(fields[3], 10, 64)
		if errTotal != nil || errUsed != nil || errAvailable != nil {
			return 0, 0, 0, fmt.Errorf("failed to df -k %s, output: %s, fields cann't been converted to int", path, string(output))
		}
		return total * 1000, used * 1000, available * 1000, nil
	} else {
		return 0, 0, 0, fmt.Errorf("failed to df -k %s, err: %v, output: %s", path, err, string(output))
	}
}

//GenerateHostName create a hostname base on the prefix and hostname
func GenerateHostName(prefix, hostname string) (string, error) {
	if hostname == "" {
		if name, err := GetHostName(); err != nil {
			return "", err
		} else {
			hostname = name
		}
	}

	return fmt.Sprintf("%s-%s", prefix, hostname), nil
}

//GetHostName get host name from /etc/hostname file
func GetHostName() (string, error) {
	io := osIOHandler{}

	data, err := io.ReadFile("/etc/hostname")
	if err != nil {
		return "", err
	} else {
		return strings.Trim(string(data), "\n"), nil
	}
}

// ExtendFS perform resize of file system
func ExtendFS(devicePath string, deviceMountPath string) (bool, error) {
	format, err := GetDiskFormat(devicePath)

	if err != nil {
		formatErr := fmt.Errorf("ExtendFS error checking format for device %s: %v", devicePath, err)
		return false, formatErr
	}

	// If disk has no format, there is no need to resize the disk because mkfs.*
	// by default will use whole disk anyways.
	if format == "" {
		return false, nil
	}

	glog.Debugf("Expanding mounted volume %s", devicePath)
	switch format {
	case "ext3", "ext4":
		return extResize(devicePath)
	case "xfs":
		return xfsResize(deviceMountPath)
	}
	return false, fmt.Errorf("ExtendFS resize of format %s is not supported for device %s mounted at %s", format, devicePath, deviceMountPath)
}

func GetDiskFormat(disk string) (string, error) {
	dataOut, err := Run("blkid", "-p", "-s", "TYPE", "-s", "PTTYPE", "-o", "export", disk)
	output := string(dataOut)
	if err != nil {
		if exit, ok := err.(utilexec.ExitError); ok {
			if exit.ExitStatus() == 2 {
				// Disk device is unformatted.
				// For `blkid`, if the specified token (TYPE/PTTYPE, etc) was
				// not found, or no (specified) devices could be identified, an
				// exit code of 2 is returned.
				return "", nil
			}
		}
		glog.Errorf("Could not determine if disk %q is formatted (%v)", disk, err)
		return "", err
	}

	var fstype, pttype string

	lines := strings.Split(output, "\n")
	for _, l := range lines {
		if len(l) <= 0 {
			// Ignore empty line.
			continue
		}
		cs := strings.Split(l, "=")
		if len(cs) != 2 {
			return "", fmt.Errorf("blkid returns invalid output: %s", output)
		}
		// TYPE is filesystem type, and PTTYPE is partition table type, according
		// to https://www.kernel.org/pub/linux/utils/util-linux/v2.21/libblkid-docs/.
		if cs[0] == "TYPE" {
			fstype = cs[1]
		} else if cs[0] == "PTTYPE" {
			pttype = cs[1]
		}
	}

	if len(pttype) > 0 {
		glog.Debugf("Disk %s detected partition table type: %s", disk, pttype)
		// Returns a special non-empty string as filesystem type, then kubelet
		// will not format it.
		return "unknown data, probably partitions", nil
	}

	return fstype, nil
}

func extResize(devicePath string) (bool, error) {
	output, err := utils.Execute("resize2fs", devicePath)
	if err == nil {
		glog.Debugf("Device %s resized successfully", devicePath)
		return true, nil
	}

	resizeError := fmt.Errorf("resize of device %s failed: %v. resize2fs output: %s", devicePath, err, string(output))
	return false, resizeError
}

func xfsResize(deviceMountPath string) (bool, error) {
	output, err := utils.Execute("xfs_growfs", "-d", deviceMountPath)
	if err == nil {
		glog.Debugf("Device %s resized successfully", deviceMountPath)
		return true, nil
	}

	resizeError := fmt.Errorf("resize of device %s failed: %v. xfs_growfs output: %s", deviceMountPath, err, string(output))
	return false, resizeError
}
