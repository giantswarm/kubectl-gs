package policyexception

import (
	polexv1alpha1 "github.com/giantswarm/kyverno-policy-operator/api/v1alpha1"
	//	polexdraftv1alpha1 "github.com/giantswarm/exception-recommender/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	Name string
}

func NewPolicyExceptionCR(config Config) (*polexv1alpha1.PolicyException, error) {

	polexCR := &polexv1alpha1.PolicyException{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PolicyException",
			APIVersion: "policy.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Name,
		},
		Spec: polexv1alpha1.PolicyExceptionSpec{},
	}

	return polexCR, nil
}
