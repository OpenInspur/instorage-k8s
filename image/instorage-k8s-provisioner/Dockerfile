#Dockerfile for Inspur Instorage external provisioner

FROM centos:7.6.1810

LABEL maintainer="instorage.csi@inspur.com"

RUN mkdir -p /opt/instorage/config

COPY provisioner /opt/instorage/provisioner

CMD ["/opt/instorage/provisioner", "--config=/opt/instorage/config/instorage.yaml"]
