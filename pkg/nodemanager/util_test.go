package nodemanager

import (
	"context"
	"fmt"
	"github.com/coreos/go-semver/semver"
	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	utils_ssh "github.com/piwriw/nodedeploy-controller/utils/ssh"
	"testing"
)

func TestGetWorkJoinArgs(t *testing.T) {
	masterHostAndPort, token, hash, err := GetWorkJoinInfo()
	if err != nil {
		fmt.Println("获取不到token， err： ", err)
	}
	fmt.Printf("kubeadm join %s --token %s  --discovery-token-ca-cert-hash %s\n", masterHostAndPort, token, hash)
	fmt.Println("masterip: ", masterHostAndPort)
	fmt.Println("token: ", token)
	fmt.Println("hash: ", hash)
}

func TestKubernetesVersion(t *testing.T) {
	_, validateErr := semver.NewVersion("1.21.0")
	if validateErr != nil {
		t.Error(validateErr)
	}
}
func TestCreateSSHClient(t *testing.T) {
	_, err := utils_ssh.NewClient("192.168.28.142", "22", "root", "Piwriw503420")
	if err != nil {
		t.Fatal(err)
	}
}
func TestUpload(t *testing.T) {
	sshclient, err := utils_ssh.NewClient("192.168.28.142", "22", "root", "Piwriw503420")
	if err != nil {
		t.Fatal(err)
	}
	//离线安装包源地址  /pkg/nodedeploy/work/arch-v1.21.tar.gz
	src_setup_pkg := fmt.Sprintf("%s/%v-%v.tar.gz", "F:\\workSpace\\goSpace\\nodedeploy-controller\\deploypkg\\work", "amd64", "1.21.14")
	//离线安装包目的地址
	dest_setup_pkg := fmt.Sprintf("/tmp/nodedeploy-%v-%v.tar.gz", "amd64", "1.21.14")
	//从本地原地址下发到目标文件地址
	err = sshclient.Upload(context.TODO(), src_setup_pkg, dest_setup_pkg)
	if err != nil {
		t.Fatal(err)
	}
}
func TestSetDockerConf(t *testing.T) {
	sshclient, err := utils_ssh.NewClient("192.168.28.142", "22", "root", "Piwriw503420")
	if err != nil {
		t.Fatal(err)
	}
	docker_config_cmd := fmt.Sprintf("bash %s/%s/03-docker_config.sh %v %v %v", FilePathPrefix, nodev1.NodeWork.String(), "admin", "Harbor12345", "http://192.168.28.130:30003")
	_, err = sshclient.Exec(context.TODO(), docker_config_cmd)
	if err != nil {
		t.Fatal(err)
	}
}
func TestLoadImage(t *testing.T) {
	sshclient, err := utils_ssh.NewClient("192.168.28.142", "22", "root", "Piwriw503420")
	if err != nil {
		t.Fatal(err)
	}
	image_load_cmd := fmt.Sprintf("bash %s/%s/05-load-image.sh images/work ", FilePathPrefix, nodev1.NodeWork.String())
	_, err = sshclient.Exec(context.TODO(), image_load_cmd)
	if err != nil {
		t.Fatal(err)
	}
}
func TestDeprecate(t *testing.T) {
	sshclient, err := utils_ssh.NewClient("192.168.28.142", "22", "root", "Piwriw503420")
	if err != nil {
		t.Fatal(err)
	}
	workerNode_disjoin_cmd := fmt.Sprintf("bash %s/%s/08-work-disjoin.sh ", FilePathPrefix, nodev1.NodeWork.String())
	_, err = sshclient.Exec(context.TODO(), workerNode_disjoin_cmd)
	if err != nil {
		t.Fatal(err)
	}
}
