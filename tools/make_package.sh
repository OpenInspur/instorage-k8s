#!/bin/bash

set -e

VERSION="1.0.0.0"
COMMAND=$0
KIND=""

BUILD_FLEX=""
BUILD_CSI=""

ARCH=""

# show help
function show_usage() {
    echo "Usage: $COMMAND [-h | -v | -a [amd64|mips64le|arm64|all] -t [13000|18000] [-p [csi|flex|all]]"
    echo ""
    echo "A tool to generate the package of inspur storage K8sPlugin for AS13000 and AS18000."
    echo ""
    echo "Options:"
    echo "    -h    show this help"
    echo "    -v    show the version"
    echo "    -a    amd64 or mips64 or arm64"
    echo "          all for amd64 and mips64 and arm64"
    echo "    -t    13000 for AS13000"
    echo "          18000 for AS18000"
    echo "    -p    csi for csi plugin"
    echo "          flex for flex and external-provisioner"
    echo "          all for both flex and csi"
}

# deal with option
while getopts "a:p:t:hv" arg
do
  case $arg in
  a)
    case ${OPTARG} in
      'amd64')
         ARCH="$ARCH amd64"
         ;;
      'mips64le')
         ARCH="$ARCH mips64le"
         ;;
      'arm64')
         ARCH="$ARCH arm64"
         ;;
      'all')
         ARCH="amd64 mips64le arm64"
         ;;
      *)
         show_usage
         exit 1
         ;;
    esac
    ;;
  t)
    case ${OPTARG} in
      '13000')
         KIND=13000
         ;;
      '18000')
         KIND=18000
         ;;
      *)
         show_usage
         exit 1
         ;;
    esac
    ;;
  p)
    case ${OPTARG} in
      'csi')
         BUILD_CSI="true"
         ;;
      'flex')
         BUILD_FLEX="true"
         ;;
      'all')
         BUILD_FLEX="true"
         BUILD_CSI="true"
         ;;
      *)
         show_usage
         exit 1
         ;;
    esac
    ;;
  h)
    show_usage
    exit 0
    ;;
  v)
    echo Version: $VERSION
    exit 0
    ;;
  ?)
    show_usage
    exit 1
    ;;
  esac
done

if [[ $KIND == "" ]]
then
  show_usage
fi

COMMIT=$(git rev-parse HEAD)
COMMIT_ID_VAR="inspur.com/storage/instorage-k8s/pkg/utils.CommitID"

CIPHER_KEY=$(grep 'cipher.key' build_config | cut -d '=' -f2)
CIPHER_KEY_VAR="inspur.com/storage/instorage-k8s/pkg/utils.CipherKey"

Timestamp=$(date +"%Y%m%d")

#build the helper
go install -mod=vendor build_helper.go

Version=$($GOBIN/build_helper -version | cut -d' ' -f2)

PACKAGE_NAME=K8sPlugin_V${Version}.Build${Timestamp}
mkdir -p $PACKAGE_NAME

$GOBIN/build_helper -sample-cfg-${KIND} > $PACKAGE_NAME/instorage.yaml

for arch in $ARCH
do
    BUILD_NAME=${PACKAGE_NAME}_${arch}

    if [[ $BUILD_FLEX == "true" ]]
    then
        CGO_ENABLED=0 GOOS=linux GOARCH=$arch go install -mod=vendor -a -ldflags "-X $COMMIT_ID_VAR=$COMMIT -X $CIPHER_KEY_VAR=$CIPHER_KEY -extldflags '-static'" -tags netgo ../pkg/cmd/flexvolume/instorage.go
        CGO_ENABLED=0 GOOS=linux GOARCH=$arch go install -mod=vendor -a -ldflags "-X $COMMIT_ID_VAR=$COMMIT -X $CIPHER_KEY_VAR=$CIPHER_KEY -extldflags '-static'" -tags netgo ../pkg/cmd/provisioner/provisioner.go

        FLEXDRIVER_DIR=$PACKAGE_NAME/$BUILD_NAME/inspur~instorage-flexvolume
        mkdir -p $FLEXDRIVER_DIR/{config,log}
        #strip $GOBIN/instorage
        mv $GOBIN/instorage $FLEXDRIVER_DIR/instorage-flexvolume
        cp $PACKAGE_NAME/instorage.yaml $FLEXDRIVER_DIR/config/instorage.yaml

        PROVISIONER_DIR=$PACKAGE_NAME/$BUILD_NAME/inspur-instorage-provisioner
        mkdir -p $PROVISIONER_DIR/config
        #strip $GOBIN/provisioner
        mv $GOBIN/provisioner $PROVISIONER_DIR/instorage-provisioner
        cp $PACKAGE_NAME/instorage.yaml $PROVISIONER_DIR/config/instorage.yaml
    fi

    if [[ $BUILD_CSI == "true" ]]
    then
        CGO_ENABLED=0 GOOS=linux GOARCH=$arch go install -mod=vendor -a -ldflags "-X $COMMIT_ID_VAR=$COMMIT -X $CIPHER_KEY_VAR=$CIPHER_KEY -extldflags '-static'" -tags netgo ../pkg/cmd/csiplugin/csiplugin.go

        # collect the csi
        CSIDRIVER_DIR=$PACKAGE_NAME/$BUILD_NAME/csiplugin
        mkdir -p $CSIDRIVER_DIR
        mv $GOBIN/csiplugin $CSIDRIVER_DIR/csiplugin

        DEMO_DIR=$CSIDRIVER_DIR/demo
        mkdir -p $DEMO_DIR
        cp ../demo/csi-pvc.yaml $DEMO_DIR
        cp ../demo/csi-storageClass.yaml $DEMO_DIR

        DEPLOY_DIR=$CSIDRIVER_DIR/deploy
        mkdir -p $DEPLOY_DIR
        cp ../image/configMap.yaml $DEPLOY_DIR
        cp ../image/instorage-k8s-csi/* $DEPLOY_DIR
        rm $DEPLOY_DIR/Dockerfile
        cp ../image/instorage-k8s-csi/Dockerfile $CSIDRIVER_DIR


    fi

    # archive the plugin
    cd $PACKAGE_NAME
    tar --owner=0 --group=0 -czvf ${BUILD_NAME}.tar.gz $BUILD_NAME
    cd ..

    # do clean
    rm -rf $PACKAGE_NAME/$BUILD_NAME
done

# do clean
rm $PACKAGE_NAME/instorage.yaml

# Generate Docker Image tar file
