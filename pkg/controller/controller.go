package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/host"
	"inspur.com/storage/instorage-k8s/pkg/storage"
	"inspur.com/storage/instorage-k8s/pkg/storage/as13000"
	"inspur.com/storage/instorage-k8s/pkg/storage/instorage"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

//IController define the general volume related operation,
//and the operations maybe need to do some work on the storage and host
type IController interface {
	//CreateVolume create a volume with given name and size, together with specific option
	CreateVolume(name string, size string, options map[string]string) (map[string]string, error)

	// CloneVolume create a volume with given name and size, together with specific option, or sourceVolumeName, or snapshotName
	CloneVolume(name string, size string, options map[string]string, sourceVolumeName string, snapshotName string) (map[string]string, error)

	//DeleteVolume delete a volume with a given name
	DeleteVolume(name string, options map[string]string) error

	//ListVolume list volumes with a given maxEnties and a given startingToken
	ListVolume(maxEnties int32, startingToken string) ([]string, []int64, string, error)
	// GetCapacity get capacity of a given storage pool
	GetCapacity(options map[string]string) (int64, error)

	//Attach just attach a volume to a host with some option
	Attach(hostname string, volumeName string, options map[string]string) (string, error)
	//IsAttached check whether a volume is attached to a host
	//IsAttached(hostname string, volumeName string, options map[string]string) (bool, error)
	//Detach used to detach a volume from a host
	Detach(hostname string, volumeName string, detachOnHost bool) error
	//MountDevice mount a volume on the specific path with specific file system
	MountDevice(volumeName string, mountPath string, fsType string, options map[string]string) (string, error)
	//UnMountDevice unmount the device which have mounted to the mountPath and then detach the volume
	UnMountDevice(mountPath string) (string, error)
	// CreateSnapshot from source volume
	CreateSnapshot(sourceVolName string, snapshotName string) (bool, string, error)
	// DeleteSnapshot  with a given name
	DeleteSnapshot(snapshotId string) error
	//ListSnapshots list snapshots with a given maxEnties and a given startingToken
	ListSnapshots(maxEnties int32, startingToken string, sourceVolName string) ([]string, []string, string, error)
	//ExtendVolume extend the given volume online
	ExtendVolume(name string, newSize string, oldSize string, devPath string, devMountPath string, options map[string]string) error
	//GetVolumeStats with given volume online
	GetVolumeStats(volumePath string) (int64, int64, int64, error)
}

const (
	//MountFSType is key for fs option
	MountFSType = "mountFSType"
)

type controller struct {
	host           string //ip:port
	login          string //username
	password       string //password for login
	deviceUserName string //username for login device(for 13000 storage)
	devicePassword string //password for login device (for 13000 storage)

	strUtil  storage.IStorageOperater
	hostUtil host.HostUtil
}

//NewController create a controller
//options should contain follows:
// host : the address of form addr:port to the storage
// login: the login for the storage
// password: the password to login into the storage
// link: iscsi or fc
func NewController(cfg utils.Config) IController {
	var hostUtil host.HostUtil
	//check the link type
	switch strings.ToLower(cfg.Host.Link) {
	case "fc":
		hostUtil = host.NewFCUtil(cfg.Host)
	case "iscsi":
		hostUtil = host.NewISCSIUtil(cfg.Host)
	default:
		return nil
	}
	switch strings.ToUpper(cfg.Storage[0].StrType) {
	case "AS18000":
		return &controller{
			host:     cfg.Storage[0].Host,
			login:    cfg.Storage[0].Username,
			password: cfg.Storage[0].Password,
			strUtil:  instorage.NewStorUtil(cfg.Storage[0]),
			hostUtil: hostUtil,
		}
	default:
		//case "AS13000"
		return &controller{
			host:           cfg.Storage[0].Host,
			login:          cfg.Storage[0].Username,
			password:       cfg.Storage[0].Password,
			deviceUserName: cfg.Storage[0].DeviceUsername,
			devicePassword: cfg.Storage[0].DevicePassword,
			strUtil:        as13000.NewStorUtil(cfg.Storage[0]),
			hostUtil:       hostUtil,
		}
	}
}

func (c *controller) Attach(hostname string, volumeName string, options map[string]string) (string, error) {
	glog.Debugf("Enter Attach(): hostname=%s, volumeName: %s, options=%+v",
		hostname, volumeName, options)
	//collect iSCSI initiator name or FC WWPN
	hostInfo, err := c.hostUtil.BuildHostInfo(hostname)
	if err != nil {
		return "", err
	}
	glog.Debugf("hostInfo: %+v", hostInfo)

	//bind the volume on storage
	property, err := c.strUtil.AttachVolume(volumeName, *hostInfo, options)
	if err != nil {
		glog.Errorf("volume attach on storage failed %s", err)
		return "", err
	}
	glog.Infof("storage attach response %s", property)

	//search the device on host
	devicePath, err := c.hostUtil.AttachDisk(property)
	if err != nil {
		glog.Errorf("volume attach on host failed %s", err)
		glog.Infof("detach volume on storage")
		c.strUtil.DetachVolume(volumeName, *hostInfo)
		return "", err
	}
	glog.Infof("volume attached, disk to use %s", devicePath)
	glog.Debugf("Exit Attach()")
	return devicePath, nil
}

func (c *controller) IsAttached(hostname string, volumeName string, options map[string]string) (bool, error) {
	//collect iSCSI initiator name or FC WWPN
	hostInfo, err := c.hostUtil.BuildHostInfo(hostname)
	if err != nil {
		return false, err
	}

	//check whether the volume is mapped on storage
	property, err := c.strUtil.GetVolumeAttachInfo(volumeName, *hostInfo)
	if err != nil {
		glog.Errorf("volume attach check failed on storage %s", err)
		return false, err
	}
	if property == nil {
		glog.Infof("volume not attach to host on storage")
		return false, nil
	}

	//check whether the volume is attached on host
	devicePath, err := c.hostUtil.GetDiskAttachPath(property)
	if err != nil {
		glog.Infof("volume attach check failed on host %s", err)
		return false, err
	}
	if devicePath == "" {
		glog.Infof("volume not attach on host, but has been attached on storage")
		return false, nil
	} else {
		return true, nil
	}
}

//Detach use the give hostname as the host name create on storage if hostname is not empty
//else it will use the /etc/hostname as the hostname
//so if kubelet use the hostname-override option, the hostname is maybe not match each other
func (c *controller) Detach(hostname string, volumeName string, detachOnHost bool) error {
	glog.Debugf("Enter Detach(): hostname=%s, volumeName=%s, detachOnHost=%+v",
		hostname, volumeName, detachOnHost)
	//collect iSCSI initiator name or FC WWPN
	hostInfo, err := c.hostUtil.BuildHostInfo(hostname)
	if err != nil {
		return err
	}

	if detachOnHost {
		//get the volume connect info so we can know which device on the host we need to remove
		property, err := c.strUtil.GetVolumeAttachInfo(volumeName, *hostInfo)
		if err != nil {
			glog.Errorf("volume(%s) attach info get failed on storage %+v", volumeName, err)
			return err
		}
		if property == nil {
			glog.Warningf("volume(%s) not attached on storage", volumeName)
			return nil
		}

		glog.Infof("volume(%s) attach info %s", volumeName, property)

		//detach the disk from host
		if _, err = c.hostUtil.DetachDisk(property); err != nil {
			glog.Errorf("volume(%s) detach from host failed %+v", volumeName, err)
			return err
		}
		glog.Infof("volume(%s) detached from host", volumeName)
	}

	//detach the disk from storage
	if err = c.strUtil.DetachVolume(volumeName, *hostInfo); err != nil {
		glog.Errorf("volume(%s) detach from storage failed %+v", volumeName, err)
		return err
	}

	glog.Infof("volume(%s) detached from storage", volumeName)
	glog.Debugf("Exit Detach()")
	return nil
}

//MountDevice mount the volume to given mountPath
func (c *controller) MountDevice(volumeName string, mountPath string, fsType string, options map[string]string) (string, error) {
	//first do an attach
	devPath, err := c.Attach("", volumeName, map[string]string{})
	if err != nil {
		return "", fmt.Errorf("attach device failed for %s", devPath)
	}

	//then mount the dev to mount path
	m := host.Mounter{}
	if err := m.FormatAndMount(devPath, mountPath, fsType, []string{}); err != nil {
		return "", fmt.Errorf("mount device failed for %s", err)
	}

	return devPath, nil
}

//UnMountDevice unmount the device from mount path,
//and also detach the device from host and storage
func (c *controller) UnMountDevice(mountPath string) (string, error) {
	glog.Infof("UnMountDevice: %s", mountPath)
	m := host.Mounter{}

	//first we need get the actual device with the mountPath
	devicePath, err := m.GetDevice(mountPath)
	if err != nil {
		glog.Errorf("can not get device path from mountPath(%s). %+v", mountPath, err)
		return "", fmt.Errorf("can not get device path from mountPath(%s).error: %+v", mountPath, err)
	}
	uid, err := host.GetDiskUID(devicePath)
	if err != nil {
		glog.Errorf("can not get device's scsi id. %+v", err)
		return "", fmt.Errorf("can not get device's scsi id. %+v", err)
	}

	glog.Debugf("device %s uid %s", devicePath, uid)

	//uid from host has a prefix 3, and is lower case, should do some transfer
	//360050760008989c0d00000000002aa53 host
	// 60050760008989C0D00000000002AA53 storage
	uid = strings.ToUpper(uid[1:])
	volumeName, err := c.strUtil.GetVolumeNameWithUID(uid)
	if err != nil {
		glog.Errorf("can not get device's scsi id. %+v", err)
		return "", fmt.Errorf("can not get volume name with uid. %+", err)
	}

	glog.Debugf("volume name: %s", volumeName)

	//then we need unmount the path
	if err := m.Unmount(mountPath); err != nil {
		return "", fmt.Errorf("unmount path failed. %s", err)
	}

	//and then detach volume from both host and storage
	return devicePath, c.Detach("", volumeName, true)
}

//CreateVolume create a volume with name and in size, with parameter from options
func (c *controller) CreateVolume(name string, size string, options map[string]string) (map[string]string, error) {
	return c.strUtil.CreateVolume(name, size, options)
}

func (c *controller) CloneVolume(name string, size string, options map[string]string, sourceVolumeName string, snapshotName string) (map[string]string, error) {
	return c.strUtil.CloneVolume(name, size, options, sourceVolumeName, snapshotName)
}

func (c *controller) DeleteVolume(name string, options map[string]string) error {
	return c.strUtil.DeleteVolume(name, options)
}

func (c *controller) ListSnapshots(maxEnties int32, startingToken string, sourceVolName string) ([]string, []string, string, error) {
	return c.strUtil.ListSnapshots(maxEnties, startingToken, sourceVolName)
}

func (c *controller) ListVolume(maxEnties int32, startingToken string) ([]string, []int64, string, error) {
	return c.strUtil.ListVolume(maxEnties, startingToken)
}

func (c *controller) GetCapacity(options map[string]string) (int64, error) {
	return c.strUtil.GetCapacity(options)
}

func (c *controller) CreateSnapshot(sourceVolName string, snapshotName string) (bool, string, error) {
	return c.strUtil.CreateSnapshot(sourceVolName, snapshotName)
}
func (c *controller) DeleteSnapshot(snapshotId string) error {
	return c.strUtil.DeleteSnapshot(snapshotId)
}

func (c *controller) GetVolumeStats(volumePath string) (int64, int64, int64, error) {
	return host.DF(volumePath)
}

func (c *controller) ExtendVolume(name string, newSize string, oldSize string, devPath string, devMountPath string, options map[string]string) error {
	//collect iSCSI initiator name or FC WWPN
	hostInfo, err := c.hostUtil.BuildHostInfo("")
	if err != nil {
		return err
	}

	//get the volume connect info so we can know which device on the host we need to remove
	property, err := c.strUtil.GetVolumeAttachInfo(name, *hostInfo)
	if err != nil {
		glog.Errorf("volume attach info get failed on storage %s", err)
		return err
	}
	if property == nil {
		glog.Warningf("Volume %s not attached on storage", name)
		return nil
	}
	glog.Infof("Volume attach info %s.", property)

	//Freeze file system
	needFreezeFS := c.strUtil.NeedFreezeFSWhenExtend(name, options)
	bT0 := time.Now()
	if needFreezeFS {
		glog.Debugf("Begin to freeze the path %s.", devMountPath)
		utils.FIFreeze(devMountPath)
		glog.Debugf("Freeze path %s finished in %s.", devMountPath, time.Since(bT0))
	}

	//extend the capacity on storage
	bT1 := time.Now()
	glog.Debugf("Begin to extend volume %s on storage.", name)
	if err := c.strUtil.ExtendVolume(name, newSize, options); err != nil {
		//We must thaw the path before return.
		if needFreezeFS {
			utils.FIThaw(devMountPath)
		}
		return fmt.Errorf("extend volume %s on storage failed for %s", name, err)
	}
	glog.Debugf("Extend volume %s from %sGB to %sGB on storage finished in %s.", name, oldSize, newSize, time.Since(bT1))

	//Thaw file system
	if needFreezeFS {
		bT2 := time.Now()
		glog.Debugf("Begin to thaw the path %s.", devMountPath)
		utils.FIThaw(devMountPath)
		glog.Debugf("Thaw path %s finished in %s.", devMountPath, time.Since(bT2))
	}
	glog.Debugf("Path %s freezed total %s during freeze, extend and thaw operation.", devMountPath, time.Since(bT0))

	//rescan each device to detect the new size.
	extendPath, err := c.hostUtil.ExtendDisk(property)
	if err != nil {
		glog.Errorf("volume extend on host failed %s, need operation by hand", err)
		return err
	}

	if extendPath != devPath {
		//when not use multipath, we choose a path with no intention,
		//so what we choose when do extend may not same as what we choose when mount disk.
		glog.Warningf("extend path %s not the same as used path %s, maybe need manual extend file system", extendPath, devPath)
	}

	//Extend the file system on device.
	if _, err := host.ExtendFS(extendPath, devMountPath); err != nil {
		glog.Errorf("file system extend failed %s, need operation by hand", err)
		return err
	}

	return nil
}
