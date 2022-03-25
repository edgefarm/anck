package resources

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// DeleteSecret deletes a secret from the cluster
func DeleteSecret(name string, namespace string) error {
	clientset, err := clientset()
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
		return err
	}
	err = clientset.CoreV1().Secrets(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		resourcesLog.Error(err, "error deleting secret")
		return err
	}
	return nil
}

// ReadSecret reads a secret from the cluster
func ReadSecret(name string, namespace string) (map[string]string, error) {
	clientset, err := clientset()
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	creds := make(map[string]string)
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		resourcesLog.Info(fmt.Sprintf("error getting secret: %s", err))
		// must return a not nil map, so it can be used by the caller
		return creds, err
	}

	for key, value := range secret.Data {
		creds[key] = string(value)
	}

	return creds, nil
}

// ExistsSecret checks if a secret exists in the cluster
func ExistsSecret(name string, namespace string) (bool, error) {
	clientset, err := clientset()
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
		return false, err
	}
	secretList, err := clientset.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		resourcesLog.Info(fmt.Sprintf("error listing secret: %s", err))
		return false, err
	}

	for _, s := range secretList.Items {
		if s.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// UpdateSecret updates a secret in the cluster
func UpdateSecret(name string, namespace string, data *map[string]string) (*v1.Secret, error) {
	clientset, err := clientset()
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		resourcesLog.Error(err, "error getting secret")
		return nil, err
	}

	newData := make(map[string][]byte)
	for network, cred := range *data {
		newData[network] = []byte(cred)
	}
	secret.Data = newData

	secret, err = clientset.CoreV1().Secrets(namespace).Update(context.Background(), secret, metav1.UpdateOptions{})
	if err != nil {
		resourcesLog.Error(err, "error updating secret")
		return nil, err
	}

	return secret, nil
}

// CreateSecret creates a secret in the cluster
func CreateSecret(name string, namespace string, data *map[string]string) (*v1.Secret, error) {
	clientset, err := clientset()
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}

	for key, value := range *data {
		secret.Data[key] = []byte(value)
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
		resourcesLog.Error(err, "error creating secret")
		return nil, err
	}

	return writtenSecret, nil
}
