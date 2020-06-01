package restApi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"net"
	"net/http"
	"time"

	"github.com/golang/glog"
	"strings"
)

type MethodType string

const (
	Get    MethodType = "Get"
	Post   MethodType = "Post"
	Put    MethodType = "Put"
	Delete MethodType = "Delete"
)

type HttpMethod interface {
	Get(shortUrl string) (*[]byte, error)
	Post(method string, parameter map[string]interface{}) (*[]byte, error)
	Delete(shortUrl string) (*[]byte, error)
	Put(method string, parameter map[string]interface{}) (*[]byte, error)

	GetEnhanced(shortURL string) (*[]byte, error)
	PostEnhanced(shortURL string, parameter map[string]interface{}) (*[]byte, error)
	PutEnhanced(shortURL string, parameter map[string]interface{}) (*[]byte, error)
	DeleteEnhanced(shortURL string) (*[]byte, error)
}

type RestApiClient struct {
	SystemLoginInfo
	DeviceLoginInfo
	BaseUrl string
	Client  *http.Client
}

type SystemLoginInfo struct {
	UserNameSystem string `json:"name"`
	PasswordSystem string `json:"password"`
	tokenSystem    string
}
type DeviceLoginInfo struct {
	UserNameDevice string `json:"name"`
	PasswordDevice string `json:"password"`
	tokenDevice    string `json:"X-Auth-Token"`
}

func NewRestApiClient(systemLoginInfo *SystemLoginInfo, deviceLoginInfo *DeviceLoginInfo, host string) (*RestApiClient, error) {
	baseUrl := fmt.Sprintf("http://%s", host)

	client := initClient()
	restApiClient := &RestApiClient{*systemLoginInfo, *deviceLoginInfo, baseUrl,
		client}
	err := restApiClient.init()

	return restApiClient, err

}

//Init client
func initClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*200) //设置建立连接超时
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * 200)) //设置发送接收数据超时
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * 200,
		},
	}
	return client
}

func (c *RestApiClient) init() error {
	if err := c.initSystemToken(); err != nil {
		return err
	}
	if err := c.initDeviceToken(); err != nil {
		return err
	}

	return nil
}
func (c *RestApiClient) initSystemToken() error {
	url := fmt.Sprintf("%s/rest/account/multidevicelogin", c.BaseUrl)
	jsonUserInfo, jsonerr := json.Marshal(&c.SystemLoginInfo)
	if jsonerr != nil {
		//fmt.Printf(jsonerr.Error())
		return jsonerr
	}
	//var jsonStr = []byte(`{"name":"admin", "password":"passw0rd"}`)
	req, errReq := http.NewRequest("post", url, bytes.NewBuffer(jsonUserInfo))
	if errReq != nil {
		//fmt.Printf("new request error:%s", errReq.Error())
		return errReq
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Content-Length", "238")

	resp, err := c.Client.Do(req)
	if err != nil {
		//fmt.Printf(err.Error())
		glog.Errorf("Init System token error: %+v", err)
		return err
	}
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	if resp.Status == "200 OK" {
		body, _ := ioutil.ReadAll(resp.Body)
		//fmt.Println("response Body:", string(body))

		responseSystemLogin := ResponseSystemLogin{}

		if err = json.Unmarshal(body, &responseSystemLogin); err != nil {
			//fmt.Printf("unmarsh error: %v\n", err)
		}
		c.SystemLoginInfo.tokenSystem = responseSystemLogin.Data.Auth_token
		//fmt.Printf("login token:%s\n", responseSystemLogin.Data.Auth_token)
		return nil
	} else {
		return errors.New(fmt.Sprintf("initSystemToken error: status,%s", resp.Status))
	}

}
func (c *RestApiClient) initDeviceToken() error {

	url := fmt.Sprintf("%s/rest/security/mulitydevicytoken", c.BaseUrl)
	//var jsonStr2 = []byte(`{"name":"superuser", "password":"00000000","X-Auth-Token":"kqJUgwo0FI5bR8uGZl8NK0GicTh3SetefQr"}`)
	dLInfo := c.DeviceLoginInfo
	dLInfo.tokenDevice = c.SystemLoginInfo.tokenSystem
	jsonUserInfo, jsonerr := json.Marshal(&dLInfo)
	if jsonerr != nil {
		//fmt.Printf("marshal error:%v\n", jsonerr)
		return jsonerr
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonUserInfo))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Content-Length", "238")
	req.Header.Set("X-Auth-Token", c.SystemLoginInfo.tokenSystem)

	resp, err := c.Client.Do(req)
	if err != nil {
		//fmt.Printf("initDeviceToken error:%v\n", err)
		return err
	}
	defer resp.Body.Close()

	//fmt.Println("response Status-device:", resp.Status)
	//fmt.Println("response Headers-device:", resp.Header)
	if resp.Status == "200 OK" {
		body, _ := ioutil.ReadAll(resp.Body)
		//fmt.Println("response Body-device:", string(body))

		responseDeviceLogin := ResponseDeviceLogin{}

		if err = json.Unmarshal(body, &responseDeviceLogin); err != nil {
			//fmt.Printf("unmarsh error: %v\n", err)
		}
		c.DeviceLoginInfo.tokenDevice = responseDeviceLogin.Data.Token
		//fmt.Printf("login token:%s\n", responseDeviceLogin.Data.Token)
		return nil
	} else {
		return errors.New(fmt.Sprintf("initDeviceToken error: status,%s", resp.Status))
	}

}

func (c *RestApiClient) request(method string, shortURL string, parameter map[string]interface{}) (*[]byte, error) {
	content, err := c.requestImpl(method, shortURL, parameter)
	if err != nil {
		return content, err
	}
	respResult := BaseResponse{}
	if err = json.Unmarshal(*content, &respResult); err != nil {
		return content, err
	}
	if code, isOk := respResult.Code.(float64); isOk == true && code == 0 {
		return content, nil
	}
	if code, isOk := respResult.Code.(string); isOk == true && code == "0" {
		return content, nil
	}
	if c.checkExpired(respResult.Message)  == false {
		return content, nil
	}
	if c.init() != nil {
		return content, nil
	}
	return c.requestImpl(method, shortURL, parameter)
}

func (c *RestApiClient) requestImpl(method string, shortURL string, parameter map[string]interface{}) (*[]byte, error) {
	url := fmt.Sprintf("%s%s", c.BaseUrl, shortURL)

	var reqLoad io.Reader
	if parameter != nil {
		jsonEncoded, err := json.Marshal(parameter)
		if err != nil {
			return nil, fmt.Errorf("json marshal failed for %s", err)
		}
		reqLoad = bytes.NewBuffer(jsonEncoded)
	}

	glog.Debugf("Request %s %s %s", method, url, reqLoad)
	req, err := http.NewRequest(method, url, reqLoad)
	if err != nil {
		return nil, fmt.Errorf("Request(%s %s) create failed for %s", method, url, err)
	}

	req.Header.Set("X-Auth-Token", c.SystemLoginInfo.tokenSystem)
	req.Header.Set("X-Target-Device-Auth-Token", c.DeviceLoginInfo.tokenDevice)

	resp, err := c.Client.Do(req)
	if err != nil {
		glog.Errorf("Request(%s %s) execute failed for %s", method, url, err)
		return nil, fmt.Errorf("Request(%s %s) execute failed for %s", method, url, err)
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response with status %v content %s", resp.StatusCode, string(content))
	}
	fmt.Println("RestApi Request, response Body:", string(content))
	return &content, err
}

// GetEnhanced do a get request and return the content fetched.
func (c *RestApiClient) GetEnhanced(shortURL string) (*[]byte, error) {
	return c.request("Get", shortURL, nil)
}

// DeleteEnhanced do a delete request and return the content respond.
func (c *RestApiClient) DeleteEnhanced(shortURL string) (*[]byte, error) {
	return c.request("Delete", shortURL, nil)
}

// PostEnhanced do a post request and return the content respond
func (c *RestApiClient) PostEnhanced(shortURL string, parameter map[string]interface{}) (*[]byte, error) {
	return c.request("Post", shortURL, parameter)
}

// PutEnhanced do a post request and return the content respond
func (c *RestApiClient) PutEnhanced(shortURL string, parameter map[string]interface{}) (*[]byte, error) {
	return c.request("Put", shortURL, parameter)
}

func (c *RestApiClient) Get(shortUrl string) (*[]byte, error) {
	content, err := c.getImpl(shortUrl)
	if err != nil {
		return content, err
	}
	respResult := BaseResponse{}
	if err = json.Unmarshal(*content, &respResult); err != nil {
		return content, err
	}
	if c.checkExpired(respResult.Message)  == false {
		return content, err
	}
	if c.init() != nil {
		return content, err
	}
	return c.getImpl(shortUrl)
}

func (c *RestApiClient) getImpl(shortUrl string) (*[]byte, error) {
	glog.Debugf("Enter RestApi Get() : url,%s", shortUrl)
	//fmt.Printf("RestApi-Get Begin: url,%s\n", shortUrl)
	//fmt.Printf(fmt.Sprintf("%s%s\n", c.BaseUrl, shortUrl))

	req, errReq := http.NewRequest("Get", fmt.Sprintf("%s%s", c.BaseUrl, shortUrl), nil)
	if errReq != nil {
		glog.Errorf("RestApi Get,url=%s%s, error: %+v", c.BaseUrl, shortUrl, errReq)
		return nil, fmt.Errorf("RestApi Get,url=%s%s, error: %+v", c.BaseUrl, shortUrl, errReq)
	}

	//req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	//req.Header.Set("Content-Length", "238")

	req.Header.Set("X-Auth-Token", c.SystemLoginInfo.tokenSystem)
	req.Header.Set("X-Target-Device-Auth-Token", c.DeviceLoginInfo.tokenDevice)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RestApi Get,url=>%s%s, error: %+v", c.BaseUrl, shortUrl, err)
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RestApi Get, response with status %v content %s", resp.StatusCode, string(content))
	}
	fmt.Println("RestApi Get, response Body:", string(content))
	return &content, err
}

func (c *RestApiClient) Post(method string, parameter map[string]interface{}) (*[]byte, error) {
	content, err := c.postImpl(method, parameter)
	if err != nil {
		return content, err
	}
	respResult := BaseResponse{}
	if err = json.Unmarshal(*content, &respResult); err != nil {
		return content, err
	}
	if c.checkExpired(respResult.Message)  == false {
		return content, err
	}
	if c.init() != nil {
		return content, err
	}
	return c.postImpl(method, parameter)
}

func (c *RestApiClient) postImpl(method string, parameter map[string]interface{}) (*[]byte, error) {
	glog.Debugf("Enter RestApi Post : method => %s, parameter => %+v", method, parameter)
	//fmt.Printf("RestApi-Get Begin:")

	jsonParameter, jsonerr := json.Marshal(parameter)
	if jsonerr != nil {
		glog.Errorf("json marshal error:%v", jsonerr)
		return nil, errors.New(fmt.Sprintf("json marshal error:%v", jsonerr))
	}

	req, errReq := http.NewRequest("Post", fmt.Sprintf("%s%s", c.BaseUrl, method), bytes.NewBuffer(jsonParameter))
	//req, errReq := http.NewRequest("Post", fmt.Sprintf("%s%s", c.BaseUrl, method), strings.NewReader("name=targetTest"))
	if errReq != nil {
		glog.Errorf("new request error:%v", errReq)
		return nil, errors.New(fmt.Sprintf("new request error:%v", errReq))
	}

	//req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	//req.Header.Set("Content-Length", "238")

	//fmt.Printf("X-Auth-Token:%s\n", c.SystemLoginInfo.tokenSystem)
	//fmt.Printf("X-Target-Device-Auth-Token:%s\n", c.DeviceLoginInfo.tokenDevice)

	req.Header.Set("X-Auth-Token", c.SystemLoginInfo.tokenSystem)
	req.Header.Set("X-Target-Device-Auth-Token", c.DeviceLoginInfo.tokenDevice)

	resp, err := c.Client.Do(req)
	if err != nil {
		//fmt.Printf("client.Do error:%v\n", err)
		glog.Errorf("client.Do error:%v\n", err)
		return nil, err
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RestApi Post, response with status %v content %s", resp.StatusCode, string(content))
	}
	fmt.Println("RestApi Post, response Body:", string(content))
	return &content, err
}

func (c *RestApiClient) Put(method string, parameter map[string]interface{}) (*[]byte, error) {
	content, err := c.putImpl(method, parameter)
	if err != nil {
		return content, err
	}
	respResult := BaseResponse{}
	if err = json.Unmarshal(*content, &respResult); err != nil {
		return content, err
	}
	if c.checkExpired(respResult.Message)  == false {
		return content, err
	}
	if c.init() != nil {
		return content, err
	}
	return c.putImpl(method, parameter)
}

func (c *RestApiClient) putImpl(method string, parameter map[string]interface{}) (*[]byte, error) {
	glog.Debugf("Enter RestApi Put : method => %s, parameter => %+v", method, parameter)
	//fmt.Printf("RestApi-Get Begin:")

	jsonParameter, jsonerr := json.Marshal(parameter)
	if jsonerr != nil {
		glog.Errorf("json marshal error:%v", jsonerr)
		return nil, errors.New(fmt.Sprintf("json marshal error:%v", jsonerr))
	}

	req, errReq := http.NewRequest("Put", fmt.Sprintf("%s%s", c.BaseUrl, method), bytes.NewBuffer(jsonParameter))
	//req, errReq := http.NewRequest("Post", fmt.Sprintf("%s%s", c.BaseUrl, method), strings.NewReader("name=targetTest"))
	if errReq != nil {
		glog.Errorf("new request error:%v", errReq)
		return nil, errors.New(fmt.Sprintf("new request error:%v", errReq))
	}

	//req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	//req.Header.Set("Content-Length", "238")

	//fmt.Printf("X-Auth-Token:%s\n", c.SystemLoginInfo.tokenSystem)
	//fmt.Printf("X-Target-Device-Auth-Token:%s\n", c.DeviceLoginInfo.tokenDevice)

	req.Header.Set("X-Auth-Token", c.SystemLoginInfo.tokenSystem)
	req.Header.Set("X-Target-Device-Auth-Token", c.DeviceLoginInfo.tokenDevice)

	resp, err := c.Client.Do(req)
	if err != nil {
		//fmt.Printf("client.Do error:%v\n", err)
		glog.Errorf("client.Do error:%v\n", err)
		return nil, err
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RestApi Put, response with status %v content %s", resp.StatusCode, string(content))
	}
	fmt.Println("RestApi Put, response Body:", string(content))
	return &content, err
}

func (c *RestApiClient) Delete(shortUrl string) (*[]byte, error) {
	content, err := c.deleteImpl(shortUrl)
	if err != nil {
		return content, err
	}
	respResult := BaseResponse{}
	if err = json.Unmarshal(*content, &respResult); err != nil {
		return content, err
	}
	if c.checkExpired(respResult.Message)  == false {
		return content, err
	}
	if c.init() != nil {
		return content, err
	}
	return c.deleteImpl(shortUrl)
}

func (c *RestApiClient) deleteImpl(shortUrl string) (*[]byte, error) {

	glog.Debugf("Enter RestApi Delete: url=%s", shortUrl)
	//fmt.Printf("RestApi-Delete Begin: url,%s\n", shortUrl)
	//fmt.Printf(fmt.Sprintf("url:%s%s\n", c.BaseUrl, shortUrl))

	req, errReq := http.NewRequest("Delete", fmt.Sprintf("%s%s", c.BaseUrl, shortUrl), nil)
	if errReq != nil {
		//fmt.Printf("RestApi-Delete NewRequest error:%v", errReq)
		glog.Errorf("RestApi Delete, NewRequest error: %v", errReq)
		return nil, errors.New(fmt.Sprintf("RestApi Delete, NewRequest error: %v", errReq))
	}

	//req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	//req.Header.Set("Content-Length", "238")

	req.Header.Set("X-Auth-Token", c.SystemLoginInfo.tokenSystem)
	req.Header.Set("X-Target-Device-Auth-Token", c.DeviceLoginInfo.tokenDevice)

	resp, err := c.Client.Do(req)
	if err != nil {
		glog.Errorf("RestApi Delete error:%v", err)
		return nil, err
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RestApi Delete, response with status %v content %s", resp.StatusCode, string(content))
	}
	fmt.Println("RestApi Delete, response Body:", string(content))
	return &content, err
}

func (c *RestApiClient) checkExpired(message string) bool {
	if strings.Index(message, "70510") != -1 {
		glog.Debugf("[RestApiClient::checkExpired] system expired")
		return true
	} else if strings.Index(message, "70517") != -1 {
		glog.Debugf("[RestApiClient::checkExpired] device expired")
		return true
	} else {
		return false
	}
}