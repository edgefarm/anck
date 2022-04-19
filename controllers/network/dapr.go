package network

import (
	"fmt"
	"strings"

	"github.com/edgefarm/anck/pkg/dapr"
	jetstreams "github.com/edgefarm/anck/pkg/jetstreams"
	"github.com/edgefarm/anck/pkg/nats"
	resources "github.com/edgefarm/anck/pkg/resources"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	daprLog = ctrl.Log.WithName("dapr")
)

func createOrUpdateComponentDaprSecrets(secret *v1.Secret, participantType string) error {
	items := make(map[string]string)

	// Update the secret if it exists or create a new one if it doesn't
	for network, cred := range secret.Data {
		skip := false
		// skip ignored files
		for _, ignored := range ignoredSecretEntries {
			if strings.Contains(network, ignored) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		rawNetworkName := strings.TrimSuffix(network, ".creds")
		daprComponentName := fmt.Sprintf("%s.yaml", rawNetworkName)
		jwt, nkey, err := jetstreams.ParseCredsString(string(cred))
		if err != nil {
			return err
		}
		opts := []dapr.Option{}
		opts = append(opts, dapr.WithCreds(jwt, nkey))
		if participantType == "edge" {
			opts = append(opts, dapr.WithNatsURL("nats://leaf-nats.nats:4222"))
		} else if participantType == "cloud" {
			natsServer, err := nats.GetNatsServerInfos()
			if err != nil {
				return err
			}
			opts = append(opts, dapr.WithNatsURL(natsServer.Addresses.NatsAddress))
		} else {
			// skip unknown participant type
			continue
		}

		config := dapr.NewDapr(rawNetworkName, opts...)
		str, err := config.ToYaml()
		if err != nil {
			return err
		}
		items[daprComponentName] = str
	}
	// only create/update secret if there are items to add. This is to avoid creating empty secrets if only ignored participant types are present
	if len(items) > 0 {
		daprSecretName := fmt.Sprintf("%s.dapr", secret.Name)
		secretExists, err := resources.ExistsSecret(daprSecretName, secret.Namespace)
		if err != nil {
			return err
		}
		if secretExists {
			_, err = resources.UpdateSecret(daprSecretName, secret.Namespace, &items)
		} else {
			_, err = resources.CreateSecret(daprSecretName, secret.Namespace, &items)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func removeNetworkFromDaprSecret(component, network, namespace string) error {
	rawNetworkName := strings.TrimSuffix(network, ".creds")
	daprLog.Info(fmt.Sprintf("Removing network '%s' from dapr secret '%s' in namespace '%s'", rawNetworkName, component, namespace))
	daprSecret := fmt.Sprintf("%s.dapr", component)
	secret, err := resources.ReadSecret(daprSecret, namespace)
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

	daprComponentName := fmt.Sprintf("%s.yaml", rawNetworkName)
	if _, ok := secret[daprComponentName]; ok {
		if len(filteredSecret) == 1 {
			// delete secret if it's the only network
			return resources.DeleteSecret(daprSecret, namespace)
		}
		delete(secret, daprComponentName)
	} else {
		// component is not participating in this network
		return nil
	}

	_, err = resources.UpdateSecret(daprSecret, namespace, &secret)
	if err != nil {
		return err
	}
	return nil
}
