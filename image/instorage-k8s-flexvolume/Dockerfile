#Dockerfile for Inspur Instorage flexvolume driver

FROM centos:7.6.1810

LABEL maintainer="instorage.csi@inspur.com"

RUN mkdir -p /opt/instorage/config

RUN mkdir -p /opt/flexmnt

COPY instorage /opt/instorage/instorage

COPY deploy.sh /deploy.sh

CMD ["/deploy.sh"]
