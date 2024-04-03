#!/bin/bash

HARBOR_USER=$1
HARBOR_PASSWD=$2
HARBOR_ADDR=$3
bashpath=$(cd `dirname $0`; pwd)

export HARBOR_ADDR=${HARBOR_ADDR}

# 检查docker config.json是否存在
isExistedDocker() {
  docker_config_path="/etc/docker/daemon.json"

  # 检查文件是否存在
  if [ -e "$docker_config_path" ]; then
     ${bashpath}/utils/docker_daemon_utils
  else
    if [ ! -d "/etc/docker" ]; then
        mkdir -p /etc/docker
    fi
    echo '{}' > /etc/docker/daemon.json
     ${bashpath}/utils/docker_daemon_utils
  fi
}


startDocker(){
# Start docker service
systemctl daemon-reload
systemctl restart docker

# Login to Harbor
docker login -u "$HARBOR_USER" -p "$HARBOR_PASSWD" "$HARBOR_ADDR"
}


isExistedDocker
startDocker
