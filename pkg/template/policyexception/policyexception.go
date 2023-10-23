package policyexception

import (
	"github.com/giantswarm/microerror"

	polexdraftv1alpha1 "github.com/giantswarm/exception-recommender/api/v1alpha1"
	polexv1alpha1 "github.com/giantswarm/kyverno-policy-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	Name string
}

func NewPolicyExceptionCR(config Config, client runtimeclient.Client) (*polexv1alpha1.PolicyException, error) {

	polexdraft := &polexdraftv1alpha1.PolicyExceptionDraft{}
	err := client.Get(ctx, runtimeclient.ObjectKey{Namespace: "policy-exception", Name: config.Name}, polexdraft)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	polexCR := &polexv1alpha1.PolicyException{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PolicyException",
			APIVersion: "policy.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Name,
		},
		Spec: polexdraft.Spec,
	}

	return polexCR, nil
}
