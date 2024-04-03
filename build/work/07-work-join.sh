#!/bin/bash
set -e
MASTERIP=$1
TOKEN=$2
DISCOVERY_TOKEN_CA_CERT_HASH=$3

kubeadm reset -f

kubeadm join ${MASTERIP}  --token  ${TOKEN}  --discovery-token-ca-cert-hash ${DISCOVERY_TOKEN_CA_CERT_HASH}