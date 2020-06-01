package csiplugin

import (
	"fmt"
	"strconv"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"inspur.com/storage/instorage-k8s/pkg/controller"
	"inspur.com/storage/instorage-k8s/pkg/csiplugin/csicommon"
	"inspur.com/storage/instorage-k8s/pkg/storage"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
	ctrl controller.IController
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	glog.Infof("[controllerServer::CreateVolume] req: %v", req)
	volName := req.GetName()

	capacity := req.GetCapacityRange().GetRequiredBytes()
	//capacity is size in bytes, transform it to size in Gib
	capacityValue := (capacity + 1 * 1024 * 1024 * 1024 - 1) / (1 * 1024 * 1024 * 1024)
	//volSize is the volume size in Gib
	volSize := strconv.FormatInt(capacityValue, 10)
	parameters := req.GetParameters()
	volumeContentSource := req.GetVolumeContentSource()
	var err error = nil
	var info map[string]string
	if volumeContentSource == nil {
		info, err = cs.ctrl.CreateVolume(volName, volSize, parameters)
	} else {
		var options map[string]string
		volumeSource := volumeContentSource.GetVolume()
		snapshotSource := volumeContentSource.GetSnapshot()
		var sourceVolumeName string
		if volumeSource != nil {
			sourceVolumeName, options, err = csicommon.ParseVolumeID(volumeSource.GetVolumeId())
			if err != nil {
				return nil, fmt.Errorf("[controllerServer::CreateVolume] failed to parse volume id, sourceVolumeId: %s, error: %v", volumeSource.GetVolumeId(), err)
			}
		}
		var snapshotName string
		if snapshotSource != nil {
			snapshotName, options, err = csicommon.ParseVolumeID(snapshotSource.GetSnapshotId())
			if err != nil {
				return nil, fmt.Errorf("[controllerServer::CreateVolume] failed to parse volume id, snapshotId: %s, error: %v", snapshotSource.GetSnapshotId(), err)
			}
		}
		info, err = cs.ctrl.CloneVolume(volName, volSize, parameters, sourceVolumeName, snapshotName, options)
	}
	if err != nil {
		return nil, fmt.Errorf("[controllerServer::CreateVolume]  failed to create volume, error: %v", err)
	}
	vol := &csi.Volume{
		CapacityBytes: capacity,
		VolumeId:      csicommon.GenerateVolumeID(volName, info),
		ContentSource: volumeContentSource,
	}
	resp := &csi.CreateVolumeResponse{
		Volume: vol,
	}
	return resp, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	glog.Infof("[controllerServer::DeleteVolume] req: %v", req)
	volumeId := req.GetVolumeId()
	name, options, err := csicommon.ParseVolumeID(volumeId)
	if err != nil {
		return nil, fmt.Errorf("[controllerServer::DeleteVolume] failed to parse volume id, sourceVolumeId: %s, error: %v", volumeId, err)
	}
	err = cs.ctrl.DeleteVolume(name, options)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%s", err))
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	glog.Infof("[controllerServer::ListVolumes] req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_LIST_VOLUMES); err != nil {
		return nil, err
	}
	if volumeMap, nextID, err := cs.ctrl.ListVolume(req.GetMaxEntries(), req.GetStartingToken()); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%s", err))
	} else {
		var entries []*csi.ListVolumesResponse_Entry
		for name, info := range volumeMap {
			_, capacity := csicommon.ParseCapacitySize(info[storage.VolumeCapacity])
			delete(info, storage.VolumeCapacity)
			var entry csi.ListVolumesResponse_Entry
			entry.Volume = &csi.Volume{
				VolumeId:      csicommon.GenerateVolumeID(name, info),
			    CapacityBytes: capacity}
			entries = append(entries, &entry)
		}
		return &csi.ListVolumesResponse{
			Entries:   entries,
			NextToken: nextID,
		}, nil
	}
}

// ControllerPublishVolume will attach the volume to the specified node
func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	glog.Infof("[controllerServer::ControllerPublishVolume] req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME); err != nil {
		return nil, err
	}
	glog.Infof("[controllerServer::ControllerPublishVolume] nodeId: %s, volumeId: %s", req.GetNodeId(), req.GetVolumeId())
	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	glog.Infof("[controllerServer::ControllerUnpublishVolume] req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME); err != nil {
		return nil, err
	}
	glog.Infof("[controllerServer::ControllerUnpublishVolume], nodeId: %s, volumeId: %s", req.GetNodeId(), req.GetVolumeId())

	volumeName := req.GetVolumeId()
	hostname := req.GetNodeId()
	err := cs.ctrl.Detach(hostname, volumeName, false)
	if err != nil {
		glog.Errorf("[controllerServer::ControllerUnpublishVolume], Exit DetachCmd exec with error:%+v", err)
		return nil, err
	} else {
		glog.Debugf("[controllerServer::ControllerUnpublishVolume], Exit DetachCmd exec")
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	glog.Infof("[controllerServer::ValidateVolumeCapabilities] req: %v", req)
	volumeCapabilities := req.GetVolumeCapabilities()
	for _, capability := range volumeCapabilities {
		if capability.GetAccessMode() == nil {
			return &csi.ValidateVolumeCapabilitiesResponse{}, nil
		}
		if capability.GetAccessMode().GetMode() > csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER {
			return &csi.ValidateVolumeCapabilitiesResponse{}, nil
		}
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: volumeCapabilities,
		},
	}, nil

}

func (cs *controllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	glog.Infof("[controllerServer::GetCapacity] req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_GET_CAPACITY); err != nil {
		return nil, err
	}
	parameters := req.GetParameters()
	availableCapacity, err := cs.ctrl.GetCapacity(parameters)
	if err != nil {
		return nil, fmt.Errorf("[controllerServer::GetCapacity] failed to get capacity, error: %v", err)
	}
	resp := &csi.GetCapacityResponse{
		AvailableCapacity: availableCapacity,
	}
	return resp, nil
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	glog.Infof("[controllerServer::CreateSnapshot] req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT); err != nil {
		return nil, err
	}
	sourceVolumeId := req.GetSourceVolumeId()
	srcVolName, options, err := csicommon.ParseVolumeID(req.GetSourceVolumeId())
	if err != nil {
		return nil, fmt.Errorf("[controllerServer::CreateSnapshot] failed to parse volume id, sourceVolumeId: %s, error: %v", sourceVolumeId, err)
	}
	snapshotName := req.GetName()
	info, err := cs.ctrl.CreateSnapshot(srcVolName, snapshotName, options)
	if err != nil {
		return nil, fmt.Errorf("[controllerServer::CreateSnapshot] failed to create snapshot, srcVolName: %s, snapshotName: %s, error: %v", srcVolName, snapshotName, err)
	}
	var timeStamp *timestamp.Timestamp
	var readyToUse bool
	if info != nil {
		timeStamp, err = csicommon.ConvertStrToTime(info[storage.SnapshotCreateTime])
		if err != nil {
			return nil, fmt.Errorf("[controllerServer::CreateSnapshot] failed to create snapshot, snapshotName: %s, error: %v", snapshotName, err)
		}
		readyToUse, err = strconv.ParseBool(info[storage.SnapshotReadyToUse])
		if err != nil {
			return nil, fmt.Errorf("[controllerServer::CreateSnapshot] failed to create snapshot, snapshotName: %s, error: %v", snapshotName, err)
		}
		delete(info, storage.SnapshotCreateTime)
		delete(info, storage.SnapshotReadyToUse)
	}
	snapshot := &csi.Snapshot{
		SnapshotId:     csicommon.GenerateVolumeID(snapshotName, info),
		SourceVolumeId: sourceVolumeId,
		ReadyToUse:     readyToUse,
		CreationTime:   timeStamp,
	}
	resp := &csi.CreateSnapshotResponse{
		Snapshot: snapshot,
	}

	return resp, nil
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	glog.Infof("[controllerServer::DeleteSnapshot] req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT); err != nil {
		return nil, err
	}
	snapshotId := req.GetSnapshotId()
	snapshotName, options, err := csicommon.ParseVolumeID(snapshotId)
	if err != nil {
		return nil, fmt.Errorf("[controllerServer::DeleteSnapshot] failed to snapshot id, snapshotId: %s, error: %v", snapshotId, err)
	}
	err = cs.ctrl.DeleteSnapshot(snapshotName, options)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%s", err))
	}

	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	glog.Infof("[controllerServer::ListSnapshots] req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS); err != nil {
		return nil, err
	}
	sourceVolumeId := req.GetSourceVolumeId()
	srcVolName, options, err := csicommon.ParseVolumeID(req.GetSourceVolumeId())
	if err != nil {
		return nil, fmt.Errorf("[controllerServer::ListSnapshots] failed to parse volume id, sourceVolumeId: %s, error: %v", sourceVolumeId, err)
	}
	if snapshotMap, nextID, err := cs.ctrl.ListSnapshots(req.GetMaxEntries(), req.GetStartingToken(), srcVolName, options); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%s", err))
	} else {
		var entries []*csi.ListSnapshotsResponse_Entry
		for snapshotName, info := range snapshotMap {
			timeStamp, errConvert := csicommon.ConvertStrToTime(info[storage.SnapshotCreateTime])
			if errConvert != nil {
				glog.Warning("[controllerServer::ListSnapshots] failed to convert string to time, %s, %v", info[storage.SnapshotCreateTime], errConvert)
				timeStamp = nil
			}
			delete(info, storage.SnapshotCreateTime)
			var entry csi.ListSnapshotsResponse_Entry
			entry.Snapshot = &csi.Snapshot{
				SourceVolumeId: req.GetSourceVolumeId(),
				SnapshotId:     csicommon.GenerateVolumeID(snapshotName, info),
				CreationTime:  timeStamp}
			entries = append(entries, &entry)
		}
		return &csi.ListSnapshotsResponse{
			Entries:   entries,
			NextToken: nextID,
		}, nil
	}
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         req.GetCapacityRange().GetRequiredBytes(),
		NodeExpansionRequired: true,
	}, nil
}