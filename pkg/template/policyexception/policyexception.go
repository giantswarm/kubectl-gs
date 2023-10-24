package policyexception

import (
	"context"

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
	ctx := context.Background()

	polexdraft := &polexdraftv1alpha1.PolicyExceptionDraft{}
	err := client.Get(ctx, runtimeclient.ObjectKey{Namespace: "policy-exceptions", Name: config.Name}, polexdraft)
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
		Spec: policyExceptionSpecDeepCopy(polexdraft.Spec),
	}

	return polexCR, nil
}

func policyExceptionSpecDeepCopy(draft polexdraftv1alpha1.PolicyExceptionDraftSpec) polexv1alpha1.PolicyExceptionSpec {
	polex := polexv1alpha1.PolicyExceptionSpec{
		Policies: draft.Policies,
		Targets:  targetsDeepCopy(draft.Targets),
	}
	return polex
}

func targetsDeepCopy(in []polexdraftv1alpha1.Target) []polexv1alpha1.Target {
	var targets []polexv1alpha1.Target

	for _, v := range in {
		out := polexv1alpha1.Target{
			Namespaces: make([]string, len(v.Namespaces)),
			Names:      make([]string, len(v.Names)),
			Kind:       v.Kind,
		}

		if v.Namespaces != nil {
			copy(out.Namespaces, v.Namespaces)
		}
		if v.Names != nil {
			copy(out.Names, v.Names)
		}
		targets = append(targets, out)
	}
	return targets
}
