package network

import (
	"context"
	"encoding/json"
	"fmt"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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

func readCredentialsFromSecret(username string, namespace string) (*anckcredentials.Credentials, error) {
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
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), username, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		setupLog.Error(err, "error deleting secret")
		return nil, err
	}

	creds := &anckcredentials.Credentials{}
	err = json.Unmarshal(secret.Data[edgefarmNetworkAccountNameSecret], creds)
	if err != nil {
		setupLog.Error(err, "error unmarshalling json")
		return nil, err
	}

	return creds, nil
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

func createOrUpdateSecrets(networkName string, namespace string, creds *anckcredentials.DesiredStateResponse) error {
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
		setupLog.Info(fmt.Sprintf("Create or update secret '%s' in namespace '%s'", secretName, namespace))

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
				Labels: map[string]string{
					"network":   networkName,
					"component": userCred.Username,
				},
			},
		}

		secret.Data = make(map[string][]byte)
		j, err := json.Marshal(userCred)
		if err != nil {
			setupLog.Error(err, "error marshalling json")
			return err
		}
		secret.Data[edgefarmNetworkAccountNameSecret] = j
		secret.Data[edgefarmSecretUsernameKey] = []byte(userCred.Username)
		secret.Data[edgefarmSecretPasswordKey] = []byte(userCred.Password)
		jwt, nkey, err := parseCredsString(userCred.Creds)
		if err != nil {
			setupLog.Error(err, "error parsing creds string")
			return err
		}
		secret.Data[edgefarmSecretNKeyKey] = []byte(nkey)
		secret.Data[edgefarmSecretJWTKey] = []byte(jwt)
		secret.Data[edgefarmSecretCredsfileKey] = []byte(userCred.Creds)

		_, err = clientset.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				setupLog.Info(fmt.Sprintf("Secret '%s' already exists in namespace '%s'. Updating.", secretName, namespace))
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

	for _, secret := range creds.DeletedUserAccountNames {
		setupLog.Info(fmt.Sprintf("Delete secret '%s' in namespace '%s'", secret, namespace))

		err = clientset.CoreV1().Secrets(namespace).Delete(context.Background(), secret, metav1.DeleteOptions{})
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
