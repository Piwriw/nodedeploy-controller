package nodemanager

import (
	"context"
	"fmt"
	"github.com/coreos/go-semver/semver"
	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	utils_ssh "github.com/piwriw/nodedeploy-controller/utils/ssh"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"strings"
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
func TestCloudJoinInfo(t *testing.T) {
	//使用configmap将master的kubeconfig文件挂载到容器中的/controllers目录下
	kubeconfig := "F:\\workSpace\\goSpace\\nodedeploy-controller\\conf"
	//kubeconfig := "config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	cloudcoreCfg, err := clientSet.CoreV1().ConfigMaps("kubeedge").Get(context.TODO(), "cloudcore", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	bytesCfg := []byte(cloudcoreCfg.Data["cloudcore.yaml"])
	cloudCfg := &CloudCoreConfig{}
	err = yaml.Unmarshal(bytesCfg, cloudCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(cloudCfg)

}
func TestCheckArch(t *testing.T) {
	sshclient, err := utils_ssh.NewClient("192.168.28.142", "22", "root", "Piwriw503420")
	if err != nil {
		t.Fatal(err)
	}
	temp_dir_str, err := sshclient.Exec(context.TODO(), mktemp_cmd)
	lines := strings.Split(temp_dir_str, "\n")
	temp_dir := lines[0]
	if err != nil {
		klog.Errorf("Failed to create new temp dir: %v", err)
	}
	klog.Infof("The temporary directory [%v] is created\n", temp_dir)

	//检测机器的类型，下载对应的离线安装包
	systemlArch, err := sshclient.Exec(context.TODO(), check_arch_cmd)
	if err != nil {
		t.Fatal(err)
	}
	systemArch, err := ParseArch(systemlArch)
	if err != nil {
		klog.Infof("The architecture of the system  is [%v]\n", systemlArch)
		t.Fatal(err)
	}
	t.Log(systemArch, temp_dir, nil)
}
func TestVersion(t *testing.T) {

	//获取kubernetes的版
	versionK8s, err := getKubernetesVersionStr("1.21.14")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(versionK8s)

}

func TestGetInfoIn(t *testing.T) {
	config, err := rest.InClusterConfig()
	if err != nil {
		t.Fatal(err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	cloudcoreCfg, err := clientSet.CoreV1().ConfigMaps("kubeedge").Get(context.TODO(), "cloudcore", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	bytesCfg := []byte(cloudcoreCfg.Data["cloudcore.yaml"])
	cloudCfg := &CloudCoreConfig{}
	err = yaml.Unmarshal(bytesCfg, cloudCfg)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := clientSet.CoreV1().Secrets("kubeedge").Get(context.Background(), "tokensecret", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	token := string(secret.Data["tokendata"])
	t.Logf("token:%s Address:%v Port：%s", token, cloudCfg.Modules.CloudHub.AdvertiseAddress, cloudCfg.Modules.CloudHub.Websocket.Port)
}
