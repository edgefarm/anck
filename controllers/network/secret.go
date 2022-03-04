package network

import (
	"context"
	"fmt"
	"strings"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// splitNetworkParticipant extracts the component and network name from the network participant name.
// The format of networkParticipant is <network>.<component>
func splitNetworkParticipant(networkParticipant string) (string, string, error) {
	parts := strings.Split(networkParticipant, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid network participant name: %s", networkParticipant)
	}
	return parts[0], parts[1], nil
}

func createOrUpdateComponentSecrets(component string, namespace string, networkCreds *anckcredentials.DesiredStateResponse) error {
	secretExists := true
	secret, err := readComponentSecret(component, namespace)
	if apierrors.IsNotFound(err) {
		setupLog.Info(fmt.Sprintf("secret not found. Creating new secret: %s", err))
		secretExists = false
	} else if err != nil {
		return err
	}

	// Update the secret if it exists or create a new one if it doesn't
	for _, active := range networkCreds.Creds {
		network, _, err := splitNetworkParticipant(active.NetworkParticipant)
		if err != nil {
			return err
		}
		secret[network] = active.Creds
	}

	// Delete networks from the secret if the component is no longer participating in
	for _, deleted := range networkCreds.DeletedParticipants {
		network, participantComponent, err := splitNetworkParticipant(deleted)
		if err != nil {
			return err
		}
		if component == participantComponent {
			delete(secret, network)
		}
	}

	if secretExists {
		_, err = updateComponentSecret(component, namespace, &secret)
	} else {
		_, err = createComponentSecret(component, namespace, &secret)
	}
	if err != nil {
		return err
	}

	return nil
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
	secret, err := readComponentSecret(component, namespace)
	if err != nil {
		return err
	}

	if _, ok := secret[network]; ok {
		delete(secret, network)
	} else {
		// component is not participating in this network
		return nil
	}

	_, err = updateComponentSecret(component, namespace, &secret)
	if err != nil {
		return err
	}

	return nil
}
