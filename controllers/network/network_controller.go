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

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
	common "github.com/edgefarm/anck/pkg/common"
	resources "github.com/edgefarm/anck/pkg/resources"
)

var (
	networkLog = ctrl.Log.WithName("network")
)

const (
	timeoutSeconds = 10

	// anckParticipant is the participant that is able to create jetstreams
	anckParticipant = "anck-this-name-shall-never-be-used"
)

// NetworksReconciler reconciles a Network object
type NetworksReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=network.edgefarm.io,resources=networks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.edgefarm.io,resources=networks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=network.edgefarm.io,resources=networks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Network object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *NetworksReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// your logic here
	// Lookup the Network instance for this reconcile request
	network := &networkv1alpha1.Network{}
	err := r.Get(ctx, req.NamespacedName, network)
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

	networkName := network.Name
	grpcContext, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
	defer cancel()

	cc, err := grpc.Dial(common.AnckcredentialsServiceURL, grpc.WithInsecure())
	if err != nil {
		errorText := "Error connecting to anckcredentials"
		networkLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}
	defer cc.Close()

	client := anckcredentials.NewConfigServiceClient(cc)
	networkLog.Info(fmt.Sprintf("Requesting credentials for network '%s'", networkName))

	participants := network.Spec.Participants
	participants = append(participants, fmt.Sprintf("%s.%s", network.Spec.App, anckParticipant))
	resp, err := client.DesiredState(grpcContext, &anckcredentials.DesiredStateRequest{
		Network:      networkName,
		Participants: participants,
	})
	if err != nil {
		errorText := "Error setting desired state"
		networkLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}

	namespace := ""
	if network.Spec.Namespace != "" {
		// first Case: create secret within that namespace.
		namespace = network.Spec.Namespace
	} else {
		// third case: create secret within the namespace the resource was defined 'network.Namespace'.
		namespace = network.Namespace
	}

	err = resources.CreateNamespace(namespace)
	if err != nil {
		errorText := "Error creating namespace"
		networkLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}
	for _, component := range participants {
		if err != nil {
			errorText := "Error splitting network participant"
			networkLog.Error(err, errorText)
			return ctrl.Result{}, fmt.Errorf("%s", errorText)
		}
		secret, err := createOrUpdateComponentSecrets(component, namespace, resp)
		if err != nil {
			errorText := "Error creating or updating component secret"
			networkLog.Error(err, errorText)
			return ctrl.Result{}, fmt.Errorf("%s", errorText)
		}

		err = createOrUpdateComponentDaprSecrets(secret)
		if err != nil {
			errorText := "Error creating or updating component dapr secret"
			networkLog.Error(err, errorText)
			return ctrl.Result{}, fmt.Errorf("%s", errorText)
		}
	}

	anckCreds, err := readCredentialsFromSecret(appComponentName(network.Spec.App, anckParticipant), network.Name, namespace)
	if err != nil {
		errorText := "Error reading credentials from secret"
		networkLog.Error(err, errorText)
		return ctrl.Result{}, fmt.Errorf("%s", errorText)
	}

	jetstream, err := NewJetstream(anckCreds)
	if err != nil {
		errorText := "Error creating jetstream"
		networkLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}

	for _, stream := range network.Spec.Streams {
		err = jetstream.Create(network.Name, stream)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("Error creating jetstream: %s", err)
		}
	}

	jetstream.Cleanup()

	return ctrl.Result{
		Requeue:      false,
		RequeueAfter: 0,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworksReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.Network{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				networkLog.Info("Delete handler for network called")

				network := e.Object.(*networkv1alpha1.Network)
				streamNames := func(streams []networkv1alpha1.StreamSpec) []string {
					var names []string
					for _, stream := range streams {
						names = append(names, fmt.Sprintf("%s_%s", network.Spec.App, stream.Name))
					}
					return names
				}(network.Spec.Streams)

				anckCreds, err := readCredentialsFromSecret(appComponentName(network.Spec.App, anckParticipant), network.Name, network.Spec.Namespace)
				if err != nil {
					errorText := "Error reading credentials from secret"
					networkLog.Error(err, errorText)
				}

				js, err := NewJetstream(anckCreds)
				if err != nil {
					networkLog.Error(err, "Error creating jetstream")
					return false
				}

				if len(streamNames) > 0 {
					networkLog.Info("Delete configured jetstreams:")
					for _, streamName := range streamNames {
						networkLog.Info(fmt.Sprintf("\t- %s\n", streamName))
					}
					err := js.Delete(network.Name, streamNames)
					if err != nil {
						networkLog.Info("Error deleting jetstreams")
						return false
					}
				}

				js.Cleanup()

				grpcContext, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
				defer cancel()

				cc, err := grpc.Dial(common.AnckcredentialsServiceURL, grpc.WithInsecure())
				if err != nil {
					errorText := "Error connecting to anckcredentials"
					networkLog.Info(errorText)
					return false
				}
				defer cc.Close()
				client := anckcredentials.NewConfigServiceClient(cc)

				networkLog.Info(fmt.Sprintf("Deleting network '%s'", network.Name))
				_, err = client.DeleteNetwork(grpcContext, &anckcredentials.DeleteNetworkRequest{
					Network: network.Name,
				})
				if err != nil {
					errorText := fmt.Sprintf("Cannot delete network '%s'", network.Name)
					fmt.Println(errorText)
					return false
				}

				namespace := ""
				if network.Spec.Namespace != "" {
					// first Case: create secret within that namespace.
					namespace = network.Spec.Namespace
				} else {
					// third case: create secret within the namespace the resource was defined 'network.Namespace'.
					namespace = network.Namespace
				}
				networkParticipants := network.Spec.Participants
				networkParticipants = append(networkParticipants, appComponentName(network.Spec.App, anckParticipant))

				for _, component := range networkParticipants {
					err = removeNetworkFromComponentSecret(component, network.Name, namespace)
					if err != nil {
						if errors.IsNotFound(err) {
							networkLog.Info(fmt.Sprintf("Secret '%s' not found. Must have been deleted by participant controller: %s", component, err))
						} else {
							networkLog.Error(err, "Error updating component secret")
							return false
						}
					}
					err = removeNetworkFromDaprSecret(component, network.Name, namespace)
					if err != nil {
						if errors.IsNotFound(err) {
							networkLog.Info(fmt.Sprintf("Secret '%s' not found. Must have been deleted by participant controller: %s", component, err))
						} else {
							networkLog.Error(err, "Error updating dapr secret")
							return false
						}
					}
				}
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

func appComponentName(app string, component string) string {
	return fmt.Sprintf("%s.%s", app, component)
}
