package organization

import (
	"context"

	v1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Service) getSelfSubjectAccessReview(ctx context.Context) (*v1.SelfSubjectAccessReview, error) {
	request := &v1.SelfSubjectAccessReview{
		Spec: v1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &v1.ResourceAttributes{
				Verb:     "list",
				Group:    "security.giantswarm.io",
				Resource: "organizations",
			},
		},
	}

	createOptions := metav1.CreateOptions{}

	return s.client.K8sClient().AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, request, createOptions)
}

func (s *Service) getSelfSubjectRulesReview(ctx context.Context) (*v1.SelfSubjectRulesReview, error) {
	request := &v1.SelfSubjectRulesReview{
		Spec: v1.SelfSubjectRulesReviewSpec{
			Namespace: "default",
		},
	}

	createOptions := metav1.CreateOptions{}

	return s.client.K8sClient().AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, request, createOptions)
}
