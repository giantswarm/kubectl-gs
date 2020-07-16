package cluster

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type CommonClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []runtime.Object `json:"items"`
}

func (in *CommonClusterList) DeepCopyObject() runtime.Object {
	return in
}

type GetOptions struct {
	ID       string
	Provider string
}

type Interface interface {
	Get(context.Context, *GetOptions) (runtime.Object, error)
}
