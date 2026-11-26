// Package managementcluster tells which management cluster the kubeconfig
// context points at.
package managementcluster

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MetadataConfigMap is the ConfigMap, as namespace/name, whose NAME key holds
// the management cluster's name. Flux writes it when the management cluster
// is set up.
const MetadataConfigMap = metadataNamespace + "/" + metadataName

const (
	metadataNamespace = "flux-giantswarm"
	metadataName      = "management-cluster-metadata"
	nameKey           = "NAME"
)

// Name returns the management cluster's name from MetadataConfigMap. It
// returns an empty name when the ConfigMap is missing, the user may not read
// it, or it holds no name, so the caller can find the management cluster
// another way.
func Name(ctx context.Context, ctrlClient client.Client) (string, error) {
	cm := &corev1.ConfigMap{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: metadataNamespace, Name: metadataName}, cm)
	if apierrors.IsNotFound(err) || apierrors.IsForbidden(err) {
		return "", nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	return cm.Data[nameKey], nil
}
