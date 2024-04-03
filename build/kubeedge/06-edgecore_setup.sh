#!/bin/bash
set -e

TEMP_DIR=$(mktemp -d)
bashpath=$(cd `dirname $0`; pwd)


isExistEdgecore(){
    set +e
    edgecore -h > /dev/null 2>&1
    edgecore_check=$?
    # Exit if Docker is running
    if [[ $edgecore_check -eq 0 ]]; then
        exit 0
    fi
    set -e
    prepareEdgecore
}

prepareEdgecore(){
    # 强制clear edgecore work dir
    rm -rf /etc/kubeedge
    # 生成edgecore工作目录
    mkdir -p /etc/kubeedge
    # Extract the edgecore file
    tar -xf "${bashpath}/dependence/edgecore.tar.gz" -C $TEMP_DIR

    # Move the edgecore file
    mv $TEMP_DIR/edgecore /usr/local/bin/edgecore
    rm -f $TEMP_DIR/edgecore

    # Set executable permissions
    chmod 0755 /usr/local/bin/edgecore

    # Move the edgecore.service file
    mv $TEMP_DIR/edgecore.service /etc/systemd/system/edgecore.service

    echo "EdgeCore Install.......OK"
}

isExistEdgecore
