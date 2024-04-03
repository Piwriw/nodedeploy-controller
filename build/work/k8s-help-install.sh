#!/bin/bash
set -e
yum install --downloadonly  --downloaddir=/root/k8s kubelet-1.21.14 kubeadm-1.21.14 kubectl-1.21.14 ntpdate
yum localinstall -y ./package/k8s/centeros_x86/*.rpm --disablerepo=*


output_dir="images"

# 创建输出目录
mkdir -p "$output_dir"

# 获取所有的K8s集群在使用镜像列表，并遍历每个镜像
# 不包括 pause镜像，pause在启动Pod的时候会用到
kubectl get pods --all-namespaces -o jsonpath="{..image}" | tr -s '[:space:]' '\n' | sort -u  | while read -r image; do
      tm=$(echo "$image" | rev | cut -d '/' -f 1 )
      tag=$(echo "$tm" | rev | cut -d ':' -f 1)
      name=$(echo "$tm" | rev|  cut -d ':' -f 2)
     output_file="$output_dir/${name}_${tag}.tar"
   docker save -o "$output_file" "$image"
   echo "Saved $output_file"
done
