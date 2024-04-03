#!/bin/bash
set -e

# Stop edgecore
systemctl stop edgecore

# Delete file example
rm -rf /etc/kubeedge