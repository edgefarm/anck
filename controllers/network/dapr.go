package network

import (
	"fmt"

	"github.com/edgefarm/anck/pkg/dapr"
	v1 "k8s.io/api/core/v1"
)

func createOrUpdateComponentDaprSecrets(secret *v1.Secret) error {
	items := make(map[string]string)

	// Update the secret if it exists or create a new one if it doesn't
	for network, cred := range secret.Data {
		daprComponentName := fmt.Sprintf("%s.yaml", network)
		jwt, nkey, err := parseCredsString(string(cred))
		if err != nil {
			return err
		}
		config := dapr.NewDapr(network, dapr.WithCreds(jwt, nkey), dapr.WithNatsURL("nats://nats.nats:4222"))
		str, err := config.ToYaml()
		if err != nil {
			return err
		}
		items[daprComponentName] = str
	}

	daprSecretName := fmt.Sprintf("%s.dapr", secret.Name)
	secretExists, err := existsSecret(daprSecretName, secret.Namespace)
	if err != nil {
		return err
	}
	if secretExists {
		_, err = updateSecret(daprSecretName, secret.Namespace, &items)
	} else {
		_, err = createSecret(daprSecretName, secret.Namespace, &items)
	}
	if err != nil {
		return err
	}

	return nil
}

func removeParticipantFromDaprSecret(component, network, namespace string) error {
	daprSecret := fmt.Sprintf("%s.dapr", component)
	secret, err := readSecret(daprSecret, namespace)
	if err != nil {
		return err
	}

	daprComponentName := fmt.Sprintf("%s.yaml", network)
	if _, ok := secret[daprComponentName]; ok {
		if len(secret) == 1 {
			// delete secret if it's the only network
			return deleteSecret(daprSecret, namespace)
		}
		delete(secret, daprComponentName)
	} else {
		// component is not participating in this network
		return nil
	}

	_, err = updateSecret(daprSecret, namespace, &secret)
	if err != nil {
		return err
	}

	return nil
}
