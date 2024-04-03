/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/go-semver/semver"
	"github.com/piwriw/nodedeploy-controller/pkg/nodemanager"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	watchtool "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"net"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	pkgtypes "github.com/piwriw/nodedeploy-controller/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	DeadlineDuration = time.Minute * 30
)

// NodeDeployReconciler reconciles a NodeDeploy object
type NodeDeployReconciler struct {
	client.Client
	Recorder  record.EventRecorder
	Scheme    *runtime.Scheme
	ClientSet *kubernetes.Clientset
}

//+kubebuilder:rbac:groups=node.nodedeploy,resources=nodedeploys,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node.nodedeploy,resources=nodedeploys/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node.nodedeploy,resources=nodedeploys/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeDeploy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *NodeDeployReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	nodeDeploy := &nodev1.NodeDeploy{}
	err := r.Get(ctx, req.NamespacedName, nodeDeploy)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//判断节点状态,如果没有节点状态说明是第一次加入
	if nodeDeploy.Status.NodeStatus == nodev1.NodeUnknown {
		r.Recorder.Event(nodeDeploy, corev1.EventTypeNormal, pkgtypes.EventChangeStatus, "init status")
		nodeDeploy.Status.NodeStatus = nodev1.NodeInit
		if err = r.Status().Update(ctx, nodeDeploy); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// reach target status
	if nodeDeploy.Spec.NodeStatus == nodeDeploy.Status.NodeStatus {
		return ctrl.Result{}, nil
	}

	// other thread are processing
	//如果处于中间状态，比如重启中和删除中
	if nodeDeploy.Status.NodeStatus == nodev1.NodeLaunching ||
		nodeDeploy.Status.NodeStatus == nodev1.NodeDeprecating {
		if nodeDeploy.Status.Deadline != nil && time.Now().After(nodeDeploy.Status.Deadline.Time) {
			beforeStatus := getBeforeStatus(nodeDeploy.Status.NodeStatus)
			r.Recorder.Event(nodeDeploy, corev1.EventTypeWarning, pkgtypes.EventChangeStatus, fmt.Sprintf("exec timeout, fallback: %s -> %s", nodeDeploy.Status.NodeStatus, beforeStatus))
			nodeDeploy.Status.NodeStatus = beforeStatus
			if err = r.Status().Update(ctx, nodeDeploy); err != nil {

				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: nodeDeploy.Status.Deadline.Time.Sub(time.Now())}, nil
		}
		return ctrl.Result{}, nil
	}
	//如果处于上线和上线失败
	if nodeDeploy.Spec.NodeStatus == nodev1.NodeActive && nodeDeploy.Status.NodeStatus == nodev1.NodeLaunchFail {
		return ctrl.Result{}, nil
		//如果处于下线和下线失败状态
	} else if nodeDeploy.Spec.NodeStatus == nodev1.NodeInactive && nodeDeploy.Status.NodeStatus == nodev1.NodeDeprecateFail {
		return ctrl.Result{}, nil
	}

	//从 secret 中获取 nodeinfo 信息
	//nodeInfo, err := r.getNodeInfoFormSecret(ctx, nodeDeploy)
	//if err != nil {
	//	r.Recorder.Event(nodeDeploy, corev1.EventTypeWarning, pkgtypes.EventPreflight, fmt.Sprintf(
	//		"get secret %s/%s failed", nodeDeploy.Namespace, getSecretName(nodeDeploy),
	//	))
	//	return ctrl.Result{}, err
	//}

	nodeInfo := &pkgtypes.NodeInfo{
		NodeName:       nodeDeploy.Spec.NodeName,
		NodeIP:         nodeDeploy.Spec.NodeIP,
		NodeType:       nodeDeploy.Spec.NodeType.String(),
		NodePort:       nodeDeploy.Spec.NodePort,
		NodeUser:       nodeDeploy.Spec.NodeUser,
		NodePwd:        nodeDeploy.Spec.NodePwd,
		HarborEndpoint: nodeDeploy.Spec.HarborEndpoint,
		HarborUser:     nodeDeploy.Spec.HarborUser,
		HarborPwd:      nodeDeploy.Spec.HarborPwd,
	}
	success, err := r.preflight(ctx, nodeDeploy, nodeInfo)
	if err != nil {
		r.Recorder.Event(nodeDeploy, corev1.EventTypeWarning, pkgtypes.EventPreflight, fmt.Sprintf("Preflight err:%s", err))
		return ctrl.Result{}, err
	}
	if !success {
		if nodeDeploy.Status.NodeStatus == nodev1.NodeInactive {
			nodeDeploy.Status.NodeStatus = nodev1.NodeDeprecateFail
		} else if nodeDeploy.Spec.NodeStatus == nodev1.NodeActive {
			nodeDeploy.Status.NodeStatus = nodev1.NodeLaunchFail
		}
		//更新 workerDeploy 资源
		if err = r.Status().Update(ctx, nodeDeploy); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if nodeDeploy.Spec.NodeStatus == nodev1.NodeActive {
		// 节点上线
		if err = r.prepare(ctx, nodeDeploy, nodev1.NodeLaunching); err != nil {
			return ctrl.Result{}, err
		}
		success, err = r.launchNode(ctx, nodeDeploy, nodeInfo)
		if !success {
			klog.Warningf("节点上线时出现异常，导致无法正常上线 ,err: %v\n", err)
			return ctrl.Result{}, nil
		}
		var status nodev1.NodeStatus
		if success {
			status = nodev1.NodeActive
		} else {
			status = nodev1.NodeLaunchFail
		}
		//上线后确定状态并更新
		if err = r.finalize(ctx, nodeDeploy, status); err != nil {
			return ctrl.Result{}, err
		}
		//下线过程
	} else if nodeDeploy.Spec.NodeStatus == nodev1.NodeInactive {
		//如果处于初始化状态
		if nodeDeploy.Status.NodeStatus == nodev1.NodeInit {
			r.Recorder.Event(nodeDeploy, corev1.EventTypeWarning, pkgtypes.EventInvalidStatus, "ignore init to inactive")
			return ctrl.Result{}, nil
		}

		//下线前确定状态
		if err = r.prepare(ctx, nodeDeploy, nodev1.NodeDeprecating); err != nil {
			return ctrl.Result{}, err
		}
		success, err = r.deprecateNode(ctx, nodeDeploy, nodeInfo)
		if err != nil {
			return ctrl.Result{}, err
		}
		var status nodev1.NodeStatus
		if success {
			status = nodev1.NodeInactive
		} else {
			status = nodev1.NodeDeprecateFail
		}

		if err = r.finalize(ctx, nodeDeploy, status); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeDeployReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("NodeDeploy")
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodev1.NodeDeploy{}).
		Complete(r)
}
func (r *NodeDeployReconciler) getNodeInfoFormSecret(ctx context.Context, nodeDeploy *nodev1.NodeDeploy) (*pkgtypes.NodeInfo, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Namespace: nodeDeploy.Namespace, Name: getSecretName(nodeDeploy)}, secret)
	if err != nil {
		return nil, err
	}

	info := pkgtypes.NodeInfo{}
	err = json.Unmarshal(secret.Data["nodeInfo"], &info)
	if err != nil {
		return nil, err
	}
	logger := log.FromContext(ctx)
	logger.Info("secret.nodeInfo", "info", info)

	return &info, nil
}

func (r *NodeDeployReconciler) preflight(ctx context.Context, nodeDeploy *nodev1.NodeDeploy, nodeInfo *pkgtypes.NodeInfo) (success bool, err error) {
	var reason string
	defer func() {
		if !success {
			r.Recorder.Event(nodeDeploy, corev1.EventTypeWarning, pkgtypes.EventStatusMessage, nodemanager.StatusMessage{
				Phase:    "Parameter validate",
				Message:  reason,
				Progress: "1/1",
			}.String())
		}
	}()
	if nodeDeploy.Spec.NodeName != nodeInfo.NodeName ||
		nodeDeploy.Spec.NodeIP != nodeInfo.NodeIP {
		reason = fmt.Sprintf("nodeName or nodeIP does not match the config in secret %s/%s", nodeDeploy.Namespace, getSecretName(nodeDeploy))
		return
	}

	// labels
	_, validateErr := labels.Set(nodeDeploy.Spec.Labels).AsValidatedSelector()
	if validateErr != nil {
		reason = fmt.Sprintf("invalid nodeLabels, err: %s", validateErr)
		return
	}

	// GET nodeIP
	parsedIP := net.ParseIP(nodeDeploy.Spec.NodeIP)
	if parsedIP == nil {
		reason = "nodeIP is an invalid ip address"
		return
	}
	//KubernetesVersion, 检测 k8s 版本
	_, validateErr = semver.NewVersion(nodeDeploy.Spec.KubernetesVersion)
	if validateErr != nil {
		reason = fmt.Sprintf("invalid KubernetesVersion, err: %s", validateErr)
		return
	}

	success = true
	return
}
func (r *NodeDeployReconciler) prepare(ctx context.Context, nodeDeploy *nodev1.NodeDeploy, status nodev1.NodeStatus) error {
	// deadline
	now := metav1.NewTime(time.Now().Add(DeadlineDuration))
	nodeDeploy.Status.Deadline = &now

	// update status
	r.Recorder.Event(nodeDeploy, corev1.EventTypeNormal, pkgtypes.EventChangeStatus, fmt.Sprintf("%s -> %s", nodeDeploy.Status.NodeStatus, status))
	nodeDeploy.Status.NodeStatus = status
	if err := r.Status().Update(ctx, nodeDeploy); err != nil {
		return err
	}

	return nil
}
func (r *NodeDeployReconciler) launchNode(ctx context.Context, nodeDeploy *nodev1.NodeDeploy, nodeInfo *pkgtypes.NodeInfo) (success bool, err error) {
	manager := nodemanager.New(r.Recorder, nodeDeploy, nodeInfo)
	err = manager.LaunchByCmd(ctx)
	if err != nil {
		return false, err
	}
	success, err = r.watchNode(ctx, time.Minute*5, nodeDeploy.Spec.NodeName, true)
	if err != nil {
		r.Recorder.Event(nodeDeploy, corev1.EventTypeWarning, pkgtypes.EventLaunchNode, "node status is not as expected")
		return false, err
	}
	if success {
		node := &corev1.Node{}
		err := retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
			err = r.Get(ctx, types.NamespacedName{Namespace: corev1.NamespaceAll, Name: nodeDeploy.Spec.NodeName}, node)
			if err != nil {
				return err
			}

			node.Labels = labels.Merge(node.Labels, nodeDeploy.Spec.Labels)
			for _, t1 := range nodeDeploy.Spec.Taints {
				var existed bool
				for _, t2 := range node.Spec.Taints {
					if t1.MatchTaint(&t2) {
						existed = true
						break
					}
				}
				if !existed {
					node.Spec.Taints = append(node.Spec.Taints, t1)
				}
			}
			for k, v := range nodeDeploy.Spec.Annotations {
				node.Annotations[k] = v
			}

			return r.Update(ctx, node)
		})
		if err != nil {
			return false, err
		}
	}

	return success, nil
}
func (r *NodeDeployReconciler) watchNode(ctx context.Context, timeout time.Duration, nodeName string, ready bool) (success bool, err error) {
	logger := log.FromContext(ctx)

	node := &corev1.Node{}
	err = r.Get(ctx, types.NamespacedName{Namespace: corev1.NamespaceAll, Name: nodeName}, node)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
	} else if nodeReady := nodeIsReady(node); nodeReady == ready {
		r.Recorder.Event(node, corev1.EventTypeNormal, pkgtypes.EventLaunchNode, fmt.Sprintf("node %s ready: %v", nodeName, nodeReady))
		return true, nil
	}
	timeOut := int64(10)
	// 创建一个ListWatch监控器
	watcher, err := watchtool.NewRetryWatcher("1", &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return r.ClientSet.CoreV1().Nodes().Watch(ctx, metav1.ListOptions{TimeoutSeconds: &timeOut})
		},
	})
	if err != nil {
		return false, err
	}
	timer := time.NewTimer(timeout)
	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
			logger.Info("Stop Watch Node", "reason", "context done")

			return false, nil
		case <-timer.C:
			watcher.Stop()
			logger.Info("Stop Watch Node", "reason", "timeout")

			return false, nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				continue
			}
			switch event.Type {
			case watch.Added, watch.Modified:
				node, ok := event.Object.(*corev1.Node)
				if !ok {
					logger.Info("Watch Node", "error", "type not match")
					continue
				}
				if node.Name != nodeName {
					continue
				}
				if nodeReady := nodeIsReady(node); nodeReady == ready {
					watcher.Stop()
					logger.Info("Stop Watch Node", "reason", "done")
					r.Recorder.Event(node, corev1.EventTypeNormal, pkgtypes.EventLaunchNode, fmt.Sprintf("node %s ready: %v", nodeName, nodeReady))

					return true, nil
				}

				logger.Info("Watch Node", "node", node.Name)
			}
		}
	}
}
func (r *NodeDeployReconciler) finalize(ctx context.Context, nodeDeploy *nodev1.NodeDeploy, status nodev1.NodeStatus) error {
	r.Recorder.Event(nodeDeploy, corev1.EventTypeNormal, pkgtypes.EventChangeStatus, fmt.Sprintf("%s -> %s", nodeDeploy.Status.NodeStatus, status))
	nodeDeploy.Status.NodeStatus = status
	nodeDeploy.Status.Deadline = nil
	if err := r.Status().Update(ctx, nodeDeploy); err != nil {
		return err
	}

	return nil
}

// 节点下线
func (r *NodeDeployReconciler) deprecateNode(ctx context.Context, nodeDeploy *nodev1.NodeDeploy, nodeInfo *pkgtypes.NodeInfo) (success bool, err error) {
	node := &corev1.Node{}
	err = r.Get(ctx, types.NamespacedName{Namespace: corev1.NamespaceAll, Name: nodeInfo.NodeName}, node)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		} else {
			return false, fmt.Errorf("try to deprecate a non-existent node")
		}
	}
	if nodeDeploy.Spec.IsEvicted {
		err = drainNode(r.ClientSet, node)
		if err != nil {
			r.Recorder.Event(nodeDeploy, corev1.EventTypeWarning, pkgtypes.EventDeprecateNode, fmt.Sprintf("failed to drain node, err: %s", err))
			return false, nil
		}
	}

	manager := nodemanager.New(r.Recorder, nodeDeploy, nodeInfo)
	// 节点下线
	err = manager.Deprecate(ctx)
	if err != nil {
		return false, err
	}

	if err = r.Delete(ctx, node); err != nil {
		r.Recorder.Event(nodeDeploy, corev1.EventTypeWarning, pkgtypes.EventDeprecateNode, "failed to delete node")
		return false, err
	}
	r.Recorder.Event(nodeDeploy, corev1.EventTypeNormal, pkgtypes.EventDeprecateNode, "node deleted successfully")

	return true, nil
}
