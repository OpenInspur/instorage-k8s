package csiplugin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"inspur.com/storage/instorage-k8s/pkg/controller"
	"inspur.com/storage/instorage-k8s/pkg/csiplugin/csicommon"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
	ctrl controller.IController
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	glog.Infof("ControllerServer CreateVolume req: %v", req)
	volName := req.GetName()

	capacity := req.GetCapacityRange().GetRequiredBytes()
	//capacity is size in bytes, transform it to size in Gib
	capacityValue := (capacity + 1*1024*1024*1024 - 1) / (1 * 1024 * 1024 * 1024)
	//volSize is the volume size in Gib
	volSize := strconv.FormatInt(capacityValue, 10)

	parameters := req.GetParameters()

	volumeContentSource := req.GetVolumeContentSource()
	var err error = nil
	if volumeContentSource == nil {
		_, err = cs.ctrl.CreateVolume(volName, volSize, parameters)
	} else {
		volumeSource := volumeContentSource.GetVolume()
		snapshotSource := volumeContentSource.GetSnapshot()
		var sourveVolumeName string = ""
		if volumeSource != nil {
			sourveVolumeName = volumeSource.GetVolumeId()
		}
		var snapshotName string = ""
		if snapshotSource != nil {
			snapshotName = snapshotSource.GetSnapshotId()
		}
		_, err = cs.ctrl.CloneVolume(volName, volSize, parameters, sourveVolumeName, snapshotName)
	}

	if err != nil {
		return nil, fmt.Errorf("create volume %s failed %s", volName, err)
	}

	vol := &csi.Volume{
		CapacityBytes: capacity,
		VolumeId:      volName,
		ContentSource: volumeContentSource,
	}

	resp := &csi.CreateVolumeResponse{
		Volume: vol,
	}

	return resp, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	glog.Infof("ControllerServer DeleteVolume req: %v", req)
	volumeID := req.GetVolumeId()
	options := map[string]string{}

	if err := cs.ctrl.DeleteVolume(volumeID, options); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%s", err))
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	glog.Infof("ControllerServer ListVolumes req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_LIST_VOLUMES); err != nil {
		return nil, err
	}
	if volumeNames, capacitiesBytes, nextID, err := cs.ctrl.ListVolume(req.GetMaxEntries(), req.GetStartingToken()); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%s", err))
	} else {
		var entries []*csi.ListVolumesResponse_Entry
		for index, volumeName := range volumeNames {
			var entry csi.ListVolumesResponse_Entry
			entry.Volume = &csi.Volume{
				VolumeId:      volumeName,
			    CapacityBytes: capacitiesBytes[index]}
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
	glog.Infof("ControllerServer ControllerPublishVolume req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME); err != nil {
		return nil, err
	}
	glog.Infof("ControllerServer ControllerPublishVolume nodeId: %s, volumeId: %s", req.GetNodeId(), req.GetVolumeId())
	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	glog.Infof("ControllerServer ControllerUnpublishVolume req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME); err != nil {
		return nil, err
	}
	glog.Infof("ControllerServer ControllerUnpublishVolume nodeId: %s, volumeId: %s", req.GetNodeId(), req.GetVolumeId())

	volumeName := req.GetVolumeId()
	hostname := req.GetNodeId()
	err := cs.ctrl.Detach(hostname, volumeName, false)
	if err != nil {
		glog.Errorf("Exit DetachCmd exec with error:%+v", err)
		return nil, err
	} else {
		glog.Debugf("Exit DetachCmd exec")
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	glog.Infof("ControllerServer ValidateVolumeCapabilities req: %v", req)
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
	glog.Infof("ControllerServer GetCapacity req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_GET_CAPACITY); err != nil {
		return nil, err
	}
	parameters := req.GetParameters()
	availableCapacity, err := cs.ctrl.GetCapacity(parameters)
	if err != nil {
		return nil, fmt.Errorf("get capacity %s failed %s", parameters, err)
	}
	resp := &csi.GetCapacityResponse{
		AvailableCapacity: availableCapacity,
	}
	return resp, nil
}

func (cs *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	glog.Infof("ControllerServer CreateSnapshot req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT); err != nil {
		return nil, err
	}

	sourceVolName := req.GetSourceVolumeId()
	snapshotName := req.GetName()
	readyToUse, ctimeStr, err := cs.ctrl.CreateSnapshot(sourceVolName, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("create snapshot %s failed %s", snapshotName, err)
	}

	var ctimeStamp *timestamp.Timestamp = nil
	if len(ctimeStr) == 12 {
		stdCtimeStr := "20" + ctimeStr[0:2] + "-" + ctimeStr[2:4] + "-" + ctimeStr[4:6] + " " + ctimeStr[6:8] + ":" + ctimeStr[8:10] + ":" + ctimeStr[10:12]
		template := "2006-01-02 15:04:05"
		ctime, errParse := time.ParseInLocation(template, stdCtimeStr, time.Local)
		if errParse == nil {
			ctimeStamp, _ = ptypes.TimestampProto(ctime)
		}
	}

	snapshot := &csi.Snapshot{
		SnapshotId:     snapshotName,
		SourceVolumeId: sourceVolName,
		ReadyToUse:     readyToUse,
		CreationTime:   ctimeStamp,
	}

	resp := &csi.CreateSnapshotResponse{
		Snapshot: snapshot,
	}

	return resp, nil
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	glog.Infof("ControllerServer DeleteSnapshot req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT); err != nil {
		return nil, err
	}
	err := cs.ctrl.DeleteSnapshot(req.GetSnapshotId())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%s", err))
	}

	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	glog.Infof("ControllerServer ListSnapshots req: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS); err != nil {
		return nil, err
	}
	sourceVolumeId := req.GetSourceVolumeId()
	if snapshotIds, sourcevolIds, nextID, err := cs.ctrl.ListSnapshots(req.GetMaxEntries(), req.GetStartingToken(), sourceVolumeId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%s", err))
	} else {
		ctime := ptypes.TimestampNow()
		var entries []*csi.ListSnapshotsResponse_Entry
		for index, snapshotId := range snapshotIds {
			var entry csi.ListSnapshotsResponse_Entry
			entry.Snapshot = &csi.Snapshot{
				SourceVolumeId: sourcevolIds[index],
				SnapshotId:     snapshotId,
				CreationTime:   ctime}
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
