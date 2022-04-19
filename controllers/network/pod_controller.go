/*
Copyright © 2021 Ci4Rail GmbH <engineering@ci4rail.com>
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

package network

import (
	"context"
	"fmt"
	"strings"
	"time"

	common "github.com/edgefarm/anck/pkg/common"
	resources "github.com/edgefarm/anck/pkg/resources"
	"github.com/ssoroka/slice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	podsLog = ctrl.Log.WithName("pods")
)

const (
	// PodFinalizer is the name of the finalizer which will be added to the
	PodFinalizer = "applications.edgefarm.io/finalizer"
)

// PodsReconiler reconciles a Participants object
type PodsReconiler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Participants object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *PodsReconiler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	if pod.Spec.NodeName == "" {
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}
	if !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDeletePod(ctx, pod)
	}
	return r.reconcileCreatePod(ctx, pod)
}

func (r *PodsReconiler) reconcileCreatePod(ctx context.Context, pod *corev1.Pod) (ctrl.Result, error) {
	podNetworks := []string{}

	labels := pod.GetLabels()
	for k, v := range labels {
		if strings.Contains(k, "participant.edgefarm.io/") {
			podNetworks = append(podNetworks, v)
		}
	}

	if len(podNetworks) > 0 {
		if pod.Spec.NodeName == "" {
			return ctrl.Result{
				RequeueAfter: 5 * time.Second,
			}, fmt.Errorf("Still waiting for pod '%s' to be scheduled onto a node", pod.Name)
		}
		podsLog.Info("Create: ", "event", pod.Name)
		pod, err := resources.GetPod(pod.Name, pod.Namespace)
		if err != nil {
			podsLog.Error(err, "error getting pod")
			return ctrl.Result{}, err
		}
		node := pod.Spec.NodeName
		nodeLabels, err := resources.GetNodeLabels(node)
		if err != nil {
			podsLog.Error(err, "error getting node labels")
			return ctrl.Result{
				RequeueAfter: 5 * time.Second,
			}, err
		}
		edgeNode := false
		fmt.Println(node)
		fmt.Println(nodeLabels)
		if _, ok := nodeLabels["node-role.kubernetes.io/edge"]; ok {
			edgeNode = true
		}
		if !edgeNode {
			node = "main"
		}
		podsLog.Info("Networks for pod:", "pod", pod.Name, "node", node, "networks", podNetworks)
		for _, networkName := range podNetworks {
			network, err := resources.GetNetwork(networkName, pod.Namespace)
			if err != nil {
				podsLog.Error(err, "error getting network")
				return ctrl.Result{}, err
			}
			// prevent from creating the same pod twice
			if !slice.Contains(network.Info.Participating.Pods[node], pod.Name) {
				network.Info.Participating.PodsCreating[node] = append(network.Info.Participating.PodsCreating[node], pod.Name)
				network.Info.Participating.PodsCreating[node] = slice.Unique(network.Info.Participating.PodsCreating[node])
				if network.Info.Participating.Nodes[node] != participatingNodeStateActive {
					if len(network.Info.Participating.PodsCreating[node]) > 0 {
						network.Info.Participating.Nodes[node] = participatingNodeStatePending
					}
				}
			}
			_, err = resources.UpdateNetwork(network, pod.Namespace)
			if err != nil {
				podsLog.Error(err, "error updating network")
				return ctrl.Result{
					RequeueAfter: 5 * time.Second,
				}, err
			}
		}

	}

	return ctrl.Result{}, nil
}

func (r *PodsReconiler) reconcileDeletePod(ctx context.Context, pod *corev1.Pod) (ctrl.Result, error) {
	podNetworks := []string{}
	labels := pod.GetLabels()
	for k, v := range labels {
		if strings.Contains(k, "participant.edgefarm.io/") {
			podNetworks = append(podNetworks, v)
		}
	}

	if len(podNetworks) > 0 {
		podsLog.Info("Delete: ", "event", pod.Name)
		pod, err := resources.GetPod(pod.Name, pod.Namespace)
		if err != nil {
			podsLog.Info("error getting pod")
			return ctrl.Result{}, err
		}
		node := pod.Spec.NodeName
		for _, networkName := range podNetworks {
			fmt.Println(networkName)
			network, err := resources.GetNetwork(networkName, pod.Namespace)
			if err != nil {
				podsLog.Error(err, "error getting network")
				return ctrl.Result{
					RequeueAfter: 5 * time.Second,
				}, err
			}
			network.Info.Participating.PodsTerminating[node] = append(network.Info.Participating.PodsTerminating[node], pod.Name)
			network.Info.Participating.PodsTerminating[node] = slice.Unique(network.Info.Participating.PodsTerminating[node])
			if common.SliceEqual(network.Info.Participating.PodsTerminating[node], network.Info.Participating.Pods[node]) {
				if !slice.Contains(network.Info.Participating.PodsCreating[node], pod.Name) {
					network.Info.Participating.Nodes[node] = participatingNodeStateTerminating
				}
			}
			_, err = resources.UpdateNetwork(network, pod.Namespace)
			if err != nil {
				podsLog.Error(err, "error updating network")
				return ctrl.Result{
					RequeueAfter: 5 * time.Second,
				}, err
			}
		}
	}
	// err = resources.RemovePodFinalizers(req.Name, req.Namespace, []string{PodFinalizer})
	// if err != nil {
	// 	podsLog.Error(err, "error removing finalizers")
	// 	return ctrl.Result{
	// 		Requeue: false,
	// 	}, err
	// }

	return ctrl.Result{
		Requeue: false,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodsReconiler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				podNetworks := []string{}
				labels := e.Object.GetLabels()
				for k, v := range labels {
					if strings.Contains(k, "participant.edgefarm.io/") {
						podNetworks = append(podNetworks, v)
					}
				}

				return len(podNetworks) > 0
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldGeneration := e.ObjectOld.GetGeneration()
				newGeneration := e.ObjectNew.GetGeneration()
				// Generation is only updated on spec changes (also on deletion),
				// not metadata or status
				// Filter out events where the generation hasn't changed to
				// avoid being triggered by status updates
				if oldGeneration == newGeneration {
					labels := e.ObjectNew.GetLabels()
					podNetworks := []string{}
					for k, v := range labels {
						if strings.Contains(k, "participant.edgefarm.io/") {
							podNetworks = append(podNetworks, v)
						}
					}
					// Allow events for pods who are participating in a network (len(podNetworks) > 0)
					// AND where the pod is not being deleted (len(e.ObjectNew.GetFinalizers()) > 0)
					return len(podNetworks) > 0 && len(e.ObjectNew.GetFinalizers()) > 0
				}
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				// Delete is handeled by the reconcile function using the finalizer
				return false
			},
		}).
		Complete(r)
}
