#!/bin/bash

# Disable SELinux
sed -i 's/^SELINUX=enforcing/SELINUX=disabled/' /etc/selinux/config
setenforce 0

# Disable firewalld
systemctl stop firewalld
systemctl disable firewalld

# Disable UFW
systemctl stop ufw
systemctl disable ufw

echo "Check  firewalld UFW SELINUX Done"