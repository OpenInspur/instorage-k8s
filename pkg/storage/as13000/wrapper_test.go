package as13000

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"inspur.com/storage/instorage-k8s/pkg/restApi"

	"github.com/go-yaml/yaml"

	//"inspur.com/storage/instorage-k8s/pkg/restapi"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

type DriverStatus struct {
	// Status of the callout. One of "Success", "Failure" or "Not supported".
	Status string `json:"status"`
	// Reason for success/failure.
	Message string `json:"message,omitempty"`
	// Path to the device attached. This field is valid only for attach calls.
	// ie: /dev/sdx
	DevicePath string `json:"device,omitempty"`
	// Cluster wide unique name of the volume.
	VolumeName string `json:"volumeName,omitempty"`
	// Represents volume is attached on the node
	Attached bool `json:"attached,omitempty"`
	// Returns capabilities of the driver.
	// By default we assume all the capabilities are supported.
	// If the plugin does not support a capability, it can return false for that capability.
	//Capabilities *DriverCapabilities `json:",omitempty"`
}

func TestNewRestApiWrapper(t *testing.T) {
	data := `
    name: storage-01                #存储名称
    type: 13000                    #存储类型
    host: 192.168.1.1:8080          #存储ip端口（用于restapi连接）
    username: username             #存储系统登陆用户名
    password: password             #存储系统登陆密码
    deviceUsername: devuser        #存储设备登陆用户名
    devicePassword: devpassword
`
	var cfg utils.StorageCfg
	err := yaml.Unmarshal([]byte(data), &cfg)
	fmt.Printf("cfg:\n%+v", cfg)

	if err != nil {
		t.Error("Unmarshal error:" + err.Error())
	}
	restApiWrapper := NewRestApiWrapper(cfg)

	fmt.Printf("restApiWrapper:\n%+v", restApiWrapper)
	fmt.Println(http.StatusOK)
}

type httpMethod struct {
	url string
}

func targetResponse() *http.Response {
	data := `{
  "code": 0,
  "data": [
    {
      "name": "targettest",
      "node": "inspurdon03,inspurdon02,inspurdon01",
      "chapUser": "chapuser01,chapuser02,chapuser03",
      "status": 1
    },
    {
      "name": "target.inspur.k8s-00000012",
      "node": "inspurdon03,inspurdon02,inspurdon01",
      "chapUser": "chapuser01,chapuser02,chapuser03",
      "status": 1
    },
    {
      "name": "target.inspur.k8s-00000015",
      "node": "inspurdon03,inspurdon02,inspurdon01",
      "chapUser": "chapuser01,chapuser02,chapuser03",
      "status": 1
    }
  ]
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func lvmResponse() *http.Response {
	data := `{
  "code": 0,
  "data": {
        "lvmList": [
            {
                "consistencyGroup": "default_group",
                "mappingState": 0,
                "createTime": "2019-08-19 15:29:52",
                "totalCapacity": "12MB",
                "usedCapacity": "0B",
                "dataPool": "pool0",
                "name": "lvm02",
                "lvmType": 1,
                "threshold": "0.8",
                "thinAttribute": 1
            }
        ],
        "pageNumber": 1,
        "pageSize": 25,
        "totalItemNumber": 1
    }
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func iqnResponse() *http.Response {
	data := `{
  "code": 0,
  "data": [
    {
      "linkStatus": 1,
      "iqnPort": "iqn.1994-05.com.redhat:b5a4f0c69231"
    },
    {
      "linkStatus": 0,
      "iqnPort": "iqntest"
    }
  ]
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func volumeMappingResponse() *http.Response {
	data := `{
  "code": 0,
  "data": [
    {
      "iqnPort": "All",
      "clientIp": "All",
      "id": "3",
      "mappingLvm": "pool0/vol0",
      "capacity":"100M",
      "naa":"66c92bf000000000000000000001001",
      "target": "targettest"
    },
    {
      "iqnPort": "ALL",
      "clientIp": "All",
      "id": "2",
      "mappingLvm": "pool0/vol02",
      "capacity":"12M",
      "naa":"66c92bf000000000000000000001002",
      "target": "targettest1"
    },
    {
      "iqnPort": "iqntest",
      "clientIp": "All",
      "id": "2",
      "mappingLvm": "pool0/vol03",
      "capacity":"12M",
      "naa":"66c92bf00000000000000000000103",
      "target": "target.inspur.k8s-00000012"
    },
    {
      "iqnPort": "iqntest01",
      "clientIp": "All",
      "id": "2",
      "mappingLvm": "pool0/vol04",
      "capacity":"12M",
      "naa":"66c92bf00000000000000000000104",
      "target": "target.inspur.k8s-00000015"
    }
  ]
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func volumeMappingk8s12Response() *http.Response {
	data := `{
  "code": 0,
  "data": [
    {
      "iqnPort": "iqntest",
      "clientIp": "All",
      "id": "2",
      "mappingLvm": "pool0/vol03",
      "capacity":"12M",
      "naa":"66c92bf00000000000000000000103",
      "target": "target.inspur.k8s-00000012"
    }
  ]
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func volumeMappingk8s15Response() *http.Response {
	data := `{
  "code": 0,
  "data": [
    {
      "iqnPort": "iqntest01",
      "clientIp": "All",
      "id": "2",
      "mappingLvm": "pool0/vol04",
      "capacity":"12M",
      "naa":"66c92bf00000000000000000000104",
      "target": "target.inspur.k8s-00000015"
    }
  ]
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func nodeGeneralResponse01() *http.Response {
	data := `{
  "code": "0",
  "data": {
	"healthDisk": "1",
	"name": "inspurdon01",
	"totalDisk": "2",
	"timeZone": "Asia/Shanghai",
	"productName": "VMware Virtual Platform",
	"manageIp": "None",
	"businessIp": "192.168.40.190",
	"healthStatus": "1",
	"firmwareVersion": "None",
	"time": "2019-07-11 15:55:28",
	"nodeIp": "192.168.40.190",
	"functionType": "Mon,Osd",
	"deviceSn": "VMware-56 4d 04 91 23 55 42 32-f8 51 2c 20 0f 41 5d 1d",
	"runningStatus": "1",
	"manufacturer": "Inspur"
}
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func nodeGeneralResponse02() *http.Response {
	data := `{
  "code": "0",
  "data": {
	"healthDisk": "1",
	"name": "inspurdon02",
	"totalDisk": "2",
	"timeZone": "Asia/Shanghai",
	"productName": "VMware Virtual Platform",
	"manageIp": "None",
	"businessIp": "192.168.40.192",
	"healthStatus": "1",
	"firmwareVersion": "None",
	"time": "2019-07-11 15:55:28",
	"nodeIp": "192.168.40.192",
	"functionType": "Mon,Osd",
	"deviceSn": "VMware-56 4d 04 91 23 55 42 32-f8 51 2c 20 0f 41 5d 1d",
	"runningStatus": "1",
	"manufacturer": "Inspur"
}
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func nodeGeneralResponse03() *http.Response {
	data := `{
  "code": "0",
  "data": {
	"healthDisk": "1",
	"name": "inspurdon03",
	"totalDisk": "2",
	"timeZone": "Asia/Shanghai",
	"productName": "VMware Virtual Platform",
	"manageIp": "None",
	"businessIp": "192.168.40.194",
	"healthStatus": "1",
	"firmwareVersion": "None",
	"time": "2019-07-11 15:55:28",
	"nodeIp": "192.168.40.194",
	"functionType": "Mon,Osd",
	"deviceSn": "VMware-56 4d 04 91 23 55 42 32-f8 51 2c 20 0f 41 5d 1d",
	"runningStatus": "1",
	"manufacturer": "Inspur"
}
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func poolResponse() *http.Response {
	data := `{
  "code": 0,
    "data": [
        {
            "availableCapacity": "190.67TB",
            "fsPool": 1,
            "fastPool": "0",
            "faultDomain": "default:host",
            "totalCapacity": "190.67TB",
            "usedCapacity": "3.17GB",
            "name": "pool0",
            "bucketBelong": "icfs37,icfs36,icfs35",
            "threshold": "0.8",
            "type": 1,
            "strategy": "2",
            "status": "0"
        },
        {
            "availableCapacity": "190.67TB",
            "fsPool": 1,
            "fastPool": "0",
            "faultDomain": "default:host",
            "totalCapacity": "190.67TB",
            "usedCapacity": "886.25MB",
            "name": "pool13000",
            "bucketBelong": "icfs37,icfs36,icfs35",
            "threshold": "0.8",
            "type": 1,
            "strategy": "2",
            "status": "0"
        },
        {
            "availableCapacity": "254.23TB",
            "fsPool": 1,
            "fastPool": "0",
            "faultDomain": "default:host",
            "totalCapacity": "254.23TB",
            "usedCapacity": "0B",
            "name": "testPool",
            "bucketBelong": "icfs37,icfs36,icfs35",
            "threshold": "0.8",
            "type": 0,
            "strategy": "2+1:0",
            "status": "0"
        },
        {
            "availableCapacity": "190.67TB",
            "fsPool": 1,
            "fastPool": "0",
            "faultDomain": "default:host",
            "totalCapacity": "190.67TB",
            "usedCapacity": "6.48KB",
            "name": "rep_pool_2",
            "bucketBelong": "icfs37,icfs36,icfs35",
            "threshold": "0.9",
            "type": 1,
            "strategy": "2",
            "status": "0"
        }
    ]
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func crateResponse() *http.Response {
	data := `{
  "code": "0",
  "message": "Successfully."
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func extendResponse() *http.Response {
	data := `{
  "code": "0",
  "message": "Successfully."
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}
func deleteResponse() *http.Response {
	data := `{
  "code": "0",
  "message": "Successfully."
}`
	return &http.Response{StatusCode: 200,
		Body: ioutil.NopCloser(strings.NewReader(data)),
	}
}

func (c *httpMethod) Get(shortUrl string) (*http.Response, error) {

	/*
		if shortUrl == "/rest/block/target" {
			return &http.Response{StatusCode: 200,
				Body: ioutil.NopCloser(strings.NewReader(data)),
			}, nil
	*/
	switch shortUrl {
	case "/rest/block/target":
		return targetResponse(), nil
	case "/rest/block/target/iqn?name=targettest&node=inspurdon03,inspurdon02,inspurdon01":
		return iqnResponse(), nil
	case "/rest/block/lun?name=targettest":
		return volumeMappingResponse(), nil
	case "/rest/block/lun?name=target.inspur.k8s-00000012":
		return volumeMappingk8s12Response(), nil
	case "/rest/block/lun?name=target.inspur.k8s-00000015":
		return volumeMappingk8s15Response(), nil
	case "/rest/cluster/node/general/inspurdon01":
		return nodeGeneralResponse01(), nil
	case "/rest/cluster/node/general/inspurdon02":
		return nodeGeneralResponse02(), nil
	case "/rest/cluster/node/general/inspurdon03":
		return nodeGeneralResponse03(), nil
	case "/rest/block/pool?type=2":
		return poolResponse(), nil
	case "/rest/block/lvm/page?pool=pool0&pageNumber=1&pageSize=25&filter=lvm02":
		return lvmResponse(), nil
	}
	return nil, nil
}
func (c *httpMethod) Post(method string, parameter map[string]interface{}) (*http.Response, error) {
	switch method {
	case "/rest/block/target":
		return crateResponse(), nil
	case "/rest/block/host":
		return crateResponse(), nil
	case "/rest/block/target/bind/iqn":
		return crateResponse(), nil
	case "/rest/block/lun/bind/iqn":
		return crateResponse(), nil
	case "/rest/block/lun":
		return crateResponse(), nil
	case "/rest/block/lvm":
		return crateResponse(), nil
	case "/rest/block/lvm/batchDeletion":
		return deleteResponse(), nil
	}
	return nil, nil
}
func (c *httpMethod) Put(method string, parameter map[string]interface{}) (*http.Response, error) {
	switch method {
	case "/rest/block/lvm":
		return extendResponse(), nil
	}
	return nil, nil
}
func (c *httpMethod) Delete(shortUrl string) (*http.Response, error) {
	switch shortUrl {
	case "/rest/block/lun?name=targettest&id=3&force=1":
		return deleteResponse(), nil
	case "/rest/block/target?name=target01":
		return deleteResponse(), nil
	}
	return nil, nil
}
func (c *httpMethod) GetEnhanced(shortURL string) (*[]byte, error) {
	return nil, nil
}
func (c *httpMethod) PostEnhanced(shortURL string, parameter map[string]interface{}) (*[]byte, error) {
	return nil, nil
}
func (c *httpMethod) PutEnhanced(shortURL string, parameter map[string]interface{}) (*[]byte, error) {
	return nil, nil
}
func (c *httpMethod) DeleteEnhanced(shortURL string) (*[]byte, error) {
	return nil, nil
}
func TestQueryTarget(t *testing.T) {

	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	target, err := restApiWrapper.queryTarget()
	if err != nil {
		t.Errorf("TestQueryTarget error:%+v", err)
	}
	fmt.Printf("target:\n%+v", target)

}
func TestQueryLvm(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	lvms, err := restApiWrapper.queryLvm("pool0", "lvm02")
	if err != nil {
		t.Errorf("TestQueryLvm error:%+v", err)
	}
	if (*lvms)[0].Name != "lvm02" {
		t.Errorf("TestQueryLvm error: get wrong result")
	}
	fmt.Printf("lvms:\n%+v", lvms)
}
func TestQueryTargetByIQN(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	target, err := restApiWrapper.queryTargetByIQN("iqn.1994-05.com.redhat:b5a4f0c69231")
	if err != nil {
		t.Errorf("TestQueryTargetByIQN error:%+v", err)
	}
	fmt.Printf("target:\n%+v", target)
}
func TestQueryIQN(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	iqn, err := restApiWrapper.queryIQN("targettest", "inspurdon03,inspurdon02,inspurdon01")
	if err != nil {
		t.Errorf("TestQueryIQN error:%+v", err)
	}
	fmt.Printf("iqn:\n%+v\n", iqn)
	fmt.Printf("length:%d", len(*iqn))
}
func TestCreateTarget(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	name, err := restApiWrapper.createTarget("targetcreate")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("target created ,Name:%s\n", name)
}
func TestAddHost(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	err := restApiWrapper.addHost("targettest", "ALL")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("addHost Successfully\n")
}
func TestBindIQN(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	err := restApiWrapper.bindIQN("targettest", "inspurdon03,inspurdon02,inspurdon01", "iqn.1994-05.com.redhat:b5a4f0c69231")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("bindIQN Successfully\n")
}
func TestBindLunIQN(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	err := restApiWrapper.bindLunIQN("target.inspur.k8s-00000015", "inspurdon03,inspurdon02,inspurdon01", "iqn.1994-05.com.redhat:b5a4f0c69231", "2")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("bindlunIQN Successfully\n")
}
func TestCreateVolumeMapping(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	err := restApiWrapper.createVolumeMapping("targettest", "vol0", "pool0", "")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("mapVolumeToTarget Successfully\n")
}
func TestQueryVolumeMapping(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	volumeMapping, err := restApiWrapper.queryVolumeMapping("targettest")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("VolumeMapping:%+v\n", volumeMapping)
}
func TestGetLunMappingID(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	lunMappingId, err := restApiWrapper.getLunMappingID("targettest", "pool0", "vol02")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("lunMappingId:%s\n", lunMappingId)
}
func TestQueryNodeGeneralInfo(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	nodeGeneral, err := restApiWrapper.queryNodeGeneralInfo("inspurdon01")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("nodeGeneral:%+v\n", nodeGeneral)
}

func TestDeletVolumeMapping(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	err := restApiWrapper.deleteVolumeMapping("targettest", "3", true)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("DeletVolumeMapping Successfully.\n")
}

func TestDeleteTarget(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	err := restApiWrapper.deleteTarget("target01")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("DeleteTarget Successfully.")
}
func TestInterface(t *testing.T) {
	var inter interface{}
	inter = 0
	switch inter.(type) {
	default:
		fmt.Println(inter.(int))
	}

}
func TestQueryVolumeMappingByNaa(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	volumeMapping, err := restApiWrapper.queryVolumeMappingByNaa("66C92BF000000000000000000001001")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("volumeMapping:%+v\n", volumeMapping)
}
func TestCreateVolume(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}

	para := make(map[string]interface{})
	para["dataPoolType"] = 1
	para["name"] = "lvm01"
	para["capacity"] = 25
	para["dataPool"] = "pool0"
	para["metaPool"] = "pool0"
	para["thinAttribute "] = 1
	para["threshold"] = 80

	err := restApiWrapper.createVolume(para)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("StorageCreateVolume Successfully.\n")
}
func TestDeleteVolume(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	err := restApiWrapper.deleteVolume("pool0", "lvm02")
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("DeleteVolume Successfully.")
}
func TestJsonUnmarshal(t *testing.T) {
	data := `{
	"status": "Success",
	"message": "Attach volume success",
	"device": "dev/sda",
	"volume": "",
	"attached": true
}`
	driver := DriverStatus{}
	err := json.Unmarshal([]byte(data), &driver)
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}

	fmt.Printf("JsonUnmarshal Successfully.")
}
func TestStringEqual(t *testing.T) {
	var a, b string
	a = "66c92bf000000000000000000001002"
	b = "66c92bf000000000000000000001002"
	fmt.Println(a == b)
}
func TestIsResponseOk(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	respStr := restApi.BaseResponse{
		Code:    "0",
		Message: "string",
	}
	if !restApiWrapper.isResponseOk(respStr.Code) {
		t.Errorf("error while string ")
	}
	//int
	respint := restApi.BaseResponse{
		Code:    0,
		Message: "string",
	}
	if !restApiWrapper.isResponseOk(respint.Code) {
		t.Errorf("error while int ")
	}
	//float
	respfloat := restApi.BaseResponse{
		Code:    0,
		Message: "string",
	}
	if !restApiWrapper.isResponseOk(respfloat.Code) {
		t.Errorf("error while float ")
	}
}
func TestQueryPool(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	pools, err := restApiWrapper.queryPool()
	if err != nil {
		t.Errorf("error:%+v\n", err)
	}
	fmt.Printf("pools: %v", pools)
}
func TestQueryPoolTypeByName(t *testing.T) {
	restApiWrapper := &RestApiWrapper{restApiClient: &httpMethod{}}
	poolType, err := restApiWrapper.queryPoolTypeByName("testPool")
	if err != nil {
		t.Errorf("error: %+v\n", err)
	}
	fmt.Printf("poolType:%+v\n", poolType)
}
