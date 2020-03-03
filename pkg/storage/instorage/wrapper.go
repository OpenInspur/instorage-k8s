package instorage

import (
	"fmt"
	"regexp"
	"strings"

	"inspur.com/storage/instorage-k8s/pkg/ssh"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

//CLIWrapper encapsulate storage command in a friendly form
type CLIWrapper struct {
	delimiter string

	sshExecutor ssh.IExecutor
	cliParser   ICLIParser
}

//NewCLIWrapper create and initialize a new CLIWrapper object
func NewCLIWrapper(cfg utils.StorageCfg) *CLIWrapper {
	return &CLIWrapper{
		delimiter: "!",

		//options["host"] is an go addr for net dial in the form of ip:port
		sshExecutor: ssh.NewExecutor(cfg.Host, cfg.Username, cfg.Password),

		cliParser: NewCLIParser(),
	}
}

func (w *CLIWrapper) createAndCheckResponse(cmd []string) (string, error) {
	cmdStr := strings.Join(cmd, " ")
	stdout, stderr, _, err := w.sshExecutor.Execute(cmdStr)
	if err != nil {
		return stderr, fmt.Errorf("create object failed %s", err)
	}

	re, _ := regexp.Compile(`\[([0-9]+)], successfully created`)
	matches := re.FindAllStringSubmatch(stdout, -1)
	if matches == nil {
		return "", fmt.Errorf("object ID not found, response parse failed")
	}
	return matches[0][1], nil
}

func (w *CLIWrapper) silentOperation(cmd []string) error {
	cmdStr := strings.Join(cmd, " ")
	stdout, stderr, _, err := w.sshExecutor.Execute(cmdStr)
	if err != nil {
		return err
	}
	if len(stdout) != 0 || len(stderr) != 0 {
		return fmt.Errorf("expected no output from command %s, actual is stdout: %s stderr: %s", cmdStr, stdout, stderr)
	}
	return nil
}

func (w *CLIWrapper) query(cmd []string, withHeader bool) ([]CLIRow, error) {
	cmdStr := strings.Join(cmd, " ")
	stdout, _, _, err := w.sshExecutor.Execute(cmdStr)
	if err != nil {
		return nil, err
	}

	cliRows := w.cliParser.Parse(stdout, withHeader, w.delimiter)

	return cliRows, nil
}

//encloser query operation
func (w *CLIWrapper) lsiogrp() ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lsiogrp", "-delim", w.delimiter}
	return w.query(cmd, true)
}

//vdisk operation

//mkvdisk create a vdisk with given information
func (w *CLIWrapper) mkvdisk(name string, size string, params []string) (string, error) {
	cmd := []string{"mcsop", "mkvdisk", "-name", name, "-size", size, "-unit", "gb"}
	cmd = append(cmd, params...)

	id, err := w.createAndCheckResponse(cmd)
	if err != nil {
		return "", err
	}

	return id, err
}

func (w *CLIWrapper) mkvolume(name string, size string, params []string) (string, error) {
	cmd := []string{"mcsop", "mkvolume", "-name", name, "-size", size, "-unit", "gb"}
	cmd = append(cmd, params...)

	id, err := w.createAndCheckResponse(cmd)
	if err != nil {
		return "", err
	}

	return id, err
}

func (w *CLIWrapper) rmvdisk(name string, force bool) error {
	cmd := []string{"mcsop", "rmvdisk"}
	if force {
		cmd = append(cmd, "-force")
	}
	cmd = append(cmd, name)

	return w.silentOperation(cmd)
}

func (w *CLIWrapper) rmvolume(name string, force bool) error {
	cmd := []string{"mcsop", "rmvolume"}
	if force {
		cmd = append(cmd, "-removehostmappings", "-removercrelationships", "-removelcmaps", "-discardimage", "-cancelbackup")
	}
	cmd = append(cmd, name)

	return w.silentOperation(cmd)
}

func (w *CLIWrapper) lsvdisk(filterKey string, filterValue string) ([]CLIRow, error) {
	filter := fmt.Sprintf("%s=%s", filterKey, filterValue)
	cmd := []string{"mcsinq", "lsvdisk", "-bytes", "-delim", w.delimiter, "-filtervalue", filter}

	cmdStr := strings.Join(cmd, " ")
	stdout, stderr, _, err := w.sshExecutor.Execute(cmdStr)
	if err != nil {
		return nil, err
	}

	if len(stderr) == 0 {
		return w.cliParser.Parse(stdout, true, w.delimiter), nil
	} else if strings.Contains(stderr, "CMMVC5754E") {
		//CMMVC5754E The specified object does not exist
		//the vdisk not found
		return nil, nil
	} else {
		return nil, fmt.Errorf("unexpected output from cmd %s", cmdStr)
	}
}

func (w *CLIWrapper) lsmdiskgrp(filterKey string, filterValue string) ([]CLIRow, error) {
	filter := fmt.Sprintf("%s=%s", filterKey, filterValue)
	cmd := []string{"mcsinq", "lsmdiskgrp", "-bytes", "-delim", w.delimiter, "-filtervalue", filter}

	cmdStr := strings.Join(cmd, " ")
	stdout, stderr, _, err := w.sshExecutor.Execute(cmdStr)
	if err != nil {
		return nil, err
	}

	if len(stderr) == 0 {
		return w.cliParser.Parse(stdout, true, w.delimiter), nil
	} else if strings.Contains(stderr, "CMMVC5754E") {
		//CMMVC5754E The specified object does not exist
		//the vdisk not found
		return nil, nil
	} else {
		return nil, fmt.Errorf("unexpected output from cmd %s", cmdStr)
	}
}

func (w *CLIWrapper) lsvdiskEx() ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lsvdisk", "-bytes", "-delim", w.delimiter}

	cmdStr := strings.Join(cmd, " ")
	stdout, stderr, _, err := w.sshExecutor.Execute(cmdStr)
	if err != nil {
		return nil, err
	}

	if len(stderr) == 0 {
		return w.cliParser.Parse(stdout, true, w.delimiter), nil
	} else if strings.Contains(stderr, "CMMVC5754E") {
		//CMMVC5754E The specified object does not exist
		//the vdisk not found
		return nil, nil
	} else {
		return nil, fmt.Errorf("unexpected output from cmd %s", cmdStr)
	}
}

func (w *CLIWrapper) lsvdiskDetail(name string) ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lsvdisk", "-bytes", "-delim", w.delimiter, name}
	return w.query(cmd, false)
}

func (w *CLIWrapper) expandvdisksizeList(args []string) error {
	return w.expandvdisksize(args[0], args[1])
}

func (w *CLIWrapper) expandvdisksize(name string, addition string) error {
	cmd := []string{"mcsop", "expandvdisksize", "-size", addition, "-unit", "gb", name}
	return w.silentOperation(cmd)
}

//The host operation
func (w *CLIWrapper) mkhost(hostName, portArg, portName, siteName string) error {
	cmd := []string{"mcsop", "mkhost", "-force", portArg, portName, "-name", hostName}
	if siteName != "" {
		cmd = append(cmd, "-site", siteName)
	}

	_, err := w.createAndCheckResponse(cmd)
	if err != nil {
		return fmt.Errorf("mkhost %s failed %s", hostName, err)
	}

	return nil
}

func (w *CLIWrapper) rmhost(hostName string) error {
	cmd := []string{"mcsop", "rmhost", hostName}
	return w.silentOperation(cmd)
}

func (w *CLIWrapper) lshost(hostName string) ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lshost", "-delim", w.delimiter}
	if hostName != "" {
		cmd = append(cmd, hostName)
	}

	return w.query(cmd, hostName == "")
}

func (w *CLIWrapper) addhostport(hostName, portArg, portName string) error {
	cmd := []string{"mcsop", "addhostport", "-force", portArg, portName, hostName}
	return w.silentOperation(cmd)
}

//The vdisk and host map operation
func (w *CLIWrapper) mkvdiskhostmap(hostName, vdiskName string) error {
	cmd := []string{"mcsop", "mkvdiskhostmap", "-force", "-host", hostName, vdiskName}
	_, err := w.createAndCheckResponse(cmd)
	if err != nil {
		return fmt.Errorf("mkvdiskhostmap %s to %s failed %s", vdiskName, hostName, err)
	}

	return nil
}

func (w *CLIWrapper) rmvdiskhostmap(hostName, vdiskName string) error {
	cmd := []string{"mcsop", "rmvdiskhostmap", "-host", hostName, vdiskName}
	return w.silentOperation(cmd)
}

func (w *CLIWrapper) lsvdiskhostmap(vdiskName string) ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lsvdiskhostmap", "-delim", w.delimiter, vdiskName}
	return w.query(cmd, true)
}

func (w *CLIWrapper) lshostvdiskmap(hostName string) ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lshostvdiskmap", "-delim", w.delimiter, hostName}
	return w.query(cmd, true)
}

// port operation
func (w *CLIWrapper) lsnode() ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lsnode", "-delim", w.delimiter}
	return w.query(cmd, true)
}

func (w *CLIWrapper) lsportip() ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lsportip", "-delim", w.delimiter}
	return w.query(cmd, true)
}

func (w *CLIWrapper) lsportfc() ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lsportfc", "-delim", w.delimiter}
	return w.query(cmd, true)
}

func (w *CLIWrapper) lsfabric(wwpn string, host string) ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lsfabric", "-delim", w.delimiter}
	if wwpn != "" {
		cmd = append(cmd, "-wwpn", wwpn)
	} else if host != "" {
		cmd = append(cmd, "-host", host)
	} else {
		return nil, fmt.Errorf("one of wwpn and host must be provide")
	}

	return w.query(cmd, true)
}

func (w *CLIWrapper) mkrcrelationshipList(args []string) error {
	return w.mkrcrelationship(args[0], args[1], args[2], args[3])
}

func (w *CLIWrapper) mkrcrelationship(master string, aux string, cluster string, mode string) error {
	cmd := []string{"mcsop", "mkrcrelationship", "-master", master, "-aux", aux, "-cluster", cluster}
	switch mode {
	case "sync":
		cmd = append(cmd, "-sync")
	case "activeactive":
		cmd = append(cmd, "-sync", "-activeactive")
	}
	_, err := w.createAndCheckResponse(cmd)
	if err != nil {
		return fmt.Errorf("mkrcrelationship master %s aux %s cluster %s failed %s", master, aux, cluster, err)
	}

	return nil
}

func (w *CLIWrapper) chrcrelationshipList(args []string) error {
	return w.chrcrelationship(args[0], args[1], args[2])
}

func (w *CLIWrapper) chrcrelationship(mode string, cvName string, rcID string) error {
	cmd := []string{"mcsop", "chrcrelationship"}
	switch mode {
	case "master":
		cmd = append(cmd, "-masterchange", cvName)
	case "aux":
		cmd = append(cmd, "-auxchange", cvName)
	default:
		return fmt.Errorf("mode %s not valid", mode)
	}

	cmd = append(cmd, rcID)
	return w.silentOperation(cmd)
}

func (w *CLIWrapper) rmrcrelationshipList(args []string) error {
	return w.rmrcrelationship(args[0])
}

func (w *CLIWrapper) rmrcrelationship(rcID string) error {
	cmd := []string{"mcsop", "rmrcrelationship", rcID}
	return w.silentOperation(cmd)
}

func (w *CLIWrapper) addvdiskaccessList(args []string) error {
	return w.addvdiskaccess(args[0], args[1])
}

func (w *CLIWrapper) addvdiskaccess(iogrp string, name string) error {
	cmd := []string{"mcsop", "addvdiskaccess", "-iogrp", iogrp, name}
	return w.silentOperation(cmd)
}

func (w *CLIWrapper) lssystem() ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lssystem", "-delim", w.delimiter}
	return w.query(cmd, false)
}

func (w *CLIWrapper) lslcmap() ([]CLIRow, error) {
	cmd := []string{"mcsinq", "lslcmap", "-delim", w.delimiter}

	cmdStr := strings.Join(cmd, " ")
	stdout, stderr, _, err := w.sshExecutor.Execute(cmdStr)
	if err != nil {
		return nil, err
	}

	if len(stderr) == 0 {
		return w.cliParser.Parse(stdout, true, w.delimiter), nil
	} else {
		return nil, fmt.Errorf("unexpected output from cmd %s", cmdStr)
	}
}

func (w *CLIWrapper) lslcmapEx(filterKey string, filterValue string) ([]CLIRow, error) {
	filter := fmt.Sprintf("%s=%s", filterKey, filterValue)
	cmd := []string{"mcsinq", "lslcmap", "-delim", w.delimiter, "-filtervalue", filter}

	cmdStr := strings.Join(cmd, " ")
	stdout, stderr, _, err := w.sshExecutor.Execute(cmdStr)
	if err != nil {
		return nil, err
	}

	if len(stderr) == 0 {
		return w.cliParser.Parse(stdout, true, w.delimiter), nil
	} else {
		return nil, fmt.Errorf("unexpected output from cmd %s", cmdStr)
	}
}

func (w *CLIWrapper) mklcmap(sourceVolName string, targetVolName string, copyRate string, cleanRate string, autoDelte bool) (string, error) {
	cmd := []string{"mcsop", "mklcmap", "-copyrate", copyRate, "-cleanrate", cleanRate, "-source", sourceVolName, "-target", targetVolName}
	if autoDelte {
		cmd = append(cmd, "-autodelete")
	}
	lcmapID, err := w.createAndCheckResponse(cmd)
	if err != nil {
		return "", fmt.Errorf("mklcmap %s to %s failed %s", sourceVolName, targetVolName, err)
	}

	return lcmapID, nil
}

func (w *CLIWrapper) startlcmap(lcmapID string) error {
	cmd := []string{"mcsop", "startlcmap", "-prep", lcmapID}
	return w.silentOperation(cmd)
}

func (w *CLIWrapper) stoplcmap(lcmapID string, force bool) error {
	cmd := []string{"mcsop", "stoplcmap"}
	if force {
		cmd = append(cmd, "-force")
	}
	cmd = append(cmd, lcmapID)

	return w.silentOperation(cmd)
}

func (w *CLIWrapper) rmlcmap(lcmapID string, force bool) error {
	cmd := []string{"mcsop", "rmlcmap"}
	if force {
		cmd = append(cmd, "-force")
	}
	cmd = append(cmd, lcmapID)

	return w.silentOperation(cmd)
}
