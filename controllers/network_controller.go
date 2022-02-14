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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	networkv1alpha1 "github.com/edgefarm/anck/api/v1alpha1"

	credsmanager "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
)

var (
	setupLog = ctrl.Log.WithName("network-controller")
)

const (
	credsmanagerServiceName          = "credsmanager"
	credsmanagerServicePort          = 6000
	credsmanagerNamespace            = "edgefarm-network"
	timeoutSeconds                   = 10
	natsCredsSecretName              = "nats-credentials"
	edgefarmNetworkAccountNameSecret = "anck-credentials-natsUserData"
)

// NetworkReconciler reconciles a Network object
type NetworkReconciler struct {
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
func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	cc, err := grpc.Dial(fmt.Sprintf("%s.%s.svc.cluster.local:%d", credsmanagerServiceName, credsmanagerNamespace, credsmanagerServicePort), grpc.WithInsecure())
	if err != nil {
		errorText := "Error connecting to credsmanager"
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

	client := credsmanager.NewConfigServiceClient(cc)
	fmt.Printf("Requesting credentials for account name '%s'\n", accountName)
	resp, err := client.DesiredState(grpcContext, &credsmanager.DesiredStateRequest{
		AccountName: accountName,
		Username:    network.Spec.Usernames,
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

	// fmt.Printf("Create or update secret '%s' in namespace '%s'\n", natsCredsSecretName, namespace)
	err = createOrUpdateSecrets(accountName, namespace, resp)
	if err != nil {
		errorText := "Error creating or updating secret"
		setupLog.Error(err, errorText)
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, fmt.Errorf("%s", errorText)
	}

	return ctrl.Result{
		Requeue:      false,
		RequeueAfter: 0,
	}, nil
}

func createNamespace(namespace string) error {
	c, err := rest.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "error getting cluster config")
		return err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return err
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		setupLog.Error(err, "error creating namespace")
		return err
	}
	return nil
}

func createOrUpdateSecrets(accountName string, namespace string, creds *credsmanager.DesiredStateResponse) error {
	c, err := rest.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "error getting cluster config")
		return err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return err
	}

	for _, userCred := range creds.Creds {
		secretName := userCred.UserAccountName
		fmt.Printf("Create or update secret '%s' in namespace '%s'\n", secretName, namespace)

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
				Labels:    map[string]string{"account": accountName},
			},
		}

		secret.Data = make(map[string][]byte)
		j, err := json.Marshal(userCred)
		if err != nil {
			setupLog.Error(err, "error marshalling json")
			return err
		}
		secret.Data[edgefarmNetworkAccountNameSecret] = j

		_, err = clientset.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				fmt.Printf("Secret '%s' already exists in namespace '%s'. Updating.\n", secretName, namespace)
				_, err = clientset.CoreV1().Secrets(namespace).Update(context.Background(), secret, metav1.UpdateOptions{})
				if err != nil {
					setupLog.Error(err, "error updating secret")
				}
				continue
			}

			setupLog.Error(err, "error creating secret")
			continue
		}
	}

	for _, deletedUserAccountName := range creds.DeletedUserAccountNames {
		fmt.Printf("Delete secret '%s' in namespace '%s'\n", deletedUserAccountName, namespace)

		err = clientset.CoreV1().Secrets(namespace).Delete(context.Background(), deletedUserAccountName, metav1.DeleteOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			setupLog.Error(err, "error deleting secret")
			continue
		}
	}
	return nil
}

func deleteSecret(name string, namespace string) error {
	c, err := rest.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "error getting cluster config")
		return err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return err
	}
	err = clientset.CoreV1().Secrets(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		setupLog.Error(err, "error deleting secret")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.Network{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				fmt.Println("Delete handler for network called")
				grpcContext, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
				defer cancel()

				cc, err := grpc.Dial(fmt.Sprintf("%s.%s.svc.cluster.local:%d", credsmanagerServiceName, credsmanagerNamespace, credsmanagerServicePort), grpc.WithInsecure())
				if err != nil {
					errorText := "Error connecting to credsmanager"
					fmt.Println(errorText)
					return false
				}
				defer cc.Close()
				client := credsmanager.NewConfigServiceClient(cc)

				accountName := ""
				if e.Object.(*networkv1alpha1.Network).Spec.Accountname != "" {
					accountName = e.Object.(*networkv1alpha1.Network).Spec.Accountname
				} else if e.Object.(*networkv1alpha1.Network).Spec.Namespace != "" {
					accountName = e.Object.(*networkv1alpha1.Network).Spec.Namespace
				} else {
					accountName = e.Object.(*networkv1alpha1.Network).Namespace
				}

				fmt.Printf("Deleting network account %s\n", e.Object.(*networkv1alpha1.Network).Spec.Accountname)
				_, err = client.DeleteAccount(grpcContext, &credsmanager.DeleteAccountRequest{
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
				for _, user := range e.Object.(*networkv1alpha1.Network).Spec.Usernames {
					secretName := fmt.Sprintf("%s.%s", accountName, user)
					fmt.Printf("Deleting network secret %s from namespace %s\n", secretName, namespace)
					err = deleteSecret(secretName, namespace)
					if err != nil {
						errorText := fmt.Sprintf("Cannot deleted network secret %s from namespace %s", secretName, namespace)
						fmt.Println(errorText)
						continue
					}
				}
				return false
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return true
			},
		}).
		Complete(r)
}
