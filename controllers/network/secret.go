package network

import (
	"context"
	"fmt"
	"strings"

	anckcredentials "github.com/edgefarm/anck-credentials/pkg/apis/config/v1alpha1"
	"github.com/edgefarm/anck/pkg/nats"
	resources "github.com/edgefarm/anck/pkg/resources"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

var secretLog = ctrl.Log.WithName("secret")

// createOrUpdateComponentSecrets creates or updates the secrets for the component.
// The component secret contains the credentials for each network the component is participant of
func createOrUpdateComponentSecrets(component string, namespace string, networkCreds *anckcredentials.DesiredStateResponse) (*v1.Secret, error) {
	secretExists := true
	secret, err := resources.ReadSecret(component, namespace)
	if apierrors.IsNotFound(err) {
		secretLog.Info(fmt.Sprintf("secret not found. Creating new secret: %s", err))
		secretExists = false
	} else if err != nil {
		return nil, err
	}

	accountPublicKey := networkCreds.Network.AccoutPublicKey
	network := networkCreds.Network.Name

	for _, active := range networkCreds.Creds {
		secret[fmt.Sprintf("%s.creds", network)] = active.Creds
	}
	natsServer, err := nats.GetNatsServerInfos()
	if err != nil {
		return nil, err
	}

	// currently fixed to sysaccounts credentials
	secret["nats-sidecar.creds"] = natsServer.SysAccount.SysCreds
	secret[fmt.Sprintf("%s.pub", network)] = accountPublicKey

	// Delete networks from the secret if the component is no longer participating in
	for _, deleted := range networkCreds.DeletedParticipants {
		_, app, participantComponent, err := splitNetworkParticipant(deleted)
		if err != nil {
			return nil, err
		}
		if component == fmt.Sprintf("%s.%s", app, participantComponent) {
			delete(secret, network)
			delete(secret, fmt.Sprintf("%s.pub", network))
		}
	}

	// Update the secret if it exists or create a new one if it doesn't
	secretv1 := &v1.Secret{}
	if secretExists {
		secretv1, err = resources.UpdateSecret(component, namespace, &secret)
	} else {
		secretv1, err = resources.CreateSecret(component, namespace, &secret)
	}
	if err != nil {
		return nil, err
	}

	return secretv1, nil
}

func readCredentialsFromSecret(component string, network string, namespace string) (string, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		secretLog.Error(err, "error getting cluster config")
		return "", err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		secretLog.Error(err, "error getting client for cluster")
		return "", err
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), component, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		secretLog.Error(err, "error deleting secret")
		return "", err
	}

	networkCredsfile := fmt.Sprintf("%s.creds", network)
	if _, ok := secret.Data[networkCredsfile]; ok {
		return string(secret.Data[networkCredsfile]), nil
	}
	return "", fmt.Errorf("network %s not found in secret %s", network, component)
}

func removeNetworkFromComponentSecret(component, network, namespace string) error {
	secretLog.Info(fmt.Sprintf("Removing network '%s' from component secret '%s' in namespace '%s'", network, component, namespace))
	secret, err := resources.ReadSecret(component, namespace)
	if err != nil {
		return err
	}

	// ignore default creds in secret to get the real length
	filteredSecret := make(map[string]string)
	for _, ignored := range ignoredSecretEntries {
		if strings.Contains(network, ignored) {
			continue
		}
		filteredSecret[network] = string(secret[network])
	}

	if _, ok := secret[credsfileFromNetwork(network)]; ok {
		if len(filteredSecret) == 1 {
			// delete secret if it's the only network
			return resources.DeleteSecret(component, namespace)
		}
		delete(secret, credsfileFromNetwork(network))
		delete(secret, pubkeyFileFromNetwork(network))
	} else {
		// component is not participating in this network
		return nil
	}

	_, err = resources.UpdateSecret(component, namespace, &secret)
	if err != nil {
		return err
	}

	return nil
}

func credsfileFromNetwork(network string) string {
	return fmt.Sprintf("%s.creds", network)
}

func pubkeyFileFromNetwork(network string) string {
	return fmt.Sprintf("%s.pub", network)
}
