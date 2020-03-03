# Notice
1. Host should have /etc/iscsi configuration set, include initiatorname.iscsi and iscsid.conf.
2. The iscsid service on host should not start, as plugin will use the iscsid service in the image.

# Steps
1. Use the host iscsi config as the container's iscsi config, so just map host volume /etc/iscsi to container volume /etc/iscsi.
2. Create the directory /var/lib/kubelet/plugins/csi-instorage as the csi socket directory.
