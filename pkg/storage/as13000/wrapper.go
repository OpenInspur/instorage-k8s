package as13000

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"

	"inspur.com/storage/instorage-k8s/pkg/restApi"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

//RestApiWrapper encapsulate storage command in a friendly form for 13000 storage
type RestApiWrapper struct {
	restApiClient restApi.HttpMethod
}

//NewRestApiWrapper create and initialize a new RestApiWrapper object
func NewRestApiWrapper(cfg utils.StorageCfg) *RestApiWrapper {
	sysLoginInfo := &restApi.SystemLoginInfo{UserNameSystem: cfg.Username, PasswordSystem: cfg.Password}
	deviceLoginInfo := &restApi.DeviceLoginInfo{UserNameDevice: cfg.DeviceUsername, PasswordDevice: cfg.DevicePassword}
	restClient, _ := restApi.NewRestApiClient(sysLoginInfo, deviceLoginInfo, cfg.Host)
	return &RestApiWrapper{restApiClient: restClient}

}

func (c *RestApiWrapper) queryTarget() (*[]restApi.DataTarget, error) {
	glog.Debugf("Enter queryTarget()")
	url := "/rest/block/target"

	resp, err := c.restApiClient.Get(url)
	if err != nil {
		glog.Errorf("queryTarget error:%v\n", err)
		return nil, fmt.Errorf("queryTarget error:%v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respTarget := restApi.ResponseQueryTarget{}

		if err = json.Unmarshal(body, &respTarget); err != nil {
			glog.Errorf("queryTarget error: %+v", err)
			return nil, fmt.Errorf("queryTarget error: %+v", err)
		}

		if c.isResponseOk(respTarget.Code) {
			glog.Debugf("Exit queryTarget():target=%+v", respTarget.Data)
			return &respTarget.Data, nil
		} else {
			glog.Errorf("queryTarget error:%+v", respTarget)
			return nil, fmt.Errorf("queryTarget error:%+v", respTarget)
		}

	} else {
		glog.Errorf("queryTarget error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("queryTarget error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}

}
func (c *RestApiWrapper) queryTargetByIQN(iqn string) (*restApi.DataTarget, error) {
	glog.Debugf("Enter queryTargetByIQN(): iqn=%s", iqn)
	var lastErr error
	dataTargetArray, err := c.queryTarget()
	if err != nil {
		return nil, err
	}
	glog.Debugf("dataTargetArray:%+v\n", dataTargetArray)
	//fmt.Printf("dataTargetArray:%+v\n", dataTargetArray)
	for _, dataTarget := range *dataTargetArray {
		glog.Debugf("look for iqn in dataTarget:%+v\n", dataTarget)
		//fmt.Printf("dataTarget:%+v\n", dataTarget)
		dataIQNArray, err := c.queryIQN(dataTarget.Name, dataTarget.Node)
		if err != nil {
			lastErr = err
			glog.Warningf("queryIQN with %+v failed %+v", dataTarget, err)
			continue
		}
		if dataIQNArray != nil {
			for _, dataIQN := range *dataIQNArray {
				glog.Debugf("find iqn: %s", dataIQN.IqnPort)
				if dataIQN.IqnPort == iqn {
					glog.Debugf("Exit queryTargetByIQN(): target=%+v", dataTarget)
					return &dataTarget, nil
				}
			}
		} else {
			glog.Debugf("get no iqn with target=%s, node=%s",
				dataTarget.Name, dataTarget.Node)
		}

	}
	if lastErr != nil {
		//fmt.Printf("Can not find target with iqn:%s", iqn)
		glog.Errorf("Can not find target with iqn:%s,search target come across error %+v", iqn, lastErr)
		return nil, fmt.Errorf("Can not find target with iqn:%s,search target come across error %+v", iqn, lastErr)
	}
	//fmt.Printf("Can not find target with iqn:%s", iqn)
	return nil, nil

}

func (c *RestApiWrapper) queryTargetByName(name string) (*restApi.DataTarget, error) {
	glog.Debugf("Enter queryTargetByName(): name=%s", name)
	dataTargetArray, err := c.queryTarget()
	if err != nil {
		return nil, err
	}
	glog.Debugf("look for in dataTargetArray:%+v\n", dataTargetArray)
	//fmt.Printf("dataTargetArray:%+v\n", dataTargetArray)
	for _, dataTarget := range *dataTargetArray {
		glog.Debugf("dataTarget:%+v\n", dataTarget)
		//fmt.Printf("dataTarget:%+v\n", dataTarget)
		if dataTarget.Name == name {
			glog.Debugf("Exit queryTargetByName(): get target=>%+v", dataTarget)
			return &dataTarget, nil
		}
	}

	glog.Warningf("Can not find target with name:%s", name)
	return nil, nil

}
func (c *RestApiWrapper) queryLvm(poolName string, volumeName string) (*[]restApi.LvmList, error) {
	glog.Debugf("Enter queryLvm()")
	url := fmt.Sprintf("/rest/block/lvm/page?pool=%s&pageNumber=1&pageSize=25&filter=%s", poolName, volumeName)

	resp, err := c.restApiClient.Get(url)
	if err != nil {
		glog.Errorf("queryLvm error:%v\n", err)
		return nil, fmt.Errorf("queryLvm error:%v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respLvm := restApi.ResponseQueryLvm{}

		if err = json.Unmarshal(body, &respLvm); err != nil {
			glog.Errorf("queryLvm error: %+v", err)
			return nil, fmt.Errorf("queryLvm error: %+v", err)
		}

		if c.isResponseOk(respLvm.Code) {
			glog.Debugf("Exit queryLvm():lvm=%+v", respLvm.Data.LvmList)
			return &respLvm.Data.LvmList, nil
		} else {
			glog.Errorf("queryLvm error:%+v", respLvm)
			return nil, fmt.Errorf("queryLvm error:%+v", respLvm)
		}

	} else {
		glog.Errorf("queryLvm error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("queryLvm error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}

}
func (c *RestApiWrapper) queryIQN(targetName string, node string) (*[]restApi.DataIQN, error) {
	url := fmt.Sprintf("/rest/block/target/iqn?name=%s&node=%s", targetName, node)
	glog.Debugf("Enter queryIQN(): targetName=%s, node=%s", targetName, node)
	//fmt.Printf("targetName:%s,node:%s\n", targetName, node)
	//fmt.Printf("url:%s\n", url)
	resp, err := c.restApiClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("queryIQN error get:%v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respIQN := restApi.ResponseQueryIQN{}

		if err = json.Unmarshal(body, &respIQN); err != nil {
			return nil, fmt.Errorf("queryIQN error Unmarshal: %+v", err)
		}
		//fmt.Printf("respIQN.Code=%g", respIQN.Code)
		if c.isResponseOk(respIQN.Code) {
			glog.Debugf("Exit queryIQN(): find iqn=>%+v", respIQN.Data)
			return &respIQN.Data, nil
		} else {
			glog.Errorf("queryIQN error Code:%+v", respIQN)
			return nil, fmt.Errorf("queryIQN error Code:%+v", respIQN)
		}

	} else {
		glog.Errorf("queryIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("queryIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) createTarget(name string) (string, error) {
	glog.Debugf("Enter createTarget(): name=%s", name)
	url := "/rest/block/target"

	targetInfo := make(map[string]interface{})
	targetInfo["name"] = name
	resp, err := c.restApiClient.Post(url, targetInfo)
	if err != nil {
		//fmt.Printf("createTarget error post:%+v\n", err)
		glog.Errorf("createTarget error post:%+v", err)
		return "", fmt.Errorf("createTarget error post:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			glog.Errorf("createTarget error Unmarshal: %+v", err)
			return "", fmt.Errorf("createTarget error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("Exit createTarget().")
			return name, nil
		} else {
			glog.Errorf("createTarget error Code:%+v", respResult)
			return "", fmt.Errorf("createTarget error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("createTarget error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return "", fmt.Errorf("createTarget error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}

}
func (c *RestApiWrapper) addHost(targetName string, hostIp string) error {
	url := "/rest/block/host"
	glog.Debugf("Enter addHost(): targetName=%s, hostIp=%s", targetName, hostIp)
	hostInfo := make(map[string]interface{})
	hostInfo["name"] = targetName
	hostInfo["hostIp"] = hostIp
	resp, err := c.restApiClient.Post(url, hostInfo)
	if err != nil {
		//fmt.Printf("addHost error post:%+v\n", err)
		glog.Errorf("addHost error post:%+v", err)
		return fmt.Errorf("addHost error post:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			glog.Errorf("addHost error Unmarshal: %+v", err)
			return fmt.Errorf("addHost error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("Exit addHost().")
			return nil
		} else {
			glog.Errorf("addHost error Code:%+v", respResult)
			return fmt.Errorf("addHost error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("addHost error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("addHost error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) bindIQN(targetName string, node string, iqnPort string) error {
	url := "/rest/block/target/bind/iqn"
	glog.Debugf("Enter bindIQN(): targetName=%s, node=%s, iqnPort=%s",
		targetName, node, iqnPort)
	iqnInfo := make(map[string]interface{})
	iqnInfo["name"] = targetName
	iqnInfo["node"] = node
	iqnInfo["iqnPort"] = iqnPort
	resp, err := c.restApiClient.Post(url, iqnInfo)
	if err != nil {
		glog.Errorf("bindIQN error post:%+v", err)
		return fmt.Errorf("bindIQN error post:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			glog.Errorf("bindIQN error Unmarshal: %+v", err)
			return fmt.Errorf("bindIQN error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("Exit bindIQN().")
			return nil
		} else {
			glog.Errorf("bindIQN error Code:%+v", respResult)
			return fmt.Errorf("bindIQN error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("bindIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("bindIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) bindLunIQN(targetName string, node string, iqnPort string, lunId string) error {
	url := "/rest/block/lun/bind/iqn"
	glog.Debugf("Enter bindLunIQN(): targetName=%s, node=%s, iqnPort=%s, lunId=%s",
		targetName, node, iqnPort, lunId)
	lunInfo := make(map[string]interface{})
	lunInfo["target"] = targetName
	lunInfo["node"] = node
	lunInfo["iqnPort"] = iqnPort
	lunInfo["lunId"] = lunId
	resp, err := c.restApiClient.Post(url, lunInfo)
	if err != nil {
		glog.Errorf("bindLunIQN error post:%+v", err)
		return fmt.Errorf("bindLunIQN error post:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			glog.Errorf("bindLunIQN error Unmarshal: %+v", err)
			return fmt.Errorf("bindLunIQN error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("Exit bindLunIQN().")
			return nil
		} else {
			glog.Errorf("bindLunIQN error Code:%+v", respResult)
			return fmt.Errorf("bindLunIQN error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("bindLunIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("bindLunIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) unBindLunIQN(targetName string, node string, iqnPort string, lunId string) error {
	url := "/rest/block/lun/unbind/iqn"
	glog.Debugf("Enter unBindLunIQN(): targetName=%s, node=%s, iqnPort=%s, lunId=%s",
		targetName, node, iqnPort, lunId)
	lunInfo := make(map[string]interface{})
	lunInfo["target"] = targetName
	lunInfo["node"] = node
	lunInfo["iqnPort"] = iqnPort
	lunInfo["lunId"] = lunId
	resp, err := c.restApiClient.Post(url, lunInfo)
	if err != nil {
		glog.Errorf("unBindLunIQN error post:%+v", err)
		return fmt.Errorf("unBindLunIQN error post:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			glog.Errorf("unBindLunIQN error Unmarshal: %+v", err)
			return fmt.Errorf("unBindLunIQN error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("Exit unBindLunIQN().")
			return nil
		} else {
			glog.Errorf("unBindLunIQN error Code:%+v", respResult)
			return fmt.Errorf("unBindLunIQN error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("unBindLunIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("unBindLunIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) createVolumeMapping(targetName string, volumeName string, poolName string, snap string) error {
	url := "/rest/block/lun"
	glog.Debugf("Enter createVolumeMapping(): targetName=%s, volumeName=%s, poolName=%s, snap=%s",
		targetName, volumeName, poolName, snap)
	mapInfo := make(map[string]interface{})
	mapInfo["name"] = targetName
	mapInfo["lvm"] = volumeName
	mapInfo["pool"] = poolName
	if snap != "" {
		mapInfo["snap"] = snap
	}
	resp, err := c.restApiClient.Post(url, mapInfo)
	if err != nil {
		//fmt.Printf("mapVolumeToTarget error post:%+v\n", err)
		glog.Errorf("mapVolumeToTarget error post:%+v", err)
		return fmt.Errorf("mapVolumeToTarget error post:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			return fmt.Errorf("mapVolumeToTarget error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("Exit createVolumeMapping(): code=%+v", respResult.Code)
			return nil
		} else {
			glog.Errorf("mapVolumeToTarget error Code:%+v", respResult)
			return fmt.Errorf("mapVolumeToTarget error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("bindIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("bindIQN error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) queryVolumeMapping(targetName string) (*[]restApi.DataVolumeMapping, error) {
	glog.Debugf("Enter queryVolumeMapping(): targetName=%s", targetName)
	url := fmt.Sprintf("/rest/block/lun?name=%s", targetName)
	//fmt.Printf("url:%s\n", url)
	resp, err := c.restApiClient.Get(url)
	if err != nil {
		glog.Errorf("queryVolumeMapping error get:%v\n", err)
		return nil, fmt.Errorf("queryVolumeMapping error get:%v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respVolumeMapping := restApi.ResponseQueryVolumeMapping{}

		if err = json.Unmarshal(body, &respVolumeMapping); err != nil {
			glog.Errorf("queryVolumeMapping error Unmarshal: %+v", err)
			return nil, fmt.Errorf("queryVolumeMapping error Unmarshal: %+v", err)
		}
		//fmt.Printf("respVolumeMapping.Code=%g", respVolumeMapping.Code)
		if c.isResponseOk(respVolumeMapping.Code) {
			glog.Debugf("Exit queryVolumeMapping(): respVolumeMapping=>%+v", respVolumeMapping.Data)
			return &respVolumeMapping.Data, nil
		} else {
			glog.Errorf("respVolumeMapping error Code:%+v", respVolumeMapping)
			return nil, fmt.Errorf("respVolumeMapping error Code:%+v", respVolumeMapping)
		}

	} else {
		glog.Errorf("respVolumeMapping error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("respVolumeMapping error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) queryVolumeMappingByNaa(naa string) (*restApi.DataVolumeMapping, error) {
	glog.Debugf("Enter queryVolumeMappingByNaa(): naa=%s", naa)
	targets, err := c.queryTarget()
	if err != nil {
		glog.Errorf("queryVolumeMappingByNaa error:%+v\n", err)
		return nil, err
	}
	for _, target := range *targets {
		volumeMappings, err := c.queryVolumeMapping(target.Name)
		if err != nil {
			glog.Errorf("queryVolumeMappingByNaa error:%+v\n", err)
			return nil, err
		}

		for _, volumeMapping := range *volumeMappings {
			glog.Debugf("find volumeMapping:%+v", volumeMapping)

			volNaaTrim := strings.TrimSpace(strings.ToUpper(volumeMapping.Naa))
			naaTrim := strings.TrimSpace(strings.ToUpper(naa))

			//glog.V(4).Infof("volNaaTrim=%s,length=%d", volNaaTrim, len(volNaaTrim))
			//glog.V(4).Infof("naaTrim=%s,length=%d", naaTrim, len(naaTrim))
			if volNaaTrim == naaTrim {
				glog.Debugf("Exit queryVolumeMappingByNaa(): volumeMapping=%+v", volumeMapping)
				return &volumeMapping, nil
			}
		}

	}
	//fmt.Printf("could not find volumeMapping with naa:%s\n", naa)
	glog.Warningf("find no volumeMapping with naa: %s", naa)
	return nil, nil
}
func (c *RestApiWrapper) getLunMappingID(targetName string, poolName string, volumeName string) (string, error) {
	glog.Debugf("Enter getLunMappingID(): targetName=%s, poolName=%s, volumeName=%s",
		targetName, poolName, volumeName)
	volumeMappingArray, err := c.queryVolumeMapping(targetName)
	if err != nil {
		//fmt.Printf("getLunMappingID error:%+v\n", err)
		glog.Errorf("getLunMappingID error:%+v\n", err)
		return "", err
	}
	for _, volumeMapping := range *volumeMappingArray {
		glog.Debugf("find volumeMapping:%+v", volumeMapping)
		if volumeMapping.MappingLvm == fmt.Sprintf("%s/%s", poolName, volumeName) {
			glog.Debugf("get LunMapping:%+v\n", volumeMapping)
			//fmt.Printf("Exit getLunMappingID(): get LunMapping:%+v\n", volumeMapping)
			return volumeMapping.Id, nil
		}
	}
	//fmt.Printf("can not find LunMapping with targetName=%s, poolName=%s, volumeName=%s\n", targetName, poolName, volumeName)
	glog.Debugf("Exit getLunMappingID(): find no lunMappingId with targetName=%s, poolName=%s, volumeName=%s", targetName, poolName, volumeName)
	return "", nil
}
func (c *RestApiWrapper) getLunMappingIDWithoutPoolName(targetName string, volumeName string) (string, error) {
	glog.Debugf("Enter getLunMappingIDWithoutPoolName(): targetName=%s, volumeName=%s",
		targetName, volumeName)
	var lunIDs []string
	var lunID string
	volumeMappingArray, err := c.queryVolumeMapping(targetName)
	if err != nil {
		//fmt.Printf("getLunMappingID error:%+v\n", err)
		glog.Errorf("getLunMappingID error:%+v\n", err)
		return "", err
	}
	for _, volumeMapping := range *volumeMappingArray {
		glog.Debugf("find volumeMapping:%+v", volumeMapping)
		_, volName, err := c.getPoolAndVolumeName(volumeMapping.MappingLvm)
		if err != nil {
			continue
		}
		if volName == volumeName {
			glog.Debugf("get LunMapping:%+v\n", volumeMapping)
			//fmt.Printf("Exit getLunMappingIDWithoutPoolName: get LunMapping:%+v\n", volumeMapping)
			lunIDs = append(lunIDs, volumeMapping.Id)
		}
	}

	//glog.V(4).Infof("Exit getLunMappingID: find no lunMappingId with targetName=%s, volumeName=%s", targetName, volumeName)

	if lunIDs != nil {
		//there is no other lunMapping with the name of volume
		if len(lunIDs) == 1 {
			lunID = lunIDs[0]
		} else {
			//there is other lunMapping with the name of volume in other pool
			glog.Warningf("there is not only one lunID with the volume :%s,lunID: %+v", volumeName, lunIDs)
			lunID = ""
		}
	}
	return lunID, nil
}
func (c *RestApiWrapper) getPoolAndVolumeName(volumeMappingName string) (string, string, error) {
	if strings.TrimSpace(volumeMappingName) == "" {
		glog.Errorf("getPoolAndVolumeName error: volumeMappingName is empty. volumeMappingName=>%s", volumeMappingName)
		return "", "", fmt.Errorf("getPoolAndVolumeName error: volumeMappingName is empty. volumeMappingName=>%s", volumeMappingName)
	}

	index := strings.Index(volumeMappingName, "/")
	if index == -1 {
		//fmt.Printf("getPoolAndVolumeName error, it should like:poolName-volumeName,but given:%s\n", poolVolumeName)
		glog.Infof("getPoolAndVolumeName(): volumeMappingName=%s", volumeMappingName)

		return "", volumeMappingName, nil
	}

	poolName := (fmt.Sprintf(volumeMappingName[0:index]))
	volumeName := (fmt.Sprintf(volumeMappingName[index+1:]))

	//return names[0], names[1:], nil
	return poolName, volumeName, nil
}
func (c *RestApiWrapper) queryNodeGeneralInfo(nodeName string) (*restApi.DataNodeGeneral, error) {
	glog.Debugf("Enter queryNodeGeneralInfo(): nodeName=%s", nodeName)
	url := fmt.Sprintf("/rest/cluster/node/general/%s", nodeName)
	//fmt.Printf("url:%s\n", url)
	resp, err := c.restApiClient.Get(url)
	if err != nil {
		glog.Errorf("queryNodeGeneralInfo error get:%v\n", err)
		return nil, fmt.Errorf("queryNodeGeneralInfo error get:%v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respNode := restApi.ResponseQueryNodeGeneral{}

		if err = json.Unmarshal(body, &respNode); err != nil {
			glog.Errorf("queryNodeGeneralInfo error Unmarshal: %+v", err)
			return nil, fmt.Errorf("queryNodeGeneralInfo error Unmarshal: %+v", err)
		}
		//fmt.Printf("respNode.Code=%g", respNode.Code)
		if c.isResponseOk(respNode.Code) {
			glog.Debugf("Exit queryNodeGeneralInfo(): %+v", respNode.Data)
			return &respNode.Data, nil
		} else {
			glog.Errorf("queryNodeGeneralInfo error Code:%+v", respNode)
			return nil, fmt.Errorf("queryNodeGeneralInfo error Code:%+v", respNode)
		}

	} else {
		glog.Errorf("queryNodeGeneralInfo error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("queryNodeGeneralInfo error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) deleteVolumeMapping(targetName string, lunID string, isForce bool) error {
	glog.Debugf("Enter deleteVolumeMapping(): targetName=%s, lunID=%s, isForce=%t", targetName, lunID, isForce)
	url := fmt.Sprintf("/rest/block/lun?name=%s&id=%s", targetName, lunID)
	if isForce {
		url = url + "&force=1"
	}

	//fmt.Printf("url:%s\n", url)
	resp, err := c.restApiClient.Delete(url)
	if err != nil {
		glog.Errorf("deletVolumeMapping error:%v\n", err)
		return fmt.Errorf("deletVolumeMapping error:%v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respDelVolume := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respDelVolume); err != nil {
			glog.Errorf("deletVolumeMapping error Unmarshal: %+v", err)
			return fmt.Errorf("deletVolumeMapping error Unmarshal: %+v", err)
		}
		//fmt.Printf("respDelVolume.Code=%g", respDelVolume.Code)
		if c.isResponseOk(respDelVolume.Code) {
			glog.Debugf("Exit deleteVolumeMapping().")
			return nil
		} else {
			glog.Errorf("deletVolumeMapping error Code:%+v", respDelVolume)
			return fmt.Errorf("deletVolumeMapping error Code:%+v", respDelVolume)
		}

	} else {
		glog.Errorf("deletVolumeMapping error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("deletVolumeMapping error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) deleteTarget(targetName string) error {
	glog.Debugf("Enter deleteTarget(): targetName=%s", targetName)
	url := fmt.Sprintf("/rest/block/target?name=%s", targetName)

	//fmt.Printf("url:%s\n", url)
	resp, err := c.restApiClient.Delete(url)
	if err != nil {
		glog.Errorf("deletTarget error get:%v\n", err)
		return fmt.Errorf("deletTarget error get:%v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respDelTarget := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respDelTarget); err != nil {
			glog.Errorf("deletTarget error Unmarshal: %+v", err)
			return fmt.Errorf("deletTarget error Unmarshal: %+v", err)
		}
		//fmt.Printf("respDelTarget.Code=%g", respDelTarget.Code)
		if c.isResponseOk(respDelTarget.Code) {
			glog.Debugf("Exit deleteTarget().")
			return nil
		} else {
			glog.Errorf("deletTarget error Code:%+v", respDelTarget)
			return fmt.Errorf("deletTarget error Code:%+v", respDelTarget)
		}

	} else {
		glog.Errorf("deletTarget error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("deletTarget error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}

//create volume with parameter
//para:
//{
// "dataPoolType":0,
//  "name": "lvm01",
//  "capacity": 25,
//  "dataPool":" rep_pool",
//  "metaPool":"rep_pool",
//  "thinAttribute ": 1,
//  "threshold":80
//}
func (c *RestApiWrapper) createVolume(para map[string]interface{}) error {
	glog.Debugf("Enter createVolume(): %+v", para)
	//fmt.Printf("createVolume para:%+v\n", para)

	url := "/rest/block/lvm"
	resp, err := c.restApiClient.Post(url, para)
	if err != nil {
		//fmt.Printf("createVolume error post:%+v\n", err)
		glog.Errorf("createVolume error post:%+v", err)
		return fmt.Errorf("createVolume error post:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			glog.Errorf("createVolume error Unmarshal: %+v", err)
			return fmt.Errorf("createVolume error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("Exit createVolume().")
			return nil
		} else {
			glog.Errorf("createVolume error Code:%+v", respResult)
			return fmt.Errorf("createVolume error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("createVolume error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("createVolume error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}

}
func (c *RestApiWrapper) extendLvm(poolName string, name string, newCapacity int64) error {
	glog.Debugf("Enter extendLvm(): poolName=%s,name=%s,newCapacity=%d", poolName, name, newCapacity)

	url := "/rest/block/lvm"
	LvmInfo := make(map[string]interface{})
	LvmInfo["pool"] = poolName
	LvmInfo["name"] = name
	LvmInfo["newCapacity"] = newCapacity
	resp, err := c.restApiClient.Put(url, LvmInfo)
	if err != nil {
		glog.Errorf("extendLvm error put:%+v", err)
		return fmt.Errorf("extendLvm error put:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			glog.Errorf("extendLvm error Unmarshal: %+v", err)
			return fmt.Errorf("extendLvm error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("Exit extendLvm().")
			return nil
		} else {
			glog.Errorf("extendLvm error Code:%+v", respResult)
			return fmt.Errorf("extendLvm error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("extendLvm error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("extendLvm error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}

}
func (c *RestApiWrapper) deleteVolume(poolName string, volumeName string) error {
	glog.Debugf("Enter deleteVolume(): poolName=%s, volumeName=%s", poolName, volumeName)
	//fmt.Printf("deleteVolume poolName:%s,volumeName:%s\n", poolName, volumeName)
	para := make(map[string]interface{})
	para["lvm"] = volumeName
	para["pool"] = poolName

	url := "/rest/block/lvm/batchDeletion"
	resp, err := c.restApiClient.Post(url, para)
	if err != nil {
		//fmt.Printf("deleteVolume error post:%+v\n", err)
		glog.Errorf("deleteVolume error post:%+v", err)
		return fmt.Errorf("deleteVolume error post:%+v", err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respResult := restApi.BaseResponse{}

		if err = json.Unmarshal(body, &respResult); err != nil {
			glog.Errorf("deleteVolume error Unmarshal: %+v", err)
			return fmt.Errorf("deleteVolume error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respResult.Code) {
			glog.Debugf("delete volume: poolName=%s,volumeName=%s", poolName, volumeName)
			glog.Debugf("Exit deleteVolume().")
			return nil
		} else {
			glog.Errorf("deleteVolume error Code:%+v", respResult)
			return fmt.Errorf("deleteVolume error Code:%+v", respResult)
		}

	} else {
		glog.Errorf("deleteVolume error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return fmt.Errorf("deleteVolume error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) isResponseOk(code interface{}) bool {
	//glog.Debugf("code:%+v", code)
	switch code.(type) {
	case string:
		if code.(string) == "0" {
			return true
		} else {
			return false
		}
	case float64:
		if int64(code.(float64)) == 0 {
			return true
		} else {
			return false
		}
	case int:
		if code.(int) == 0 {
			return true
		} else {
			return false
		}
	default:
		return false
	}
}

func (c *RestApiWrapper) getDirectoryDetail(dirPath string) (*map[string]interface{}, error) {
	url := fmt.Sprintf("/rest/file/directory/detail?path=%s", dirPath)

	content, err := c.restApiClient.GetEnhanced(url)
	if err != nil {
		if content != nil {
			response := restApi.BaseResponse{}
			json.Unmarshal(*content, &response)
			code, err := response.GetCode()
			if err == nil && code == "400" && strings.Contains(response.Message, "Error(71000):") {
				return nil, nil
			}
		}

		return nil, fmt.Errorf("get directory detail failed %s", err)
	}

	detail := restApi.ResponseMultiObject{}
	if err := json.Unmarshal(*content, &detail); err == nil {
		if len(detail.Data) == 0 {
			return nil, nil
		}

		return &detail.Data[0], nil
	}

	singleDetail := restApi.ResponseSingleObject{}
	if err := json.Unmarshal(*content, &singleDetail); err == nil {
		return &singleDetail.Data, nil
	}

	nullDetail := restApi.ResponseNullObject{}
	if err := json.Unmarshal(*content, &nullDetail); err == nil {
		if nullDetail.Data == "" {
			return nil, nil
		}
	}

	return nil, fmt.Errorf("directory detail json unmarshal as null object failed %s", err)
}

func (c *RestApiWrapper) getPoolDetail() ([]map[string]interface{}, error) {
	url := "/rest/block/pool?type=2"

	content, err := c.restApiClient.GetEnhanced(url)
	if err != nil {
		return nil, fmt.Errorf("get pool detail failed %s", err)
	}

	pools := restApi.ResponseMultiObject{}
	if err := json.Unmarshal(*content, &pools); err != nil {
		glog.Errorf("response json unmarshal failed %s", err)
		return nil, fmt.Errorf("response json unmarshal failed %s", err)
	}

	return pools.Data, nil
}

func (c *RestApiWrapper) buildDirCreatePoolList(parentPoolList interface{}) ([]map[string]interface{}, error) {
	pPoolList, ok := parentPoolList.([]interface{})
	if !ok {
		return nil, fmt.Errorf("parentPoolList type transfer failed.")
	}

	poolDetail, err := c.getPoolDetail()
	if err != nil {
		return nil, err
	}

	poolList := []map[string]interface{}{}

	for _, pPoolInterface := range pPoolList {
		pPool := pPoolInterface.(map[string]interface{})
		for _, detail := range poolDetail {
			if pPool["poolName"] == detail["name"] {
				pool := make(map[string]interface{})
				pool["poolName"] = detail["name"]
				pool["type"] = detail["type"]
				pool["strategy"] = detail["strategy"]
				if pool["type"].(float64) == 0 {
					pool["strip"] = pPool["strip"]
				} else {
					pool["strip"] = "--"
				}

				poolList = append(poolList, pool)

				break
			}
		}
	}

	return poolList, nil
}

func (c *RestApiWrapper) createDirectoryV2(dirName string, parentDirDetail *map[string]interface{}) error {
	url := "/rest/file/directory"

	poolList, err := c.buildDirCreatePoolList((*parentDirDetail)["poolList"])
	if err != nil {
		return err
	}

	dirInfo := make(map[string]interface{})
	dirInfo["name"] = dirName
	dirInfo["parentPath"] = (*parentDirDetail)["path"]
	dirInfo["poolList"] = poolList
	dirInfo["authorityInfo"] = (*parentDirDetail)["authorityInfo"]
	dirInfo["poolStrategy"] = (*parentDirDetail)["poolStrategy"]

	if _, err := c.restApiClient.PostEnhanced(url, dirInfo); err != nil {
		glog.Errorf("create directory failed for %s", err)
		return fmt.Errorf("create directory failed for %s", err)
	}

	return nil
}

func (c *RestApiWrapper) listSubDirectory(dirPath string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("/rest/file/directory?path=%s", dirPath)

	content, err := c.restApiClient.GetEnhanced(url)
	if err != nil {
		return nil, fmt.Errorf("list sub-directory failed %s", err)
	}

	lists := restApi.ResponseMultiObject{}
	if err := json.Unmarshal(*content, &lists); err != nil {
		glog.Errorf("response json unmarshal failed %s", err)
		return nil, fmt.Errorf("response json unmarshal failed %s", err)
	}

	return lists.Data, nil
}

func (c *RestApiWrapper) createDirectory(dirName string, parentDirDetail *map[string]interface{}) error {
	url := "/rest/file/directory"

	dirInfo := make(map[string]interface{})
	dirInfo["name"] = dirName
	dirInfo["parentPath"] = (*parentDirDetail)["path"]
	dirInfo["poolName"] = (*parentDirDetail)["poolName"]
	dirInfo["authorityInfo"] = (*parentDirDetail)["authorityInfo"]
	dirInfo["dataProtection"] = (*parentDirDetail)["dataProtection"]

	if _, err := c.restApiClient.PostEnhanced(url, dirInfo); err != nil {
		glog.Errorf("create directory failed for %s", err)
		return fmt.Errorf("create directory failed for %s", err)
	}

	return nil
}

func (c *RestApiWrapper) createNFSShare(dirPath string, client string) error {
	url := "/rest/file/share/nfs"

	clientInfo := make(map[string]interface{})
	clientInfo["name"] = client
	clientInfo["type"] = 0
	clientInfo["authority"] = "rw"

	clientList := []map[string]interface{}{clientInfo}

	shareInfo := make(map[string]interface{})
	shareInfo["path"] = dirPath
	shareInfo["pathAuthority"] = "rw"
	shareInfo["sync"] = "true"
	shareInfo["clientList"] = clientList

	if _, err := c.restApiClient.PostEnhanced(url, shareInfo); err != nil {
		glog.Errorf("create NFS share failed for %s", err)
		return fmt.Errorf("create NFS share failed for %s", err)
	}

	return nil
}

// setDirectoryQuota, quota should in Gi unit.
func (c *RestApiWrapper) setDirectoryQuota(dirPath string, quota string) error {
	url := "/rest/file/quota/directory"

	q, _ := strconv.Atoi(quota)

	quotaInfo := make(map[string]interface{})
	quotaInfo["path"] = dirPath
	quotaInfo["hardthreshold"] = q
	quotaInfo["hardunit"] = 2
	quotaInfo["softthreshold"] = 0
	quotaInfo["softunit"] = 2

	if _, err := c.restApiClient.PostEnhanced(url, quotaInfo); err != nil {
		if _, err := c.restApiClient.PutEnhanced(url, quotaInfo); err != nil {
			return fmt.Errorf("set directory quota on %s failed for %s", dirPath, err)
		}
	}

	return nil
}

func (c *RestApiWrapper) getClusterVirtualIP() ([]string, error) {
	url := "/rest/ctdb/set"

	content, err := c.restApiClient.GetEnhanced(url)
	if err != nil {
		return nil, fmt.Errorf("get ctdb set failed %s", err)
	}

	single := restApi.ResponseSingleObject{}
	if err := json.Unmarshal(*content, &single); err != nil {
		return nil, fmt.Errorf("get ctdb response parse failed %s", err)
	}

	ips := []string{}
	virtualIPList := single.Data["virtualIpList"].([]interface{})
	for _, item := range virtualIPList {
		value := item.(map[string]interface{})
		ip := value["ip"].(string)
		ips = append(ips, ip[:strings.Index(ip, "/")])
	}

	return ips, nil
}

func (c *RestApiWrapper) getClusterNodeIP() ([]string, error) {
	url := "/rest/cluster/node"

	content, err := c.restApiClient.GetEnhanced(url)
	if err != nil {
		return nil, fmt.Errorf("get cluster node failed %s", err)
	}

	multi := restApi.ResponseMultiObject{}
	if err := json.Unmarshal(*content, &multi); err != nil {
		return nil, fmt.Errorf("get cluster node response parse failed %s", err)
	}

	ips := []string{}
	for _, value := range multi.Data {
		status := value["healthStatus"].(string)
		ip := value["ip"].(string)
		if status == "1" && ip != "" {
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

func (c *RestApiWrapper) getClusterAccessIP() []string {
	ips := []string{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	if vips, err := c.getClusterVirtualIP(); err == nil {
		r.Shuffle(len(vips), func(i, j int) {
			vips[i], vips[j] = vips[j], vips[i]
		})
		ips = append(ips, vips...)
	}

	if nips, err := c.getClusterNodeIP(); err == nil {
		r.Shuffle(len(nips), func(i, j int) {
			nips[i], nips[j] = nips[j], nips[i]
		})
		ips = append(ips, nips...)
	}

	return ips
}

func (c *RestApiWrapper) getNFSShareInfo(dirPath string) (map[string]interface{}, error) {
	url := fmt.Sprintf("/rest/file/share/nfs?path=%s", dirPath)

	content, err := c.restApiClient.GetEnhanced(url)
	if err != nil {
		if content != nil {
			response := restApi.BaseResponse{}
			json.Unmarshal(*content, &response)
			//Maybe this response indicate the specific share does not exist.
			if code, isOk := response.Code.(string); isOk == true && code == "400" && strings.Contains(response.Message, "Error(706)") {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("get NFS share %s failed for %+v", dirPath, err)
	}

	detail := restApi.ResponseSingleObject{}
	if err := json.Unmarshal(*content, &detail); err != nil {
		return nil, fmt.Errorf("get NFS share detail response parse failed for %s", err)
	}

	return detail.Data, nil
}

func (c *RestApiWrapper) updateNFSShare(dirPath string, desire map[string]interface{}) error {
	url := "/rest/file/share/nfs"

	if _, err := c.restApiClient.PutEnhanced(url, desire); err != nil {
		return fmt.Errorf("update NFS share %s failed for %s", dirPath, err)
	}

	return nil
}

func (c *RestApiWrapper) deleteNFSShare(dirPath string) error {
	url := fmt.Sprintf("/rest/file/share/nfs?path=%s", dirPath)

	if _, err := c.restApiClient.DeleteEnhanced(url); err != nil {
		return fmt.Errorf("delete NFS share %s failed for %s", dirPath, err)
	}

	return nil
}

func (c *RestApiWrapper) deleteDirectory(dirPath string) error {
	url := fmt.Sprintf("/rest/file/directory?path=%s", dirPath)

	if _, err := c.restApiClient.DeleteEnhanced(url); err != nil {
		return fmt.Errorf("delete directory %s failed for %s", dirPath, err)
	}
	glog.Infof("delete directory: %s", dirPath)
	return nil
}
func (c *RestApiWrapper) queryPool() (*[]restApi.DataPool, error) {
	glog.Debugf("Enter queryPool()")
	url := fmt.Sprintf("/rest/block/pool?type=2")
	//fmt.Printf("url:%s\n", url)
	resp, err := c.restApiClient.Get(url)
	if err != nil {
		glog.Errorf("queryPool error get:%+v\n", err)
		return nil, fmt.Errorf("queryPool error get:%+v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		respPool := restApi.ResponseQueryPool{}

		if err = json.Unmarshal(body, &respPool); err != nil {
			glog.Errorf("queryPool error Unmarshal: %+v", err)
			return nil, fmt.Errorf("queryPool error Unmarshal: %+v", err)
		}

		if c.isResponseOk(respPool.Code) {
			glog.Debugf("Exit queryPool(): respPool=%+v", respPool.Data)
			return &respPool.Data, nil
		} else {
			glog.Errorf("respPool error Code:%+v", respPool)
			return nil, fmt.Errorf("respPool error Code:%+v", respPool)
		}

	} else {
		glog.Errorf("respPool error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
		return nil, fmt.Errorf("respPool error: response StatusCode=%d,response Body=%s",
			resp.StatusCode, string(body))
	}
}
func (c *RestApiWrapper) queryPoolTypeByName(name string) (int, error) {
	pools, err := c.queryPool()
	if err != nil {
		glog.Errorf("queryPoolTypeByName error: %+v", err)
		return -1, fmt.Errorf("queryPoolTypeByName error: %+v", err)
	}
	for _, pool := range *pools {
		if pool.Name == name {
			return pool.Type, nil
		}
	}
	glog.Errorf("could not find pool type with pool name: %s", name)
	return -1, fmt.Errorf("could not find pool type with pool name: %s", name)
}
