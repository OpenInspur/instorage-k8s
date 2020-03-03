package as13000

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetTargetNameWithHostInfo(t *testing.T) {
	host := HostInfo{
		Hostname:  "test",
		Link:      "iscsi",
		Initiator: "iqn.1994-05.com.redhat:b5a4f0c69231",
		WWPNs:     []string{},
	}
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	targetName, err := storage13000Util.GetTargetNameWithHostInfo(host)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("targetName:%s", targetName)
}

/*
func TestStorageCreateTarget(t *testing.T) {
	host := HostInfo{
		Hostname:  "test",
		Link:      "iscsi",
		Initiator: "iqn.1994-05.com.redhat:b5a4f0c69231",
		WWPNs:     []string{},
	}
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	targetName, err := storage13000Util.CreateTarget(host)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("target created:%s\n", targetName)
}
*/
func TestMapVolumeToTarget(t *testing.T) {
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	lunId, err := storage13000Util.mapVolumeToTarget("targettest", "pool0", "vol0", "")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("lunId:%s\n", lunId)
}

func TestGetPortal(t *testing.T) {
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	portals, err := storage13000Util.getPortal("targettest")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("portals:%v\n", portals)
}
func TestBuildConnProperty(t *testing.T) {
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	connProperty, err := storage13000Util.BuildConnProperty("iscsi", "1", "targettest")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("connProperty:%+v\n", connProperty)
}
func TestAttachVolume(t *testing.T) {
	host := HostInfo{
		Hostname:  "test",
		Link:      "iscsi",
		Initiator: "iqn.1994-05.com.redhat:b5a4f0c69231",
		WWPNs:     []string{},
	}
	options := make(map[string]string)
	options["dataPool"] = "pool0"
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	connProperty, err := storage13000Util.AttachVolume("vol04", host, options)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("AttachVolume connProperty:%+v\n", connProperty)
}
func TestGetVolumeAttachInfo(t *testing.T) {
	host := HostInfo{
		Hostname:  "test",
		Link:      "iscsi",
		Initiator: "iqn.1994-05.com.redhat:b5a4f0c69231",
		WWPNs:     []string{},
	}
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	connProperty, err := storage13000Util.GetVolumeAttachInfo("pool0/vol0", host)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("GetVolumeAttachInfo connProperty:%+v\n", connProperty)
}
func TestDetachVolume(t *testing.T) {
	host := HostInfo{
		Hostname:  "test",
		Link:      "iscsi",
		Initiator: "iqn.1994-05.com.redhat:b5a4f0c69231",
		WWPNs:     []string{},
	}
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	err := storage13000Util.DetachVolume("pool0/vol0", host)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("DetachVolume Successfully.")
}
func TestGetVolumeNameWithUID(t *testing.T) {
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	volumeMappingName, err := storage13000Util.GetVolumeNameWithUID("66c92bf000000000000000000001001")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("volumeMappingName:%s\n", volumeMappingName)
}
func TestStorageCreateVolume(t *testing.T) {
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}

	option := make(map[string]string)
	//option["dataPoolType"] = "1"
	option["dataPool"] = "pool0"
	//option["metaPool"] = "pool0"
	option["thinAttribute "] = "1"
	option["threshold"] = "80"
	option["devKind"] = "block"

	volumeName, err := storage13000Util.CreateVolume("lvm02", "2", option)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("CreateVolume Successfully.volumeName:%s\n", volumeName)
}
func TestStorageDeleteVolume(t *testing.T) {
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	info := make(map[string]string)
	info["devKind"] = "block"
	info["dataPool"] = "pool0"
	err := storage13000Util.DeleteVolume("lvm02", info)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("StorageDeleteVolume Successfully.\n")
}
func TestSplit(t *testing.T) {
	poolVolumeName := "pool0-test-13000"
	index := strings.Index(poolVolumeName, "-")
	fmt.Printf(fmt.Sprintf(poolVolumeName[0:index]))
	fmt.Printf(fmt.Sprintf(poolVolumeName[index+1:]))
}
func TestGetVolumeMappingByVolumeName(t *testing.T) {
	volumeMappingName := "pool0/vol0"
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	volumeMappings, err := storage13000Util.getVolumeMappingByVolumeName(volumeMappingName)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("volumeMappings:%v\n", *volumeMappings)
}
func TestGetTargetForVolume(t *testing.T) {
	storage13000Util := &Storage13000Util{restApiWrapper: RestApiWrapper{restApiClient: &httpMethod{}}}
	/*
		//1)lun not exist ,not bind iqn
		isLunExist, lunId, isLunBindIqn, targetName, targetNode, err := storage13000Util.GetTargetForVolume("volNotMapping", "pool0", "iqntest")
		if err != nil {
			t.Errorf("error:%+v\n", err)
		}
		if isLunExist || isLunBindIqn || targetName != "target.inspur.k8s-00000016" {
			t.Errorf("1) not expected result.")
		}
		fmt.Printf("[lun not exist, not bind iqn]isLunExist:%v,lunID:%s, isLunBindIqn:%v, targetName:%s,targetNode:%s\n", isLunExist, lunId, isLunBindIqn, targetName, targetNode)
	*/
	//2)lun exist, not bind iqn
	isLunExist, lunId, isLunBindIqn, targetName, targetNode, err := storage13000Util.GetTargetForVolume("vol04", "pool0", "iqntest")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	if !isLunExist || isLunBindIqn || targetName != "target.inspur.k8s-00000015" {
		t.Errorf("2) not expected result.")
	}
	fmt.Printf("[lun exist, not bind iqn]isLunExist:%v,lunID:%s, isLunBindIqn:%v, targetName:%s,targetNode:%s\n", isLunExist, lunId, isLunBindIqn, targetName, targetNode)

	//3)lun exist ,bind iqn
	isLunExist, lunId, isLunBindIqn, targetName, targetNode, err = storage13000Util.GetTargetForVolume("vol03", "pool0", "iqntest")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	if !isLunExist || !isLunBindIqn || targetName != "target.inspur.k8s-00000012" {
		t.Errorf("3) not expected result.")
	}
	fmt.Printf("[lun exist ,bind iqn]isLunExist:%v,lunID:%s, isLunBindIqn:%v, targetName:%s,targetNode:%s\n", isLunExist, lunId, isLunBindIqn, targetName, targetNode)
}
