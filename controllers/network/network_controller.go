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
	"reflect"
	"time"

	retry "github.com/avast/retry-go/v4"
	slice "github.com/merkur0/go-slices"
	unique "github.com/ssoroka/slice"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	grpcClient "github.com/edgefarm/anck/pkg/grpc"
	"github.com/edgefarm/anck/pkg/nats"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
	common "github.com/edgefarm/anck/pkg/common"
	jetstreams "github.com/edgefarm/anck/pkg/jetstreams"
	resources "github.com/edgefarm/anck/pkg/resources"
)

var (
	networkLog          = ctrl.Log.WithName("network")
	nodeparticipantsLog = ctrl.Log.WithName("node-participants")
	createJetstreamsLog = ctrl.Log.WithName("create-jetstreams")
)

const (
	timeoutSeconds = 10

	// anckParticipant is the participant that is able to create jetstreams
	anckParticipant = "anck-this-name-shall-never-be-used"

	// NetworkFinalizer is the name of the finalizer which will be added to the
	NetworkFinalizer = "network.edgefarm.io/finalizer"
)

type nodeNetworkStateType int

const (
	new nodeNetworkStateType = iota
	stillPartitipatingCreate
	stillPartitipatingDelete
	deleted
	invalid
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
	network := &networkv1alpha1.Network{}
	err := r.Get(ctx, req.NamespacedName, network)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	if !network.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, network)
	}
	return r.reconcileCreate(ctx, network)
}

func (r *NetworksReconciler) reconcileCreate(ctx context.Context, network *networkv1alpha1.Network) (ctrl.Result, error) {
	_, networkError := r.reconcileCreateNetwork(ctx, network)
	if networkError != nil {
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, networkError
	}
	_, mainErr := r.createMainJetstreams(ctx, network)
	_, err := r.reconcileJetstreams(ctx, network)
	if err != nil || mainErr != nil {
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}
	return ctrl.Result{
		RequeueAfter: 5 * time.Second,
	}, nil
}

func (r *NetworksReconciler) reconcileDelete(ctx context.Context, network *networkv1alpha1.Network) (ctrl.Result, error) {
	_, mainErr := r.deleteMainJetstreams(ctx, network)
	_, nodeErr := r.reconcileJetstreams(ctx, network)
	_, networkError := r.reconcileDeleteNetwork(ctx, network)
	err := fmt.Sprintf("mainErr: %v, nodeErr: %v, networkError: %v", mainErr, nodeErr, networkError)
	if networkError != nil || nodeErr != nil || mainErr != nil {
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, fmt.Errorf(err)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworksReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.Network{}).
		WithEventFilter(predicate.Funcs{
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

				return oldGeneration != newGeneration && len(e.ObjectNew.GetFinalizers()) > 0
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				// Delete is handeled by the reconcile function using the finalizer
				return false
			},
		}).
		Complete(r)
}

func (r *NetworksReconciler) reconcileCreateNetwork(ctx context.Context, network *networkv1alpha1.Network) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	networkName := network.Name
	grpcContext, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
	defer cancel()

	cc, err := grpcClient.Dial(common.AnckcredentialsServiceURL, time.Second*10, time.Second*1)
	if err != nil {
		errorText := "Error connecting to anckcredentials"
		networkLog.Error(err, errorText)
		return ctrl.Result{
			Requeue: false,
		}, fmt.Errorf("%s", errorText)
	}
	defer cc.Close()

	client := anckcredentials.NewConfigServiceClient(cc)
	networkLog.Info(fmt.Sprintf("Requesting credentials for network '%s'", networkName))

	participantsMap := network.Info.Participating.Components
	participantsMap[fmt.Sprintf("%s.%s", network.Spec.App, anckParticipant)] = "unknownParticipantType"
	participants := []string{}
	for participant := range participantsMap {
		participants = append(participants, participant)
	}

	resp, err := client.DesiredState(grpcContext, &anckcredentials.DesiredStateRequest{
		Network:      networkName,
		Participants: participants,
	})
	if err != nil {
		errorText := "Error setting desired state"
		networkLog.Error(err, errorText)
		return ctrl.Result{
			Requeue: false,
		}, fmt.Errorf("%s", errorText)
	}
	networkCopy := network.DeepCopy()
	if network.Info.UsedAccount == "" {
		network, err = resources.SetNetworkAccountName(network, resp.Network.AccountName)
		if err != nil {
			errorText := "Error setting network account name"
			networkLog.Error(err, errorText)
			return ctrl.Result{
				Requeue: false,
			}, fmt.Errorf("%s", errorText)
		}
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

	natsServer, err := nats.GetNatsServerInfos()
	if err != nil {
		networkLog.Error(err, "Error reading Nats Server Infos")
		return ctrl.Result{
			Requeue: false,
		}, fmt.Errorf("%s", err.Error())
	}
	// This secret is used by nats-leafnode-client and *-registry to create the leafnode connection to the
	// correct nats server.
	_, err = resources.CreateSecret("nats-server-info", namespace, &map[string]string{
		"NATS_ADDRESS": natsServer.Addresses.NatsAddress,
		"LEAF_ADDRESS": natsServer.Addresses.LeafAddress,
	})
	if err != nil && !errors.IsAlreadyExists(err) {
		errorText := "Error creating secret"
		networkLog.Error(err, errorText)
		return ctrl.Result{
			Requeue: false,
		}, fmt.Errorf("%s", errorText)
	}

	for component, participantType := range participantsMap {
		secret, err := createOrUpdateComponentSecrets(component, namespace, resp)
		if err != nil {
			errorText := "Error creating or updating component secret"
			networkLog.Error(err, errorText)
			return ctrl.Result{}, fmt.Errorf("%s", errorText)
		}

		err = createOrUpdateComponentDaprSecrets(secret, participantType)
		if err != nil {
			errorText := "Error creating or updating component dapr secret"
			networkLog.Error(err, errorText)
			return ctrl.Result{}, fmt.Errorf("%s", errorText)
		}
	}

	err = r.updateInfoAndReturn(ctx, network, networkCopy)
	if err != nil {
		return ctrl.Result{
			RequeueAfter: time.Second * 5,
		}, err
	}

	return ctrl.Result{
		RequeueAfter: time.Second * 5,
	}, nil
}

func appComponentName(app string, component string) string {
	return fmt.Sprintf("%s.%s", app, component)
}

func (r *NetworksReconciler) reconcileDeleteNetwork(ctx context.Context, network *networkv1alpha1.Network) (ctrl.Result, error) {
	networkLog.Info("Delete handler for network called")

	grpcContext, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
	defer cancel()

	cc, err := grpcClient.Dial(common.AnckcredentialsServiceURL, time.Second*10, time.Second*1)
	if err != nil {
		errorText := "Error connecting to anckcredentials"
		networkLog.Info(errorText)
		return ctrl.Result{}, fmt.Errorf("err: %s:%s", errorText, err.Error())
	}
	defer cc.Close()
	client := anckcredentials.NewConfigServiceClient(cc)

	networkLog.Info(fmt.Sprintf("Deleting network '%s'", network.Name))
	_, err = client.DeleteNetwork(grpcContext, &anckcredentials.DeleteNetworkRequest{
		Network: network.Name,
	})
	if err != nil {
		errorText := fmt.Sprintf("Cannot delete network '%s'", network.Name)
		networkLog.Info(errorText)
		return ctrl.Result{}, fmt.Errorf("err: %s:%s", errorText, err.Error())
	}

	namespace := ""
	if network.Spec.Namespace != "" {
		// first Case: create secret within that namespace.
		namespace = network.Spec.Namespace
	} else {
		// third case: create secret within the namespace the resource was defined 'network.Namespace'.
		namespace = network.Namespace
	}
	participantsMap := network.Info.Participating.Components
	participants := make([]string, 0, len(participantsMap))
	for participant := range participantsMap {
		participants = append(participants, participant)
	}
	participants = append(participants, appComponentName(network.Spec.App, anckParticipant))

	for _, component := range participants {
		err = removeNetworkFromComponentSecret(component, network.Name, namespace)
		if err != nil {
			if errors.IsNotFound(err) {
				networkLog.Info(fmt.Sprintf("Secret '%s' not found. Must have been deleted by participant controller: %s", component, err))
			} else {
				errorText := "Error updating component secret"
				networkLog.Info(errorText)
				return ctrl.Result{}, fmt.Errorf("err: %s:%s", errorText, err.Error())
			}
		}
		err = removeNetworkFromDaprSecret(component, network.Name, namespace)
		if err != nil {
			if errors.IsNotFound(err) {
				networkLog.Info(fmt.Sprintf("Secret '%s' not found. Must have been deleted by participant controller: %s", component, err))
			} else {
				errorText := "Error updating dapr secret"
				networkLog.Info(errorText)
				return ctrl.Result{}, fmt.Errorf("err: %s:%s", errorText, err.Error())
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *NetworksReconciler) networkNodeState(network *networkv1alpha1.Network, node string) nodeNetworkStateType {
	if len(network.Info.Participating.PodsCreating[node]) >= 1 && len(network.Info.Participating.Pods[node]) == 0 {
		return new
	}
	if len(network.Info.Participating.PodsCreating[node]) >= 1 && len(network.Info.Participating.Pods[node]) >= 1 {
		return stillPartitipatingCreate
	}
	if common.SliceEqual(network.Info.Participating.PodsTerminating[node], network.Info.Participating.Pods[node]) && len(network.Info.Participating.Pods[node]) >= 1 {
		return stillPartitipatingDelete
	}
	if len(network.Info.Participating.PodsTerminating[node]) >= 1 {
		return deleted
	}
	return invalid
}

func (r *NetworksReconciler) reconcileJetstreams(ctx context.Context, network *networkv1alpha1.Network) (ctrl.Result, error) {
	anckCreds, err := readCredentialsFromSecret(appComponentName(network.Spec.App, anckParticipant), network.Name, network.Spec.Namespace)
	if err != nil {
		errorText := "Error reading credentials from secret"
		networkLog.Error(err, errorText)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	js, err := jetstreams.NewJetstreamController(anckCreds)
	if err != nil {
		errorText := "Error creating jetstream manager"
		networkLog.Info(errorText)
		return ctrl.Result{}, fmt.Errorf("err: %s:%s", errorText, err.Error())
	}
	defer js.Cleanup()

	networkCopy := network.DeepCopy()
	errorsCreatingJetstreams := make(map[string]bool)
	domainMessages := jetstreams.NewDomainMessages()
	removedNetworkFinalizers := []string{}
	addedNetworkFinalizers := []string{}
	for node := range network.Info.Participating.Nodes {
		// networkInfoModified := false
		// Check if the node is participating in the network yet
		switch r.networkNodeState(network, node) {
		case new:
			nodeparticipantsLog.Info("Adding new pods for network", "node", node, "network", network.Name, "podsCreating", network.Info.Participating.PodsCreating, "pods", network.Info.Participating.Pods, "podsTerminating", network.Info.Participating.PodsTerminating)
			if network.Info.Participating.Nodes[node] == participatingNodeStatePending {
				nodeparticipantsLog.Info("Creating node participation", "node", node, "network", network.Name, "state", network.Info.Participating.Nodes[node])
				retry.Do(func() error {
					for _, stream := range network.Spec.Streams {
						if stream.Location == "node" {
							exists, err := js.Exists(node, stream.Name)
							if err != nil {
								errorText := fmt.Sprintf("Error checking if jetstream exists: {%s: %s, %s: %s, %s: %s}", "domain", node, "stream", stream.Name, "error", err.Error())
								nodeparticipantsLog.Info(errorText)
								errorsCreatingJetstreams[node] = true
								return fmt.Errorf("")
							}
							if !exists {
								err = js.Create(node, network.Name, stream, network.Spec.Subjects)
								if err != nil {
									errorText := fmt.Sprintf("Error creating jetstream: {%s: %s, %s: %s, %s: %s}", "domain", node, "stream", stream.Name, "error", err.Error())
									fmt.Println(errorText)
									domainMessages.Error(node, errorText)
									errorsCreatingJetstreams[node] = true
									return fmt.Errorf("")
								}
								domainMessages.Ok(node, fmt.Sprintf("Successfully created jetstream: {domain: %s, stream: %s}", node, stream.Name))
							}
						}
					}
					return nil
				}, retry.Attempts(5), retry.Delay(time.Second*1))
				fmt.Println(domainMessages.ErrMap)
				fmt.Println(errorsCreatingJetstreams)
				if errorsCreatingJetstreams[node] {
					nodeparticipantsLog.Info("Done creating node participation with errors!!!", "node", node, "network", network.Name, "state", network.Info.Participating.Nodes[node])
				} else {
					nodeparticipantsLog.Info("Done creating node participation", "node", node, "network", network.Name, "state", network.Info.Participating.Nodes[node])
					network.Info.Participating.Nodes[node] = participatingNodeStateActive
					addedNetworkFinalizers = append(addedNetworkFinalizers, node)
					// move pods from PodsCreating to Pods
					for _, pods := range network.Info.Participating.PodsCreating[node] {
						network.Info.Participating.Pods[node] = append(network.Info.Participating.Pods[node], pods)
					}
					network.Info.Participating.Pods[node] = unique.Unique(network.Info.Participating.Pods[node])
					delete(network.Info.Participating.PodsCreating, node) // TODO: delete value from slice. if slice is empty, delete node key from map
					nodeparticipantsLog.Info("Done adding new pods for network", "node", node, "network", network.Name, "podsCreating", network.Info.Participating.PodsCreating, "pods", network.Info.Participating.Pods, "podsTerminating", network.Info.Participating.PodsTerminating)
				}
			}
		case stillPartitipatingCreate:
			// Node is already particpating in the network - create pod case
			nodeparticipantsLog.Info("Adding new nodes to already participating nodes", "node", node, "network", network.Name, "podsCreating", network.Info.Participating.PodsCreating, "pods", network.Info.Participating.Pods, "podsTerminating", network.Info.Participating.PodsTerminating)
			// move pods from PodsCreating to Pods
			for node, pods := range network.Info.Participating.PodsCreating {
				network.Info.Participating.Pods[node] = append(network.Info.Participating.Pods[node], pods...)
				network.Info.Participating.Pods[node] = unique.Unique(network.Info.Participating.Pods[node])
				delete(network.Info.Participating.PodsCreating, node)
			}
			nodeparticipantsLog.Info("Done adding new nodes to already participating nodes", "node", node, "network", network.Name, "podsCreating", network.Info.Participating.PodsCreating, "pods", network.Info.Participating.Pods, "podsTerminating", network.Info.Participating.PodsTerminating)
		case stillPartitipatingDelete:
			// Node isn't participating in the network anymore
			// if !networkInfoModified && sliceEqual(network.Info.Participating.PodsTerminating[node], network.Info.Participating.Pods[node]) && len(network.Info.Participating.Pods[node]) >= 1 {
			nodeparticipantsLog.Info("Removing nodes participating", "node", node, "network", network.Name, "podsCreating", network.Info.Participating.PodsCreating, "pods", network.Info.Participating.Pods, "podsTerminating", network.Info.Participating.PodsTerminating)
			if network.Info.Participating.Nodes[node] == participatingNodeStateTerminating {
				// Stream deletion for domains is handled on the domain level
				// Handle only the logic for the network resource
				delete(network.Info.Participating.Pods, node)
				delete(network.Info.Participating.Nodes, node)
				delete(network.Info.Participating.PodsTerminating, node)
				removedNetworkFinalizers = append(removedNetworkFinalizers, node)
				nodeparticipantsLog.Info("Done removing nodes participation", "node", node, "network", network.Name, "podsCreating", network.Info.Participating.PodsCreating, "pods", network.Info.Participating.Pods, "podsTerminating", network.Info.Participating.PodsTerminating)
			}
		// Node is still particpating in the network - delete pod case
		case deleted:
			nodeparticipantsLog.Info("Removing nodes from still participating nodes", "node", node, "network", network.Name, "podsCreating", network.Info.Participating.PodsCreating, "pods", network.Info.Participating.Pods, "podsTerminating", network.Info.Participating.PodsTerminating)
			// move pods from PodsCreating to Pods
			for _, pod := range network.Info.Participating.PodsTerminating[node] {
				// remove pods from network.Info.Participating.Pods[node]
				for i, podName := range network.Info.Participating.Pods[node] {
					if podName == pod {
						network.Info.Participating.Pods[node] = append(network.Info.Participating.Pods[node][:i], network.Info.Participating.Pods[node][i+1:]...)
						break
					}
				}
				nodeparticipantsLog.Info("Removing finalizer from pod", "pod", pod, "node", node, "network", network.Name, "pods", network.Info.Participating.Pods[node])
				delete(network.Info.Participating.PodsTerminating, node)
			}
			nodeparticipantsLog.Info("Done removing nodes from still participating nodes", "node", node, "network", network.Name, "podsCreating", network.Info.Participating.PodsCreating, "pods", network.Info.Participating.Pods, "podsTerminating", network.Info.Participating.PodsTerminating)
			removedNetworkFinalizers = append(removedNetworkFinalizers, node)
		}
	}

	nodeparticipantsLog.Info("Deleting network finalizer for nodes", "nodes", removedNetworkFinalizers, "network", network.Name, "finalizers", network.ObjectMeta.Finalizers)
	removeNetworkFinalizers(network, removedNetworkFinalizers)
	nodeparticipantsLog.Info("Done deleting network finalizer for nodes", "nodes", removedNetworkFinalizers, "network", network.Name, "finalizers", network.ObjectMeta.Finalizers)
	nodeparticipantsLog.Info("Deleting adding finalizer for nodes", "nodes", addedNetworkFinalizers, "network", network.Name, "finalizers", network.ObjectMeta.Finalizers)
	addNetworkFinalizer(network, addedNetworkFinalizers)
	nodeparticipantsLog.Info("Done adding network finalizer for nodes", "nodes", addedNetworkFinalizers, "network", network.Name, "finalizers", network.ObjectMeta.Finalizers)

	for _, node := range domainMessages.OkMap {
		for _, message := range node {
			nodeparticipantsLog.Info(message)
		}
	}
	for _, node := range domainMessages.ErrMap {
		for _, message := range node {
			nodeparticipantsLog.Info(message)
		}
	}

	err = r.updateInfoAndReturn(ctx, network, networkCopy)
	if err != nil {
		return ctrl.Result{
			Requeue: true,
		}, err
	}

	if len(domainMessages.ErrMap) > 0 {
		if err != nil {
			return ctrl.Result{
				RequeueAfter: 5 * time.Second,
			}, fmt.Errorf("Error handling jestreams")
		}
	}

	return ctrl.Result{
		Requeue: true,
	}, nil
}

func (r *NetworksReconciler) updateInfoAndReturn(ctx context.Context, network *networkv1alpha1.Network, copy *networkv1alpha1.Network) error {
	if !reflect.DeepEqual(copy.Info, network.Info) {
		networkLog.Info("info has changed, updating")
		err := r.Update(ctx, network)
		if err != nil {
			networkLog.Error(err, "failed to update info")
			return err
		}
	}
	return nil
}

// removeNetworkFinalizers removes the finalizers from a network
func removeNetworkFinalizers(network *networkv1alpha1.Network, removeFinalizers []string) *networkv1alpha1.Network {
	finalizers := network.ObjectMeta.Finalizers
	for i, v := range removeFinalizers {
		if slice.ContainsString(finalizers, v) {
			finalizers = append(finalizers[:i], finalizers[i+1:]...)
		}
	}

	network.ObjectMeta.Finalizers = finalizers
	return network
}

// addNetworkFinalizer adds the finalizers from a network
func addNetworkFinalizer(network *networkv1alpha1.Network, finalizers []string) {
	network.ObjectMeta.Finalizers = append(network.ObjectMeta.Finalizers, finalizers...)
	network.ObjectMeta.Finalizers = unique.Unique(network.ObjectMeta.Finalizers)
}
