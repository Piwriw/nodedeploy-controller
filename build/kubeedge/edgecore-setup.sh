#!/bin/bash
set -e

printEnv() {
    echo  "已设置环境变量参数："
    echo  "HARBOR_ADDR="${HARBOR_ADDR}
    echo  "HARBOR_USER="${HARBOR_USER}
    echo  "HARBOR_PASSWD="${HARBOR_PASSWD}
    echo "CLOUD_HOST="${CLOUD_HOST}
    echo "TOKEN="${TOKEN}
    echo "HOSTNAME="${HOSTNAME}
    echo ""
}

TEMP_DIR=$(mktemp -d)
export TEMP_DIR="$TEMP_DIR"
edgecore_model=$1
EDGESTACK_ARCH=$(cat ./dependence/arch.txt)

bashpath=$(cd `dirname $0`; pwd)
checkARCH(){

   formatted_arch=$(uname -m)
    if [[ $formatted_arch == "x86_64" || $formatted_arch == "amd64" ]]; then
        formatted_arch="amd64"
    elif [[ $formatted_arch == "aarch64" ]]; then
        formatted_arch="arm64"
    fi
    if [[ $EDGESTACK_ARCH != $formatted_arch ]];then
      echo "安装包支持架构和系统不一致"
      exit 1
    elif [[ $EDGESTACK_ARCH == $formatted_arch ]]; then
        doExec
    else
      echo "出现了一些意料之外的架构问题"
    fi

}

doExec(){
  if [ "$edgecore_model" = "join" ]; then
    bash ${bashpath}/01-docker_install.sh $EDGESTACK_ARCH
    bash ${bashpath}/02-docker_config.sh
    bash ${bashpath}/03-edgecore_install.sh
    bash ${bashpath}/04-edgecore_config.sh
    bash ${bashpath}/05-stopwalld.sh
    bash ${bashpath}/06-edgecore_join.sh
  elif [ "$edgecore_model" = "disjoin"  ]; then
    bash ${bashpath}/07-edgecore_disjoin.sh
  else
    echo "NO Command ,$edgecore_model"
  fi
}
if [ ${#HARBOR_USER} -eq 0 ]  || [ ${#HARBOR_ADDR} -eq 0 ] ||
 [ ${#HARBOR_PASSWD} -eq 0 ] || [ ${#CLOUD_HOST} -eq 0 ] ||  [ ${#TOKEN} -eq 0 ] ||[ ${#HOSTNAME} -eq 0 ]; then
    echo -e "\033[31;1m缺少环境变量参数 \033[0m"
    printEnv
    exit 1
fi

checkARCH
