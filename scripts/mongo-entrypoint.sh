#!/bin/sh
set -eu
# Persist the replica-set membership key separately from application credentials.
if [ ! -s /data/configdb/mlm-keyfile ]; then
  umask 077
  head -c 756 /dev/urandom | base64 > /data/configdb/mlm-keyfile
fi
chown mongodb:mongodb /data/configdb/mlm-keyfile
chmod 400 /data/configdb/mlm-keyfile
exec /usr/local/bin/docker-entrypoint.sh mongod --replSet mlm --keyFile /data/configdb/mlm-keyfile --bind_ip_all
