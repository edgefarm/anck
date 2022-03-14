package network

import (
	"context"
	"fmt"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func createOrUpdateComponentSecrets(component string, namespace string, networkCreds *anckcredentials.DesiredStateResponse) (*v1.Secret, error) {
	secretExists := true
	secret, err := readSecret(component, namespace)
	if apierrors.IsNotFound(err) {
		setupLog.Info(fmt.Sprintf("secret not found. Creating new secret: %s", err))
		secretExists = false
	} else if err != nil {
		return nil, err
	}

	// Update the secret if it exists or create a new one if it doesn't
	for _, active := range networkCreds.Creds {
		network, _, err := splitNetworkParticipant(active.NetworkParticipant)
		if err != nil {
			return nil, err
		}
		secret[network] = active.Creds
	}

	// Delete networks from the secret if the component is no longer participating in
	for _, deleted := range networkCreds.DeletedParticipants {
		network, participantComponent, err := splitNetworkParticipant(deleted)
		if err != nil {
			return nil, err
		}
		if component == participantComponent {
			delete(secret, network)
		}
	}

	secretv1 := &v1.Secret{}
	if secretExists {
		secretv1, err = updateSecret(component, namespace, &secret)
	} else {
		secretv1, err = createSecret(component, namespace, &secret)
	}
	if err != nil {
		return nil, err
	}

	return secretv1, nil
}

func readCredentialsFromSecret(component string, network string, namespace string) (string, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "error getting cluster config")
		return "", err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		setupLog.Error(err, "error getting client for cluster")
		return "", err
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), component, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		setupLog.Error(err, "error deleting secret")
		return "", err
	}

	if val, ok := secret.Data[network]; ok {
		return string(val), nil
	}
	return "", fmt.Errorf("network %s not found in secret %s", network, component)
}

func removeParticipantFromComponentSecret(component, network, namespace string) error {
	secret, err := readSecret(component, namespace)
	if err != nil {
		return err
	}

	if _, ok := secret[network]; ok {
		if len(secret) == 1 {
			// delete secret if it's the only network
			return deleteSecret(component, namespace)
		}
		delete(secret, network)
	} else {
		// component is not participating in this network
		return nil
	}

	_, err = updateSecret(component, namespace, &secret)
	if err != nil {
		return err
	}

	return nil
}
