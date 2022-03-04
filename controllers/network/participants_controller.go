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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	networkclientset "github.com/edgefarm/anck/pkg/client/networkclientset"
)

// ParticipantsReconciler reconciles a Participants object
type ParticipantsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=network.edgefarm.io,resources=participants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.edgefarm.io,resources=participants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=network.edgefarm.io,resources=participants/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Participants object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ParticipantsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	// Lookup the Network instance for this reconcile request
	participant := &networkv1alpha1.Participants{}
	err := r.Get(ctx, req.NamespacedName, participant)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{
			Requeue: true,
		}, err
	}

	clientset, err := setupNetworkClientset()
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return ctrl.Result{}, err
	}

	setupLog.Info(fmt.Sprintf("Reconceiling participant '%s' in network '%s' in namespace '%s'", participant.Spec.Component, participant.Spec.Network, participant.Namespace))
	desiredNetwork := participant.Spec.Network
	if networkExists(desiredNetwork, participant.Namespace) {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			network, err := clientset.NetworkV1alpha1().Networks(participant.Namespace).Get(ctx, desiredNetwork, metav1.GetOptions{})
			if err != nil {
				return err
			}
			network = addParticipant(network, participant.Spec.Component)
			_, err = clientset.NetworkV1alpha1().Networks(network.Namespace).Update(context.Background(), network, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		return ctrl.Result{
			Requeue: true,
		}, nil
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 5 * time.Second,
	}, nil
}

func networkExists(network string, namespace string) bool {
	clientset, err := setupNetworkClientset()
	if err != nil {
		return false
	}
	_, err = clientset.NetworkV1alpha1().Networks(namespace).Get(context.Background(), network, metav1.GetOptions{})
	if err != nil {
		setupLog.Info(fmt.Sprintf("Error getting network: %s", err))
		return false
	}
	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// addParticipant adds the participant to the network object.
func addParticipant(network *networkv1alpha1.Network, participant string) *networkv1alpha1.Network {
	if !contains(network.Spec.Participants, participant) {
		network.Spec.Participants = append(network.Spec.Participants, participant)
	}
	return network
}

// removeParticipant removes the participant from the network object.
func removeParticipant(network *networkv1alpha1.Network, participant string) *networkv1alpha1.Network {
	for i, p := range network.Spec.Participants {
		if p == participant {
			network.Spec.Participants = append(network.Spec.Participants[:i], network.Spec.Participants[i+1:]...)
			break
		}
	}

	return network
}

// setupNetworkClientset returns a clientset for the network v1alpha1
func setupNetworkClientset() (*networkclientset.Clientset, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "error getting cluster config")
		return nil, err
	}
	clientset, err := networkclientset.NewForConfig(c)
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	return clientset, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ParticipantsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.Participants{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				clientset, err := setupNetworkClientset()
				if err != nil {
					setupLog.Error(err, "error getting client for cluster")
					return false
				}
				participant := e.Object.(*networkv1alpha1.Participants)
				retry.RetryOnConflict(retry.DefaultRetry, func() error {
					network, err := clientset.NetworkV1alpha1().Networks(participant.Namespace).Get(context.Background(), participant.Spec.Network, metav1.GetOptions{})
					if err != nil {
						return err
					}
					network = removeParticipant(network, participant.Spec.Component)
					_, err = clientset.NetworkV1alpha1().Networks(network.Namespace).Update(context.Background(), network, metav1.UpdateOptions{})
					if err != nil {
						return err
					}
					err = removeParticipantFromComponentSecret(participant.Spec.Component, participant.Spec.Network, participant.Namespace)
					return err
				})

				return false
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldGeneration := e.ObjectOld.GetGeneration()
				newGeneration := e.ObjectNew.GetGeneration()
				// Generation is only updated on spec changes (also on deletion),
				// not metadata or status
				// Filter out events where the generation hasn't changed to
				// avoid being triggered by status updates
				return oldGeneration != newGeneration
			},
		}).
		Complete(r)
}
