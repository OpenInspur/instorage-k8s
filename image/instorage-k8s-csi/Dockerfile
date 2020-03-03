#Dockerfile for Inspur Instorage CSI Driver

FROM centos:7.6.1810

LABEL maintainer="instorage.csi@inspur.com"

COPY csiplugin csiplugin

RUN yum -y install sysfsutils device-mapper device-mapper-multipath iscsi-initiator-utils e2fsprogs xfsprogs && yum clean all

RUN /sbin/mpathconf --enable

ENTRYPOINT ["/csiplugin"]
