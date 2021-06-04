package resources

import (
	"context"
	"fmt"
	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	publicAPIKey     = "publicApiKey"
	privateAPISecret = "privateApiSecret"
)

// ConnectionAPIKeys encapsulates crunchybridge connectivity information that is necessary to generate token for performing API requests
type ConnectionAPIKeys struct {
	PublicKey     string
	PrivateSecret string
}

func ReadAPIKeysFromSecret(kubeClient client.Client, ctx context.Context, inventory *dbaasredhatcomv1alpha1.CrunchyBridgeInventory) (ConnectionAPIKeys, error) {
	secret := &corev1.Secret{}
	selector := client.ObjectKey{
		Namespace: inventory.Spec.CredentialsRef.Namespace,
		Name:      inventory.Spec.CredentialsRef.Name,
	}

	if err := kubeClient.Get(ctx, selector, secret); err != nil {
		return ConnectionAPIKeys{}, err
	}
	secretData := make(map[string]string)
	for k, v := range secret.Data {
		secretData[k] = string(v)
	}

	if err := validateAPIKeysSecret(inventory, secretData); err != nil {
		return ConnectionAPIKeys{}, err
	}

	return ConnectionAPIKeys{
		PublicKey:     secretData["publicApiKey"],
		PrivateSecret: secretData["privateApiSecret"],
	}, nil
}

func validateAPIKeysSecret(inventory *dbaasredhatcomv1alpha1.CrunchyBridgeInventory, secretData map[string]string) error {
	var missingFields []string
	requiredKeys := []string{publicAPIKey, privateAPISecret}

	for _, key := range requiredKeys {
		if _, ok := secretData[key]; !ok {
			missingFields = append(missingFields, key)
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("the following fields are missing in the Secret %v: %v", inventory.Spec.CredentialsRef.Name, missingFields)
	}
	return nil
}
