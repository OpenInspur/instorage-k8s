package glog

import (
	"flag"
	"fmt"
	"os"

	//"os"
	"testing"
)

func TestFindAvaliableLogName(t *testing.T) {
	fileName := "d:/test/inspur-flexvolume.test.log"
	availableName := FindAvaliableLogName(fileName)
	fmt.Printf("availableName:%s", availableName)
}

/*
func TestCreate(t *testing.T) {
	_, _, err := create("Info", true)
	if err != nil {
		t.Errorf("error in TestCreate:%v", err)
	}
}
*/
/*
func TestGetFileSize(t *testing.T) {
	fileTest, _ := os.OpenFile("C:/Users/chendonghe/AppData/Local/Temp/glog.test.exe.chendonghe00.HOME_chendonghe.log.Info.log", os.O_APPEND, 0666)
	buf_len, _ := fileTest.Seek(0, os.SEEK_END)
	fmt.Println("buf_len", buf_len)
	str2 := "hello"
	data2 := []byte(str2)
	fileTest.Write(data2)
	//获取buf_len后把文件指针重新定位于文件开始
	//fileTest.Seek(0, os.SEEK_SET)

	//buf := make([]byte, buf_len)
	//fileTest.Read(buf)
	//fmt.Println(string(buf[:]), len(buf))
}
*/
func TestGlog(t *testing.T) {

	//SetLevelString("error")
	logArgs := []string{
		os.Args[0],
		fmt.Sprintf("-log_dir=%s", "d:/test"),
	}
	SetLevelString("debug")
	os.Args = logArgs
	flag.Parse()
	defer Flush()
	Debugf("Debugf information")
	Infof("Infof information")
	Warningf("Warning information")
	Errorf("Errorf information")

}
