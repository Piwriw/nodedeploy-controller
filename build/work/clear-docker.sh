#!/bin/bash
set -e
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
    docker-buildx-plugin \
    docker-ce-rootless-extra

