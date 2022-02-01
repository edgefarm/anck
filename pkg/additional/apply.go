package additional

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyOrUpdate applies or updates the given object.
func ApplyOrUpdate(client client.Client, obj client.Object) error {
	err := client.Create(context.Background(), obj)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
		err = client.Update(context.Background(), obj)
		if err != nil {
			return err
		}
	}
	return nil
}

// ApplyIgnoreExisting applies the given object ignoring if it already exists.
func ApplyIgnoreExisting(client client.Client, obj client.Object) error {
	err := client.Create(context.Background(), obj)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}
