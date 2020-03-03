package csiplugin

import (
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"inspur.com/storage/instorage-k8s/pkg/controller"
	"inspur.com/storage/instorage-k8s/pkg/csiplugin/csicommon"
)

type nodeServer struct {
	*csicommon.DefaultNodeServer
	ctrl controller.IController
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	glog.Infof("nodeServer NodePublishVolume req: %v", req)
	targetPath := req.GetTargetPath()
	volumeName := req.GetVolumeId()
	fsType := "ext4"
	options := map[string]string{}

	_, err := ns.ctrl.MountDevice(volumeName, targetPath, fsType, options)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	glog.Infof("nodeServer NodeUnpublishVolume req: %v", req)
	targetPath := req.GetTargetPath()

	_, err := ns.ctrl.UnMountDevice(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
	glog.Infof("nodeServer NodeExpandVolume req: %v", req)
	volumeId := req.GetVolumeId()
	volumePath := req.GetVolumePath()
	requiredBytes := req.GetCapacityRange().RequiredBytes
	var requiredGB int64 = requiredBytes / 1024 / 1024 / 1024
	volSpec := make(map[string]string)
	err := ns.ctrl.ExtendVolume(volumeId, strconv.FormatInt(requiredGB, 10), "0", "", volumePath, volSpec)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodeExpandVolumeResponse{
		CapacityBytes: requiredBytes,
	}, nil
}

func (ns *nodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	glog.Infof("nodeServer NodeGetVolumeStats req: %v", req)
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
