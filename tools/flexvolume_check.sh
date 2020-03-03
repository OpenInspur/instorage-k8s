#volPoolName    = "volPoolName"
#volAuxPoolName = "volAuxPoolName"
#volIOGrp       = "volIOGrp"
#volAuxIOGrp    = "volAuxIOGrp"
#volThin        = "volThin"
#volCompress    = "volCompress" //bool indicate compressed volume
#volInTier      = "volInTier"   //bool indicate InTier volume
#
#volLevel = "volLevel" //basic or mirror or aa
#
#volThinResize    = "volThinResize"
#volThinGrainSize = "volThinGrainSize"
#volThinWarning   = "volThinWarning"
#volAutoExpand    = "volAutoExpand"

#create general volume
instorage create "test-001" 1 '{"inspur.com/volPoolName": "Pool0"}'

instorage create "test-002" 1 '{"inspur.com/volPoolName": "Pool0", "inspur.com/volThin": "true"}'

#create thin volume
instorage create "test-003" 1 '{"inspur.com/volPoolName": "Pool0", "inpsur.com/volIOGrp": "0", "inspur.com/volThin": "true", "inspur.com/volLevel": "basic", "inspur.com/volThinResize" : "10", "inspur.com/volThinGrainSize": "128", "inspur.com/volThinWarning": "60", "inspur.com/volAutoExpand": "true"}'

#create compress volume
instorage create "test-004" 1 '{"inspur.com/volPoolName": "Pool0", "inspur.com/volCompress": "true"}'

#create intier volume
instorage create "test-005" 1 '{"inspur.com/volPoolName": "Pool0", "inspur.com/volInTier": "true"}'

#create thin mirror volume
instorage create "test-006" 1 '{"inspur.com/volPoolName": "Pool0", "inspur.com/volAuxPoolName": "Pool1", "inspur.com/volThin": "true", "inspur.com/volLevel": "mirror"}'

#create aa thin volume
# autoexpand, intier is invalid for aa volume
instorage create "test-007" 1 '{"inspur.com/volPoolName": "Pool0", "inspur.com/volAuxPoolName": "Pool1", "inspur.com/volIOGrp": "0", "inspur.com/volAuxIOGrp": "1", "inspur.com/volThin": "true", "inspur.com/volLevel": "aa"}'

#attach volume
instorage waitforattach "/dev/sdabc" '{"kubernetes.io/pvOrVolumeName": "test-006"}'

#umount volume
instorage unmountdevice '/mnt'

#detach volume
instorage detach "test-006" "hostname"
#delete volume
instorage delete "test-007"

# expand volume
./instorage expandfs '{"kubernetes.io/pvOrVolumeName": "test-v001", "volLevel": "aa"}' '/dev/dm-2' '/mnt' 4294967296 2147483648