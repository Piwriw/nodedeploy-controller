#!/bin/bash
set -e
# arm64 kubeedge-1.12.3 部署包
cp ./dependence/docker_arm64.tar.gz ./dependence/docker.tar.gz
cp ./dependence/edgecore_arm64_1-12-x.tar.gz ./dependence/edgecore.tar.gz
cp ./arm64-version.txt ./dependence/arch.txt
cp ./build/docker_daemon_utils.arm64 ./dependence/docker_daemon_utils
chmod 777   ./dependence/docker_daemon_utils
chmod +x  ./dependence/docker_daemon_utils
tar -czvf ../../deploypkg/kubeedge/arm64-1.12.tar.gz \
         01-stopwalld.sh  \
          02-docker_install.sh   \
          03-docker_config.sh  \
          04-homename_setup.sh  \
          05-load-image.sh \
          06-edgecore_setup.sh    \
          07-edgecore_join.sh \
          08-edgecore_disjoin.sh \
         ./dependence/docker.tar.gz \
         ./dependence/edgecore.tar.gz \
         ./dependence/arch.txt \
         ./dependence/docker_daemon_utils \
         images

rm -rf ./dependence/docker.tar.gz
rm -rf ./dependence/edgecore.tar.gz
rm -rf ./dependence/arch.txt
rm -rf ./dependence/docker_daemon_utils

# x86 kubeedge-1.12.3 部署包
cp ./dependence/docker_amd64.tar.gz ./dependence/docker.tar.gz
cp ./dependence/edgecore_x86_1-12-x.tar.gz ./dependence/edgecore.tar.gz
cp ./x86-version.txt ./dependence/arch.txt
cp ./build/docker_daemon_utils.amd64 ./dependence/docker_daemon_utils
chmod +x  ./dependence/docker_daemon_utils
chmod 777   ./dependence/docker_daemon_utils
tar -czvf ../../deploypkg/kubeedge/amd64-1.12.tar.gz \
         01-stopwalld.sh  \
          02-docker_install.sh   \
          03-docker_config.sh  \
          04-homename_setup.sh  \
          05-load-image.sh \
          06-edgecore_setup.sh    \
          07-edgecore_join.sh \
          08-edgecore_disjoin.sh \
         ./dependence/docker.tar.gz \
         ./dependence/edgecore.tar.gz \
         ./dependence/arch.txt  \
         ./dependence/docker_daemon_utils \
         images


rm -rf ./dependence/docker.tar.gz
rm -rf ./dependence/edgecore.tar.gz
rm -rf ./dependence/arch.txt
rm -rf   ./dependence/docker_daemon_utils
