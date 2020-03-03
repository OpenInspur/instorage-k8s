package instorage

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/storage"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

//StorageUtil encapulate the general operation with a storage
type StorageUtil struct {
	cliWrapper CLIWrapper

	aaExtendFailedBarrierPath string
}

func NewStorUtil(cfg utils.StorageCfg) storage.IStorageOperater {
	return &StorageUtil{
		cliWrapper:                *NewCLIWrapper(cfg),
		aaExtendFailedBarrierPath: path.Join(cfg.BarrierPath, "aa-extend-failure.barrier"),
	}
}

func (u *StorageUtil) aaSpecificOptionCheck(volumeName string, options map[string]string) bool {
	if options == nil {
		rows, err := u.cliWrapper.lsvdisk("volume_name", volumeName)
		if err != nil {
			return true
		}

		if len(rows) == 4 {
			return true
		}
		return false
	}

	level, ok := options[storage.VolLevel]
	if ok {
		if level == "aa" {
			return true
		}
	}
	return false
}

//NeedFreezeFSWhenExtend check whether we need freeze the FS when do extend.
func (u *StorageUtil) NeedFreezeFSWhenExtend(volumeName string, options map[string]string) bool {
	//Now during active-active volume extend, no I/O can be happend, we need Freeze the file system.
	return u.aaSpecificOptionCheck(volumeName, options)
}

func (u *StorageUtil) buildPoolParameter(level string, options map[string]string) ([]string, error) {
	pool1, ok1 := options[storage.VolPoolName]
	pool2, ok2 := options[storage.VolAuxPoolName]
	if level == "aa" {
		if ok1 && ok2 {
			return []string{"-pool", fmt.Sprintf("%s:%s", pool1, pool2)}, nil
		} else {
			return nil, fmt.Errorf("two pool should be supplied for AA volume")
		}
	} else if level == "mirror" {
		if ok1 && ok2 {
			return []string{"-mdiskgrp", fmt.Sprintf("%s:%s", pool1, pool2)}, nil
		} else {
			return nil, fmt.Errorf("two pool should be supplied for mirror volume")
		}
	} else {
		//else level is basic
		if ok1 {
			return []string{"-mdiskgrp", pool1}, nil
		} else {
			return nil, fmt.Errorf("pool should be set when create volume")
		}
	}
}

func (u *StorageUtil) buildIOGrpParameter(level string, options map[string]string) ([]string, error) {
	iogrp1, ok1 := options[storage.VolIOGrp]
	iogrp2, ok2 := options[storage.VolAuxIOGrp]
	if level == "aa" {
		if ok1 && ok2 {
			return []string{"-iogrp", fmt.Sprintf("%s:%s", iogrp1, iogrp2)}, nil
		} else {
			return nil, fmt.Errorf("AA or Mirror volume should supply two iogrp")
		}
	} else {
		if ok1 {
			if level == "mirror" {
				return []string{"-accessiogrp", iogrp1, "-iogrp", iogrp1}, nil
			} else {
				return []string{"-iogrp", iogrp1}, nil
			}
		} else {
			return []string{}, nil
		}
	}
}

func (u *StorageUtil) buildThinParameter(level string, compressed bool, options map[string]string) ([]string, error) {
	parameters := []string{}

	resize, ok := options[storage.VolThinResize]
	if ok {
		p, e := strconv.Atoi(resize)
		if e != nil {
			return nil, fmt.Errorf("resize should be integer")
		}
		if p < 0 || p > 100 {
			return nil, fmt.Errorf("resize should be in the range [0, 100]")
		}
	} else {
		resize = "2"
	}
	if level == "aa" {
		parameters = append(parameters, "-buffersize", fmt.Sprintf("%s%%", resize))
	} else {
		parameters = append(parameters, "-rsize", fmt.Sprintf("%s%%", resize))
	}

	if warning, ok := options[storage.VolThinWarning]; ok {
		p, e := strconv.Atoi(warning)
		if e != nil {
			return nil, fmt.Errorf("warning should be integer")
		}
		if p <= 0 || p >= 100 {
			return nil, fmt.Errorf("warning should be in the range (0, 100)")
		}

		parameters = append(parameters, "-warning", fmt.Sprintf("%s%%", warning))
	}

	autoexpand := true
	if _, ok := options[storage.VolAutoExpand]; ok {
		var err error
		if autoexpand, err = strconv.ParseBool(options[storage.VolAutoExpand]); err != nil {
			return nil, fmt.Errorf("autoExpand should be bool %s", err)
		}
	}
	if level != "aa" && autoexpand {
		parameters = append(parameters, "-autoexpand")
	} else if level == "aa" && autoexpand == false {
		parameters = append(parameters, "-noautoexpand")
	}

	if compressed {
		parameters = append(parameters, "-compressed")
	} else {
		if level == "aa" {
			parameters = append(parameters, "-thin")
		}

		if grainsize, ok := options[storage.VolThinGrainSize]; ok {
			if grainsize != "32" && grainsize != "64" && grainsize != "128" && grainsize != "256" {
				return nil, fmt.Errorf("thin GrainSize can only be 32 or 64 or 128 or 256")
			}

			parameters = append(parameters, "-grainsize", grainsize)
		}
	}

	return parameters, nil
}

func (u *StorageUtil) buildInTierParameter(options map[string]string) ([]string, error) {
	parameters := []string{}

	var err error

	intier := false
	if _, ok := options[storage.VolInTier]; ok {
		if intier, err = strconv.ParseBool(options[storage.VolInTier]); err != nil {
			return nil, err
		}
	}

	if intier {
		parameters = append(parameters, "-intier", "on")
	} else {
		parameters = append(parameters, "-intier", "off")
	}

	return parameters, nil
}

func (u *StorageUtil) buildVolumeCreateParameter(level string, options map[string]string) ([]string, error) {
	parameters := []string{}
	var err error

	if level == "mirror" {
		parameters = append(parameters, "-copies", "2")
	}

	if ret, err := u.buildPoolParameter(level, options); err == nil {
		parameters = append(parameters, ret...)
	} else {
		return nil, err
	}

	if ret, err := u.buildIOGrpParameter(level, options); err == nil {
		parameters = append(parameters, ret...)
	} else {
		return nil, err
	}

	compress := false
	if _, ok := options[storage.VolCompress]; ok {
		if compress, err = strconv.ParseBool(options[storage.VolCompress]); err != nil {
			return nil, err
		}
	}

	thin := false
	if _, ok := options[storage.VolThin]; ok {
		if thin, err = strconv.ParseBool(options[storage.VolThin]); err != nil {
			return nil, err
		}
	}

	if thin || compress {
		if ret, err := u.buildThinParameter(level, compress, options); err != nil {
			return nil, err
		} else {
			parameters = append(parameters, ret...)
		}
	}

	if level != "aa" {
		if ret, err := u.buildInTierParameter(options); err != nil {
			return nil, err
		} else {
			parameters = append(parameters, ret...)
		}
	}

	return parameters, nil
}

//CreateVolume create a volume on the storage with given name, size and other options
func (u *StorageUtil) CreateVolume(name string, size string, options map[string]string) (map[string]string, error) {
	glog.Infof("CreateVolume, name: %s, size: %s", name, size)
	level, ok := options[storage.VolLevel]
	if ok {
		if level != "basic" && level != "mirror" && level != "aa" {
			return nil, fmt.Errorf("volume level should in basic, mirror, aa")
		}
	} else {
		level = "basic"
	}

	parameters, err := u.buildVolumeCreateParameter(level, options)
	if err != nil {
		return nil, err
	}

	id := ""
	if level == "aa" {
		id, err = u.cliWrapper.mkvolume(name, size, parameters)
	} else {
		id, err = u.cliWrapper.mkvdisk(name, size, parameters)
	}

	if err != nil {
		return nil, err
	}

	info := map[string]string{}
	info["id"] = id

	return info, nil
}

func (u *StorageUtil) CloneVolume(name string, size string, options map[string]string, sourceVolumeName string, snapshotName string) (map[string]string, error) {
	glog.Infof("CloneVolume, name: %s, size: %s, sourceVolumeName: %s, snapshotName: %s", name, size, sourceVolumeName, snapshotName)
	// 1 check options
	level, ok := options[storage.VolLevel]
	if ok {
		if level == "aa" {
			return nil, fmt.Errorf("CloneVolume, activeactive unsupported.")
		}
	}

	// 2 check sourceVolumeName and snapshotName
	if sourceVolumeName == "" && snapshotName == "" {
		return nil, fmt.Errorf("CloneVolume, sourceVolumeName == \"\" and snapshotName == \"\".")
	}

	// 3 create volume
	info, err := u.CreateVolume(name, size, options)
	if err != nil {
		return info, err
	}

	// 4 make lcmap
	var lcmapId string = ""
	var errMKLcmap error = nil
	if sourceVolumeName != "" {
		lcmapId, errMKLcmap = u.cliWrapper.mklcmap(sourceVolumeName, name, "100", "100", true)
	} else {
		lcmapId, errMKLcmap = u.cliWrapper.mklcmap(snapshotName, name, "100", "100", true)
	}
	if errMKLcmap != nil {
		u.cliWrapper.rmvdisk(name, true)
		return nil, errMKLcmap
	}

	// 5 start lcmap
	errStartLcmap := u.cliWrapper.startlcmap(lcmapId)
	if errStartLcmap != nil {
		u.cliWrapper.stoplcmap(lcmapId, true)
		u.cliWrapper.rmvdisk(name, true)
		return nil, errStartLcmap
	}

	return info, nil
}

//DeleteVolume just delete the volume on the storage with given name
func (u *StorageUtil) DeleteVolume(volumeName string, options map[string]string) error {
	// 1 get volume
	rows, err := u.cliWrapper.lsvdisk("volume_name", volumeName)
	if err != nil {
		return err
	}
	size := len(rows)
	if size == 0 {
		glog.Warningf("volume %s does not exist.", volumeName)
		return nil
	}
	if (size != 1 && size != 4) {
		return fmt.Errorf("volumes with volume name %s count invalid", volumeName)
	}
	
	
	// 2 check volume
	rows, err = u.cliWrapper.lslcmapEx("source_vdisk_name", volumeName)
	if err != nil {
		return err
	}
	lsmapSize := len(rows)
	if size == 1 {
		if lsmapSize != 0 {
			return fmt.Errorf("volume is source vdisk of other local maps")
		} else {
			return u.cliWrapper.rmvdisk(volumeName, true)
		}
	}
	if size == 4 {
		if lsmapSize > 1 {
			return fmt.Errorf("volume is source vdisk of other local maps")
		} else {
			return u.cliWrapper.rmvolume(volumeName, true)
		}
	}
	return nil
}

//ListVolume list volumes with a given maxEnties and a given startingToken
func (u *StorageUtil) ListVolume(maxEnties int32, startingToken string) ([]string, []int64, string, error) {
	var volumeNames []string
	var capacitiesBytes []int64
	var nextID string

	// 1 init maxCount and startingID
	var maxCount int32 = 64
	if maxEnties > 0 {
		maxCount = maxEnties
	}
	var startID = -1
	if startingToken != "" {
		startingTokenInt, err := strconv.Atoi(startingToken)
		if err != nil {
			glog.Warningf("ListVolume, fail to Atoi startingToken: %s.", startingToken)
			return volumeNames, capacitiesBytes, nextID, err
		}
		startID = startingTokenInt
	}

	// 2 query from storage
	rows, err := u.cliWrapper.lsvdiskEx()
	if err != nil {
		return volumeNames, capacitiesBytes, nextID, err
	}
	if len(rows) == 0 {
		return volumeNames, capacitiesBytes, nextID, nil
	}
	for _, row := range rows {
		id, err := strconv.Atoi(row["id"])
		if err != nil {
			glog.Warningf("ListVolume, fail to ParseInt volumeID: %s.", row["id"])
			continue
		}
		if id < startID {
			continue
		}
		if int32(len(volumeNames)) < maxCount {
			_, capacity := u.parseStrSize(row["capacity"])
			volumeNames = append(volumeNames, row["name"])
			capacitiesBytes = append(capacitiesBytes, capacity)
		} else if int32(len(volumeNames)) == maxCount {
			nextID = row["id"]
			break
		}
	}
	return volumeNames, capacitiesBytes, nextID, nil
}

func (u *StorageUtil) GetCapacity(options map[string]string) (int64, error) {
	var availableCapacity int64 = -1
	poolName, ok := options[storage.VolPoolName]
	if ok == false {
		return availableCapacity, fmt.Errorf("GetCapacity should supply one pool.")
	}

	rows, err := u.cliWrapper.lsmdiskgrp("name", poolName)
	if err != nil {
		return availableCapacity, err
	}
	if len(rows) == 1 {
		_, availableCapacity := u.parseStrSize(rows[0]["free_capacity"])
		return availableCapacity * 1024 * 1024 * 1024, nil
	} else {
		return availableCapacity, fmt.Errorf("GetCapacity cann't find the pool: %s.", poolName)
	}
}

func (u *StorageUtil) CreateSnapshot(sourceVolName string, snapshotName string) (bool, string, error) {
	// 0 check snapshot
	rows, err := u.cliWrapper.lslcmapEx("target_vdisk_name", snapshotName)
	if err != nil {
		return false, "", err
	}
	if len(rows) != 0 {
		snapshot := rows[0]
		if snapshot["source_vdisk_name"] == sourceVolName {
			id := snapshot["id"]
			// status := snapshot["status"]
			startTime := snapshot["start_time"]
			if /*status == "idle_or_copied" && */ startTime != "" {
				return true, startTime, nil
			} else {
				u.cliWrapper.startlcmap(id)
				return false, startTime, nil
			}
		}
	}

	// 1  get source volume
	rows, err = u.cliWrapper.lsvdisk("volume_name", sourceVolName)
	if err != nil {
		return false, "", err
	}
	switch len(rows) {
	case 0:
		return false, "", fmt.Errorf("volume %s does not exist.", sourceVolName)
	case 1:
		break
	case 4:
		return false, "", fmt.Errorf("volume %s is aa, unsupported.", sourceVolName)
	default:
		return false, "", fmt.Errorf("volumes with volume name %s count invalid", sourceVolName)
	}

	// 2 create target volume
	rows, err = u.cliWrapper.lsvdiskDetail(sourceVolName)
	if err != nil {
		return false, "", err
	}
	row_count := len(rows)
	if row_count != 2 && row_count != 3 {
		return false, "", fmt.Errorf("row_count of volume %s is not 2 or 3.", sourceVolName)
	}
	options := make(map[string]string)
	options[storage.VolLevel] = "basic"
	options[storage.VolIOGrp] = rows[0]["IO_group_name"]
	options[storage.VolPoolName] = rows[1]["mdisk_grp_id"]
	options[storage.VolAutoExpand] = "true"
	options[storage.VolThin] = "true"
	options[storage.VolThinResize] = "0"
	options[storage.VolInTier] = "true"
	size, _ := u.parseStrSize(rows[0]["capacity"])
	_, errCreateVolume := u.CreateVolume(snapshotName, size, options)
	if errCreateVolume != nil {
		return false, "", errCreateVolume
	}

	// 3 make lcmap
	lcmapId, errMKLcmap := u.cliWrapper.mklcmap(sourceVolName, snapshotName, "0", "0", false)
	if errMKLcmap != nil {
		u.cliWrapper.rmvdisk(snapshotName, true)
		return false, "", errMKLcmap
	}

	// 4 start lcmap
	errStartLcmap := u.cliWrapper.startlcmap(lcmapId)
	if errStartLcmap != nil {
		u.cliWrapper.rmlcmap(lcmapId, true)
		u.cliWrapper.rmvdisk(snapshotName, true)
		return false, "", errStartLcmap
	}

	// 5 get createTime
	rows, err = u.cliWrapper.lslcmapEx("target_vdisk_name", snapshotName)
	if err != nil {
		u.cliWrapper.stoplcmap(lcmapId, true)
		u.cliWrapper.rmlcmap(lcmapId, true)
		u.cliWrapper.rmvdisk(snapshotName, true)
		return false, "", err
	}
	startTime := ""
	if len(rows) != 0 {
		snapshot := rows[0]
		if snapshot["source_vdisk_name"] == sourceVolName {
			startTime = snapshot["start_time"]
		} else {
			u.cliWrapper.stoplcmap(lcmapId, true)
			u.cliWrapper.rmlcmap(lcmapId, true)
			u.cliWrapper.rmvdisk(snapshotName, true)
		}
	}

	return true, startTime, nil
}

func (u *StorageUtil) DeleteSnapshot(snapshotId string) error {
	// 1  get snapshot
	rows, err := u.cliWrapper.lslcmapEx("target_vdisk_name", snapshotId)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		glog.Warningf("snapshot %s does not exist.", snapshotId)
		return nil
	}
	snapshot := rows[0]
	
	// 2 check snapshot
	rows, err = u.cliWrapper.lslcmapEx("source_vdisk_name", snapshotId)
	if err != nil {
		return err
	}
	if len(rows) != 0 {
		return fmt.Errorf("snapshot is source volume of other local maps")
	}

	// 3 stop snapshot
	u.cliWrapper.stoplcmap(snapshot["id"], true)

	// 4 query snapshot
	timeout := 20
	for i := 0; i < timeout; i++ {
		time.Sleep(time.Duration(3) * time.Second)
		rows, err = u.cliWrapper.lslcmapEx("target_vdisk_name", snapshotId)
		if err != nil {
			glog.Warningf("DeleteSnapshot, fail to lslcmapEx, %s.", err.Error())
			break
		}
		if len(rows) == 0 {
			glog.Warningf("DeleteSnapshot, snapshot does not exist, snapshotId: %s", snapshotId)
			break
		}
		snapshot = rows[0]
		if snapshot["status"] == "idle_or_copied" || snapshot["status"] == "stopped" {
			break
		}
	}

	// 5 remove lcmap and vdisk
	u.cliWrapper.rmlcmap(snapshot["id"], true)
	return u.cliWrapper.rmvdisk(snapshotId, true)
}

func (u *StorageUtil) ListSnapshots(maxEnties int32, startingToken string, sourceVolName string) ([]string, []string, string, error) {
	var snapshotvolNames []string
	var sourcevolNames []string
	var nextID string
	var rows []CLIRow
	var err error

	// 1 init maxCount and startingID
	var maxCount int32 = 64
	if maxEnties > 0 {
		maxCount = maxEnties
	}
	var startID = -1
	if startingToken != "" {
		startingTokenInt, err := strconv.Atoi(startingToken)
		if err != nil {
			glog.Warningf("ListSnapshots, fail to Atoi startingToken: %s.", startingToken)
			return snapshotvolNames, sourcevolNames, nextID, err
		}
		startID = startingTokenInt
	}

	// 2 query from storage
	if sourceVolName == "" {
		rows, err = u.cliWrapper.lslcmap()
	} else {
		rows, err = u.cliWrapper.lslcmapEx("source_vdisk_name", sourceVolName)
	}

	if err != nil {
		return snapshotvolNames, sourcevolNames, nextID, err
	}
	if len(rows) == 0 {
		return snapshotvolNames, sourcevolNames, nextID, nil
	}
	for _, row := range rows {
		id, err := strconv.Atoi(row["id"])
		if err != nil {
			glog.Warningf("ListSnapshots, fail to ParseInt lcmapID: %s.", row["id"])
			continue
		}
		if id < startID {
			continue
		}
		if int32(len(snapshotvolNames)) < maxCount {
			snapshotvolNames = append(snapshotvolNames, row["target_vdisk_name"])
			sourcevolNames = append(sourcevolNames, row["source_vdisk_name"])
		} else if int32(len(snapshotvolNames)) == maxCount {
			nextID = row["id"]
			break
		}
	}
	return snapshotvolNames, sourcevolNames, nextID, nil
}

func (u *StorageUtil) parseStrSize(capacity string) (string, int64) {
	reg := regexp.MustCompile(`[A-Z]+`)
	unit := reg.FindString(capacity)
	bareCapacity := strings.Replace(capacity, unit, "", -1)
	sizeOrignal, _ := strconv.ParseFloat(bareCapacity, 64)
	size := int64(sizeOrignal)

	var unitKB int64 = 1024
	unitMB := 1024 * unitKB
	unitGB := 1024 * unitMB

	var sizeGB int64
	switch unit {
	case "B", "":
		sizeGB = size / unitGB
	case "KB":
		sizeGB = size / unitMB
	case "MB":
		sizeGB = size / unitKB
	case "GB":
		sizeGB = size
	case "TB":
		sizeGB = size * unitKB
	case "PB":
		sizeGB = size * unitMB
	case "EB":
		sizeGB = size * unitGB
	}

	return strconv.FormatInt(sizeGB, 10), sizeGB
}

type volumeInfo struct {
	iogrpName  string
	poolName   string
	vdiskID    string
	vdiskName  string
	vdiskSize  string
	cvName     string
	formatting string
	lcMapCount string
}

func (u *StorageUtil) buildVolumeInfos(volumeName string) (*volumeInfo, *volumeInfo, error) {
	var master *volumeInfo
	var aux *volumeInfo
	var masterCV *volumeInfo
	var auxCV *volumeInfo
	var err error

	rows, err := u.cliWrapper.lsvdisk("volume_name", volumeName)
	if err != nil {
		return nil, nil, err
	}

	for _, row := range rows {
		volInfo := &volumeInfo{
			iogrpName:  row["IO_group_name"],
			poolName:   row["mdisk_grp_name"],
			vdiskID:    row["id"],
			vdiskName:  row["name"],
			vdiskSize:  row["capacity"],
			formatting: row["formatting"],
			lcMapCount: row["lc_map_count"],
		}

		switch row["function"] {
		case "master", "":
			master = volInfo
		case "aux":
			aux = volInfo
		case "master_change":
			masterCV = volInfo
		case "aux_change":
			auxCV = volInfo
		}
	}

	if master == nil {
		err = fmt.Errorf("volume %s does not exist", volumeName)
	}

	if aux != nil {
		master.cvName = masterCV.vdiskName
		aux.cvName = auxCV.vdiskName
	}

	return master, aux, err
}

func (u *StorageUtil) canVolumeBeExtend(master *volumeInfo, aux *volumeInfo, addtionSize int64) (bool, error) {
	//A formatting volume can not be extend.
	if master != nil && master.formatting == "yes" {
		return false, fmt.Errorf("volume %s is formatting, can not extend", master.vdiskName)
	}

        //get master pool free capacity
	masterPools, err := u.cliWrapper.lsmdiskgrp("name", master.poolName)
	if err != nil || masterPools == nil || len(masterPools) == 0 {
		return false, fmt.Errorf("can not get pool %s capacity for %s", master.poolName, err)
	}
	_, masterFreeCapacity := u.parseStrSize(masterPools[0]["free_capacity"])

	if aux != nil {
		if aux.formatting == "yes" {
			return false, fmt.Errorf("volume %s is formatting, can not extend", aux.vdiskName)
		}

		auxPools, err := u.cliWrapper.lsmdiskgrp("name", aux.poolName)
		if err != nil || auxPools == nil || len(auxPools) == 0 {
			return false, fmt.Errorf("can not get pool %s capacity for %s", aux.poolName, err)
		}
		_, auxFreeCapacity := u.parseStrSize(auxPools[0]["free_capacity"])

		//An active-active volume has 2 local copy, from and to the change volume,
		//so if local copy not equal 2, they must have other relationship, they can not be extend.
		errPattern := "active-active volume has only 2 local copy, volume %s havs %s local copy, can not extend"
		if master.lcMapCount != "2" {
			return false, fmt.Errorf(errPattern, master.vdiskName, master.lcMapCount)
		}
		if aux.lcMapCount != "2" {
			return false, fmt.Errorf(errPattern, aux.vdiskName, aux.lcMapCount)
		}

		if masterFreeCapacity < (addtionSize * 2) {
			return false, fmt.Errorf("master pool free capacity is not enough for extend available %d need %d", masterFreeCapacity, addtionSize * 2)
		}
		if auxFreeCapacity < (addtionSize * 2) {
			return false, fmt.Errorf("aux pool free capacity is not enough for extend available %d need %d", auxFreeCapacity, addtionSize * 2)
		}
	} else {
		//A general volume with lcmap can not be extend.
		if master.lcMapCount != "0" {
			return false, fmt.Errorf("volume %s has local copy, can not extend", master.vdiskName)
		}

		if masterFreeCapacity < addtionSize {
			return false, fmt.Errorf("pool free capacity is not enough for extend available %d need %d", masterFreeCapacity, addtionSize)
		}
	}

	return true, nil
}

//ExtendVolume extend the give volume to the specific size
func (u *StorageUtil) ExtendVolume(volumeName string, size string, options map[string]string) error {
	if u.aaExtendFailedBarrierExistCheck() {
		return fmt.Errorf("active-active volume extend failed barrier exist")
	}

	master, aux, err := u.buildVolumeInfos(volumeName)
	if err != nil {
		return fmt.Errorf("build volume info failed for %s", err)
	}

	newSize, _ := strconv.ParseInt(size, 10, 64)
	_, oldSize := u.parseStrSize(master.vdiskSize)
	if newSize == oldSize {
		return nil
	}
	if newSize < oldSize {
		return fmt.Errorf("new size (%dGB) should large than old size (%dGB)", newSize, oldSize)
	}

	additionSize := newSize - oldSize
	addition := strconv.FormatInt(additionSize, 10)

	//TODO better do a pre-check before actually expand to circumvent meaningless action.
	canDo, err := u.canVolumeBeExtend(master, aux, additionSize)
	if err != nil || canDo == false {
		return fmt.Errorf("volume %s extend check failed for err(%v) and can extend(%v)", master.vdiskName, err, canDo)
	}

	if aux == nil {
		//extend a general volume
		return u.cliWrapper.expandvdisksize(master.vdiskName, addition)
	}

	//extend an active-active volume
	var clusterName string
	if rows, err := u.cliWrapper.lssystem(); err == nil {
		clusterName = rows[0]["name"]
	} else {
		return fmt.Errorf("get cluster name failed")
	}

	if err := u.extendActiveActiveVolumeV2(master, aux, clusterName, addition); err != nil {
		u.createAAExtendFailedBarrier(volumeName, err)
		return err
	}

	return nil
}

func (u *StorageUtil) aaExtendFailedBarrierExistCheck() bool {
	f, err := os.Open(u.aaExtendFailedBarrierPath)
	if err != nil {
		return false
	}
	f.Close()

	glog.Errorf("AA volume extend failed barrier %s exist, please do recovery the volume on storage system and then remove the barrier.", u.aaExtendFailedBarrierPath)
	return true
}

func (u *StorageUtil) createAAExtendFailedBarrier(volumeName string, err error) {
	content := strings.Join([]string{
		"!!!!!! ATTENTION !!!!!!",
		fmt.Sprintf("Extend Active-Active volume %s failed for %s.", volumeName, err),
		"Read the log file to find what we have done and which step went wrong.",
		"Then correct the active-active volume on storage system.",
		"After volume corrected, remove this file, and operation such as extend/attach/detach can continue.",
	}, "\n")

	f, err := os.Create(u.aaExtendFailedBarrierPath)
	if err != nil {
		glog.Errorf("generate barrier %s failed for %s", u.aaExtendFailedBarrierPath, err)
		return
	}

	f.WriteString(content)
	f.Close()
}

func (u *StorageUtil) extendActiveActiveVolumeV2(master *volumeInfo, aux *volumeInfo, clusterName string, addition string) error {
	type task struct {
		fun  func([]string) error
		args []string
		err  string
		desc string
	}

	//RC relationship id is same as master volume id.
	rcID := master.vdiskID

	tasks := []task{
		task{
			fun:  u.cliWrapper.rmrcrelationshipList,
			args: []string{rcID},
			err:  fmt.Sprintf("rmrcrelationship %s between %s -> %s failed for %%s", rcID, master.vdiskName, aux.vdiskName),
			desc: fmt.Sprintf("rmrcrelationship %s between %s -> %s", rcID, master.vdiskName, aux.vdiskName),
		},
		task{
			fun:  u.cliWrapper.expandvdisksizeList,
			args: []string{master.vdiskName, addition},
			err:  fmt.Sprintf("expand master %s failed %%s", master.vdiskName),
			desc: fmt.Sprintf("expand master %s", master.vdiskName),
		},
		task{
			fun:  u.cliWrapper.expandvdisksizeList,
			args: []string{aux.vdiskName, addition},
			err:  fmt.Sprintf("expand aux %s failed %%s", aux.vdiskName),
			desc: fmt.Sprintf("expand aux %s", aux.vdiskName),
		},
		task{
			fun:  u.cliWrapper.expandvdisksizeList,
			args: []string{master.cvName, addition},
			err:  fmt.Sprintf("expand master change volume %s failed %%s", master.cvName),
			desc: fmt.Sprintf("expand master change volume %s", master.cvName),
		},
		task{
			fun:  u.cliWrapper.expandvdisksizeList,
			args: []string{aux.cvName, addition},
			err:  fmt.Sprintf("expand aux change volume %s failed %%s", aux.cvName),
			desc: fmt.Sprintf("expand aux change volume %s", aux.cvName),
		},
		task{
			fun:  u.cliWrapper.mkrcrelationshipList,
			args: []string{master.vdiskName, aux.vdiskName, clusterName, "activeactive"},
			err:  fmt.Sprintf("mkrcrelationship in %s between %s -> %s failed for %%s", clusterName, master.vdiskName, aux.vdiskName),
			desc: fmt.Sprintf("mkrcrelationship in %s between %s -> %s", clusterName, master.vdiskName, aux.vdiskName),
		},
		task{
			fun:  u.cliWrapper.addvdiskaccessList,
			args: []string{aux.iogrpName, master.vdiskName},
			err:  fmt.Sprintf("addvdiskaccess for %s in %s failed for %%s", master.vdiskName, aux.iogrpName),
			desc: fmt.Sprintf("addvdiskaccess for %s in %s", master.vdiskName, aux.iogrpName),
		},
		task{
			fun:  u.cliWrapper.chrcrelationshipList,
			args: []string{"master", master.cvName, rcID},
			err:  fmt.Sprintf("chrcrelationship for master %s in relationshp %s failed %%s", master.cvName, rcID),
			desc: fmt.Sprintf("chrcrelationship for master %s in relationshp %s", master.cvName, rcID),
		},
		task{
			fun:  u.cliWrapper.chrcrelationshipList,
			args: []string{"aux", aux.cvName, rcID},
			err:  fmt.Sprintf("chrcrelationship for aux %s in relationshp %s failed %%s", aux.cvName, rcID),
			desc: fmt.Sprintf("chrcrelationship for aux %s in relationshp %s", aux.cvName, rcID),
		},
	}

	glog.Infof("Start expanding active-active volume {m: %s, a: %s, mCV: %s, aCV: %s, rcID: %s}", master.vdiskName, aux.vdiskName, master.cvName, aux.cvName, rcID)

	for idx, task := range tasks {
		glog.Debugf("Expand active-active volume %s, step%d %s.", master.vdiskName, idx, task.desc)
		if err := task.fun(task.args); err != nil {
			glog.Errorf("Expand active-active volume %s failed, step%d %s failed for %s.", master.vdiskName, idx, task.desc, err)

			for i := idx + 1; i < len(tasks); i++ {
				glog.Errorf("Expand active-active volume %s failed, step%d %s need to complete.", master.vdiskName, i, tasks[i].desc)
			}

			newErr := fmt.Errorf(task.err, err)
			glog.Errorf("Failed expanding active-active volume {m: %s, a: %s, mCV: %s, aCV: %s, rcID: %s} from %s.", master.vdiskName, aux.vdiskName, master.cvName, aux.cvName, rcID, newErr)

			return newErr
		}
		glog.Infof("Expand active-active volume %s, step%d %s success.", master.vdiskName, idx, task.desc)
	}

	glog.Infof("Success expanding active-active volume {m: %s, a: %s, mCV: %s, aCV: %s, rcID: %s}.", master.vdiskName, aux.vdiskName, master.cvName, aux.cvName, rcID)
	return nil
}

//GetVolumeNameWithUID get the underlay volume name base on the UID.
func (u *StorageUtil) GetVolumeNameWithUID(uid string) (string, error) {
	//first get the volume name with uid
	rows, err := u.cliWrapper.lsvdisk("vdisk_UID", uid)
	if err != nil {
		return "", err
	}

	if len(rows) == 0 {
		return "", fmt.Errorf("volume with uid %s does not exist", uid)
	}

	return rows[0]["name"], nil
}

func (u *StorageUtil) getHostWithHostInfo(host storage.HostInfo) (string, error) {
	var lastErr error
	//check host by WWPNs information
	if len(host.WWPNs) != 0 {
		for _, wwpn := range host.WWPNs {
			wwpnRows, err := u.cliWrapper.lsfabric(wwpn, "")
			if err != nil {
				glog.Warningf("lsfabric with %s failed %s", wwpn, err)
				lastErr = err
				continue
			}

			for _, row := range wwpnRows {
				if row["remote_wwpn"] != "" && row["name"] != "" && strings.ToLower(row["remote_wwpn"]) == strings.ToLower(wwpn) {
					glog.Infof("find host with target wwpn %s", row["name"])
					return row["name"], nil
				}
			}
		}
	}

	//check host by initiator
	if host.Initiator != "" {
		//then do a exhaustive search
		allHostRows, err := u.cliWrapper.lshost("")
		if err != nil {
			return "", err
		}

		for _, allHostRow := range allHostRows {
			limitedHostRows, err := u.cliWrapper.lshost(allHostRow["name"])
			if err != nil {
				glog.Warningf("lshost with %s failed %s", allHostRow["name"], err)
				lastErr = err
				continue
			}

			for _, limitedHostRow := range limitedHostRows {
				if limitedHostRow["iscsi_name"] == host.Initiator {
					return allHostRow["name"], nil
				}
			}
		}
	}

	//check whether we have the same name
	if host.Hostname != "" {
		hostRows, err := u.cliWrapper.lshost(host.Hostname)
		if err != nil {
			return "", err
		}
		if len(hostRows) > 0 {
			return hostRows[0]["name"], nil
		}
	}

	if lastErr != nil {
		glog.Errorf("search host come across error, and not found wanted host")
		return "", fmt.Errorf("search host come across error %s", lastErr)
	} else {
		glog.Warningf("does not find a matched host with give host info %s", host)
		return "", nil
	}
}

func (u *StorageUtil) createHost(hostInfo storage.HostInfo, siteName string) (string, error) {
	//host name is decided, do not try to add prefix or suffix
	hostname := hostInfo.Hostname

	if hostInfo.Initiator == "" && len(hostInfo.WWPNs) == 0 {
		return "", fmt.Errorf("neither initiator nor wwpn specified")
	}

	if hostInfo.Initiator != "" {
		if err := u.cliWrapper.mkhost(hostname, "-iscsiname", hostInfo.Initiator, siteName); err != nil {
			return "", err
		}
	} else {
		if err := u.cliWrapper.mkhost(hostname, "-hbawwpn", hostInfo.WWPNs[0], siteName); err != nil {
			return "", err
		}

		hostInfo.WWPNs = hostInfo.WWPNs[1:]
	}

	for _, wwpn := range hostInfo.WWPNs {
		if err := u.cliWrapper.addhostport(hostname, "-hbawwpn", wwpn); err != nil {
			u.deleteHost(hostname)
			return "", fmt.Errorf("bind wwpn %s to host %s failed %s", wwpn, hostname, err)
		}
	}

	return hostname, nil
}

func (u *StorageUtil) deleteHost(hostname string) error {
	return u.cliWrapper.rmhost(hostname)
}

func (u *StorageUtil) mapVolumeToHost(volumeName string, hostName string, multihostmap bool) (map[string]string, error) {
	lunIDMap, err := u.getLunID(volumeName, hostName)
	if err == nil && lunIDMap != nil {
		return lunIDMap, nil
	}

	err = u.cliWrapper.mkvdiskhostmap(hostName, volumeName)
	if err != nil {
		return nil, err
	}

	return u.getLunID(volumeName, hostName)
}

//UnmapVolumeFromHost remove the map of volumeName to hostName,
//if hostName is not set and the volume only mapped to one host, the only map will be removed
//if hostName is not set and the volume has mapped to more than one host, err will be return
//else it will
func (u *StorageUtil) UnmapVolumeFromHost(volumeName, hostName string) error {
	mapRows, err := u.cliWrapper.lsvdiskhostmap(volumeName)
	if err != nil {
		return fmt.Errorf("lsvdiskhostmap failed %s", err)
	}
	if hostName == "" && len(mapRows) > 1 {
		return fmt.Errorf("volume mapped to multihost, need specify a host to unmap")
	}

	foundHost := false
	for _, row := range mapRows {
		if hostName == "" || row["host_name"] == hostName {
			hostName = row["host_name"]
			foundHost = true
			break
		}
	}

	if foundHost {
		return u.cliWrapper.rmvdiskhostmap(hostName, volumeName)
	}

	return nil
}

func (u *StorageUtil) volumesMappedToHost(hostName string) ([]string, error) {
	mapRows, err := u.cliWrapper.lshostvdiskmap(hostName)
	if err != nil {
		return nil, fmt.Errorf("lshostvdiskmap failed %s", err)
	}

	volumes := []string{}
	for _, row := range mapRows {
		volumes = append(volumes, row["vdisk_name"])
	}

	return volumes, nil
}

// getLunID get the SCSI ID the volume mapped to the given host.
func (u *StorageUtil) getLunID(volume string, host string) (map[string]string, error) {
	mapRows, err := u.cliWrapper.lsvdiskhostmap(volume)
	if err != nil {
		return nil, fmt.Errorf("lsvdiskhostmap failed %s", err)
	}

	lunID := make(map[string]string)
	for _, row := range mapRows {
		if row["host_name"] == host {
			lunID[row["IO_group_id"]] = row["SCSI_id"]
		}
	}

	if len(lunID) > 0 {
		return lunID, nil
	}

	return nil, nil
}

func (u *StorageUtil) getiSCSITargets(lunIDMap map[string]string) ([]string, []string, []string, error) {
	ipRows, err := u.cliWrapper.lsportip()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("lsportip failed")
	}

	nodeRows, err := u.cliWrapper.lsnode()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("lsnode failed")
	}

	targets := make([]string, 0, len(ipRows))
	portals := make([]string, 0, len(ipRows))
	luns := make([]string, 0, len(ipRows))
	for _, ipRow := range ipRows {
		ipState := ipRow["state"]
		ipv4 := ipRow["IP_address"]
		ipv6 := ipRow["IP_address_6"]
		if ((ipState == "configured" && ipRow["link_state"] == "active") || ipState == "online") && (ipv4 != "" || ipv6 != "") {
			for _, nodeRow := range nodeRows {
				if nodeRow["status"] == "online" && ipRow["node_id"] == nodeRow["id"] {
					portal := ""
					if ipv4 != "" {
						portal = fmt.Sprintf("%s:3260", ipv4)
					} else if ipv6 != "" {
						portal = fmt.Sprintf("[%s]:3260", ipv6)
					}

					portals = append(portals, portal)
					targets = append(targets, nodeRow["iscsi_name"])
					luns = append(luns, lunIDMap[nodeRow["IO_group_id"]])
				}
			}
		}
	}

	return targets, portals, luns, nil
}

func (u *StorageUtil) getFCTargets(lunIDMap map[string]string, hostname string) ([]string, []string, error) {
	nodeMap := make(map[string]string)
	nodeRows, err := u.cliWrapper.lsnode()
	if err != nil {
		return nil, nil, err
	}
	for _, row := range nodeRows {
		nodeMap[row["name"]] = row["IO_group_id"]
	}

	WWPNs := []string{}
	LUNs := []string{}

	//first check can we only return the visibility port to that host
	//port include NPIV port
	wwpnRows, err := u.cliWrapper.lsfabric("", hostname)
	if err != nil {
		glog.Warningf("lsfabric for %s failed %s", hostname, err)
	}

	for _, row := range wwpnRows {
		if len(row["local_wwpn"]) > 0 {
			exist := false
			for _, wwpn := range WWPNs {
				if wwpn == row["local_wwpn"] {
					exist = true
					break
				}
			}
			if exist == false {
				WWPNs = append(WWPNs, row["local_wwpn"])
				LUNs = append(LUNs, lunIDMap[nodeMap[row["node_name"]]])
			}
		}
	}

	if len(WWPNs) > 0 {
		return WWPNs, LUNs, nil
	}

	//if not, we will return all the port number
	//however lsportfc not return the NPIV port
	wwpnRows, err = u.cliWrapper.lsportfc()
	if err != nil {
		return nil, nil, err
	}

	for _, row := range wwpnRows {
		if row["status"] == "active" {
			WWPNs = append(WWPNs, row["WWPN"])
			LUNs = append(LUNs, lunIDMap[nodeMap[row["node_name"]]])
		}
	}

	return WWPNs, LUNs, nil
}

func (u *StorageUtil) buildConnProperty(link string, lunIDMap map[string]string, hostname string) (*storage.ConnProperty, error) {
	if link == storage.HostLinkFC {
		wwpns, luns, err := u.getFCTargets(lunIDMap, hostname)
		if err != nil {
			return nil, err
		}

		return &storage.ConnProperty{
			Protocol: storage.HostLinkFC,
			WWPNs:    wwpns,
			LunIDs:   luns,
		}, nil
	} else if link == storage.HostLinkiSCSI {
		targets, portals, luns, err := u.getiSCSITargets(lunIDMap)
		if err != nil {
			return nil, err
		}

		return &storage.ConnProperty{
			Protocol: storage.HostLinkiSCSI,
			Targets:  targets,
			Portals:  portals,
			LunIDs:   luns,
		}, nil
	} else {
		return nil, fmt.Errorf("link %s not support", link)
	}
}

func (u *StorageUtil) getVolumePreferSiteName(volumeName string) (string, error) {
	preferIOGrpName := ""

	vdisks, err := u.cliWrapper.lsvdisk("volume_name", volumeName)
	if err != nil {
		return "", err
	}
	for _, vdisk := range vdisks {
		if vdisk["function"] == "master" {
			preferIOGrpName = vdisk["IO_group_name"]
			break
		}
	}
	if preferIOGrpName == "" {
		return "", nil
	}

	iogrps, err := u.cliWrapper.lsiogrp()
	if err != nil {
		return "", err
	}
	for _, iogrp := range iogrps {
		if iogrp["name"] == preferIOGrpName {
			return iogrp["site_name"], nil
		}
	}

	return "", fmt.Errorf("can not found suitable site for aa volume")
}

// AttachVolume search the host with hostInfo or create a new one if necessary, and map the volume to host.
func (u *StorageUtil) AttachVolume(volumeName string, hostInfo storage.HostInfo, options map[string]string) (*storage.ConnProperty, error) {
	//first attach volume
	hostName, err := u.getHostWithHostInfo(hostInfo)
	if err != nil {
		return nil, err
	}

	if hostName == "" {
		glog.Infof("Host does not exist, try to create the host.")

		siteName := ""
		if siteName, err = u.getVolumePreferSiteName(volumeName); err != nil {
			return nil, err
		}

		if hostName, err = u.createHost(hostInfo, siteName); err != nil {
			return nil, err
		}
	}

	lunIDMap, err := u.mapVolumeToHost(volumeName, hostName, true)
	if err != nil {
		return nil, err
	}

	return u.buildConnProperty(hostInfo.Link, lunIDMap, hostName)
}

// GetVolumeAttachInfo get the connect info of the volume mapped to host.
func (u *StorageUtil) GetVolumeAttachInfo(volumeName string, hostInfo storage.HostInfo) (*storage.ConnProperty, error) {
	//volume already attached, just build the attach info
	hostName, err := u.getHostWithHostInfo(hostInfo)
	if err != nil {
		return nil, err
	}
	//no host info exist, volume not attached to that host
	if hostName == "" {
		return nil, nil
	}

	lunIDMap, err := u.getLunID(volumeName, hostName)
	if err != nil {
		return nil, err
	}
	//volume not attached to that host
	if lunIDMap == nil {
		return nil, nil
	}

	return u.buildConnProperty(hostInfo.Link, lunIDMap, hostName)
}

// DetachVolume remove the map of the volume and host on storage.
func (u *StorageUtil) DetachVolume(volumeName string, hostInfo storage.HostInfo) error {
	//first unbind the volume from the host
	hostName, err := u.getHostWithHostInfo(hostInfo)
	if err != nil {
		return err
	}
	//no host info exist, volume not attached to that host
	if hostName == "" {
		return nil
	}

	if err := u.UnmapVolumeFromHost(volumeName, hostName); err != nil {
		return err
	}

	volumes, err := u.volumesMappedToHost(hostName)
	if err != nil {
		glog.Warningf("get volume mapped to host %s failed, unable to decide whether need to delete the host", hostName)
		return nil
	}

	if len(volumes) == 0 {
		//no volume mapped to the host, delete the host.
		return u.deleteHost(hostName)
	}

	glog.Info(fmt.Sprintf("other volume mapped to host %s, do nothing", hostName))
	return nil
}
