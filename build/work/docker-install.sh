#!/bin/bash
set -e


# 在 master 节点和 worker 节点都要执行

# 安装 docker
# 参考文档如下
# https://docs.docker.com/install/linux/docker-ce/centos/
# https://docs.docker.com/install/linux/linux-postinstall/

# 卸载旧版本
yum remove  -y docker \
    docker-client \
    docker-client-latest \
    docker-common \
    docker-latest \
    docker-latest-logrotate \
    docker-logrotate \
    docker-engine \
    docker-ce \
    docker-ce-cli \
    docker-buildx-plugin


# 安装并启动 docker
yum install -y docker-ce-20.10.0 docker-ce-cli-20.10.0 containerd.io
systemctl enable docker
systemctl start docker
