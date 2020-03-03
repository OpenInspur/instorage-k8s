package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/golang/glog"
	ctrl "github.com/kubernetes-sigs/sig-storage-lib-external-provisioner/controller"

	"inspur.com/storage/instorage-k8s/pkg/controller"
	"inspur.com/storage/instorage-k8s/pkg/utils"
)

var (
	kubeconfig           = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Needs to be set if the provisioner is running out of cluster.")
	strconfig            = flag.String("strconfig", "", "Absolute path to the storage config file, if not set, the config/instorage.yaml in the same fold of this binary will be used.")
	provisionerName      = flag.String("provisioner-name", "inspur/instorage", "Set the provisioner name of this plugin.")
	flexVolumeDriverName = flag.String("flex-volume-driver-name", "inspur/instorage-flexvolume", "The flex volume driver name to be set when create a new PV.")
	sampleConfig         = flag.Bool("samplecfg", false, "Generate the sample storage config.")
	insecure             = flag.Bool("insecure", false, "Use insecure connection.")
	about                = flag.Bool("about", false, "Show the information about this binary.")
	threadCount          = flag.Int("threads", 1, "Set the threads for work.")
)

const (
	StorageClassProvisionerFSType = "provisioner/fsType"
)

//InStorageProvisioner container information to access storage, and has method to create and delete volume
// ControllerOption has option like follows
// host -> xxx
// login ->
// password ->
type InStorageProvisioner struct {
	config utils.Config
	lock   sync.Mutex
}

func (p *InStorageProvisioner) stickProvisionerPrefix(info map[string]string) map[string]string {
	newInfo := map[string]string{}
	for k, v := range info {
		newInfo[fmt.Sprintf("%s-%s", *provisionerName, k)] = v
	}

	return newInfo
}

func (p *InStorageProvisioner) stripProvisionerPrefix(info map[string]string) map[string]string {
	prefix := fmt.Sprintf("%s-", *provisionerName)

	newInfo := map[string]string{}
	for k, v := range info {
		if strings.HasPrefix(k, prefix) {
			newInfo[k[len(prefix):]] = v
		}
	}

	return newInfo
}

//Provision is use to create a volume in storage asset with options
func (p *InStorageProvisioner) Provision(options ctrl.ProvisionOptions) (*v1.PersistentVolume, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	glog.Infof("provision called for %s", options.PVName)
	glog.Debugf("Provision options: %+v", options)
	name := options.PVName

        scParameters := options.StorageClass.Parameters

	//capacity is volume size in bytes
	capacity := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	//transform size in bytes to size in Gib
	capacityValue := (capacity.Value() + 1*1024*1024*1024 - 1) / (1 * 1024 * 1024 * 1024)
	//size is the volume size in Gib
	size := strconv.FormatInt(capacityValue, 10)

	fsType := "ext4"
	if fs, ok := scParameters[StorageClassProvisionerFSType]; ok == true {
		fsType = fs
	}

	c := controller.NewController(p.config)
	//options.Parameter is map[string]string, which is the parameter from storageClass
	info, err := c.CreateVolume(name, size, scParameters)
	if err != nil {
		glog.Errorf("create volume %s failed %s", name, err)
		return nil, fmt.Errorf("create volume %s failed %s", name, err)
	}

	annotations := p.stickProvisionerPrefix(info)
	pv := &v1.PersistentVolume{}

	devKind := "block"
	if kind, ok := scParameters["devKind"]; ok == true {
		devKind = kind
	}

	switch devKind {
	case "block":
		glog.Infof("Provision block")
		pv = &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:        options.PVName,
				Labels:      map[string]string{},
				Annotations: annotations,
			},
			Spec: v1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: *options.StorageClass.ReclaimPolicy,
				AccessModes:                   options.PVC.Spec.AccessModes,
				Capacity: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
				},
				PersistentVolumeSource: v1.PersistentVolumeSource{
					FlexVolume: &v1.FlexPersistentVolumeSource{
						Driver:  *flexVolumeDriverName,
						FSType:  fsType,
						Options: scParameters,
					},
				},
			},
		}
	case "share":
		glog.Infof("Provision share")
		pv = &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:        options.PVName,
				Labels:      map[string]string{},
				Annotations: annotations,
			},
			Spec: v1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: *options.StorageClass.ReclaimPolicy,
				AccessModes:                   options.PVC.Spec.AccessModes,
				Capacity: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
				},
				PersistentVolumeSource: v1.PersistentVolumeSource{
					NFS: &v1.NFSVolumeSource{
						Server: info["server"],
						Path:   info["path"],
					},
				},
			},
		}
	}

	glog.Infof("Provision %s finish.", name)
	return pv, nil
}

//Delete is use to delete a volume base on the PV info
func (p *InStorageProvisioner) Delete(volume *v1.PersistentVolume) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	glog.Infof("Delete volume %s.", volume.ObjectMeta.Name)

	options := p.stripProvisionerPrefix(volume.ObjectMeta.Annotations)

	c := controller.NewController(p.config)

	return c.DeleteVolume(volume.ObjectMeta.Name, options)
}

//ShowAboutInfo show the system information
func ShowAboutInfo() {
	fmt.Printf("Version: %s\n", utils.GenerateVersionStr())
	fmt.Printf("ProvisionerName: %s\n", *provisionerName)
	fmt.Printf("FlexVolumeDriverName: %s\n", *flexVolumeDriverName)
}

//ShowSampleCfg is used to generate a sample storage configure to refer
func ShowSampleCfg() {
	fmt.Print(utils.Dump13000SampleConfig())
}

//Run is the main routine
func Run(storageConfig string, kubeConfig string) {
	strcfg, err := utils.LoadConfig(storageConfig, "")
	if err != nil {
		glog.Fatalf("Failed to load config file %s.", err)
		os.Exit(1)
	}

	kubecfg, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
		os.Exit(1)
	}
	if *insecure {
		glog.Info("Use insecure, so ignore the cafile.")
		kubecfg.Insecure = true
		kubecfg.CAFile = ""
	}

	clientset, err := kubernetes.NewForConfig(kubecfg)
	if err != nil {
		glog.Fatalf("Failed to create client: %v.", err)
		os.Exit(1)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v.", err)
		os.Exit(1)
	}

	glog.Infof("Server version is %s.", serverVersion)

	provisioner := &InStorageProvisioner{
		config: *strcfg,
	}

	pc := ctrl.NewProvisionController(
		clientset,
		*provisionerName,
		provisioner,
		serverVersion.GitVersion,
		ctrl.Threadiness(*threadCount),
	)

	pc.Run(wait.NeverStop)
}

func main() {
	flag.Parse()

	//get the base directory
	baseDir := filepath.Dir(os.Args[0])
	//parse the configuration
	strCfgPath := filepath.Join(baseDir, "config", "instorage.yaml")

	if *about {
		ShowAboutInfo()
	} else if *sampleConfig {
		ShowSampleCfg()
	} else {
		if *strconfig != "" {
			strCfgPath = *strconfig
		}

		Run(strCfgPath, *kubeconfig)
	}
}
