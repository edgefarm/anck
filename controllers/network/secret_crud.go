package network

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

// Currently unused. Needed later when deleting complete components
func deleteComponentSecret(name string, namespace string) error {
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

func readComponentSecret(component string, namespace string) (map[string]string, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "error getting cluster config")
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	creds := make(map[string]string)
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), component, metav1.GetOptions{})
	if err != nil {
		setupLog.Info(fmt.Sprintf("error getting secret: %s", err))
		return creds, err
	}

	for network, networkCred := range secret.Data {
		creds[network] = string(networkCred)
	}

	return creds, nil
}

func updateComponentSecret(component string, namespace string, creds *map[string]string) (*v1.Secret, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "error getting cluster config")
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), component, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		setupLog.Error(err, "error getting secret")
		return nil, err
	}

	newData := make(map[string][]byte)
	for network, cred := range *creds {
		newData[network] = []byte(cred)
	}
	secret.Data = newData

	secret, err = clientset.CoreV1().Secrets(namespace).Update(context.Background(), secret, metav1.UpdateOptions{})
	if err != nil {
		setupLog.Error(err, "error updating secret")
		return nil, err
	}

	return secret, nil
}

func createComponentSecret(component string, namespace string, creds *map[string]string) (*v1.Secret, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "error getting cluster config")
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      component,
			Namespace: namespace,
			Labels: map[string]string{
				"component": component,
			},
		},
		Type: v1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}

	for network, cred := range *creds {
		secret.Data[network] = []byte(cred)
	}
	writtenSecret := &v1.Secret{}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		writtenSecret, err = clientset.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		setupLog.Error(err, "error creating secret")
		return nil, err
	}

	return writtenSecret, nil
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
