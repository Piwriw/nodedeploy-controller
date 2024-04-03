#!/bin/bash
set -e

bashpath=$(cd `dirname $0`; pwd)
TEMP_DIR=$(mktemp -d)

ARCH=$1
# 检查docker是否存在
isExistedDocker(){
  set +e
  docker ps > /dev/null 2>&1
  docker_check=$?
  # Exit if Docker is running
  if [[ $docker_check -eq 0 ]]; then
      exit 0
  fi
  echo "Docker不存在，进入安装Docker……"
  set -e
  prepareDocker
}


# 解压Docker安装包
prepareDocker(){

  # Extract the docker file
  mkdir -p  $TEMP_DIR/docker
  tar -xf ${bashpath}/dependence/docker.tar.gz -C $TEMP_DIR/docker

  # Change owner and group of docker_setup
  chown root:root $TEMP_DIR/docker/setup.sh

  chmod 0755 $TEMP_DIR/docker/setup.sh

# Check if setup.sh exists and is executable
if [[ -x $TEMP_DIR/docker/setup.sh ]]; then
    setup_script=1
else
    setup_script=0
fi

# Run setup.sh to install Docker
if [[ $setup_script -eq 1 ]]; then
    docker_filename="docker_20_${ARCH}.tar.gz"

    bash $TEMP_DIR/docker/setup.sh $docker_filename
    setup_result=$?
    if [[ $setup_result -eq 0 ]]; then
        echo "Docker installation completed"
    else
        echo "Docker installation failed"
    fi
fi

}

isExistedDocker