package nodemanager

import (
	"context"
	"errors"
	"fmt"
	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	utils_ssh "github.com/piwriw/nodedeploy-controller/utils/ssh"
	"k8s.io/klog/v2"
	"strings"
)

const (
	PkgPathPrefix  = "/pkg/nodedeploy"
	FilePathPrefix = "/path/to"
	check_arch_cmd = "uname -m"
	mktemp_cmd     = "mktemp -d"
)

type NodeJoin interface {
	CheckArch(ctx context.Context, sshclient *utils_ssh.Client) (string, string, error)
	CheckKubernetes(kubernetesVersion string) (string, error)
	PrepareSetupPkg(ctx context.Context, sshclient *utils_ssh.Client, systemArch, version string) error
	StopFirewalld(ctx context.Context, sshclient *utils_ssh.Client) error
	InstallDocker(ctx context.Context, sshclient *utils_ssh.Client, tempDir, systemArch string) error
	SetDockerConf(ctx context.Context, sshclient *utils_ssh.Client, harborUser, harborPwd, harborEndpoint string) error
	SetHostName(ctx context.Context, sshclient *utils_ssh.Client, hostname string) error
	LoadImage(ctx context.Context, sshclient *utils_ssh.Client) error
	SetK8sComponents(ctx context.Context, sshclient *utils_ssh.Client, tempDir string) error
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

	workerNode_join_cmd := fmt.Sprintf("bash %s/07-work-join.sh %v %v %v", FilePathPrefix, masterHostAndPort, token, hash)
	klog.Infof("上线时具体操作为: %v\n", workerNode_join_cmd)
	_, err = sshclient.Exec(ctx, workerNode_join_cmd)
	if err != nil {
		return errors.New("workerNode start failed")
	}
	return nil
}

func (wn WorkNode) SetK8sComponents(ctx context.Context, sshclient *utils_ssh.Client, tempDir string) error {
	node_setup_cmd := fmt.Sprintf("bash %s/06-k8s-setup.sh %v", FilePathPrefix, tempDir)
	_, err := sshclient.Exec(ctx, node_setup_cmd)
	if err != nil {
		return fmt.Errorf("workerNode doesn't exist, %w", err)
	}
	return nil
}
func (wn WorkNode) LoadImage(ctx context.Context, sshclient *utils_ssh.Client) error {
	image_load_cmd := fmt.Sprintf("bash %s/05-load-image.sh ", FilePathPrefix)
	_, err := sshclient.Exec(ctx, image_load_cmd)
	if err != nil {
		return fmt.Errorf("the image file does not exist, %w", err)
	}
	return nil
}
func (wn WorkNode) SetHostName(ctx context.Context, sshclient *utils_ssh.Client, hostName string) error {
	set_hostname_cmd := fmt.Sprintf("bash %s/04-homename_setup.sh %v", FilePathPrefix, hostName)
	_, err := sshclient.Exec(ctx, set_hostname_cmd)
	if err != nil {
		return fmt.Errorf("The node has already been set up, %w", err)
	}
	return nil
}
func (wn WorkNode) SetDockerConf(ctx context.Context, sshclient *utils_ssh.Client, harborUser, harborPwd, harborEndpoint string) error {
	docker_config_cmd := fmt.Sprintf("bash %s/03-docker_config.sh %v %v %v", FilePathPrefix)
	_, err := sshclient.Exec(ctx, docker_config_cmd)
	if err != nil {
		return fmt.Errorf("docker config.json doesn't exist, %w", err)
	}
	return nil
}
func (wn WorkNode) InstallDocker(ctx context.Context, sshclient *utils_ssh.Client, tempDir, systemArch string) error {
	docker_install_cmd := fmt.Sprintf("bash %s/02-docker_install.sh %v %v", FilePathPrefix, tempDir, systemArch)
	_, err := sshclient.Exec(ctx, docker_install_cmd)
	if err != nil {
		return fmt.Errorf("docker doesn't exist, %w", err)
	}
	return nil
}
func (wn WorkNode) StopFirewalld(ctx context.Context, sshclient *utils_ssh.Client) error {
	stop_firewall_cmd := fmt.Sprintf("bash %s/01-stopwalld.sh", FilePathPrefix)
	_, err := sshclient.Exec(ctx, stop_firewall_cmd)
	if err != nil {
		return fmt.Errorf("Failed to turn off the system firewall setting, %w", err)
	}
	return nil
}

func (wn WorkNode) PrepareSetupPkg(ctx context.Context, sshclient *utils_ssh.Client, systemArch, version string) error {
	//离线安装包源地址  /pkg/nodedeploy/work/arch-v1.21.tar.gz
	src_setup_pkg := fmt.Sprintf("%s/%s/%v-%v.tar.gz", PkgPathPrefix, strings.ToLower(nodev1.NodeWork.String()), systemArch, version)
	//离线安装包目的地址
	dest_setup_pkg := fmt.Sprintf("/tmp/nodedeploy-%v-%v.tar.gz", systemArch, version)
	//从本地原地址下发到目标文件地址
	err := sshclient.Upload(ctx, src_setup_pkg, dest_setup_pkg)
	if err != nil {
		return fmt.Errorf("Upload failed, err: %s", err)
	}
	// /path/to/nodedeploy/work
	mkdir_cmd := fmt.Sprintf("mkdir -p %s/nodedeploy/%s", FilePathPrefix, strings.ToLower(nodev1.NodeWork.String()))
	_, err = sshclient.Exec(ctx, mkdir_cmd)
	unzip_cmd := fmt.Sprintf("tar -xzvf %v -C %v", dest_setup_pkg, FilePathPrefix)
	//将离线安装包解压到指定目录下 /path/to/
	_, err = sshclient.Exec(ctx, unzip_cmd)
	if err != nil {
		return fmt.Errorf("Failed to extract the zip file, err: %s", err)
	}
	return nil
}

func (wn WorkNode) CheckKubernetes(version string) (string, error) {
	//获取kubernetes的版本
	versionK8s, err := getKubernetesVersionStr(version)
	if err != nil {
		return versionK8s, fmt.Errorf("invalid kubernetes version,currently version:%s, err: %s", versionK8s, err)
	}
	return versionK8s, err

}

func (wn WorkNode) CheckArch(ctx context.Context, sshclient *utils_ssh.Client) (string, string, error) {
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
		return "", "", fmt.Errorf("系统架构异常，请检查当前系统架构, %w", err)
	}
	systemArch, err := ParseArch(systemlArch)
	if err != nil {
		klog.Infof("The architecture of the system  is [%v]\n", systemlArch)
		return "", "", err
	}
	return systemArch, temp_dir, nil
}

type KubEdgeNode struct {
}

func (kn KubEdgeNode) Join() {

}
