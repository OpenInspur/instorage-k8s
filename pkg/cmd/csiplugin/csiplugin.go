package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"

	"inspur.com/storage/instorage-k8s/pkg/csiplugin"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

var (
	strconfig  = flag.String("strconfig", "", "Absolute path to the storage config file, if not set, the config/instorage.yaml in the same fold of this binary will be used.")
	endpoint   = flag.String("endpoint", "", "The endpoint plugin listen on.")
	nodeID     = flag.String("nodeid", "", "The node id when plugin run as node worker.")
	runmode    = flag.String("mode", "all-in-one", "The run mode of the plugin either of nodeworker, controller, all-in-one.")
	driverName = flag.String("driver-name", "csi-instorage", "The driver name of this csi driver to announce.")
	version    = flag.Bool("version", false, "Show the version.")
	enpassword   = flag.String("encrypt-password", "", "Encrypt the password to a shadow one.")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("Version: %s\n", utils.GenerateVersionStr())
		os.Exit(0)
	}

	if *enpassword != "" {
		cipher := utils.Cipher{}
		fmt.Printf("Shadow is %s\n", cipher.Encrypt(*enpassword))
		os.Exit(0)
	}

	if *endpoint == "" {
		fmt.Printf("endpoint must be set.")
		os.Exit(1)
	}
	if *nodeID == "" {
		fmt.Printf("nodeid must be set.")
		os.Exit(1)
	}
	if *runmode != "all-in-one" && *runmode != "nodeworker" && *runmode != "controller" {
		fmt.Printf("mode should be nodeworker or controller or all-in-one.")
		os.Exit(1)
	}

	//get the base directory
	baseDir := filepath.Dir(os.Args[0])
	//parse the configuration
	strCfgPath := filepath.Join(baseDir, "config", "instorage.yaml")
	if *strconfig != "" {
		strCfgPath = *strconfig
	}

	strCfg, err := utils.LoadConfig(strCfgPath, baseDir)
	if err != nil {
		glog.Fatalf("Failed to load storage configuration file %s %s.", strCfgPath, err)
		os.Exit(1)
	}

	driver := csiplugin.NewDriver(*runmode, *driverName, *nodeID, *endpoint, *strCfg)

	driver.Run()
}
