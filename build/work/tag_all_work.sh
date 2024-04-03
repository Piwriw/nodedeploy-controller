#!/bin/bash
set  -e

cp ./utils/docker_daemon_utils.amd64 ./utils/docker_daemon_utils
chmod +x ./utils/docker_daemon_utils
tar -czvf ../../deploypkg/work/amd64-1.21.tar.gz 01-stopwalld.sh          04-homename_setup.sh      07-work-join.sh                 images/work     \
                                   02-docker_install.sh     05-load-image.sh          08-work-disjoin.sh             package/docker  package/k8s/centeros-amd64.tar.gz    ./utils/docker_daemon_utils \
                                   03-docker_config.sh      06-k8s-setup.sh
rm -rf ./utils/docker_daemon_utils