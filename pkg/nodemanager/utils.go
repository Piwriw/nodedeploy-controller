package nodemanager

import (
	"fmt"
	"github.com/coreos/go-semver/semver"
	"strings"
)

func getKubernetesVersionStr(version string) (string, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.%d", v.Major, v.Minor), nil
}
func ParseArch(systemArch string) (string, error) {
	//如果检测系统架构的类型是 aarch64 或者 arm64 就定义为Arm64
	if strings.Contains(systemArch, "aarch64") || strings.Contains(systemArch, "arm64") {
		systemArch = "arm64"
		//如果检测系统架构的类型是 x86_64 或者 amd64 amd64
	} else if strings.Contains(systemArch, "x86_64") || strings.Contains(systemArch, "amd64") {
		systemArch = "amd64"
		//否则就定义为未知类型，并且上报

	} else {
		return systemArch, fmt.Errorf("系统架构异常，请检查当前系统架构")
	}
	return systemArch, nil
}
