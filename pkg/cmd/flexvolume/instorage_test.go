// instorage_test.go
package main

import (
	"testing"
)

func TestPrintResponse(t *testing.T) {
	status := DriverStatus{
		Status: "success",
		// Reason for success/failure.
		Message:    "/dev/sdx",
		DevicePath: "/dev/sdx",
		VolumeName: "",
	}
	err := printResponse(status)
	if err != nil {
		t.Error("printResponse error:" + err.Error())
	}
}

func TestProcessExtentCmd_checkCfg(t *testing.T) {
	var argv = []string{"D:/test/inspur~instorage/instorage", "ext-check-cfg"}
	processExtentCmd(argv)

	argv[1] = "ext-help"
	processExtentCmd(argv)
}
