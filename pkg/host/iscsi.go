package host

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/storage"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

func updateISCSIDiscoverydb(username, password string) error {
	return nil
}

func updateISCSINode(username, password string) error {
	return nil
}

type iSCSITool struct{}

func (t *iSCSITool) Rescan(portal, target string) error {
	out, err := utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "-R")
	if err != nil {
		glog.Infof("iscsi: rescan failed %s (%v)", string(out), err)
	}

	return err
}

func (t *iSCSITool) Discovery(portal, target string) error {
	out, err := utils.Execute("iscsiadm", "-m", "discoverydb", "-t", "sendtargets", "-p", portal, "--discover")
	if err != nil {
		glog.Errorf("iscsi: discovery failed %s (%v)", string(out), err)
	}

	return err
}

func (t *iSCSITool) Login(portal, target string) error {
	out, err := utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "--login")
	if err != nil {
		glog.Errorf("iscsi: login failed %s %s", string(out), err)
	}

	return err
}

func (t *iSCSITool) Logout(portal, target string) error {
	out, err := utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "--logout")
	if err != nil {
		glog.Errorf("iscsi: logout failed %s %s", string(out), err)
	}

	return err
}

func (t *iSCSITool) NodeDelete(portal, target string) error {
	out, err := utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "-o", "delete")
	if err != nil {
		glog.Errorf("iscsi: delete node failed %s %s", string(out), err)
	}

	return err
}

func (t *iSCSITool) DiscoveryAndLogin(portal, target string) error {
	// build discoverydb and discover iscsi target
	if _, err := utils.Execute("iscsiadm", "-m", "discoverydb", "-t", "sendtargets", "-p", portal, "-o", "new"); err != nil {
		glog.Errorf("create discoverydb sendtargets failed")
		return err
	}

	// update discoverydb with CHAP secret
	if err := updateISCSIDiscoverydb("", ""); err != nil {
		glog.Errorf("updateISCSIDiscoverydb error: %+v", err)
		return err
	}

	// do discovery
	if _, err := utils.Execute("iscsiadm", "-m", "discoverydb", "-t", "sendtargets", "-p", portal, "--discover"); err != nil {
		// delete discoverydb record
		glog.Errorf("do discover error: %+v", err)
		utils.Execute("iscsiadm", "-m", "discoverydb", "-t", "sendtargets", "-p", portal, "-o", "delete")
		return err
	}

	if err := updateISCSINode("", ""); err != nil {
		// failure to update node db is rare. But deleting record will likely impact those who already start using it.
		glog.Errorf("updateISCSINode error: %+v", err)
		return err
	}

	// login to iscsi target
	if _, err := utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "--login"); err != nil {
		// delete the node record from database
		utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "-o", "delete")
		// delete sendtarget from database
		utils.Execute("iscsiadm", "-m", "discoverydb", "-t", "sendtargets", "-p", portal, "-o", "delete")
		glog.Errorf("login to iscsi target error: %+v", err)
		return err
	}
	// in case of node failure/restart, explicitly set to manual login so it doesn't hang on boot
	if _, err := utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "-o", "update", "-n", "node.startup", "-v", "manual"); err != nil {
		// don't fail if we can't set startup mode, but log warning so there is a clue
		glog.Warningf("Warning: Failed to set iSCSI login mode to manual. Error: %v", err)
	}

	return nil
}

func (t *iSCSITool) LogoutAndDelete(portal, target string) error {
	//logout
	if _, err := utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "--logout"); err != nil {
		glog.Errorf("logout failed %s", err)
		return err
	}

	//delete the node
	if _, err := utils.Execute("iscsiadm", "-m", "node", "-p", portal, "-T", target, "-o", "delete"); err != nil {
		glog.Errorf("delete node failed %s", err)
		return err
	}

	//delete the node in discoverydb
	if _, err := utils.Execute("iscsiadm", "-m", "discoverydb", "-p", portal, "-t", "sendtargets", "-o", "delete"); err != nil {
		glog.Errorf("delete node failed %s", err)
		return err
	}

	return nil
}

//ISCSIUtil define the actions used when host is in iscsi environment.
type ISCSIUtil struct {
	hostCfg utils.HostCfg
}

//NewISCSIUtil create a ISCSIUtil for use.
func NewISCSIUtil(cfg utils.HostCfg) *ISCSIUtil {
	return &ISCSIUtil{
		hostCfg: cfg,
	}
}

func (u *ISCSIUtil) countISCSISessionDevices(portals []string, targets []string) (int, error) {
	io := &osIOHandler{}

	devPrefixSet := []string{}
	for idx, portal := range portals {
		devPrefixSet = append(devPrefixSet, fmt.Sprintf("ip-%s-iscsi-%s", portal, targets[idx]))
	}

	dirs, err := io.ReadDir("/dev/disk/by-path")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, f := range dirs {
		name := f.Name()
		for _, devPrefix := range devPrefixSet {
			if strings.Contains(name, devPrefix) {
				glog.Debugf("name=%s,devPrefix=%s", name, devPrefix)
				count++
				break
			}
		}
	}

	return count, nil
}

func (u *ISCSIUtil) findDiskByPath(portal, target, lun string) (string, error) {
	io := &osIOHandler{}
	devicePath := fmt.Sprintf("/dev/disk/by-path/ip-%s-iscsi-%s-lun-%s", portal, target, lun)

	if _, err := io.Lstat(devicePath); err != nil {
		return "", err
	} else {
		return io.EvalSymlinks(devicePath)
	}
}

//FindDiskByPath get the device path with given Portal, Target, LUN info.
func (u *ISCSIUtil) FindDiskByPath(portal, target, lun string, retryCount int) (string, error) {
	for i := 0; i < retryCount; i++ {
		if i != 0 {
			time.Sleep(time.Duration(u.hostCfg.ISCSIPathCheckWaitInterval) * time.Second)
		}

		if devicePath, err := u.findDiskByPath(portal, target, lun); err == nil {
			return devicePath, nil
		}
	}

	return "", fmt.Errorf("could not find wanted device for %s %s %s", portal, target, lun)
}

func (u *ISCSIUtil) GetInitiatorName() (string, error) {
	io := &osIOHandler{}
	contents, err := io.ReadFile("/etc/iscsi/initiatorname.iscsi")
	if err != nil {
		return "", err
	}

	//contents of /etc/iscsi/initiatorname.iscsi like follows:
	//
	//## DO NOT EDIT OR REMOVE THIS FILE!
	//## If you remove this file, the iSCSI daemon will not start.
	//## If you change the InitiatorName, existing access control lists
	//## may reject this initiator.  The InitiatorName must be unique
	//## for each iSCSI initiator.  Do NOT duplicate iSCSI InitiatorNames.
	//InitiatorName=iqn.1993-08.org.debian:01:422365e27ee3
	//
	//Attention, last line have a '\n'

	parts := strings.Split(string(contents), "InitiatorName=")
	initiator := strings.Trim(parts[1], "\n")
	return initiator, nil
}

//ExtendDisk extend the path device and multipath device on the host.
func (u *ISCSIUtil) ExtendDisk(p *storage.ConnProperty) (string, error) {
	io := &osIOHandler{}
	mpathUtil := &multiPathUtil{
		resizeDelay: u.hostCfg.MultiPathResizeDelay,
	}

	devPaths := []string{}
	for idx, portal := range p.Portals {
		target := p.Targets[idx]
		lun := p.LunIDs[idx]

		if devPath, err := u.FindDiskByPath(portal, target, lun, 1); err == nil {
			devPaths = append(devPaths, devPath)
		}
	}

	//rescan each disk
	for _, disk := range devPaths {
		if err := RescanBlockDevice(disk, io); err != nil {
			glog.Errorf("rescan device %s failed for %s", disk, err)
		}
	}

	//find the multipath device
	dm, err := mpathUtil.FindMultiPathDev(devPaths, 1, 0)
	if err == nil {
		//resize the multipath device
		return dm, mpathUtil.ResizeMultiPathDev(dm)
	}

	if u.hostCfg.ForceUseMultiPath {
		return "", fmt.Errorf("no multipath device found")
	}

	glog.Warningf("can not find the multipath device, use a path instead.")
	return devPaths[0], nil
}

//SearchDisk is a generic routine to get a device with iscsi target info.
func (u *ISCSIUtil) SearchDisk(portals []string, targets []string, luns []string, doScan bool, doLogin bool) (string, error) {
	tool := iSCSITool{}
	mpathUtil := &multiPathUtil{}

	devicePaths := []string{}
	for idx, portal := range portals {
		target := targets[idx]
		lun := luns[idx]

		//if session already exist, just rescan is ok
		if doScan {
			tool.Rescan(portal, target)
			time.Sleep(time.Duration(u.hostCfg.SCSIScanWaitInterval) * time.Second)
		}

		if devicePath, err := u.FindDiskByPath(portal, target, lun, 1); err == nil {
			devicePaths = append(devicePaths, devicePath)
			continue
		}

		if doLogin {
			//we need discovery and login into the node.
			if err := tool.DiscoveryAndLogin(portal, target); err != nil {
				glog.Errorf("iscsi discovery and login failed %s", err)
				continue
			}

			//it's better to do a wait after login.
			time.Sleep(time.Duration(u.hostCfg.SCSIScanWaitInterval) * time.Second)

			if devicePath, err := u.FindDiskByPath(portal, target, lun, u.hostCfg.ISCSIPathCheckRetryTimes); err == nil {
				devicePaths = append(devicePaths, devicePath)
				continue
			} else {
				glog.Errorf("Device find for %s %s %s failed.", portal, target, lun)
			}
		}
	}

	if len(devicePaths) == 0 {
		glog.Warningf("Failed to get any path.")
		return "", fmt.Errorf("no device path found")
	}

	dm, err := mpathUtil.FindMultiPathDev(devicePaths, u.hostCfg.MultiPathSearchRetryTimes, u.hostCfg.MultiPathSearchWaitInterval)
	if err == nil {
		glog.Infof("Find MultiPath Device: %+v", dm)
		return dm, nil
	}

	if u.hostCfg.ForceUseMultiPath {
		glog.Errorf("no multipath device found")
		return "", fmt.Errorf("no multipath device found")
	}

	glog.Warningf("can not find the multipath device, use a path instead.")
	return devicePaths[0], nil
}

func (u *ISCSIUtil) GetDiskAttachPath(p *storage.ConnProperty) (string, error) {
	return u.SearchDisk(p.Portals, p.Targets, p.LunIDs, false, false)
}

//AttachDisk attach the disk specified by portal target and lun. And return the disk path to use
func (u *ISCSIUtil) AttachDisk(p *storage.ConnProperty) (string, error) {
	return u.SearchDisk(p.Portals, p.Targets, p.LunIDs, true, true)
}

func (u *ISCSIUtil) DetachDisk(p *storage.ConnProperty) (string, error) {
	devicePath, err := u.SearchDisk(p.Portals, p.Targets, p.LunIDs, false, false)
	if err != nil {
		return "", err
	}
	if devicePath == "" {
		return "", nil
	}

	//delete the device path
	if err := RemoveDisk(devicePath); err != nil {
		glog.Errorf("remove device path failed %s", devicePath)
		return "", err
	}

	//check whether need to logout and delete db
	time.Sleep(time.Duration(2) * time.Second) //wait for 2 second
	count, err := u.countISCSISessionDevices(p.Portals, p.Targets)
	if err != nil {
		glog.Errorf("count session device failed %s", err)
		return "", err
	}

	if count > 0 {
		glog.Infof("session has other device, do not logout")
		return devicePath, nil
	}

	tool := iSCSITool{}
	//no device belong to the session, logout
	for idx, portal := range p.Portals {
		//failed just failed, what can we do?
		if err := tool.LogoutAndDelete(portal, p.Targets[idx]); err != nil {
			glog.Errorf("session %s %s logout failed %s", portal, p.Targets[idx], err)
		}
	}

	return devicePath, nil
}

func (u *ISCSIUtil) BuildHostInfo(hostname string) (*storage.HostInfo, error) {
	name, err := GenerateHostName("iscsi", hostname)
	if err != nil {
		return nil, fmt.Errorf("generate host name failed")
	}

	hostInfo := &storage.HostInfo{
		Hostname: name,
		Link:     storage.HostLinkiSCSI,
	}

	if hostname != "" {
		return hostInfo, nil
	}

	//get local iSCSI initiator
	initiator, err := u.GetInitiatorName()
	if err != nil {
		return nil, err
	}

	hostInfo.Initiator = initiator

	return hostInfo, nil
}
