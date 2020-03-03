package host

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/utils"
	utilexec "k8s.io/utils/exec"
)

const (
	// 'fsck' found errors and corrected them
	fsckErrorsCorrected = 1
	// 'fsck' found errors but exited without correcting them
	fsckErrorsUncorrected = 4
)

type MountPoint struct {
	Device string
	Path   string
	Type   string
	Opts   []string
	Freq   int
	Pass   int
}

//Mounter do mount and unmount operation on the system
type Mounter struct{}

//FormatAndMount format the disk and mount it to the mountPath
func (m *Mounter) FormatAndMount(devicePath string, mountPath string, fsType string, options []string) error {
	readOnly := false
	for _, option := range options {
		if option == "ro" {
			readOnly = true
			break
		}
	}

	options = append(options, "defaults")

	if !readOnly {
		// Run fsck on the disk to fix repairable issues, only do this for volumes requested as rw.
		glog.Infof("Checking for issues with fsck on disk: %s", devicePath)
		args := []string{"-a", devicePath}
		out, err := Run("fsck", args...)
		if err != nil {
			ee, isExitError := err.(utilexec.ExitError)
			switch {
			case err == utilexec.ErrExecutableNotFound:
				glog.Warningf("'fsck' not found on system; continuing mount without running 'fsck'.")
			case isExitError && ee.ExitStatus() == fsckErrorsCorrected:
				glog.Infof("Device %s has errors which were corrected by fsck.", devicePath)
			case isExitError && ee.ExitStatus() == fsckErrorsUncorrected:
				return fmt.Errorf("'fsck' found errors on device %s but could not correct them: %s.", devicePath, string(out))
			case isExitError && ee.ExitStatus() > fsckErrorsUncorrected:
				glog.Infof("`fsck` error %s", string(out))

			}
			//return fmt.Errorf("'fsck' found errors on device %s but could not correct them: %s.", devicePath, string(out))
		}
	}

	// Try to mount the disk
	glog.Infof("Attempting to mount disk: %s %s %s", fsType, devicePath, mountPath)
	mountErr := m.Mount(devicePath, mountPath, fsType, options)
	if mountErr != nil {
		// Mount failed. This indicates either that the disk is unformatted or
		// it contains an unexpected filesystem.
		existingFormat, err := GetDiskFormat(devicePath)
		if err != nil {
			return err
		}
		if existingFormat == "" {
			if readOnly {
				// Don't attempt to format if mounting as readonly, return an error to reflect this.
				return errors.New("failed to mount unformatted volume as read only")
			}

			// Disk is unformatted so format it.
			args := []string{devicePath}
			// Use 'ext4' as the default
			if len(fsType) == 0 {
				fsType = "ext4"
			}

			if fsType == "ext4" || fsType == "ext3" {
				args = []string{
					"-F",  // Force flag
					"-m0", // Zero blocks reserved for super-user
					devicePath,
				}
			}

			glog.Infof("Disk %q appears to be unformatted, attempting to format as type: %q with options: %v", devicePath, fsType, args)
			_, err := Run("mkfs."+fsType, args...)
			if err == nil {
				// the disk has been formatted successfully try to mount it again.
				glog.Infof("Disk successfully formatted (mkfs): %s - %s %s", fsType, devicePath, mountPath)
				return m.Mount(devicePath, mountPath, fsType, options)
			}

			glog.Errorf("format of disk %q failed: type:(%q) target:(%q) options:(%q)error:(%v)", devicePath, fsType, mountPath, options, err)
			return err
		} else {
			// Disk is already formatted and failed to mount
			if len(fsType) == 0 || fsType == existingFormat {
				// This is mount error
				return mountErr
			} else {
				// Block device is formatted with unexpected filesystem, let the user know
				return fmt.Errorf("failed to mount the volume as %q, it already contains %s. Mount error: %v", fsType, existingFormat, mountErr)
			}
		}
	}
	return mountErr
}

// makeMountArgs makes the arguments to the mount(8) command.
func (m *Mounter) makeMountArgs(source, target, fstype string, options []string) []string {
	// Build mount command as follows:
	//   mount [-t $fstype] [-o $options] [$source] $target
	mountArgs := []string{}
	if len(fstype) > 0 {
		mountArgs = append(mountArgs, "-t", fstype)
	}
	if len(options) > 0 {
		mountArgs = append(mountArgs, "-o", strings.Join(options, ","))
	}
	if len(source) > 0 {
		mountArgs = append(mountArgs, source)
	}
	mountArgs = append(mountArgs, target)

	return mountArgs
}

// Mount mounts source to target as fstype with given options. 'source' and 'fstype' must
// be an empty string in case it's not required, e.g. for remount, or for auto filesystem
// type, where kernel handles fstype for you. The mount 'options' is a list of options,
// currently come from mount(8), e.g. "ro", "remount", "bind", etc. If no more option is
// required, call Mount with an empty string list or nil.
func (m *Mounter) Mount(source string, target string, fstype string, options []string) error {

	mountArgs := m.makeMountArgs(source, target, fstype, options)

	glog.Infof("Mounting cmd mount with arguments (%s)", mountArgs)

	output, err := utils.Execute("mount", mountArgs...)

	if err != nil {
		args := strings.Join(mountArgs, " ")
		glog.Errorf("Mount failed: %v\nMounting command: mount\nMounting arguments: %s\nOutput: %s\n", err, args, string(output))
		return fmt.Errorf("mount failed: %v\nMounting command: mount\nMounting arguments: %s\nOutput: %s\n", err, args, string(output))
	}
	return nil
}

// Unmount unmounts the target.
func (m *Mounter) Unmount(target string) error {
	glog.Infof("Unmounting %s", target)

	output, err := utils.Execute("umount", target)
	if err != nil {
		glog.Errorf("Unmount failed: %v\nUnmounting arguments: %s\nOutput: %s\n", err, target, string(output))
		return fmt.Errorf("Unmount failed: %v\nUnmounting arguments: %s\nOutput: %s\n", err, target, string(output))
	}
	return nil
}

func (m *Mounter) GetDevice(target string) (string, error) {
	mps, _ := m.GetMountList()
	for _, mp := range mps {
		if mp.Path == target {
			return mp.Device, nil
		}
	}

	return "", fmt.Errorf("can not get the device for path %s", target)
}

func (m *Mounter) GetMountList() ([]MountPoint, error) {
	io := osIOHandler{}
	data, _ := io.ReadFile("/proc/mounts")

	mps := []MountPoint{}

	lines := strings.Split(strings.Trim(string(data), "\n"), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)

		mp := MountPoint{
			Device: fields[0],
			Path:   fields[1],
			Type:   fields[2],
			Opts:   strings.Split(fields[3], ","),
		}

		freq, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, err
		}
		mp.Freq = freq

		pass, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, err
		}
		mp.Pass = pass

		mps = append(mps, mp)
	}

	return mps, nil
}
