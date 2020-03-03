// config_test.go
package utils

import (
	"fmt"
	"testing"

	"github.com/go-yaml/yaml"
)

func TestLoadConfig(t *testing.T) {

	configPath := "D:/test/inspur~instorage/config/instorage.yaml"
	//fmt.Printf(configPath)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Error("LoadConfig error:" + err.Error())
	}

	fmt.Printf("config:\n%+v", cfg)
}

func TestCheckConfig(t *testing.T) {
	var data = `    
    log:
      enabled: false    #是否开启日志
      logdir: log       #日志文件保存目录
      level: debug          #显示的日志级别
    host:
      link: iscsi                  #k8s节点主机与存储连接方式（只支持iscsi）
      forceUseMultipath: false      #是否启用多路径
      scsiScanRetryTimes: 3        #scsi扫描重试次数
      scsiScanWaitInterval: 1        #scsi扫描间隔等待时间
      iscsiPathCheckRetryTimes: 3    #iscsi路径检查重试次数
      iscsiPathCheckWaitInterval: 1    #iscsi路径检测间隔等待时间
      multipathSearchRetryTimes: 3    #多路径发现重试次数
      multipathSearchWaitInterval: 1    #多路径发现间隔等待时间
    storage:
    - name: storage-01                #存储名称
      type: AS13000                    #存储类型
      host: 192.168.1.100:8080          #存储ip端口（用于restapi连接）
      username: username             #存储系统登陆用户名
      password: password             #存储系统登陆密码
      deviceUsername: devuser        #存储设备登陆用户名
      devicePassword: devpassword  #存储设备登陆密码
    `

	var cfg Config
	err := yaml.Unmarshal([]byte(data), &cfg)

	if err != nil {
		t.Error("Unmarshal error:" + err.Error())
	}

	if err := cfg.CheckConfig(); err != nil {
		t.Error("CheckConfig error:" + err.Error())
	}

}
func TestDump18000SampleConfig(t *testing.T) {
	fmt.Printf(Dump18000SampleConfig())
}
func TestDump13000SampleConfig(t *testing.T) {
	fmt.Printf(Dump13000SampleConfig())
}
