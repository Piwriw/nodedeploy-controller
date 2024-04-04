package nodemanager

import (
	"context"
	"fmt"
	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	utils_ssh "github.com/piwriw/nodedeploy-controller/utils/ssh"
	"k8s.io/klog/v2"
	"strings"
)

const (
	PkgPathPrefix  = "/pkg/nodedeploy"
	FilePathPrefix = "/path/to/nodedeploy"
	check_arch_cmd = "uname -m"
	mktemp_cmd     = "mktemp -d"
)

var ()

type NodeJoin interface {
	CheckArch(ctx context.Context, sshclient *utils_ssh.Client) (string, error)
	CheckNodeVersion(nodeVersion string) (string, error)
	PrepareSetupPkg(ctx context.Context, sshclient *utils_ssh.Client, systemArch, version string) error
	StopFirewalld(ctx context.Context, sshclient *utils_ssh.Client) error
	InstallDocker(ctx context.Context, sshclient *utils_ssh.Client, systemArch string) error
	SetDockerConf(ctx context.Context, sshclient *utils_ssh.Client, harborUser, harborPwd, harborEndpoint string) error
	SetHostName(ctx context.Context, sshclient *utils_ssh.Client, hostname string) error
	LoadImage(ctx context.Context, sshclient *utils_ssh.Client) error
	SetNodeComponents(ctx context.Context, sshclient *utils_ssh.Client) error
	Join(ctx context.Context, sshclient *utils_ssh.Client) error
}

type WorkNode struct {
}

func (wn WorkNode) Join(ctx context.Context, sshclient *utils_ssh.Client) error {
	masterHostAndPort, token, hash, err := GetWorkJoinInfo()
	if err != nil {
		klog.Fatalf("Failed to get k8s token and hash: %v", err)
	}
	klog.Infof("成功获取到k8s的接入信息，kubeadm join %s --token %s  --discovery-token-ca-cert-hash %s\n", masterHostAndPort, token, hash)

	workerNode_join_cmd := fmt.Sprintf("bash %s/%s/07-work-join.sh %v %v %v", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()), masterHostAndPort, token, hash)
	klog.Infof("上线时具体操作为: %v\n", workerNode_join_cmd)
	_, err = sshclient.Exec(ctx, workerNode_join_cmd)
	if err != nil {
		return fmt.Errorf("workerNode start failed,err:%s", err)
	}
	return nil
}

func (wn WorkNode) SetNodeComponents(ctx context.Context, sshclient *utils_ssh.Client) error {
	node_setup_cmd := fmt.Sprintf("bash %s/%s/06-k8s-setup.sh", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()))
	_, err := sshclient.Exec(ctx, node_setup_cmd)
	if err != nil {
		return fmt.Errorf("workerNode doesn't exist, %w", err)
	}
	return nil
}
func (wn WorkNode) LoadImage(ctx context.Context, sshclient *utils_ssh.Client) error {
	image_load_cmd := fmt.Sprintf("bash %s/%s/05-load-image.sh %s ", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()), "images/work")
	_, err := sshclient.Exec(ctx, image_load_cmd)
	if err != nil {
		return fmt.Errorf("the image file does not exist, %w", err)
	}
	return nil
}
func (wn WorkNode) SetHostName(ctx context.Context, sshclient *utils_ssh.Client, hostName string) error {
	set_hostname_cmd := fmt.Sprintf("bash %s/%s/04-homename_setup.sh %v", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()), hostName)
	_, err := sshclient.Exec(ctx, set_hostname_cmd)
	if err != nil {
		return fmt.Errorf("The node has already been set up, %w", err)
	}
	return nil
}
func (wn WorkNode) SetDockerConf(ctx context.Context, sshclient *utils_ssh.Client, harborUser, harborPwd, harborEndpoint string) error {
	klog.Infof("Habror Info: User:%s,PassWord:%s,Addr:%s", harborUser, harborPwd, harborEndpoint)
	docker_config_cmd := fmt.Sprintf("bash %s/%s/03-docker_config.sh %v %v %v", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()), harborUser, harborPwd, harborEndpoint)
	_, err := sshclient.Exec(ctx, docker_config_cmd)
	if err != nil {
		return fmt.Errorf("docker config.json doesn't exist, %w", err)
	}
	return nil
}
func (wn WorkNode) InstallDocker(ctx context.Context, sshclient *utils_ssh.Client, systemArch string) error {
	docker_install_cmd := fmt.Sprintf("bash %s/%s/02-docker_install.sh %v ", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()), systemArch)
	_, err := sshclient.Exec(ctx, docker_install_cmd)
	if err != nil {
		return fmt.Errorf("docker doesn't exist, %w", err)
	}
	return nil
}
func (wn WorkNode) StopFirewalld(ctx context.Context, sshclient *utils_ssh.Client) error {
	stop_firewall_cmd := fmt.Sprintf("bash %s/%s/01-stopwalld.sh", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()))
	_, err := sshclient.Exec(ctx, stop_firewall_cmd)
	if err != nil {
		return fmt.Errorf("Failed to turn off the system firewall setting, %w", err)
	}
	return nil
}

func (wn WorkNode) PrepareSetupPkg(ctx context.Context, sshclient *utils_ssh.Client, systemArch, version string) error {
	//离线安装包源地址  /pkg/nodedeploy/work/arch-v1.21.tar.gz
	src_setup_pkg := fmt.Sprintf("%s/%s/%v-%v.tar.gz", PkgPathPrefix, strings.ToLower(nodev1.NodeWork.String()), systemArch, version)
	klog.Infof("Offic Pkg Path:%s", src_setup_pkg)
	//离线安装包目的地址
	dest_setup_pkg := fmt.Sprintf("/tmp/nodedeploy-%v-%v.tar.gz", systemArch, version)
	//从本地原地址下发到目标文件地址
	err := sshclient.Upload(ctx, src_setup_pkg, dest_setup_pkg)
	if err != nil {
		return fmt.Errorf("Upload failed, err: %s", err)
	}
	// /path/to/nodedeploy/work
	mkdir_cmd := fmt.Sprintf("mkdir -p %s/%s", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()))
	_, err = sshclient.Exec(ctx, mkdir_cmd)
	unzip_cmd := fmt.Sprintf("tar -xzvf %v -C %s/%s", dest_setup_pkg, FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()))
	//将离线安装包解压到指定目录下 /path/to/nodedeploy/work
	_, err = sshclient.Exec(ctx, unzip_cmd)
	if err != nil {
		return fmt.Errorf("Failed to extract the zip file, err: %s", err)
	}
	return nil
}

func (wn WorkNode) CheckNodeVersion(version string) (string, error) {
	//获取kubernetes的版本
	versionK8s, err := getKubernetesVersionStr(version)
	if err != nil {
		return "", fmt.Errorf("invalid kubernetes version, err: %s", err)
	}
	return versionK8s, nil

}

func (wn WorkNode) CheckArch(ctx context.Context, sshclient *utils_ssh.Client) (string, error) {
	//获取临时文件夹名称
	temp_dir_str, err := sshclient.Exec(ctx, mktemp_cmd)
	lines := strings.Split(temp_dir_str, "\n")
	temp_dir := lines[0]
	if err != nil {
		klog.Fatalf("Failed to create new temp dir: %v", err)
	}
	klog.Infof("The temporary directory [%v] is created\n", temp_dir)

	//检测机器的类型，下载对应的离线安装包
	systemlArch, err := sshclient.Exec(ctx, check_arch_cmd)
	if err != nil {
		return "", fmt.Errorf("系统架构异常，请检查当前系统架构, %w", err)
	}
	systemArch, err := ParseArch(systemlArch)
	if err != nil {
		klog.Infof("The architecture of the system  is [%v]\n", systemlArch)
		return "", err
	}
	return systemArch, nil
}

type KubEdgeNode struct {
}

func (k KubEdgeNode) CheckArch(ctx context.Context, sshclient *utils_ssh.Client) (string, error) {
	//获取临时文件夹名称
	temp_dir_str, err := sshclient.Exec(ctx, mktemp_cmd)
	lines := strings.Split(temp_dir_str, "\n")
	temp_dir := lines[0]
	if err != nil {
		klog.Fatalf("Failed to create new temp dir: %v", err)
	}
	klog.Infof("The temporary directory [%v] is created\n", temp_dir)

	//检测机器的类型，下载对应的离线安装包
	systemlArch, err := sshclient.Exec(ctx, check_arch_cmd)
	if err != nil {
		return "", fmt.Errorf("系统架构异常，请检查当前系统架构, %w", err)
	}
	systemArch, err := ParseArch(systemlArch)
	if err != nil {
		klog.Infof("The architecture of the system  is [%v]\n", systemlArch)
		return "", err
	}
	return systemArch, nil
}

func (k KubEdgeNode) CheckNodeVersion(nodeVersion string) (string, error) {
	//获取Kubeedge的版本
	versionK8s, err := getKubernetesVersionStr(nodeVersion)
	if err != nil {
		return "", fmt.Errorf("invalid Kubeedge version, err: %s", err)
	}
	return versionK8s, nil
}

func (k KubEdgeNode) PrepareSetupPkg(ctx context.Context, sshclient *utils_ssh.Client, systemArch, version string) error {
	//离线安装包源地址  /pkg/nodedeploy/kubeedge/arch-v1.12.3.tar.gz
	src_setup_pkg := fmt.Sprintf("%s/%s/%v-%v.tar.gz", PkgPathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()), systemArch, version)
	klog.Infof("Offic Pkg Path:%s", src_setup_pkg)
	//离线安装包目的地址
	dest_setup_pkg := fmt.Sprintf("/tmp/nodedeploy-%v-%v.tar.gz", systemArch, version)
	//从本地原地址下发到目标文件地址
	err := sshclient.Upload(ctx, src_setup_pkg, dest_setup_pkg)
	if err != nil {
		return fmt.Errorf("Upload failed, err: %s", err)
	}
	// /path/to/nodedeploy/work
	mkdir_cmd := fmt.Sprintf("mkdir -p %s/%s", FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()))
	_, err = sshclient.Exec(ctx, mkdir_cmd)
	unzip_cmd := fmt.Sprintf("tar -xzvf %v -C %s/%s", dest_setup_pkg, FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()))
	//将离线安装包解压到指定目录下 /path/to/nodedeploy/kubeedge
	_, err = sshclient.Exec(ctx, unzip_cmd)
	if err != nil {
		return fmt.Errorf("Failed to extract the zip file, err: %s", err)
	}
	return nil
}

func (k KubEdgeNode) StopFirewalld(ctx context.Context, sshclient *utils_ssh.Client) error {
	stop_firewall_cmd := fmt.Sprintf("bash %s/%s/01-stopwalld.sh", FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()))
	_, err := sshclient.Exec(ctx, stop_firewall_cmd)
	if err != nil {
		return fmt.Errorf("Failed to turn off the system firewall setting, %w", err)
	}
	return nil
}

func (k KubEdgeNode) InstallDocker(ctx context.Context, sshclient *utils_ssh.Client, systemArch string) error {
	docker_install_cmd := fmt.Sprintf("bash %s/%s/02-docker_install.sh %s", FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()), systemArch)
	_, err := sshclient.Exec(ctx, docker_install_cmd)
	if err != nil {
		return fmt.Errorf("docker doesn't exist, %w", err)
	}
	return nil
}

func (k KubEdgeNode) SetDockerConf(ctx context.Context, sshclient *utils_ssh.Client, harborUser, harborPwd, harborEndpoint string) error {
	klog.Infof("Habror Info: User:%s,PassWord:%s,Addr:%s", harborUser, harborPwd, harborEndpoint)
	docker_config_cmd := fmt.Sprintf("bash %s/%s/03-docker_config.sh %s %s %s", FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()), harborUser, harborPwd, harborEndpoint)
	_, err := sshclient.Exec(ctx, docker_config_cmd)
	if err != nil {
		return fmt.Errorf("docker config.json doesn't exist, %w", err)
	}
	return nil
}

func (k KubEdgeNode) SetHostName(ctx context.Context, sshclient *utils_ssh.Client, hostName string) error {
	set_hostname_cmd := fmt.Sprintf("bash %s/%s/04-homename_setup.sh %v", FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()), hostName)
	_, err := sshclient.Exec(ctx, set_hostname_cmd)
	if err != nil {
		return fmt.Errorf("The node has already been set up, %w", err)
	}
	return nil
}

func (k KubEdgeNode) LoadImage(ctx context.Context, sshclient *utils_ssh.Client) error {
	image_load_cmd := fmt.Sprintf("bash %s/%s/05-load-image.sh %s ", FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()), "images")
	_, err := sshclient.Exec(ctx, image_load_cmd)
	if err != nil {
		return fmt.Errorf("the image file does not exist, %w", err)
	}
	return nil
}

func (k KubEdgeNode) SetNodeComponents(ctx context.Context, sshclient *utils_ssh.Client) error {
	// 下载EdgeCore
	egdecore_install_cmd := fmt.Sprintf("bash %s/%s/06-edgecore_setup.sh", FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()))
	_, err := sshclient.Exec(ctx, egdecore_install_cmd)
	if err != nil {
		return fmt.Errorf("edgecore install failed, %s", err)
	}

	return nil
}

func (k KubEdgeNode) Join(ctx context.Context, sshclient *utils_ssh.Client) error {
	cloudHosts, port, token, err := GetKubeEdgeJoinInfo()
	if err != nil {
		klog.Fatalf("Failed to get KubeEdge CloudHost  and token: %v", err)
	}
	klog.Infof("成功获取到KubeEdge CloudCore的接入信息，keadm join %v%s --token %s  ", cloudHosts, port, token)

	edgeNode_join_cmd := fmt.Sprintf("bash %s/%s/07-edgecore_join.sh %v %v %v", FilePathPrefix, strings.ToLower(nodev1.NodeKubeedge.String()), cloudHosts[0], port, token)
	klog.Infof("上线时具体操作为: %v\n", edgeNode_join_cmd)
	_, err = sshclient.Exec(ctx, edgeNode_join_cmd)
	if err != nil {
		return fmt.Errorf("EdgeNode start failed,err:%s", err)
	}
	return nil
}
