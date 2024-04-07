package nodemanager

import (
	"context"
	"encoding/json"
	"fmt"
	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	"github.com/piwriw/nodedeploy-controller/pkg/types"
	utils_ssh "github.com/piwriw/nodedeploy-controller/utils/ssh"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

type NodeManager struct {
	recorder   record.EventRecorder
	nodeDeploy *nodev1.NodeDeploy
	nodeInfo   *types.NodeInfo
}

func New(recorder record.EventRecorder, nodeDeploy *nodev1.NodeDeploy, nodeInfo *types.NodeInfo) *NodeManager {
	return &NodeManager{
		recorder:   recorder,
		nodeDeploy: nodeDeploy,
		nodeInfo:   nodeInfo,
	}
}

// Deprecate 节点下线
func (n *NodeManager) Deprecate(ctx context.Context) error {
	var nodedisJoin NodeDisJoin
	switch n.nodeInfo.NodeType {
	case nodev1.NodeWork.String():
		nodedisJoin = new(WorkNode)
	case nodev1.NodeKubeedge.String():
		nodedisJoin = new(KubEdgeNode)
	}
	progress := NewProgress(1)

	sshclient, err := utils_ssh.NewClient(n.nodeInfo.NodeIP, n.nodeInfo.NodePort, n.nodeInfo.NodeUser, n.nodeInfo.NodePwd)
	if err != nil {
		klog.Errorf("Failed to create new client: %v", err)
	}

	n.logMessage("Deprecate Node", "shutdown workerNode", progress.Add())
	err = nodedisJoin.DisJoin(ctx, sshclient)
	if err != nil {
		n.logMessage("Deprecate Node", "fail to deprecate", progress.String())
		return err
	}

	n.recorder.Event(n.nodeDeploy, corev1.EventTypeNormal, "Deprecate", "deprecate success")
	n.logMessage("Deprecate Node", "deprecate success", progress.String())

	return nil
}

// LaunchByCmd 节点上线
func (n *NodeManager) LaunchByCmd(ctx context.Context) error {
	var nodeJoin NodeJoin
	switch n.nodeInfo.NodeType {
	case nodev1.NodeWork.String():
		nodeJoin = new(WorkNode)
	case nodev1.NodeKubeedge.String():
		nodeJoin = new(KubEdgeNode)
	}
	sshclient, err := utils_ssh.NewClient(n.nodeInfo.NodeIP, n.nodeInfo.NodePort, n.nodeInfo.NodeUser, n.nodeInfo.NodePwd)
	if err != nil {
		klog.Errorf("Create ssh client failed,err:%s", err)
		return errors.Errorf("Create ssh client failed")
	}
	progress := NewProgress(10)
	//1.检查系统架构
	n.logMessage("Check architecture", "Check architecture", progress.Add())
	systemArch, err := nodeJoin.CheckArch(ctx, sshclient)
	if err != nil {
		n.logMessage("Check architecture", "Check architecture", progress.String())
		return err
	}

	//2. 检测 kubernetes 版本
	n.logMessage("Check Kubernetes Version", "Check Kubernetes Version", progress.Add())
	versionK8s, err := nodeJoin.CheckNodeVersion(n.nodeDeploy.Spec.NodeVersion)
	if err != nil {
		n.logMessage("Kubernetes Install", "invalid kubernetes version", progress.String())
		return err
	}

	//3. 上传离线安装包
	n.logMessage("Prepare Package", "Prepare the offline installation package", progress.Add())
	if err := nodeJoin.PrepareSetupPkg(ctx, sshclient, systemArch, versionK8s); err != nil {
		n.logMessage("Prepare Package", "Prepare the offline installation package failed", progress.String())
		return err
	}

	//4. 关闭 防火墙设置
	n.logMessage("Stop Firewalld", "Turn off the system firewall settings", progress.Add())
	if err := nodeJoin.StopFirewalld(ctx, sshclient); err != nil {
		n.logMessage("Stop Firewalld", fmt.Sprintf("Failed to turn off the system firewall setting, %s", err), progress.String())
		return err
	}

	//5. 安装 docker
	n.logMessage("Docker Install", "Installing docker", progress.Add())
	if err = nodeJoin.InstallDocker(ctx, sshclient, systemArch); err != nil {
		n.logMessage("Docker Install", fmt.Sprintf("docker doesn't exist, %s", err), progress.String())
		return err
	}

	//6. 配置 docker config
	n.logMessage("Docker Config", "Configure the docker configuration file\n", progress.Add())
	if err = nodeJoin.SetDockerConf(ctx, sshclient, n.nodeInfo.HarborUser, n.nodeInfo.HarborPwd, n.nodeInfo.HarborEndpoint); err != nil {
		n.logMessage("Docker Config", fmt.Sprintf("docker config.json doesn't exist, %s", err), progress.String())
		return err
	}

	//7. 设置主机名称
	n.logMessage("SetUp Hostname", "Setup hostname", progress.Add())
	if err = nodeJoin.SetHostName(ctx, sshclient, n.nodeInfo.NodeName); err != nil {
		n.logMessage("SetUp Hostname ", fmt.Sprintf(" The node has already been set up, %s", err), progress.String())
		return err
	}

	//8. 导入 docker 镜像
	n.logMessage("Load  Images", "Loading Images", progress.Add())
	if err = nodeJoin.LoadImage(ctx, sshclient); err != nil {
		n.logMessage("Load  Images", fmt.Sprintf(" the image file does not exist, %s", err), progress.String())
		return err
	}

	//9. 给节点安装 k8s 组件
	n.logMessage(" Node Setup", "Setup workerNode", progress.Add())
	if err = nodeJoin.SetNodeComponents(ctx, sshclient); err != nil {
		n.logMessage("Node Setup ", fmt.Sprintf("workerNode doesn't exist, %s", err), progress.String())
		return err
	}
	//10. 节点上线操作操作
	n.logMessage("Node Join", "start workerNode", progress.Add())
	if err = nodeJoin.Join(ctx, sshclient); err != nil {
		n.logMessage("Node start", "workerNode Join failed", progress.String())
		return err
	}
	n.logMessage("Launch Success", "Nodes are added to the cluster", progress.String())

	return nil

}
func (n *NodeManager) logMessage(phase, message, progress string) {
	n.recorder.Event(n.nodeDeploy, corev1.EventTypeNormal, types.EventStatusMessage, StatusMessage{
		Phase:    phase,
		Message:  message,
		Progress: progress,
	}.String())
}

type StatusMessage struct {
	Phase    string `json:"phase,omitempty"`
	Message  string `json:"message,omitempty"`
	Progress string `json:"progress,omitempty"`
}

func (m StatusMessage) String() string {
	data, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(data)
}
