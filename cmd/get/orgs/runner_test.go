package orgs

import (
	"bytes"
	"github.com/giantswarm/apiextensions/v6/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/pkg/scheme"
	"github.com/giantswarm/kubectl-gs/test/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

func Test_run(t *testing.T) {
	testCases := []struct {
		name               string
		storage            []runtime.Object
		args               []string
		expectedGoldenFile string
		errorMatcher       func(error) bool
	}{
		{
			name: "case0: get orgs for admin user",
			storage: []runtime.Object{
				newOrg("test-1", "org-test-1"),
				newOrg("test-2", "org-test-2"),
				newOrg("test-3", "org-test-3"),
			},
			args: nil,
			expectedGoldenFile: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeKubeConfig := kubeconfig.CreateFakeKubeConfig()
			flag := &flag{
				print: genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault),
			}

			out := new(bytes.Buffer)
			runner := &runner{
				commonConfig: commonconfig.New(genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig)),
				flag: flag,
				service: newOrgService(t),
				stdout: out,
			}

			err := runner.Run(nil, tc.args)
			require.NoError(t, err)
		})
	}
}

func newOrg(name, namespace string) *v1alpha1.Organization {
	return &v1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.giantswarm.io/v1alpha1",
			Kind: "Organization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: v1alpha1.OrganizationSpec{},
		Status: v1alpha1.OrganizationStatus{
			Namespace: namespace,
		},
	}
}

func newOrgService(t *testing.T, object ...runtime.Object) *organization.Service {
	clientScheme, err := scheme.NewScheme()
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	/*
	apiVersion: rbac.authorization.k8s.io/v1
	kind: ClusterRole
	metadata:
	  annotations:
	    kubectl.kubernetes.io/last-applied-configuration: |
	      {"apiVersion":"rbac.authorization.k8s.io/v1","kind":"ClusterRole","metadata":{"annotations":{},"name":"all-orgs"},"rules":[{"apiGroups":["security.giantswarm.io"],"resources":["organizations"],"verbs":["list"]}]}
	  creationTimestamp: "2021-03-08T12:59:04Z"
	  name: all-orgs
	  resourceVersion: "261464181"
	  selfLink: /apis/rbac.authorization.k8s.io/v1/clusterroles/all-orgs
	  uid: c6dc57be-d9b8-4f9c-96d1-1c4a3ae6ab52
	rules:
	- apiGroups:
	  - security.giantswarm.io
	  resources:
	  - organizations
	  verbs:
	  - list
	*/

	listOrgsClusterRole := &v1.ClusterRole{
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{"security.giantswarm.io"},
				Resources: []string{"organizations"},
				Verbs: []string{"list"},
			},
		},
	}

	listOrgsClusterRoleBinding := &v1.ClusterRoleBinding{
		Subjects: []v1.Subject{
			{
				Kind: "User",
				Name: "clean",
				Namespace: "default",
			},
		},
	}

	/*simpleClientSet := k8sfake.NewSimpleClientset(listOrgsClusterRole, listOrgsClusterRoleBinding)

	fakeRbac := fake2.FakeRbacV1{&simpleClientSet.Fake}
	fakeRbac.ClusterRoles().Create()*/

	clients := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
		CtrlClient: fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(object...).Build(),
		K8sClient: k8sfake.NewSimpleClientset(listOrgsClusterRole, listOrgsClusterRoleBinding),
	})

	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	service, err := organization.New(organization.Config{
		Client: clients,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}



	return service.(*organization.Service)

	/*
		clientScheme, err := scheme.NewScheme()
		if err != nil {
			t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
		}

		clients := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
			CtrlClient: fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(object...).Build(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
		}

		service, err := nodepool.New(nodepool.Config{
			Client: clients.CtrlClient(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
		}

		return service
	 */
}

/*
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: "2022-08-11T05:39:25Z"
  labels:
    giantswarm.io/managed-by: rbac-operator
  name: organization-vaclav-v-read
  resourceVersion: "784438506"
  selfLink: /apis/rbac.authorization.k8s.io/v1/clusterroles/organization-vaclav-v-read
  uid: 601aa381-b258-459e-b914-6eda976cd1c4
rules:
- apiGroups:
  - security.giantswarm.io
  resourceNames:
  - vaclav-v
  resources:
  - organizations
  verbs:
  - get
 */