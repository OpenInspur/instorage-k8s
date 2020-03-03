package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/controller"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

const (
	// Driver calls
	initCmd          = "init"
	getVolumeNameCmd = "getvolumename"

	isAttached = "isattached"

	attachCmd        = "attach"
	waitForAttachCmd = "waitforattach"
	mountDeviceCmd   = "mountdevice"

	detachCmd        = "detach"
	unmountDeviceCmd = "unmountdevice"

	mountCmd   = "mount"
	unmountCmd = "unmount"

	expandVolumeCmd = "expandvolume"
	expandFSCmd     = "expandfs"

	//Extend for general flex api
	createCmd = "create"
	deleteCmd = "delete"

	// Option keys
	optionFSType         = "kubernetes.io/fsType"
	optionReadWrite      = "kubernetes.io/readwrite"
	optionKeySecret      = "kubernetes.io/secret"
	optionFSGroup        = "kubernetes.io/fsGroup"
	optionMountsDir      = "kubernetes.io/mountsDir"
	optionPVorVolumeName = "kubernetes.io/pvOrVolumeName"

	optionKeyPodName      = "kubernetes.io/pod.name"
	optionKeyPodNamespace = "kubernetes.io/pod.namespace"
	optionKeyPodUID       = "kubernetes.io/pod.uid"

	optionKeyServiceAccountName = "kubernetes.io/serviceAccount.name"

	//for create command which is extend for general flex api
	optionVolPoolName      = "inspur.com/volPoolName"
	optionVolAuxPoolName   = "inspur.com/volAuxPoolName"
	optionVolIOGrp         = "inspur.com/volIOGrp"
	optionVolAuxIOGrp      = "inspur.com/volAuxIOGrp"
	optionVolThin          = "inspur.com/volThin"
	optionVolCompress      = "inspur.com/volCompress"
	optionVolInTier        = "inspur.com/volInTier"
	optionVolLevel         = "inspur.com/volLevel"
	optionVolThinResize    = "inspur.com/volThinResize"
	optionVolThinGrainSize = "inspur.com/volThinGrainSize"
	optionVolThinWarning   = "inspur.com/volThinWarning"
	optionVolAutoExpand    = "inspur.com/volAutoExpand"
)

const (
	// StatusSuccess represents the successful completion of command.
	StatusSuccess = "Success"
	// StatusFailure represents the failure of the command
	StatusFailure = "Failure"
	// StatusNotSupported represents that the command is not supported.
	StatusNotSupported = "Not supported"
)

//GlobalConfig contain the configuration from configure file
//configure file is in the 'config' folder as name 'instorage.yaml'
//the 'config' folder is in the same folder as the driver
var GlobalConfig utils.Config

// DriverCapabilities represents what driver can do
type DriverCapabilities struct {
	Attach           bool `json:"attach"`
	SELinuxRelabel   bool `json:"selinuxRelabel"`
	SupportsMetrics  bool `json:"supportsMetrics"`
	FSGroup          bool `json:"fsGroup"`
	RequiresFSResize bool `json:"requiresFSResize"`
}

func defaultCapabilities() *DriverCapabilities {
	return &DriverCapabilities{
		Attach:           true,
		SELinuxRelabel:   false,
		SupportsMetrics:  false,
		FSGroup:          false,
		RequiresFSResize: true,
	}
}

//DriverStatus represents the return value of the driver callout.
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
	Capabilities *DriverCapabilities `json:",omitempty"`
	// Returns the actual size of the volume after resizing is done, the size is in bytes.
	ActualVolumeSize int64 `json:"volumeNewSize,omitempty"`
}

//CmdProcessor define the command execute interface
type CmdProcessor interface {
	//exec execute with given argv and return the resulte status
	exec(argv []string) DriverStatus
}

type unSupportCmd struct{}

func (c *unSupportCmd) exec(argv []string) DriverStatus {
	return DriverStatus{Status: StatusNotSupported}
}

//InitCmd do init and return capability
type InitCmd struct{}

func (c *InitCmd) exec(argv []string) DriverStatus {
	return DriverStatus{
		Status:  StatusSuccess,
		Message: "Plugin init successfully",
	}
}

//GetVolumeNameCmd return a volume name represent the volume
type GetVolumeNameCmd struct {
	unSupportCmd
}

//IsAttachedCmd checks if the volume is attached to the node
//<driver executable> isattached <json options> <node name> (v >= 1.6)
type IsAttachedCmd struct {
	unSupportCmd
}

//AttachCmd attaches a volume to a node
//<driver executable> attach <json options> <node name> (v=1.5 with json options, v >= 1.6 json options and node name)
type AttachCmd struct {
	unSupportCmd
}

//WaitForAttachCmd attach a device with give option to the node in which this binary run
//<driver executable> waitforattach <device path> <json volume option>
type WaitForAttachCmd struct{}

func (c *WaitForAttachCmd) exec(argv []string) DriverStatus {
	if len(argv) < 2 {
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("Wait for attach subcmd arguments not enough."),
		}
	}

	attachOptions := make(map[string]string)
	err := json.Unmarshal([]byte(argv[1]), &attachOptions)
	if err != nil {
		glog.Errorf("volume json options unmarshal failed %s %s", err, argv[0])
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("volume json options unmarshal failed %s %s", err, argv[0]),
		}
	}

	hostname := ""
	volumeName := attachOptions[optionPVorVolumeName]

	ctrl := controller.NewController(GlobalConfig)
	device, err := ctrl.Attach(hostname, volumeName, attachOptions)
	if err != nil {
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("Attach volume failed %s", err),
		}
	} else {
		return DriverStatus{
			Status:     StatusSuccess,
			Message:    "Attach volume successfully",
			DevicePath: device,
		}
	}
}

//DetachCmd detach a volume from a node
//<driver executable> detach <volume name> <node name>
type DetachCmd struct{}

func (c *DetachCmd) exec(argv []string) DriverStatus {
	glog.Debugf("Enter DetachCmd exec: argv=%+v", argv)
	if len(argv) < 2 {
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("detach subcmd arguments not enough"),
		}
	}

	volumeName := argv[0]
	hostname := argv[1]
	ctrl := controller.NewController(GlobalConfig)
	err := ctrl.Detach(hostname, volumeName, false)
	if err != nil {
		glog.Errorf("Exit DetachCmd exec with error:%+v", err)
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("detach volume failed %s", err),
		}
	} else {
		glog.Debugf("Exit DetachCmd exec")
		return DriverStatus{
			Status:  StatusSuccess,
			Message: "Detach volume successfully",
		}
	}
}

//MountDeviceCmd mount a volume's file system to a global directory
type MountDeviceCmd struct {
	unSupportCmd
}

//UnMountDeviceCmd unmount the file system of a give volume from the global directory
//<driver executable> unmountdevice mountpath
type UnMountDeviceCmd struct{}

func (c *UnMountDeviceCmd) exec(argv []string) DriverStatus {

	mountPath := argv[0]

	ctrl := controller.NewController(GlobalConfig)
	device, err := ctrl.UnMountDevice(mountPath)
	if err != nil {
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("unmount device from %s failed %s", mountPath, err),
		}
	} else {
		return DriverStatus{
			Status:     StatusSuccess,
			Message:    "unmount device successfully",
			DevicePath: device,
		}
	}
}

//MountCmd mount the file system to a Pod's local directory
type MountCmd struct {
	unSupportCmd
}

//UnMountCmd unmount the file system from a Pod's local directory
type UnMountCmd struct {
	unSupportCmd
}

//ExpandVolume expand a volume
type ExpandVolumeCmd struct {
	unSupportCmd
}

//ExpandFSCmd expand a volume online
//<driver executable> expandfs <volume-json-spec> device-path mount-path new-size old-size
type ExpandFSCmd struct{}

func (c *ExpandFSCmd) transB2GB(size string) string {
	bSize, _ := strconv.ParseInt(size, 10, 64)
	gbSize := bSize / (1024 * 1024 * 1024)

	return strconv.FormatInt(gbSize, 10)
}

func (c *ExpandFSCmd) exec(argv []string) DriverStatus {
	if len(argv) < 5 {
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("expandfs command arguments not enough."),
		}
	}

	volSpec := make(map[string]string)
	err := json.Unmarshal([]byte(argv[0]), &volSpec)
	if err != nil {
		glog.Errorf("volume json options unmarshal failed %s %s", err, argv[0])
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("volume json options unmarshal failed %s %s", err, argv[0]),
		}
	}

	devPath := argv[1]
	devMountPath := argv[2]
	newSize := c.transB2GB(argv[3])
	oldSize := c.transB2GB(argv[4])

	ctrl := controller.NewController(GlobalConfig)
	err = ctrl.ExtendVolume(volSpec[optionPVorVolumeName], newSize, oldSize, devPath, devMountPath, volSpec)
	if err != nil {
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("extend volume failed %s", err),
		}
	} else {
		actualSize, _ := strconv.ParseInt(argv[3], 10, 64)
		return DriverStatus{
			Status:           StatusSuccess,
			Message:          "extend volume success",
			ActualVolumeSize: actualSize,
		}
	}
}

//CreateCmd create a volume on the storage
//<driver executable> create <name> <size in GB> <json volume option>
type CreateCmd struct{}

func (c *CreateCmd) transOptions(in map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		newKey := k[len("inspur.com/"):]
		out[newKey] = v
	}

	return out
}

func (c *CreateCmd) exec(argv []string) DriverStatus {
	glog.Debugf("Enter CreateCmd exec: argv=%+v", argv)
	createOptions := make(map[string]string)
	if err := json.Unmarshal([]byte(argv[2]), &createOptions); err != nil {
		glog.Errorf("volume json options unmarshal failed %s %s.", err, argv[2])
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("volume json options unmarshal failed %s %s", err, argv[0]),
		}
	}

	name := argv[0]
	size := argv[1]
	options := c.transOptions(createOptions)

	ctrl := controller.NewController(GlobalConfig)
	info, err := ctrl.CreateVolume(name, size, options)
	if err != nil {
		glog.Errorf("volume create failed for %s.", err)
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("volume create failed for %s.", err),
		}
	} else {
		ret, _ := json.Marshal(info)
		glog.Debugf("Exit CreateCmd exec")
		return DriverStatus{
			Status:  StatusSuccess,
			Message: fmt.Sprintf("Create volume successfully, with response <%s>", string(ret)),
		}
	}
}

//DeleteCmd delete a volume on the storage
//<driver executable> delete <name> <json volume option>
type DeleteCmd struct{}

func (c *DeleteCmd) exec(argv []string) DriverStatus {
	glog.Debugf("Enter DeleteCmd exec: argv=%+v", argv)
	deleteOptions := make(map[string]string)
	if err := json.Unmarshal([]byte(argv[1]), &deleteOptions); err != nil {
		glog.Errorf("volume json options unmarshal failed %s %s.", err, argv[1])
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("volume json options unmarshal failed %s %s", err, argv[0]),
		}
	}

	name := argv[0]

	ctrl := controller.NewController(GlobalConfig)
	if err := ctrl.DeleteVolume(name, deleteOptions); err != nil {
		glog.Errorf("volume delete failed for %s.", err)
		return DriverStatus{
			Status:  StatusFailure,
			Message: fmt.Sprintf("volume delete failed for %s.", err),
		}
	} else {
		glog.Debugf("Exit DeleteCmd exec")
		return DriverStatus{
			Status:  StatusSuccess,
			Message: "Delete volume successfully",
		}
	}
}

//getCmdProcessor get the actually cmd processor base on cmd name
func getCmdProcessor(cmd string) CmdProcessor {
	switch cmd {
	case initCmd:
		return &InitCmd{}
	case getVolumeNameCmd:
		return &GetVolumeNameCmd{}
	case isAttached:
		return &IsAttachedCmd{}
	case attachCmd:
		return &AttachCmd{}
	case waitForAttachCmd:
		return &WaitForAttachCmd{}
	case detachCmd:
		return &DetachCmd{}
	case mountDeviceCmd:
		return &MountDeviceCmd{}
	case unmountDeviceCmd:
		return &UnMountDeviceCmd{}
	case mountCmd:
		return &MountCmd{}
	case unmountCmd:
		return &UnMountCmd{}
	case createCmd:
		return &CreateCmd{}
	case deleteCmd:
		return &DeleteCmd{}
	case expandFSCmd:
		return &ExpandFSCmd{}
	default:
		return &unSupportCmd{}
	}
}

func dumpHelp() {
	fmt.Print("Support following subcommand and extent subcommand:\n")
	fmt.Print("ext-help\n")
	fmt.Print("    Show this help.\n")
	fmt.Print("ext-sample-18000-cfg\n")
	fmt.Print("    Generate a 18000 storage sample configure for use.\n")
	fmt.Print("ext-sample-13000-cfg\n")
	fmt.Print("    Generate a 13000 storage sample configure for use.\n")
	fmt.Print("ext-version\n")
	fmt.Print("    Show the version of this program.\n")
	fmt.Print("ext-check-cfg\n")
	fmt.Print("    Check the configuration.\n")
	fmt.Print("ext-encrypt-password [plain password]\n")
	fmt.Print("    Generate the encrypted password from plain password\n")
}

//processExtendCmd just deal with the extend command for better use
func processExtentCmd(argv []string) bool {
	if len(argv) < 2 {
		return false
	}

	switch argv[1] {
	case "ext-help":
		dumpHelp()
		return true
	case "ext-sample-18000-cfg":
		fmt.Print(utils.Dump18000SampleConfig())
		return true
	case "ext-sample-13000-cfg":
		fmt.Print(utils.Dump13000SampleConfig())
		return true
	case "ext-version":
		fmt.Printf("Version: %s\n", utils.GenerateVersionStr())
		return true
	case "ext-encrypt-password":
		cipher := utils.Cipher{}
		fmt.Printf("%s\n", cipher.Encrypt(argv[2]))
		return true
	//case "ext-decrypt-password":
	//	cipher := utils.Cipher{}
	//	fmt.Printf("%s\n", cipher.Decrypt(argv[2]))
	//	return true
	case "ext-freeze-path":
		utils.FIFreeze(argv[2])
		return true
	case "ext-thaw-path":
		utils.FIThaw(argv[2])
		return true
	case "ext-check-cfg":
		//get the base directory
		baseDir := filepath.Dir(argv[0])
		//parse the configuration
		configPath := filepath.Join(baseDir, "config", "instorage.yaml")
		cfg, err := utils.LoadConfig(configPath, baseDir)
		if err != nil {
			fmt.Printf("check config error:%+v\n", err)
		}
		if cfg != nil {
			fmt.Printf("check config ok.\n")
		}
		return true
	default:
		return false
	}
}

//printResponse print the DriverStatus to stander output,
//so that flexVolume can parse and get the response,
func printResponse(status DriverStatus) error {
	responseBytes, err := json.Marshal(status)
	if err != nil {
		glog.Errorf("json marshal the response failed %s %+v", err, status)
		return err
	}

	fmt.Printf("%s", string(responseBytes))
	return nil
}

func exitWithFailure(message string) {
	status := DriverStatus{
		Status:  StatusFailure,
		Message: message,
	}
	printResponse(status)

	os.Exit(1)
}

func main() {
	if processExtentCmd(os.Args[0:]) {
		return
	}

	//get the base directory
	baseDir := filepath.Dir(os.Args[0])

	//parse the configuration
	configPath := filepath.Join(baseDir, "config", "instorage.yaml")
	cfg, err := utils.LoadConfig(configPath, baseDir)
	if err != nil {
		exitWithFailure(fmt.Sprintf("Failed to load config file %s", err))
	}
	GlobalConfig = *cfg

	//set the glog configure if enabled
	if GlobalConfig.Log.Enabled {
		logDir := GlobalConfig.Log.LogDir
		if logDir[0:1] != "/" {
			logDir = filepath.Join(baseDir, GlobalConfig.Log.LogDir)
		}
		logArgs := []string{
			os.Args[0],
			fmt.Sprintf("-log_dir=%s", logDir),
		}
		glog.SetLevelString(GlobalConfig.Log.Level)
		originalArgs := os.Args
		os.Args = logArgs
		flag.Parse()
		os.Args = originalArgs
		defer glog.Flush()
		glog.Infof("log level:%s", GlobalConfig.Log.Level)
	}

	//hello world
	glog.Infof("Hello, I am called with %s", os.Args)

	if len(os.Args) <= 1 {
		exitWithFailure("No sub-command assigned")
	}

	// process sub cmd
	cmd := getCmdProcessor(os.Args[1])
	status := cmd.exec(os.Args[2:])
	status.Capabilities = defaultCapabilities()
	printResponse(status)

	// say goodbye
	glog.Infof("GoodBye, I accomplished the job with %+v", status)
}
