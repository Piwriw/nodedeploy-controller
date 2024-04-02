package controllers

import (
	"fmt"
	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
)

func nodeIsReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return true
		}
	}

	return false
}
func getBeforeStatus(status nodev1.NodeStatus) nodev1.NodeStatus {
	switch status {
	case nodev1.NodeLaunching:
		return nodev1.NodeInactive
	case nodev1.NodeDeprecating:
		return nodev1.NodeActive
	default:
		return nodev1.NodeUnknown
	}
}
func getSecretName(nodeDeploy *nodev1.NodeDeploy) string {
	return fmt.Sprintf("nodedeploy-%s", nodeDeploy.Name)
}
