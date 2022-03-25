package resources

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a namespace in the cluster
func CreateNamespace(namespace string) error {
	clientset, err := clientset()
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
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
		resourcesLog.Error(err, "error creating namespace")
		return err
	}
	return nil
}
