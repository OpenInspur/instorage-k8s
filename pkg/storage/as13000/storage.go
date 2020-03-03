package as13000

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/restApi"
	"inspur.com/storage/instorage-k8s/pkg/storage"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

//"github.com/golang/glog"
const POOL_TYPE_DUPLICATE, POOL_TYPE_ERASURE = 1, 0

//Storage13000Util encapulate the general operation with a 13000storage
var max_target_num = 64                      //max target number that AS13000 support
var max_lun_num_per_target = 64              //max lun number that AS13000 support per target
var optimal_target_num_for_k8s = 32          // advice target number that cinder use
var k8s_target_prefix = "target.inspur.k8s-" // k8s target name prefix

type Storage13000Util struct {
	restApiWrapper RestApiWrapper
}

//NewStorageUtil create a StorageUtil with given options
func NewStorUtil(cfg utils.StorageCfg) storage.IStorageOperater {
	return &Storage13000Util{
		restApiWrapper: *NewRestApiWrapper(cfg),
	}
}

//NeedFreezeFSWhenExtend check whether we need freeze the FS when do extend.
func (u *Storage13000Util) NeedFreezeFSWhenExtend(volumeName string, options map[string]string) bool {
	return false
}

//ExtendVolume extend the give volume capacity.
func (u *Storage13000Util) ExtendVolume(name string, size string, options map[string]string) error {
	glog.Debugf("Enter ExtendVolume(): name=%s,size=%s,options=%+v", name, size, options)
	//extend volume
	poolName := options["dataPool"]
	gSize, _ := strconv.ParseInt(size, 10, 64)
	mSize := gSize * 1024
	err := u.restApiWrapper.extendLvm(poolName, name, mSize)
	if err != nil {
		glog.Errorf("ExtendVolume() error:%+v", err)
		return err
	}
	glog.Debugf("Exit ExtendVolume()")
	return nil
}

func (u *Storage13000Util) AttachVolume(volumeName string, hostInfo storage.HostInfo, options map[string]string) (*storage.ConnProperty, error) {
	glog.Debugf("Enter AttachVolume(): volumeName=%s,hostInfo=%+v,options=%+v", volumeName, hostInfo, options)
	poolName, ok := options["dataPool"]
	if !ok {
		glog.Errorf("dataPool is not configured in flexVolume, please check.")
		return nil, fmt.Errorf("dataPool is not configured in flexVolume, please check.")
	}
	lunID := ""
	iqn := hostInfo.Initiator
	//
	isLunExist, lunID, isLunBindIqn, targetName, targetNode, err := u.GetTargetForVolume(volumeName, poolName, iqn)
	if err != nil {
		return nil, err
	}
	//lun not exist, map volume to target
	if !isLunExist {
		lunID, err = u.mapVolumeToTarget(targetName, poolName, volumeName, "")
		if err != nil {
			return nil, err
		}
		glog.Infof("map volume to target: targetName=%s,poolName=%s,volumeName=%s",
			targetName, poolName, volumeName)
	}
	//if lun has been binded to iqn ,then return
	if isLunBindIqn {
		glog.Infof("lun has binded iqn")
		glog.Debugf("Exit AttachVolume:lun is binding iqn, link=%s, lunID=%s, targetName=%s", hostInfo.Link, lunID, targetName)
		return u.BuildConnProperty(hostInfo.Link, lunID, targetName)
	}
	//bind lun to iqn
	err = u.restApiWrapper.bindLunIQN(targetName, targetNode, iqn, lunID)
	if err != nil {
		return nil, err
	}
	glog.Infof("bind lun with iqn:targetName=%s,lunID=%s,iqn=%s", targetName, lunID, iqn)
	glog.Debugf("Exit AttachVolume(): link=%s, lunID=%s, targetName=%s", hostInfo.Link, lunID, targetName)
	return u.BuildConnProperty(hostInfo.Link, lunID, targetName)

	//return nil, nil
}
func (u *Storage13000Util) GetTargetForVolume(volumeName string, poolName string, iqn string) (bool, string, bool, string, string, error) {
	glog.Debugf("Enter GetTargetForVolume(): volumeName=%s, poolName=%s, iqn=%s", volumeName, poolName, iqn)
	isLunExist := false
	isLunBindIqn := false
	targetName := ""
	targetNode := ""
	lunID := ""

	current_target_max_suffix_num := 0

	targets, err := u.restApiWrapper.queryTarget()
	if err != nil {
		return isLunExist, lunID, isLunBindIqn, targetName, targetNode, err
	}

	current_target_num := len(*targets)
	k8s_target_num := 0
	k8s_target_lun_num := max_lun_num_per_target
	for _, target := range *targets {
		if strings.HasPrefix(target.Name, k8s_target_prefix) {
			k8s_target_num++
			//get max k8s target name suffix number
			suffix_num, _ := strconv.Atoi(target.Name[len(k8s_target_prefix):])
			if suffix_num > current_target_max_suffix_num {
				current_target_max_suffix_num = suffix_num
			}
			//get target with least lun mapping
			luns, err := u.restApiWrapper.queryVolumeMapping(target.Name)
			if err != nil {
				return isLunExist, lunID, isLunBindIqn, targetName, targetNode, err
			}
			if len(*luns) < k8s_target_lun_num {
				targetName = target.Name
				targetNode = target.Node
				k8s_target_lun_num = len(*luns)
			}
			//check if lun mapping exist, then return
			for _, lun := range *luns {
				mappingLvm := lun.MappingLvm
				poolVolName := fmt.Sprintf("%s/%s", poolName, volumeName)
				lunsIqn := strings.Split(lun.IqnPort, ",")
				if mappingLvm == poolVolName {
					//lun mapping exist
					isLunExist = true
					lunID = lun.Id
					targetName = target.Name
					targetNode = target.Node
					for _, lunIqn := range lunsIqn {
						if lunIqn == iqn {
							isLunBindIqn = true
						}
					}
					glog.Debugf("Exit GetTargetForVolume()")
					return isLunExist, lunID, isLunBindIqn, targetName, targetNode, nil
				}
			}
		}
	}

	//
	if current_target_num < max_target_num {
		if (k8s_target_num < optimal_target_num_for_k8s) || (k8s_target_num >= optimal_target_num_for_k8s && k8s_target_lun_num == max_lun_num_per_target) {
			//create target
			str_suffix_num := fmt.Sprintf("%08v", current_target_max_suffix_num+1)

			targetName = fmt.Sprintf("%s%s", k8s_target_prefix, str_suffix_num)

			tgtName, err := u.restApiWrapper.createTarget(targetName)
			if err != nil {
				//fmt.Printf("createTarget error:%+v\n", err)
				return isLunExist, lunID, isLunBindIqn, targetName, targetNode, fmt.Errorf("createTarget error:%+v", err)
			}
			//get target node
			dataTarget, err := u.restApiWrapper.queryTargetByName(tgtName)
			if err != nil {
				return isLunExist, lunID, isLunBindIqn, targetName, targetNode, err
			}
			targetNode = dataTarget.Node
			//bind host ip ALL
			err = u.restApiWrapper.addHost(tgtName, "ALL")
			if err != nil {
				//fmt.Printf("createTarget error addHost:%+v\n", err)
				return isLunExist, lunID, isLunBindIqn, targetName, targetNode, fmt.Errorf("createTarget error addHost:%+v", err)
			}
		}
	} else {
		return isLunExist, lunID, isLunBindIqn, targetName, targetNode, fmt.Errorf("Get target failed, because the maxium number of targets has been reached.")
	}
	glog.Debugf("Exist GetTargetForVolume()")
	return isLunExist, lunID, isLunBindIqn, targetName, targetNode, nil
}
func (u *Storage13000Util) GetVolumeAttachInfo(volumeMappingName string, hostInfo storage.HostInfo) (*storage.ConnProperty, error) {
	glog.Debugf("Enter GetVolumeAttachInfo(): volumeMappingName=%s, hostInfo=%+v", volumeMappingName, hostInfo)
	//volume already attached, just build the attach info

	//no target info exist, volume not attached to that target
	var lunID string
	var targetName string
	poolName, volName, err := u.getPoolAndVolumeName(volumeMappingName)
	if err != nil {
		return nil, err
	}
	targets, err := u.restApiWrapper.queryTarget()
	if err != nil {
		return nil, err
	}

	for _, target := range *targets {
		if strings.HasPrefix(target.Name, k8s_target_prefix) {

			//get target with lun mapping
			luns, err := u.restApiWrapper.queryVolumeMapping(target.Name)
			if err != nil {
				return nil, err
			}
			//check if lun mapping exist, then return
			for _, lun := range *luns {
				mappingLvm := lun.MappingLvm
				poolVolName := fmt.Sprintf("%s/%s", poolName, volName)
				if poolName != "" {
					if mappingLvm == poolVolName {
						lunID = lun.Id
						targetName = target.Name
						glog.Debugf("Exit GetVolumeAttachInfo(): link=%s, lunID=%s, targetName=%s",
							hostInfo.Link, lunID, targetName)
						return u.BuildConnProperty(hostInfo.Link, lunID, targetName)
					}
				} else {
					_, lunVolName, _ := u.getPoolAndVolumeName(mappingLvm)
					if volName == lunVolName {
						lunID = lun.Id
						targetName = target.Name
						glog.Debugf("Exit GetVolumeAttachInfo(): link=%s, lunID=%s, targetName=%s",
							hostInfo.Link, lunID, targetName)
						return u.BuildConnProperty(hostInfo.Link, lunID, targetName)
					}
				}

			}
		}
	}

	//
	glog.Warningf("can not get volume attach info with volumeMappingName:%s", volumeMappingName)
	return nil, nil

}

func (u *Storage13000Util) DetachVolume(volumeMappingName string, hostInfo storage.HostInfo) error {
	glog.Debugf("Enter DetachVolume(): volumeMappingName=%s, hostInfo=%+v", volumeMappingName, hostInfo)

	var targetName string
	var lunID string
	volumeMapings, err := u.getVolumeMappingByVolumeName(volumeMappingName)
	if err != nil {
		glog.Errorf("get volume mapping by volume name(%s) error: %+v", volumeMappingName, err)
		return err
	}
	if len(*volumeMapings) == 0 {
		glog.Infof("volume(%s) has been detached from storage", volumeMappingName)
		return nil
	}
	if len(*volumeMapings) != 1 {
		glog.Errorf("there are not only one volumeMapping with volumeName:%s", volumeMappingName)
		return fmt.Errorf("there are not only one volumeMapping with volumeName:%s", volumeMappingName)
	}
	targetName = (*volumeMapings)[0].Target
	//get target node
	dataTarget, err := u.restApiWrapper.queryTargetByName(targetName)
	if err != nil {
		return err
	}
	targetNode := dataTarget.Node

	lunID = (*volumeMapings)[0].Id
	//unbind from iqn
	lunsIqn := strings.Split((*volumeMapings)[0].IqnPort, ",")
	isLunBindIqn := false
	for _, lunIqn := range lunsIqn {
		if lunIqn == hostInfo.Initiator {
			isLunBindIqn = true
		}
	}
	glog.Infof("targetName= %s, targetNode= %s, lunID= %s,hostIqn= %s, isLunBindIqn= %+v",
		targetName, targetNode, lunID, hostInfo.Initiator, isLunBindIqn)
	if isLunBindIqn {
		err = u.restApiWrapper.unBindLunIQN(targetName, targetNode, hostInfo.Initiator, lunID)
		if err != nil {
			glog.Errorf("unBindLunIQN failed: %+v", err)
			return err
		}
		glog.Infof("unbind lun iqn")
	}
	if len(lunsIqn) == 1 && (isLunBindIqn || lunsIqn[0] == "ALL") {
		//unmap from target
		if err := u.unMapVolumeFromTarget(targetName, lunID); err != nil {
			glog.Errorf("unMap volume from target failed(targetName=%s,lunID=%s): %+v",
				targetName, lunID, err)
			return err
		}
		glog.Infof("unMapVolumeFromTarget: targetName=%s, lunID=%s", targetName, lunID)
		volumes, err := u.restApiWrapper.queryVolumeMapping(targetName)
		if err != nil {
			glog.Warningf("get volume mapped to target %s failed, unable to decide whether need to delete the target", targetName)
			return nil
		}
		glog.Debugf("volume mapping=%+v", volumes)
		if len(*volumes) == 0 {
			//no volume mapped to the target, delete it
			glog.Infof("delete target: %s", targetName)
			glog.Debugf("Exit DetachVolume(). deleteTarget=%s", targetName)
			return u.restApiWrapper.deleteTarget(targetName)
		} else {
			glog.Info(fmt.Sprintf("other volume mapped to target(%s), do not delete target", targetName))
			glog.Debugf("Exit DetachVolume(): do not delete target")
			return nil
		}
	}
	glog.Debugf("Exit DetachVolume()")
	return nil
}

func (u *Storage13000Util) getVolumeMappingByVolumeName(volumeMappingName string) (*[]restApi.DataVolumeMapping, error) {
	volumeMapings := []restApi.DataVolumeMapping{}
	poolName, volName, err := u.getPoolAndVolumeName(volumeMappingName)
	if err != nil {
		return nil, err
	}
	dataTargetArray, err := u.restApiWrapper.queryTarget()
	if err != nil {
		return nil, err
	}
	for _, dataTarget := range *dataTargetArray {
		dataVolumeMappings, err := u.restApiWrapper.queryVolumeMapping(dataTarget.Name)
		if err != nil {
			return nil, err
		}

		for _, dataVolumeMap := range *dataVolumeMappings {
			if poolName != "" {
				if volumeMappingName == dataVolumeMap.MappingLvm {
					volumeMapings = append(volumeMapings, dataVolumeMap)
					//mappingLvm is unique
					return &volumeMapings, nil
				}
			} else {
				_, volumeName, err := u.restApiWrapper.getPoolAndVolumeName(dataVolumeMap.MappingLvm)
				if err != nil {
					continue
				}
				if volName == volumeName {
					volumeMapings = append(volumeMapings, dataVolumeMap)
				}
			}
		}
	}
	return &volumeMapings, nil
}

//return volumeMapping.MappingLvm
func (u *Storage13000Util) GetVolumeNameWithUID(uid string) (string, error) {
	glog.Debugf("Enter GetVolumeNameWithUID(): uid=%s", uid)
	volumeMapping, err := u.restApiWrapper.queryVolumeMappingByNaa(uid)
	if err != nil {
		return "", err
	}
	if volumeMapping != nil {
		glog.Debugf("Exit GetVolumeNameWithUID(). volumeMapping=%+v", volumeMapping)
		return volumeMapping.MappingLvm, nil
	}

	//find no volumeMapping with naa
	glog.Warningf("Exit GetVolumeNameWithUID: find no volumeMapping with naa:%s", uid)
	return "", nil

}

//CreateVolume create a volume on the storage with given name, size and other options
func (u *Storage13000Util) CreateBlockVolume(name string, size string, options map[string]string) (map[string]string, error) {
	glog.Debugf("Enter CreateVolume(): name=%s, size=%s, options=%+v", name, size, options)
	para := make(map[string]interface{})
	para["name"] = name

	capacity, _ := strconv.Atoi(size)
	para["capacity"] = capacity * 1024
	/*
		para["dataPoolType"], _ = strconv.Atoi(options["dataPoolType"])
		para["dataPool"], _ = options["dataPool"]
		para["metaPool"], _ = options["metaPool"]
		para["thinAttribute "], _ = strconv.Atoi(options["thinAttribute "])
		para["threshold"], _ = strconv.Atoi(options["threshold"])
	*/
	for k, v := range options {
		if k == "thinAttribute" || k == "threshold" {
			para[k], _ = strconv.Atoi(v)
		} else {
			para[k] = v
		}

	}
	//get dataPoolType from dataPool name
	poolType, err := u.restApiWrapper.queryPoolTypeByName(para["dataPool"].(string))
	if err != nil {
		return nil, err
	}
	para["dataPoolType"] = poolType
	//if dataPoolType is 1(duplicate),then set metadataPool name equals dataPool name
	if poolType == POOL_TYPE_DUPLICATE {
		para["metaPool"] = para["dataPool"]
	}
	//check if volume exist
	lvms, err := u.restApiWrapper.queryLvm(para["dataPool"].(string), name)
	if err != nil {
		glog.Errorf("query Lvm error:%+v", err)
		return nil, err
	}
	if len(*lvms) == 0 {
		//volume not exist
		err = u.restApiWrapper.createVolume(para)
		if err != nil {
			glog.Errorf("create volume error:%+v", err)
			return nil, err
		}
		glog.Infof("create volume :%+v", para)
	} else {
		glog.Infof("volume already exists: poolName=%s,volumeName=%s", para["dataPool"].(string), name)
	}
	glog.Debugf("Exit CreateVolume().poolVolName=%s/%s", para["dataPool"], para["name"])

	info := map[string]string{}
	info["dataPool"] = para["dataPool"].(string)

	return info, nil
}

//DeleteVolume just delete the volume on the storage with given name
func (u *Storage13000Util) DeleteBlockVolume(volumeName string, options map[string]string) error {
	glog.Debugf("Enter DeleteBlockVolume(): volumeName=%s options=%+v", volumeName, options)
	//fmt.Printf("DeleteVolume volumeName:%s", volumeName)
	poolName, ok := options["dataPool"]
	if !ok {
		glog.Errorf("DeleteBlockVolume error: can not get dataPool from options%+v", options)
		return fmt.Errorf("DeleteBlockVolume error: can not get dataPool from options%+v", options)
	}
	//check if volume deleted
	lvms, err := u.restApiWrapper.queryLvm(poolName, volumeName)
	if err != nil {
		glog.Errorf("query Lvm error:%+v", err)
		return err
	}
	if len(*lvms) == 0 {
		//volume already deleted
		glog.Infof("volume already deleted: poolName=%s,volumeName=%s", poolName, volumeName)
		return nil
	}
	glog.Debugf("Exit DeleteBlockVolume(): poolName=%s, volumeName=%s", poolName, volumeName)
	return u.restApiWrapper.deleteVolume(poolName, volumeName)
}

func (u *Storage13000Util) CreateVolume(name string, size string, options map[string]string) (map[string]string, error) {
	switch options[storage.DevKind] {
	case "block":
		ret, err := u.CreateBlockVolume(name, size, options)
		if err != nil {
			return ret, err
		}

		ret[storage.DevKind] = "block"
		return ret, nil
	case "share":
		ret, err := u.CreateNFSShare(name, size, options)
		if err != nil {
			return ret, err
		}

		ret[storage.DevKind] = "share"
		return ret, nil
	}

	return nil, fmt.Errorf("device kind option is not valid %s not in (block, share)", options[storage.DevKind])
}

// CloneVolume create a volume with given name and size, together with specific option, or sourceVolumeName, or snapshotName
func (u *Storage13000Util) CloneVolume(name string, size string, options map[string]string, sourceVolumeName string, snapshotName string) (map[string]string, error) {
	return nil, fmt.Errorf("CloneVolume Unimplemented.")
}

//DeleteVolume just delete the volume on the storage with given name
func (u *Storage13000Util) DeleteVolume(name string, options map[string]string) error {
	switch options[storage.DevKind] {
	case "block":
		return u.DeleteBlockVolume(name, options)
	case "share":
		return u.DeleteNFSShare(name, options)
	}
	glog.Errorf("device kind option is not valid %s not in (block, share)", options[storage.DevKind])
	return fmt.Errorf("device kind option is not valid %s not in (block, share)", options[storage.DevKind])
}

func (u *Storage13000Util) ListVolume(maxEnties int32, startingToken string) ([]string, []int64, string, error) {
	return []string{}, []int64{}, "", fmt.Errorf("ListVolume Unimplemented.")
}

func (u *Storage13000Util) GetCapacity(options map[string]string) (int64, error) {
	return int64(-1), fmt.Errorf("GetCapacity Unimplemented.")
}

func (u *Storage13000Util) CreateSnapshot(sourceVolName string, snapshotName string) (bool, string, error) {
	return false, "", fmt.Errorf("CreateSnapshot Unimplemented.")
}

func (u *Storage13000Util) DeleteSnapshot(snapshotId string) error {
	return fmt.Errorf("DeleteSnapshot Unimplemented.")
}

func (u *Storage13000Util) ListSnapshots(maxEnties int32, startingToken string, sourceVolName string) ([]string, []string, string, error) {
	return []string{}, []string{}, "", fmt.Errorf("ListSnapshots Unimplemented.")
}

func (u *Storage13000Util) GetTargetNameWithHostInfo(host storage.HostInfo) (string, error) {
	glog.Debugf("Enter GetTargetNameWithHostInfo(): host=%+v", host)
	//fmt.Printf("host.Initiator = %s\n", host.Initiator)
	if host.Initiator != "" {
		//then do a exhaustive search
		dataTarget, err := u.restApiWrapper.queryTargetByIQN(host.Initiator)
		if err != nil {
			glog.Errorf("error: %+v", err)
			return "", err
		}

		//fmt.Printf("dataTarget:%+v\n", dataTarget)
		if dataTarget != nil {
			glog.Debugf("Exit GetTargetNameWithHostInfo(): targetName=%s", dataTarget.Name)
			return dataTarget.Name, nil
		} else {
			glog.Warningf("Exit GetTargetNameWithHostInfo(): find no target")
			return "", nil
		}

	} else {
		glog.Warningf("can not find target, because host initiator is empty. host:%+v", host)
		return "", fmt.Errorf("can not find target, because host initiator is empty. host:%+v", host)
	}

}

func (u *Storage13000Util) CreateTarget(host storage.HostInfo) (string, error) {
	glog.Debugf("Enter CreateTarget(): host=%+v", host)
	random := fmt.Sprintf("%08v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(100000000))
	//random := fmt.Sprintf("%d", 52080532) //for unit test
	targetName := fmt.Sprintf("target.k8s.%s-%s", host.Hostname, random)

	//fmt.Printf("targetName:%s", targetName)

	tgtName, err := u.restApiWrapper.createTarget(targetName)
	if err != nil {
		//fmt.Printf("createTarget error:%+v\n", err)
		return "", fmt.Errorf("createTarget error:%+v", err)
	}
	//fmt.Printf("tgtName:%s\n", tgtName)
	dataTarget, err := u.restApiWrapper.queryTargetByName(tgtName)
	if err != nil {
		//fmt.Printf("createTarget error queryTargetByName:%+v\n", err)
		return "", fmt.Errorf("createTarget error queryTargetByName:%+v", err)
	}
	err = u.restApiWrapper.addHost(tgtName, "ALL")
	if err != nil {
		//fmt.Printf("createTarget error addHost:%+v\n", err)
		return "", fmt.Errorf("createTarget error addHost:%+v", err)
	}

	err = u.restApiWrapper.bindIQN(tgtName, dataTarget.Node, host.Initiator)
	if err != nil {
		//fmt.Printf("createTarget error bindIQN:%+v\n", err)
		return "", fmt.Errorf("createTarget error bindIQN:%+v", err)
	}
	glog.Debugf("Exit CreateTarget(): tgtName=%s", tgtName)
	return tgtName, nil
}

func (u *Storage13000Util) BuildConnProperty(link string, lunID string, targetName string) (*storage.ConnProperty, error) {
	glog.Debugf("Enter BuildConnProperty(): link=%s, lunID=%s, targetName=%s", link, lunID, targetName)
	if link == storage.HostLinkiSCSI {

		portals, err := u.getPortal(targetName)
		if err != nil {
			return nil, err
		} else {
			count := len(portals)
			targets := make([]string, 0, count)
			for i := 0; i < count; i += 1 {
				targets = append(targets, targetName)
			}
			conProperty := &storage.ConnProperty{
				Protocol: storage.HostLinkiSCSI,
				Targets:  targets,
				Portals:  portals,
				LunIDs:   u.buildLunIDs(lunID, len(portals)),
			}
			glog.Debugf("Exit BuildConnProperty(): conPropery=%+v", conProperty)
			return conProperty, nil
		}
	} else {
		glog.Errorf("link %s not support", link)
		return nil, fmt.Errorf("link %s not support", link)
	}
}

func (u *Storage13000Util) mapVolumeToTarget(targetName string, poolName string, volumeName string, snap string) (string, error) {
	glog.Debugf("Enter mapVolumeToTarget(): targetName=%s, poolName=%s, volumeName=%s, snap=%s", targetName, poolName,
		volumeName, snap)
	lunID, err := u.restApiWrapper.getLunMappingID(targetName, poolName, volumeName)
	if err != nil {
		//fmt.Printf("mapVolumeToTarget error getLunMappingID:%+v", err)
		return "", err
	}
	if lunID != "" {
		return lunID, nil
	}
	//create volumeMapping
	err = u.restApiWrapper.createVolumeMapping(targetName, volumeName, poolName, "")
	if err != nil {
		return "", fmt.Errorf("createVolumeMapping error:%+v", err)
	}
	glog.Debugf("Exit mapVolumeToTarget().")
	return u.restApiWrapper.getLunMappingID(targetName, poolName, volumeName)
}

func (c *Storage13000Util) unMapVolumeFromTarget(targetName string, lunID string) error {
	return c.restApiWrapper.deleteVolumeMapping(targetName, lunID, true)
}

func (c *Storage13000Util) getPortal(targetName string) ([]string, error) {
	glog.Debugf("Enter getPortal(): targetName=%s", targetName)
	target, err := c.restApiWrapper.queryTargetByName(targetName)
	if err != nil {
		return nil, err
	}

	nodeArray := strings.Split(target.Node, ",")

	//fmt.Printf("nodeArray:%v\n", nodeArray)
	if len(nodeArray) < 1 {
		return nil, fmt.Errorf("getPortal error: the target with no nodes:targetName=%s", targetName)
	}

	portals := make([]string, 0, len(nodeArray))
	for _, node := range nodeArray {
		nodeGeneralInfo, err := c.restApiWrapper.queryNodeGeneralInfo(node)
		if err != nil {
			//fmt.Printf("getPortal warning:%+v", err)
			continue
		}
		portals = append(portals, fmt.Sprintf("%s:3260", nodeGeneralInfo.BusinessIp))
	}
	//fmt.Printf("protals:%v\n", portals)
	glog.Debugf("Exit getPortal(): portals=%+v", portals)
	return portals, nil
}
func (u *Storage13000Util) buildLunIDs(lunID string, count int) []string {
	lunIDs := make([]string, count)
	for i := 0; i < count; i += 1 {
		lunIDs[i] = lunID
	}

	return lunIDs
}
func (u *Storage13000Util) getPoolAndVolumeName(volumeMappingName string) (string, string, error) {
	if strings.TrimSpace(volumeMappingName) == "" {
		return "", "", fmt.Errorf("getPoolAndVolumeName error: volumeMappingName is empty. volumeMappingName=>%s", volumeMappingName)
	}

	index := strings.Index(volumeMappingName, "/")
	if index == -1 {

		glog.Infof("getPoolAndVolumeName():volumeMappingName=%s", volumeMappingName)

		return "", volumeMappingName, nil
	}

	poolName := (fmt.Sprintf(volumeMappingName[0:index]))
	volumeName := (fmt.Sprintf(volumeMappingName[index+1:]))

	//return names[0], names[1:], nil
	return poolName, volumeName, nil
}

func (u *Storage13000Util) generateNFSShareName(hintName string) string {
	if strings.HasPrefix(hintName, "pvc-") {
		hintName = hintName[len("pvc-"):]
	}

	hintName = strings.Replace(hintName, "-", "", -1)
	if len(hintName) > 32 {
		hintName = hintName[0:32]
	}

	return hintName
}

//CreateNFSShare create a NFS share on the storage base on the given name, size and other options
//function may return options as map, which will pass as options argument for DeleteNFSShare call.
func (u *Storage13000Util) CreateNFSShare(name string, size string, options map[string]string) (map[string]string, error) {
	pool, ok := options[storage.SharePoolName]
	if ok == false {
		glog.Errorf("the pool in which NFS share create should be set")
		return nil, fmt.Errorf("the pool in which NFS share create should be set")
	}

	client, ok := options[storage.ShareAccessClient]
	if ok == false {
		glog.Errorf("share access client ip info should be set")
		return nil, fmt.Errorf("share access client ip info should be set")
	}

	baseDir := "/" + pool
	baseDirDetail, err := u.restApiWrapper.getDirectoryDetail(baseDir)
	if err != nil {
		glog.Errorf("get base directory detail failed for %s", err)
		return nil, fmt.Errorf("get base directory detail failed for %s", err)
	}
	glog.Infof("base directory %s detail is %v", baseDir, baseDirDetail)

	dirName := u.generateNFSShareName(name)
	shareDir := baseDir + "/" + dirName
	//check if directory exist
	dirDetailInfo, err := u.restApiWrapper.getDirectoryDetail(shareDir)
	if err != nil {
		return nil, fmt.Errorf("get Directory Detail info failed for %s", err)
	}
	if dirDetailInfo == nil {
		//directory not exist
		if err := u.restApiWrapper.createDirectoryV2(dirName, baseDirDetail); err != nil {
			if err := u.restApiWrapper.createDirectory(dirName, baseDirDetail); err != nil {
				glog.Errorf("create share directory failed for %s", err)
				return nil, fmt.Errorf("create share directory failed for %s", err)
			}
		}
		glog.Infof("create directory: %s", dirName)
	}
	//check if NFS share exist
	shareInfo, err := u.restApiWrapper.getNFSShareInfo(shareDir)
	if err != nil {
		return nil, fmt.Errorf("get NFS share info failed for %s", err)
	}
	//share not exist
	if shareInfo == nil {
		if err := u.restApiWrapper.createNFSShare(shareDir, client); err != nil {
			glog.Errorf("create NFS share failed for %s", err)
			return nil, fmt.Errorf("create NFS share failed for %s", err)
		}
		glog.Infof("create NFS share: shareDir=%s, client=%+v", shareDir, client)
	}
	if err := u.restApiWrapper.setDirectoryQuota(shareDir, size); err != nil {
		glog.Errorf("set directory quota failed for %s", err)
		return nil, fmt.Errorf("set directory quota failed for %s", err)
	}

	ips := u.restApiWrapper.getClusterAccessIP()
	if len(ips) == 0 {
		glog.Errorf("no service ip is available")
		return nil, fmt.Errorf("no service ip is available")
	}

	info := map[string]string{}
	info["server"] = ips[0]
	info["path"] = shareDir
	glog.Debugf("NFS share created: %+v", info)
	return info, nil
}

func (u *Storage13000Util) removeClientList(shareDir string, detail map[string]interface{}) error {
	desire := make(map[string]interface{})
	desire["path"] = detail["path"]
	desire["pathAuthority"] = detail["pathAuthority"]
	desire["sync"] = detail["sync"]
	desire["editedClientList"] = []interface{}{}
	desire["addedClientList"] = []interface{}{}
	desire["deletedClientList"] = detail["clientList"]

	return u.restApiWrapper.updateNFSShare(shareDir, desire)
}

func (u *Storage13000Util) removeDirectoryRecursively(directory string) error {
	//first get all sub-directory
	lists, err := u.restApiWrapper.listSubDirectory(directory)
	if err != nil {
		return fmt.Errorf("list subdirectory failed for %s", err)
	}

	//then delete the sub-directory
	for _, item := range lists {
		if dirName, ok := item["name"].(string); ok == true {
			subDir := fmt.Sprintf("%s/%s", directory, dirName)
			if err := u.removeDirectoryRecursively(subDir); err != nil {
				return fmt.Errorf("remove subdirectory %s failed for %s", subDir, err)
			}
		}
	}

	//finally delete the directory itself
	return u.restApiWrapper.deleteDirectory(directory)
}

//DeleteNFSShare delete the NFS share from storage
func (u *Storage13000Util) DeleteNFSShare(name string, options map[string]string) error {
	shareDir := options["path"]
	info, err := u.restApiWrapper.getNFSShareInfo(shareDir)
	if err != nil {
		return fmt.Errorf("get NFS share info failed for %s", err)
	}
	//share maybe already deleted
	if info != nil {
		//first we need remove all the clientList from the NFS share
		if err := u.removeClientList(shareDir, info); err != nil {
			return fmt.Errorf("remove client list failed for %s", err)
		}
		if err := u.restApiWrapper.deleteNFSShare(shareDir); err != nil {
			return fmt.Errorf("delete NFS share failed for %s", err)
		}
	}

	//check whether the backend directory have already be deleted
	detail, err := u.restApiWrapper.getDirectoryDetail(shareDir)
	if err != nil {
		return fmt.Errorf("get directory detail failed %s", err)
	}
	if detail == nil {
		return nil
	}

	//delete the backend directory
	return u.restApiWrapper.deleteDirectory(shareDir)
}
