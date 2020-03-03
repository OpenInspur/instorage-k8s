package utils

import (
	"bytes"
	"os/exec"

	"github.com/golang/glog"
)

//Execute execute command on the system.
func Execute(command string, args ...string) (string, error) {
	glog.Debugf("Execute command: %s %s", command, args)
	cmd := exec.Command(command, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	stdErr := string(stderr.Bytes())
	stdOut := string(stdout.Bytes())

	glog.Debugf("Execute response: [%s] %s %s", err, stdOut, stdErr)

	if err != nil {
		return stdErr, err
	} else {
		return stdOut, nil
	}
}
