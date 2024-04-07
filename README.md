# nodedeploy-controller
## Introduce
这是一个节点离线自动上下限的控制器，目前支持K8s、Kubeedge的节点上下线
## Deploy-Heml
```shell
helm upgrade --install node-deploy node-deploy   -n controller --create-namespace
```
## CHANGELOG
- 2024.4.3  镜像：piwriw/nodedeploy-controller:0.1 支持K8s 1.21.14 node节点上下线
- 2024.4.3  镜像：piwriw/nodedeploy-controller:kubeeedge-0.1 仅支持kubeedge节点上下线
- 2024.4.3  镜像：piwriw/nodedeploy-controller:0.2  支持K8s 1.21.14 node节点上下线和支持kubeedge 1.12.3节点上下线
- 2024.4.3  镜像：piwriw/nodedeploy-controller:0.3  优化：使用集群内认证token
- 2024.4.7  镜像：piwriw/nodedeploy-controller:0.4  增强：支持Retry，重复次数
## CRD设计
```yaml
apiVersion: node.nodedeploy/v1
kind: NodeDeploy
metadata:
  name: nodedeploy-sample
  namespace: controller
spec:
  # 节点名称
  nodeName: "k8s-worker-test1"
  # 节点IP
  nodeIP: 192.168.28.142
  # 节点类型 work|kubeedge
  nodeType: kubeedge
  nodePort: "22"
  nodeUser: "root"
  nodePwd: "Piwriw503420"
  harborEndpoint: "http://192.168.28.130:30003"
  harborUser: "admin"
  harborPwd: "Harbor12345"
  # 节点上线后添加的label
  labels:
    test-key: test-value
  # 节点上线后添加的annotations
  annotations:
    test-an: test-value
  # 节点上线后添加的污点
  taints:
    - key: test-key
      value: test-value
      effect: NoSchedule
  # nodeVersion 版本
  nodeVersion: "1.12.3"
  # 节点下线时是否驱逐pod
  isEvicted: true
  # 节点目标状态，active/inactive 分别对应上线和下线
  nodeStatus: active
  # 最大重试次数，默认值3，可以不传此参数
  maxRetry: 3

status:
# 当前节点状态
#  nodeStatus: inactive
```




