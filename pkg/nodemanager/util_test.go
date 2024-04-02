package nodemanager

import (
	"fmt"
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
