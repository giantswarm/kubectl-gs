package policyexception

import (
	polexdraftv1alpha1 "github.com/giantswarm/exception-recommender/api/v1alpha1"
	polexv1alpha1 "github.com/giantswarm/kyverno-policy-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	Name string
}

func NewPolicyExceptionCR(polexdraft polexdraftv1alpha1.PolicyExceptionDraft) *polexv1alpha1.PolicyException {
	polexCR := &polexv1alpha1.PolicyException{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PolicyException",
			APIVersion: "policy.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      polexdraft.GetName(),
			Namespace: polexdraft.GetNamespace(),
		},
		Spec: policyExceptionSpecDeepCopy(polexdraft.Spec),
	}

	return polexCR
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
