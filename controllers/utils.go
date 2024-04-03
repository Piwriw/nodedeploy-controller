package controllers

import (
	"fmt"
	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
	"os"
)

// 给节点打上污点
func drainNode(clientset kubernetes.Interface, node *corev1.Node) error {
	h := drain.NewCordonHelper(node)
	h.UpdateIfRequired(true)
	err, patchErr := h.PatchOrReplace(clientset, false)
	if patchErr != nil {
		return patchErr
	}
	if err != nil {
		return err
	}
	c := &drain.Helper{
		GracePeriodSeconds: -1,
		Out:                os.Stdout,
		ErrOut:             os.Stderr,
		ChunkSize:          500,
		Client:             clientset,
	}
	list, errs := c.GetPodsForDeletion(node.Name)
	if errs != nil {
		return utilerrors.NewAggregate(errs)
	}
	if err := c.DeleteOrEvictPods(list.Pods()); err != nil {
		return err
	}

	return nil
}

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
