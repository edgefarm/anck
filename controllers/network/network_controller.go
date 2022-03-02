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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
)

var (
	setupLog = ctrl.Log.WithName("anck-controller")
)

const (
	anckcredentialsServiceName       = "anck-credentials"
	anckcredentialsServicePort       = 6000
	anckcredentialsNamespace         = "anck"
	timeoutSeconds                   = 10
	edgefarmNetworkAccountNameSecret = "anck-credentials-natsUserData"
	edgefarmSecretUsernameKey        = "username"
	edgefarmSecretPasswordKey        = "password"
	edgefarmSecretNKeyKey            = "nkey"
	edgefarmSecretJWTKey             = "jwt"
	edgefarmSecretCredsfileKey       = "credsfile"
	anckParticipant                  = "anck-this-name-shall-never-be-used"
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

	grpcContext, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
	defer cancel()

	cc, err := grpc.Dial(fmt.Sprintf("%s.%s.svc.cluster.local:%d", anckcredentialsServiceName, anckcredentialsNamespace, anckcredentialsServicePort), grpc.WithInsecure())
	if err != nil {
		errorText := "Error connecting to anckcredentials"
		setupLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}
	defer cc.Close()

	accountName := ""
	if network.Spec.Accountname != "" {
		accountName = network.Spec.Accountname
	} else if network.Spec.Namespace != "" {
		accountName = network.Spec.Namespace
	} else {
		accountName = network.Namespace
	}

	client := anckcredentials.NewConfigServiceClient(cc)
	setupLog.Info(fmt.Sprintf("Requesting credentials for account name '%s'", accountName))

	participants := network.Spec.Participants
	participants = append(participants, anckParticipant)
	resp, err := client.DesiredState(grpcContext, &anckcredentials.DesiredStateRequest{
		AccountName: accountName,
		Username:    participants,
	})
	if err != nil {
		errorText := "Error setting desired state"
		setupLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}

	namespace := ""
	if network.Spec.Namespace != "" {
		// first Case: create secret within that namespace.
		namespace = network.Spec.Namespace
	} else if network.Spec.Accountname != "" {
		// second Case: create secret within a namespaced with name of accountname. If this namespace does not exist, create it.
		namespace = network.Spec.Accountname
	} else {
		// third case: create secret within the namespace the resource was defined 'network.Namespace'.
		namespace = network.Namespace
	}

	err = createNamespace(namespace)
	if err != nil {
		errorText := "Error creating namespace"
		setupLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}

	err = createOrUpdateSecrets(accountName, namespace, resp)
	if err != nil {
		errorText := "Error creating or updating secret"
		setupLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}

	anckCreds, err := readCredentialsFromSecret(fmt.Sprintf("%s.%s", accountName, anckParticipant), namespace)
	if err != nil {
		errorText := "Error reading credentials from secret"
		setupLog.Error(err, errorText)
	}

	jetstream, err := NewJetstream(anckCreds.Creds)
	if err != nil {
		errorText := "Error creating jetstream"
		setupLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}

	for _, stream := range network.Spec.Streams {
		err = jetstream.Create(stream)
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
				setupLog.Info("Delete handler for network called")

				streamNames := func(streams []networkv1alpha1.StreamSpec) []string {
					var names []string
					for _, stream := range streams {
						names = append(names, stream.Name)
					}
					return names
				}(e.Object.(*networkv1alpha1.Network).Spec.Streams)

				anckCreds, err := readCredentialsFromSecret(fmt.Sprintf("%s.%s", e.Object.(*networkv1alpha1.Network).Spec.Accountname, anckParticipant), e.Object.(*networkv1alpha1.Network).Spec.Namespace)
				if err != nil {
					errorText := "Error reading credentials from secret"
					setupLog.Error(err, errorText)
				}

				js, err := NewJetstream(anckCreds.Creds)
				if err != nil {
					setupLog.Error(err, "Error creating jetstream")
					return false
				}

				if len(streamNames) > 0 {
					setupLog.Info("Delete configured jetstreams:")
					for _, streamName := range streamNames {
						setupLog.Info(fmt.Sprintf("\t- %s\n", streamName))
					}
					err := js.Delete(streamNames)
					if err != nil {
						setupLog.Info("Error deleting jetstreams")
						return false
					}
				}

				js.Cleanup()

				grpcContext, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
				defer cancel()

				cc, err := grpc.Dial(fmt.Sprintf("%s.%s.svc.cluster.local:%d", anckcredentialsServiceName, anckcredentialsNamespace, anckcredentialsServicePort), grpc.WithInsecure())
				if err != nil {
					errorText := "Error connecting to anckcredentials"
					setupLog.Info(errorText)
					return false
				}
				defer cc.Close()
				client := anckcredentials.NewConfigServiceClient(cc)

				accountName := ""
				if e.Object.(*networkv1alpha1.Network).Spec.Accountname != "" {
					accountName = e.Object.(*networkv1alpha1.Network).Spec.Accountname
				} else if e.Object.(*networkv1alpha1.Network).Spec.Namespace != "" {
					accountName = e.Object.(*networkv1alpha1.Network).Spec.Namespace
				} else {
					accountName = e.Object.(*networkv1alpha1.Network).Namespace
				}

				setupLog.Info(fmt.Sprintf("Deleting network account %s", e.Object.(*networkv1alpha1.Network).Spec.Accountname))
				_, err = client.DeleteAccount(grpcContext, &anckcredentials.DeleteAccountRequest{
					AccountName: accountName,
				})
				if err != nil {
					errorText := fmt.Sprintf("Cannot deleted network account %s", accountName)
					fmt.Println(errorText)
					return false
				}

				namespace := ""
				if e.Object.(*networkv1alpha1.Network).Spec.Namespace != "" {
					// first Case: create secret within that namespace.
					namespace = e.Object.(*networkv1alpha1.Network).Spec.Namespace
				} else if e.Object.(*networkv1alpha1.Network).Spec.Accountname != "" {
					// second Case: create secret within a namespaced with name of accountname. If this namespace does not exist, create it.
					namespace = e.Object.(*networkv1alpha1.Network).Spec.Accountname
				} else {
					// third case: create secret within the namespace the resource was defined 'network.Namespace'.
					namespace = e.Object.(*networkv1alpha1.Network).Namespace
				}
				for _, user := range e.Object.(*networkv1alpha1.Network).Spec.Participants {
					secretName := fmt.Sprintf("%s.%s", accountName, user)
					setupLog.Info(fmt.Sprintf("Deleting network secret %s from namespace %s", secretName, namespace))
					err = deleteSecret(secretName, namespace)
					if err != nil {
						errorText := fmt.Sprintf("Cannot deleted network secret %s from namespace %s", secretName, namespace)
						setupLog.Info(errorText)
						continue
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
