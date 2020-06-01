package csiplugin

import (
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"fmt"

	"inspur.com/storage/instorage-k8s/pkg/controller"
	"inspur.com/storage/instorage-k8s/pkg/csiplugin/csicommon"
	"inspur.com/storage/instorage-k8s/pkg/host"
	"inspur.com/storage/instorage-k8s/pkg/storage"
)

type nodeServer struct {
	*csicommon.DefaultNodeServer
	ctrl controller.IController
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	glog.Infof("[nodeServer::NodePublishVolume] req: %v", req)
	name, options, err := csicommon.ParseVolumeID(req.GetVolumeId())
	if err != nil {
		return nil, fmt.Errorf("[nodeServer::NodePublishVolume] failed to parse volume id, sourceVolumeId: %s, error: %v", req.GetVolumeId(), err)
	}
	//mount block volume
	if options == nil || options[storage.DevKind] == "block" { //is AS18000 storage, or AS13000 block
		_, err = ns.ctrl.MountDevice(name, req.GetTargetPath(), "ext4", options)
	} else { //mount nfs
		_, err = ns.ctrl.MountDevice(name, req.GetTargetPath(), "nfs", options)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	glog.Infof("[nodeServer::NodeUnpublishVolume] req: %v", req)
	_, options, err := csicommon.ParseVolumeID(req.GetVolumeId())
	if err != nil {
		return nil, fmt.Errorf("[nodeServer::NodeUnpublishVolume] failed to parse volume id, sourceVolumeId: %s, error: %v", req.GetVolumeId(), err)
	}

	if options == nil || options[storage.DevKind] == "block" { //is AS18000 storage, or AS13000 block

		_, err := ns.ctrl.UnMountDevice(req.GetTargetPath())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		m := host.Mounter{}
		_, err := m.GetDevice(req.GetTargetPath()) //whether mounted or not
		if err == nil {                            //mounted
			if err := m.Unmount(req.GetTargetPath()); err != nil {
				return &csi.NodeUnpublishVolumeResponse{}, fmt.Errorf("unmount path failed. %s", err)
			}
		}
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	glog.Infof("[nodeServer::NodeExpandVolume] req: %v", req)

	// 1 get require capacity
	requiredBytes := req.GetCapacityRange().RequiredBytes
	var requiredGB int64 = requiredBytes / 1024 / 1024 / 1024

	// 2 get volume name and options
	name, options, err := csicommon.ParseVolumeID(req.GetVolumeId())
	if err != nil {
		return nil, fmt.Errorf("[nodeServer::NodePublishVolume] failed to parse volume id, sourceVolumeId: %s, error: %v", req.GetVolumeId(), err)
	}

	// 3 expand volume
	err = ns.ctrl.ExtendVolume(name, strconv.FormatInt(requiredGB, 10), "0", "", req.GetVolumePath(), options)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodeExpandVolumeResponse{
		CapacityBytes: requiredBytes,
	}, nil
}

func (ns *nodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	glog.Infof("[nodeServer::NodeGetVolumeStats] req: %v", req)
	volumePath := req.GetVolumePath()
	total, used, available, err := ns.ctrl.GetVolumeStats(volumePath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var usage []*csi.VolumeUsage
	item := &csi.VolumeUsage{
		Total:     total,
		Used:      used,
		Available: available,
		Unit:      csi.VolumeUsage_BYTES,
	}
	usage = append(usage, item)
	return &csi.NodeGetVolumeStatsResponse{
		Usage: usage,
	}, nil
}
