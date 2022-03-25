package resources

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

var resourcesLog = ctrl.Log.WithName("resources")

func clientset() (*kubernetes.Clientset, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		resourcesLog.Error(err, "error getting cluster config")
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	return clientset, nil
}
