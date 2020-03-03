package csiplugin

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"inspur.com/storage/instorage-k8s/pkg/controller"
	"inspur.com/storage/instorage-k8s/pkg/utils"

	"inspur.com/storage/instorage-k8s/pkg/csiplugin/csicommon"
)

type driver struct {
	csiDriver *csicommon.CSIDriver
	ctrl      controller.IController
	endpoint  string

	mode string

	ids *csicommon.DefaultIdentityServer
	ns  *nodeServer

	cap   []*csi.VolumeCapability_AccessMode
	cscap []*csi.ControllerServiceCapability
}

// NewDriver create an instance of instorage csi driver
func NewDriver(mode string, driverName string, nodeID string, endpoint string, config utils.Config) *driver {
	version := utils.GenerateVersionStr()

	glog.Infof("Driver: %v version: %v", driverName, version)

	d := &driver{}

	csiDriver := csicommon.NewCSIDriver(driverName, version, nodeID)
	csiDriver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER})
	csiDriver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_GET_CAPACITY,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
		csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	})
	csiDriver.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
	})

	d.csiDriver = csiDriver
	d.ctrl = controller.NewController(config)
	d.endpoint = endpoint
	d.mode = mode

	return d
}

func newNodeServer(d *driver) *nodeServer {
	return &nodeServer{
		DefaultNodeServer: csicommon.NewDefaultNodeServer(d.csiDriver),
		ctrl:              d.ctrl,
	}
}

func newControllerServer(d *driver) *controllerServer {
	return &controllerServer{
		DefaultControllerServer: csicommon.NewDefaultControllerServer(d.csiDriver),
		ctrl:                    d.ctrl,
	}
}

func (d *driver) Run() {
	s := csicommon.NewNonBlockingGRPCServer()

	switch d.mode {
	case "nodeworker":
		s.Start(d.endpoint,
			csicommon.NewDefaultIdentityServer(d.csiDriver),
			nil,
			newNodeServer(d))
	case "controller":
		s.Start(d.endpoint,
			csicommon.NewDefaultIdentityServer(d.csiDriver),
			newControllerServer(d),
			nil)
	case "all-in-one":
		s.Start(d.endpoint,
			csicommon.NewDefaultIdentityServer(d.csiDriver),
			newControllerServer(d),
			newNodeServer(d))
	}
	s.Wait()
}
