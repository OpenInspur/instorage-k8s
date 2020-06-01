package storage

const (
	DevKind = "devKind" // block or share

	VolPoolName    = "volPoolName"
	VolAuxPoolName = "volAuxPoolName"
	VolIOGrp       = "volIOGrp"
	VolAuxIOGrp    = "volAuxIOGrp"
	VolThin        = "volThin"
	VolCompress    = "volCompress" //bool indicate compressed volume
	VolInTier      = "volInTier"   //bool indicate InTier volume

	VolLevel = "volLevel" //basic or mirror or aa

	VolThinResize    = "volThinResize"
	VolThinGrainSize = "volThinGrainSize"
	VolThinWarning   = "volThinWarning"
	VolAutoExpand    = "volAutoExpand"

	SharePoolName     = "sharePoolName"
	ShareAccessClient = "shareAccessClient"

	VolumeCapacity    = "volumeCapacity"

	SnapshotSourceName = "sn"
	SnapshotPoolName   = "pn"
	SnapshotCreateTime = "snapshotCreateTime"
	SnapshotReadyToUse = "snapshotReadyToUse"

	True  = "true"
	False = "false"
)

// ConnProperty contain the LUN's connect info which is used to search the device on host.
type ConnProperty struct {
	Protocol string /* fc or iscsi */

	WWPNs   []string
	Targets []string
	Portals []string
	LunIDs  []string
}

const (
	HostLinkFC    = "fc"
	HostLinkiSCSI = "iscsi"
)

//HostInfo keep connect information about a host.
type HostInfo struct {
	Hostname  string
	Link      string
	Initiator string
	WWPNs     []string
}

//IManager is the interface use to manage a storage
type IManager interface {
	CreateVolume(name string, size string, options map[string]string) (string, error)
	DeleteVolume(volumeName string) error
}

//IStorageOperater define the general storage related operation.
type IStorageOperater interface {
	AttachVolume(volumeName string, hostInfo HostInfo, options map[string]string) (*ConnProperty, error)
	DetachVolume(volumeName string, hostInfo HostInfo) error
	GetVolumeAttachInfo(volumeName string, hostInfo HostInfo) (*ConnProperty, error)
	GetVolumeNameWithUID(uid string) (string, error)
	CreateVolume(name string, size string, options map[string]string) (map[string]string, error)
	CloneVolume(name string, size string, parameters map[string]string, sourceVolumeName string, snapshotName string, options map[string]string) (map[string]string, error)
	DeleteVolume(volumeName string, options map[string]string) error
	ExtendVolume(name string, size string, options map[string]string) error
	NeedFreezeFSWhenExtend(volumeName string, options map[string]string) bool
	ListVolume(maxEnties int32, startingToken string) (map[string]map[string]string, string, error)
	GetCapacity(options map[string]string) (int64, error)
	CreateSnapshot(sourceVolName string, snapshotName string, options map[string]string) (map[string]string, error)
	DeleteSnapshot(snapshotId string, options map[string]string) error
	ListSnapshots(maxEnties int32, startingToken string, sourceVolName string, options map[string]string) (map[string]map[string]string, string, error)
}
