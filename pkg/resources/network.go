package resources

import (
	"context"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	"github.com/edgefarm/anck/pkg/client/networkclientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

var networkLog = ctrl.Log.WithName("network")

// GetNetwork returns a network
func GetNetwork(name string, namespace string) (*networkv1alpha1.Network, error) {
	clientset, err := SetupNetworkClientset()
	if err != nil {
		return nil, err
	}
	network, err := clientset.NetworkV1alpha1().Networks(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return network, nil
}

// UpdateNetwork updates a network
func UpdateNetwork(network *networkv1alpha1.Network, namespace string) (*networkv1alpha1.Network, error) {
	clientset, err := SetupNetworkClientset()
	if err != nil {
		return nil, err
	}
	network, err = clientset.NetworkV1alpha1().Networks(namespace).Update(context.Background(), network, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return network, nil
}

// // AddNetworkFinalizer adds the finalizers from a network
// func AddNetworkFinalizer(name string, namespace string, finalizers []string) error {
// 	clientset, err := SetupNetworkClientset()
// 	if err != nil {
// 		networkLog.Error(err, "error getting client for cluster")
// 		return err
// 	}
// 	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
// 		network, err := clientset.NetworkV1alpha1().Networks(namespace).Get(context.Background(), name, metav1.GetOptions{})
// 		if err != nil {
// 			networkLog.Info(fmt.Sprintf("error listing networks: %s", err))
// 			return err
// 		}

// 		network.ObjectMeta.Finalizers = append(network.ObjectMeta.Finalizers, finalizers...)
// 		network.ObjectMeta.Finalizers = unique.Unique(network.ObjectMeta.Finalizers)

// 		_, err = clientset.NetworkV1alpha1().Networks(namespace).Update(context.Background(), network, metav1.UpdateOptions{})
// 		if err != nil {
// 			networkLog.Info(fmt.Sprintf("error listing networks: %s", err))
// 			return err
// 		}
// 		return nil
// 	})
// 	return err
// }

// SetNetworkAccountName sets the account name of a network
func SetNetworkAccountName(network *networkv1alpha1.Network, accountName string) (*networkv1alpha1.Network, error) {
	clientset, err := SetupNetworkClientset()
	if err != nil {
		return nil, err
	}
	network, err = clientset.NetworkV1alpha1().Networks(network.Namespace).Get(context.Background(), network.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	networkLog.Info("Setting network account name", "network", network.Name, "accountName", accountName)
	network.Info.UsedAccount = accountName
	network, err = clientset.NetworkV1alpha1().Networks(network.Namespace).Update(context.Background(), network, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return network, nil
}

// SetupNetworkClientset returns a clientset for the network v1alpha1
func SetupNetworkClientset() (*networkclientset.Clientset, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := networkclientset.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
