# 说明
浪潮存储CSI驱动遵从CSI接口协议规范，利用浪潮存储CSI驱动，在Kubernetes的使用过程中，可以直接利用Kubernetes的管理命令来触发浪潮存储上的卷，快照，主机信息等的管理操作，并可以将存储上的卷提供给应用服务使用，用于服务的持久化数据存储。

浪潮存储CSI驱动，主要包括Identity、Controller、Node三个服务。Identity服务负责驱动信息、能力、状态的获取；Controller服务负责卷的创建、删除、克隆和快照的创建、删除；Node服务负责卷的挂载、卸载。浪潮存储CSI驱动拥有三种工作模式，‘all-in-one’、‘controller’、‘nodeworker’。‘all-in-one’模式同时包含上述三个服务，部署在Kubernetes集群的Master节点；‘controller’模式包含Identity和Controller两个服务，部署在Kubernetes的Master节点；‘nodeworker’模式包含Identity和Node两个服务，部署在Kubernetes的所有节点。

浪潮存储CSI驱动兼容CSI v1.3版本接口规范。支持功能列表如下：

编号 | 功能 | 说明
-----|------|------
1 | 卷挂载 | 支持FC，iSCSI链路，支持多路径设备管理
2 | 卷卸载 |
3 | 挂载卷文件系统到主机 |
4 | 从主机上卸载卷文件系统 |
5 | 卷创建 | 普通卷，镜像卷，双活卷
6 | 删除卷 |
7 | 卷克隆 | 基于卷的克隆，基于快照的克隆
8 | 卷在线扩容 |
9 | 快照创建 |
10 | 快照删除 |

## 1 编译与部署
### 1.1 编译驱动
1. 克隆驱动源码仓库到本地
    ```
    git clone https://github.com/OpenInspur/instorage-k8s.git
    ```
2. 进入tools目录，在目录中利用make_package.sh脚本编译驱动。通过./make_package.sh -h可获取帮助信息，当前可以支持amd64, mips64le, arm64等平台选项，可选的支持csi, flex两种编译结果输出。
    ```
    dev@lab:~/instorage-k8s/tools$ ./make_package.sh -h
    Usage: ./make_package.sh [-h | -v | -a [amd64|mips64le|arm64|all] -t [13000|18000] [-p [csi|flex|all]]
    
    A tool to generate the package of inspur storage K8sPlugin for AS13000 and AS18000.
    
    Options:
        -h    show this help
        -v    show the version
        -a    amd64 or mips64 or arm64
              all for amd64 and mips64 and arm64
        -t    13000 for AS13000
              18000 for AS18000
        -p    csi for csi plugin
              flex for flex and external-provisioner
              all for both flex and csi
    ```
3. 在tools目录下创建build_config配置文件，加入编译配置信息。示例如下：
    ```
    dev@lab:~/instorage-k8s/tools$ ls
    build_config  build_helper.go  flexvolume_check.sh  make_package.sh
    dev@lab:~/instorage-k8s/tools$ cat build_config
    cipher.key=MY_CIPHER_KEY
    ```
    **编译配置文件支持如下配置选项**

    配置项名称 | 说明
    -----------|-----
    cipher.key | 加密函数所使用的加密密钥

4. 如编译amd64平台，AS18000存储系列的CSI驱动
    ```
    dev@lab:~/instorage-k8s/tools$ ./make_package.sh -a amd64 -t 18000 -p csi
    ```
5. 编译完成后，会在当前目录下生成驱动版本包，如
    ```
    dev@lab:~/instorage-k8s/tools$ ls
    build_config  build_helper.go  flexvolume_check.sh  K8sPlugin_V2.1.0.Build20200226  make_package.sh
    dev@lab:~/instorage-k8s/tools$ tree
    .
    ├── build_config
    ├── build_helper.go
    ├── flexvolume_check.sh
    ├── K8sPlugin_V2.1.0.Build20200226
    │   └── K8sPlugin_V2.1.0.Build20200226_amd64.tar.gz
    └── make_package.sh
    
    1 directory, 5 files
    dev@lab:~/instorage-k8s/tools$ tar -tzvf K8sPlugin_V2.1.0.Build20200226/K8sPlugin_V2.1.0.Build20200226_amd64.tar.gz
    drwxrwxr-x dev/dev         0 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/
    drwxrwxr-x dev/dev         0 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/
    -rwxrwxr-x dev/dev  15809675 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/csiplugin
    drwxrwxr-x dev/dev         0 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/
    -rw-rw-r-- dev/dev      3310 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/csi-rbac.yaml
    -rw-rw-r-- dev/dev       321 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/Dockerfile
    -rw-rw-r-- dev/dev       604 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/configMap.yaml
    -rw-rw-r-- dev/dev       434 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/README.md
    -rw-rw-r-- dev/dev      6711 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/csi-deploy.yaml
    drwxrwxr-x dev/dev         0 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/demo/
    -rw-rw-r-- dev/dev       251 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/demo/csi-storageClass.yaml
    -rw-rw-r-- dev/dev       446 2020-02-26 08:26 K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/demo/csi-pvc.yaml
    ```

### 1.2 制作容器镜像
1. 解压并进入驱动插件目录
    ```
    dev@lab:~$ tar -xzvf K8sPlugin_V2.1.0.Build20200226_amd64.tar.gz
    K8sPlugin_V2.1.0.Build20200226_amd64/
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/csiplugin
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/csi-rbac.yaml
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/Dockerfile
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/configMap.yaml
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/README.md
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/deploy/csi-deploy.yaml
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/demo/
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/demo/csi-storageClass.yaml
    K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin/demo/csi-pvc.yaml
    dev@lab:~$ cd K8sPlugin_V2.1.0.Build20200226_amd64/
    dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64$ ls
    csiplugin
    dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64$ cd csiplugin/
    dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin$ ls
    csiplugin  demo  deploy
    ```
2. 根据驱动包中的Dockerfile制作制作镜像，Dockerfile内容如下
    ```
    dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin$ cat deploy/Dockerfile
    #Dockerfile for Inspur Instorage CSI Driver
    
    FROM centos:7.6.1810
    
    LABEL maintainer="instorage.csi@inspur.com"
    
    COPY csiplugin csiplugin
    
    RUN yum -y install sysfsutils device-mapper device-mapper-multipath iscsi-initiator-utils e2fsprogs xfsprogs && yum clean all
    
    RUN /sbin/mpathconf --enable
    
    ENTRYPOINT ["/csiplugin"]
    ```
3. 制作容器镜像
    ```
    dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin$ docker build -t csiplugin:2.1.0 -f deploy/Dockerfile .
    ```
4. 完成容器镜像制作后，可根据具体需要，将镜像推送到指定仓库，供后续使用。

### 1.3 部署插件
本章节将演示在基于kubeadm方式部署的Kubernetes集群中使用浪潮存储CSI驱动的方法。

1. CSI驱动支持快照，克隆等操作，可以根据Kubernetes版本的需要设置kube-apiserver启动参数，在/etc/kubernetes/manifests/kube-apiserver.yaml文件中增加如下内容
    ```
    --feature-gates=VolumeSnapshotDataSource=true
    --feature-gates=VolumePVCDataSource=true
    --feature-gates=ExpandInUserPersistentVolumes=true
    --feature-gates=ExpandCSIVolume=true
    ```
2. CSI驱动支持卷的扩容操作，可以根据Kubernetes版本的需要设置kube-controller-manager启动参数，在/etc/kubernetes/manifests/kube-controller-manager.yaml文件中增加如下内容
    ```
    --feature-gates=ExpandInUserPersistentVolumes=true
    --feature-gates=ExpandCSIVolume=true
    ```
3. CSI驱动支持卷的扩容操作，可以根据Kubernetes版本的需要设置kubelet启动参数，在/usr/lib/system/system/kubelet.service.d/10-kubeadm.conf文件中增加如下内容
    ```
    --feature-gates=ExpandInUserPersistentVolumes=true
    --feature-gates=ExpandCSIVolume=true
    ```
    * 配置文件修改后，需要对kubelet服务进行重启，如利用systemctl restart kubelet来重启服务。
4. CSI驱动在启动以后，会通过unix域来接受gRPC请求，在部署时我们利用主机上的/var/lib/kubelet/plugins/csi-instorage目录来统一存放CSI驱动的unix域socket文件。csi-instorage目录需要在所有节点上提前创建。
    ```
    root@k8s:~# mkdir -p /var/lib/kubelet/plugins/csi-instorage
    ```
5. 在使用iSCSI链路场景下，驱动要求将主机上的/etc/iscsi目录映射到驱动所在容器的/etc/iscsi目录中，利用该目录下的initiatorname.iscsi, iscsid.conf 等配置会作为容器内部iscsi服务的标识，配置。在实际使用中，iSCSI链路通过驱动容器内部的iscsi客户端软件来管理链路。为避免服务异常，主机上不能安装iscsi客户端软件。

6. 需要在Kubernetes集群中为驱动创建合适的访问用户，并为用户赋予相应权限，可以利用示例的RBAC文件来创建相应内容。
    ```
    dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin$ kubectl create -f deploy/csi-rbac.yaml
    ```
7. CSI驱动访问存储系统的配置信息存放在Kubernetes的configMap中，需要根据实际部署修改configMap内容，可参考image/configMap.yaml文件，完成调整后，在Kubernetes系统中创建对应的configMap配置
    ```
    dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin$ kubectl create -f deploy/configMap.yaml
    ```
8. 根据驱动的容器镜像实际所在仓库位置及tag信息，调整驱动部署示例image/instorage-k8s-csi/csi-deploy.yaml文件内容，并通过kubectl部署驱动
    ```
    dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin$ kubectl create -f deploy/csi-deploy.yaml
    ```
至此，浪潮存储CSI驱动完成部署，之后便可以在Kubernetes平台上根据实际使用要求创建StorageClass，并通过PVC等方式使用存储上的资源。

## 2 使用说明
### 注意
1. 存储系统在使用前，需要完成存储集群初始化配置，并完成存储池的创建与初始化。
2. 存储系统的管理网络需要与所有存储CSI驱动所在的节点互通，CSI驱动会通过存储的管理网络访问并管理存储系统。
3. 所有使用存储上存储资源的节点，需要与存储的数据网络互通，如通过以太网方式或FC方式互通。
4. 在生产环境中，建议使用多路径，增加系统可靠性。

### 2.1 创建StorageClass，定义存储类型。
可参考如下示例创建StorageClass，供后续使用。
```
dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin$ cat demo/csi-storageClass.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: instorage-csi
provisioner: csi-instorage
parameters:
  volPoolName: Pool1
  volThin: "true"
  volThinResize: "2"
  volThinGrainSize: "256"
  volThinWarning: "20"
reclaimPolicy: Delete
```

**驱动支持以下配置参数，来定义不同的StorageClass类型**

名称 | 类型 | 说明 | 是否必须
-----|------|------|---------
volPoolName | string | 创建卷时所在的池的名称。| 必须
volAuxPoolName | string | 创建卷时辅助卷所在的池的名称。 | 双活/镜像卷时必须
volIOGrp | string | 创建卷时所在的IO组的ID号。| 双活卷时必须
volAuxIOGrp | string | 创建卷时辅助卷所在的IO组的IO号。| 双活卷时必须
volThin | string | 是否创建精简卷。值为true或false字符串。开启压缩时，自动开启精简卷。 | 默认false
volCompress | string | 是否创建压缩卷。值为true或false字符串。开启压缩时，自动开启精简卷。 | 默认false
volInTier | string | 是否开启分层。值为true或false字符串。 | 默认false
volLevel | string | 卷的等级类型。普通卷为basic，镜像卷为mirror，双活卷为aa。 | 默认basic，非必须。
volThinResize | string | 创建精简卷时，初始的卷大小占实际容量的百分比。 | 精简卷时必须
volThinGrainSize | string | 精简卷增长时的块大小，单位为KiB。可选32, 64, 128，256等值。	| 精简卷时有效，非必须。
volThinWarning | string | 精简卷告警阈值。警告时容量占实际容量的百分比。 | 精简卷时有效，非必须。
volAutoExpand | string | 是否开启自动扩张。值为true或false字符串。 | 默认false

### 2.2 存储CSI驱动配置文件说明
通过将驱动配置文件定义在configMap中，可以在部署驱动服务的时候，利用Kubernetes将configMap中的驱动配置提取到驱动所在容器中，供驱动使用，示例如下：
```
dev@lab:~/K8sPlugin_V2.1.0.Build20200226_amd64/csiplugin$ cat deploy/configMap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: inspur-instorage-01
data:
  instorage.yaml: |
    log:
      enabled: false
      logdir: log
      level: ""
      logrotatemaxsize: 0
    host:
      link: iscsi
      forceUseMultipath: false
      scsiScanRetryTimes: 3
      scsiScanWaitInterval: 1
      iscsiPathCheckRetryTimes: 3
      iscsiPathCheckWaitInterval: 1
      multipathSearchRetryTimes: 3
      multipathSearchWaitInterval: 1
      multipathResizeDelay: 1
    storage:
    - name: storage-01
      type: AS18000
      host: 192.168.10.100:22
      username: username
      password: password
```

**驱动配置文件支持以下配置项**

配置项名称 | 说明
---------|------
log.enabled | 是否打开插件日志。 true打开，false不打开。
log.logdir | 日志输出目录。
log.level | 日志输出级别。支持 debug/info/warning/error。
Log.logrotatemaxsize | 日志文件滚动大小阈值
host.link | 数据通道连接类型。iscsi表示使用iSCSI连接方式。fc表示使用FC连接方式。
host.forceUseMultipath | 是否强制使用多路径。true强制，false不强制。
host.scsiScanRetryTimes | SCSI设备扫描尝试次数。
host.scsiScanWaitInterval | SCSI设备扫描失败后等待间隔。单位为秒。
host.iscsiPathCheckRetryTimes | iSCSI路径扫描检查尝试次数。
host.iscsiPathCheckWaitInterval | iSCSI路径扫描失败后等待间隔，单位为秒。
host.multipathSearchRetryTimes | 多路径设备查找重试次数。
host.multipathSearchWaitInterval | 多路径设备查找失败后等待间隔，单位为秒。
host.multipathResizeDelay | 在线扩容时，多路径resize命令延迟时间。单位为秒。
host.attachExtendFileLockPath | 挂载卸载扩容操作并发控制文件锁路径。默认使用配置文件。
storage[].name | 存储名称。配置文件范围内唯一，区分多个存储。当前只支持一个存储。
storage[].type | 存储类型，必须配置。类型为AS18000。
storage[].host | 存储SSH访问路径。格式为IP:Port，如10.0.0.1:22。
storage[].username | 存储SSH访问时的用户名。
storage[].password | 存储SSH访问时的密码明文。
storage[].shadow | 存储SSH访问时的密码明文加密后的密文。*1

备注：
1. 针对storage[].password和storage[].shadow配置参数说明如下：
    > 当设置了password参数时，优先使用password设置的密码明文，当password未设置时，使用shadow设置的密码密文。密码密文可以通过执行./csiplugin ext-encrypt-password [password]获得。

### 2.3 多路径使用说明
浪潮存储CSI驱动提供了对多路径设备管理的支持，在Linux环境下，浪潮存储利用Linux系统自带的device-mapper-multipath服务进行多路径聚合，如果需要启用多路径，首先需要保证Kubernetes集群中各工作节点均按照要求部署安装multipathd服务，并对Kubernetes环境中的各工作节点设置多路径相关的配置，以正确使用多路径。CSI驱动所在的容器中也安装了device-mapper-multipath服务安装包，在实际使用中路径设备的聚合采用主机上的device-mapper-multipath服务来处理，驱动容器中只是使用了device-mapper-multipath中的客户端部分与主机上的多路径服务进行消息交互。

浪潮存储有推荐的多路径配置参数，且推荐的多路径配置参数也已经合入到多路径工具的社区版本中，针对使用旧版本多路径工具的场景，需要在配置文件中加入浪潮推荐的多路径配置。即在devices配置组中增加浪潮存储的device配置内容。推荐配置信息如下：
```
devices{
  device {
    vendor                  "INSPUR"
    product                 "MCS"
    path_grouping_policy    group_by_prio
    path_selector           "round-robin 0"
    path_checker            tur
    features                "1 queue_if_no_path"
    hardware_handler        "0"
    prio                    alua
    failback                immediate
    rr_weight               uniform
    rr_min_io               1000
  }
}
```
