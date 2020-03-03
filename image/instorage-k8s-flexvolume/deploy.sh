#!/bin/sh

set -o errexit
set -o pipefail

VENDOR=inspur
DRIVER=instorage

driver_dir="${VENDOR}~${DRIVER}"

# create the driver base directory
if [ ! -d "/opt/flexmnt/$driver_dir" ]; then
  mkdir "/opt/flexmnt/$driver_dir"
fi

# create the config directory
if [ ! -d "/opt/flexmnt/$driver_dir/config" ]; then
  mkdir "/opt/flexmnt/$driver_dir/config"
fi

# create the log directory
if [ ! -d "/opt/flexmnt/$driver_dir/log" ]; then
  mkdir "/opt/flexmnt/$driver_dir/log"
fi

# copy the config
cp /opt/instorage/config/instorage.yaml /opt/flexmnt/$driver_dir/config/instorage.yaml

# copy the driver
tmp_driver=.tmp_$DRIVER
cp /opt/instorage/instorage "/opt/flexmnt/$driver_dir/$tmp_driver"
mv -f "/opt/flexmnt/$driver_dir/$tmp_driver" "/opt/flexmnt/$driver_dir/$DRIVER"

while : ; do
  sleep 3600
done