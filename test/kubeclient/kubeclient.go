package kubeclient

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8scrdclient"
	v1 "k8s.io/api/authorization/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/kubectl-gs/v2/pkg/scheme"
)

type fakeK8sClient struct {
	ctrlClient client.Client
	k8sClient  *fakek8s.Clientset
}

type Interface interface {
	k8sclient.Interface
	AddSubjectAccessResolver(accessResolver authAccessResolver)
	AddSubjectAccess(accessAllowed bool)
	AddSubjectResourceRules(namespace string, rules []v1.ResourceRule)
}

type authAccessResolver func(action testing.Action) bool

func FakeK8sClient(objects ...runtime.Object) Interface {
	var k8sClient *fakeK8sClient
	{
		scheme, err := scheme.NewScheme()

		if err != nil {
			panic(err)
		}

		clientSet := fakek8s.NewSimpleClientset()

		k8sClient = &fakeK8sClient{
			ctrlClient: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build(),
			k8sClient:  clientSet,
		}
	}

	return k8sClient
}

func (f *fakeK8sClient) CRDClient() k8scrdclient.Interface {
	return nil
}

func (f *fakeK8sClient) CtrlClient() client.Client {
	return f.ctrlClient
}

func (f *fakeK8sClient) DynClient() dynamic.Interface {
	return nil
}

func (f *fakeK8sClient) ExtClient() apiextensionsclient.Interface {
	return nil
}

func (f *fakeK8sClient) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *fakeK8sClient) RESTClient() rest.Interface {
	return nil
}

func (f *fakeK8sClient) RESTConfig() *rest.Config {
	return nil
}

func (f *fakeK8sClient) Scheme() *runtime.Scheme {
	return nil
}

func (f *fakeK8sClient) AddSubjectAccess(accessAllowed bool) {
	f.AddSubjectAccessResolver(func(action testing.Action) bool {
		return accessAllowed
	})
}

func (f *fakeK8sClient) AddSubjectAccessResolver(accessResolver authAccessResolver) {
	f.k8sClient.PrependReactor("create", "selfsubjectaccessreviews", newSelfSubjectAccessReviewReaction(accessResolver))
}

func (f *fakeK8sClient) AddSubjectResourceRules(namespace string, rules []v1.ResourceRule) {
	f.k8sClient.PrependReactor("create", "selfsubjectrulesreviews", newSelfSubjectRulesReviewReaction(namespace, rules))
}

func newSelfSubjectAccessReviewReaction(accessResolver authAccessResolver) testing.ReactionFunc {
	return func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		selfSubjectAccessReview := &v1.SelfSubjectAccessReview{
			Spec: v1.SelfSubjectAccessReviewSpec{},
			Status: v1.SubjectAccessReviewStatus{
				Allowed: accessResolver(action),
			},
		}
		return true, selfSubjectAccessReview, nil
	}
}

func newSelfSubjectRulesReviewReaction(namespace string, resourceRules []v1.ResourceRule) testing.ReactionFunc {
	return func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		selfSubjectResourceRules := &v1.SelfSubjectRulesReview{
			Spec: v1.SelfSubjectRulesReviewSpec{
				Namespace: namespace,
			},
			Status: v1.SubjectRulesReviewStatus{
				ResourceRules: resourceRules,
			},
		}
		return true, selfSubjectResourceRules, nil
	}
}
