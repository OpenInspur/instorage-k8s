package restApi

import "fmt"

type BaseResponse struct {
	Code        interface{} `json:"code"`
	Description string      `json:"description"`
	Suggestion  string      `json:"suggestion"`
	Message     string      `json:"message"`
}

//GetCode parse the code and return string format.
func (r *BaseResponse) GetCode() (string, error) {
	if code, isOk := r.Code.(float64); isOk == true {
		return fmt.Sprintf("%.0f", code), nil
	}

	if code, isOk := r.Code.(string); isOk == true {
		return code, nil
	}

	return "", fmt.Errorf("Parse code failed %s", r.Code)
}

type ResponseSystemLogin struct {
	BaseResponse
	Data DataSystemLogin
}

type DataSystemLogin struct {
	Expire_time string `json:"Expire_time"`
	Auth_token  string `json:"Auth_token"`
}

type ResponseDeviceLogin struct {
	BaseResponse
	Data DataDeviceLogin
}
type DataDeviceLogin struct {
	ExpireTime string `json:"expireTime"`
	Role       string `json:"role"`
	Token      string `json:"token"`
}

type ResponseQueryTarget struct {
	BaseResponse
	Data []DataTarget
}
type DataTarget struct {
	Name     string `json:"name"`
	Node     string `json:"node"`
	ChapUser string `json:"chapUser"`
	Status   int    `json:"status"`
}
type ResponseQueryLvm struct {
	BaseResponse
	Data DataLvm
}
type DataLvm struct {
	LvmList         []LvmList `json:"lvmList"`
	PageNumber      int       `json:"pageNumber"`
	PageSize        int       `json:"pageSize"`
	TotalItemNumber int       `json:"totalItemNumber"`
}
type LvmList struct {
	ConsistencyGroup string `json:"consistencyGroup"`
	MappingState     int    `json:"mappingState"`
	CreateTime       string `json:"createTime"`
	TotalCapacity    string `json:"totalCapacity"`
	UsedCapacity     string `json:"usedCapacity"`
	DataPool         string `json:"dataPool"`
	Name             string `json:"name"`
	LvmType          int    `json:"lvmType"`
	Threshold        string `json:"threshold"`
	ThinAttribute    int    `json:"thinAttribute"`
}
type ResponseQueryIQN struct {
	BaseResponse
	Data []DataIQN
}
type DataIQN struct {
	LinkStatus int    `json:"linkStatus"`
	IqnPort    string `json:"iqnPort"`
}

type ResponseQueryVolumeMapping struct {
	BaseResponse
	Data []DataVolumeMapping
}
type DataVolumeMapping struct {
	Id         string `json:"id"`
	Capacity   string `json:"capacity"`
	MappingLvm string `json:"mappingLvm"`
	ClientIp   string `json:"clientIp"`
	Naa        string `json:"naa"`
	IqnPort    string `json:"iqnPort"`
	Target     string `json:"target"`
}
type ResponseQueryNodeGeneral struct {
	BaseResponse
	Data DataNodeGeneral
}
type DataNodeGeneral struct {
	HealthDisk      string `json:"healthDisk"`
	Name            string `json:"name"`
	TotalDisk       string `json:"totalDisk"`
	TimeZone        string `json:"timeZone"`
	ProductName     string `json:"productName"`
	ManageIp        string `json:"manageIp"`
	BusinessIp      string `json:"businessIp"`
	HealthStatus    string `json:"healthStatus"`
	FirmwareVersion string `json:"firmwareVersion"`
	Time            string `json:"time"`
	NodeIp          string `json:"nodeIp"`
	FunctionType    string `json:"functionType"`
	DeviceSn        string `json:"deviceSn"`
	RunningStatus   string `json:"runningStatus"`
	Manufacturer    string `json:"manufacturer"`
}
type ResponseQueryPool struct {
	BaseResponse
	Data []DataPool
}
type DataPool struct {
	AvailableCapacity string `json:"availableCapacity"`
	FsPool            int    `json:"fsPool"`
	FastPool          string `json:"fastPool"`
	FaultDomain       string `json:"faultDomain"`
	TotalCapacity     string `json:"totalCapacity"`
	UsedCapacity      string `json:"usedCapacity"`
	Name              string `json:"name"`
	BucketBelong      string `json:"bucketBelong"`
	Threshold         string `json:"threshold"`
	Type              int    `json:"type"`
	Strategy          string `json:"strategy"`
	Status            string `json:"status"`
}
type ResponseNFSShareDetail struct {
	BaseResponse
	Data map[string]interface{} `json:"data"`
}

type ResponseMultiObject struct {
	BaseResponse
	Data []map[string]interface{} `json:"data"`
}

type ResponseSingleObject struct {
	BaseResponse
	Data map[string]interface{} `json:"data"`
}

type ResponseNullObject struct {
	BaseResponse
	Data string `json:"data"`
}
