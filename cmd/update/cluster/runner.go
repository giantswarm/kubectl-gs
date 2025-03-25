package cluster

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/kubectl-gs/v5/internal/label"
	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/cluster"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger

	service cluster.Interface

	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		_ = cmd.Help()
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error
	var scheduledTime string
	var t time.Time

	name := r.flag.Name
	targetRelease := r.flag.ReleaseVersion
	provider := r.flag.Provider

	if r.flag.ScheduledTime != "" {
		{
			layout := "2006-01-02 15:04"
			t, err = time.Parse(layout, r.flag.ScheduledTime)
			if err != nil {
				fmt.Println(err)
				return microerror.Maskf(notAllowedError, "Scheduled time has not the right time format, please use 'YYYY-MM-DD HH:MM' as the time format.")
			}
			scheduledTime = t.Format(time.RFC822)
		}
	}

	{
		err = r.getService()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	namespace, _, err := r.commonConfig.GetNamespace()

	if err != nil {
		return microerror.Mask(err)
	}

	getOptions := cluster.GetOptions{
		Name:      name,
		Namespace: namespace,
		Provider:  provider,
	}

	c, err := r.service.Get(ctx, getOptions)

	if cluster.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "Cluster with name '%s' cannot be found in the '%s' namespace.\n", getOptions.Name, getOptions.Namespace)
	} else if err != nil {
		return microerror.Mask(err)
	}

	resource, ok := c.(*cluster.Cluster)

	if !ok {
		return microerror.Maskf(notFoundError, "Cluster with name '%s' cannot be found in the '%s' namespace.\n", getOptions.Name, getOptions.Namespace)
	}

	var patches cluster.PatchOptions
	var msg string

	if scheduledTime != "" {
		patchSpecs := make([]cluster.PatchSpec, 0)
		if resource.Cluster.Annotations == nil {
			patchSpecs = append(patchSpecs, cluster.PatchSpec{
				Op:    "add",
				Path:  "/metadata/annotations",
				Value: make(map[string]string, 0),
			})
		}
		patchSpecs = append(patchSpecs, cluster.PatchSpec{
			Op:    "add",
			Path:  fmt.Sprintf("/metadata/annotations/%s", replaceToEscape(annotation.UpdateScheduleTargetRelease)),
			Value: targetRelease,
		})
		patchSpecs = append(patchSpecs, cluster.PatchSpec{
			Op:    "add",
			Path:  fmt.Sprintf("/metadata/annotations/%s", replaceToEscape(annotation.UpdateScheduleTargetTime)),
			Value: scheduledTime,
		})

		patches = cluster.PatchOptions{
			PatchSpecs: patchSpecs,
		}
		messageFormat := "An upgrade of cluster %s to release %s has been scheduled for\n\n    %v, (%v)"
		msg = fmt.Sprintf(messageFormat, name, targetRelease, t.Format(time.RFC1123), t.Local().Format(time.RFC1123))
	} else if isCapiProvider(resource) {

		currentVersion := getReleaseVersion(resource)
		if currentVersion == "" {
			return microerror.Maskf(notFoundError, "Release version not found in cluster '%s'", name)
		}

		k8sclient, err := r.commonConfig.GetClient(r.logger)
		if err != nil {
			return microerror.Mask(err)
		}

		cm := &corev1.ConfigMap{}
		err = k8sclient.CtrlClient().Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-userconfig", resource.Cluster.GetName()), Namespace: resource.Cluster.GetNamespace()}, cm)
		if err != nil {
			return microerror.Mask(err)
		}

		// Extract the values field and replace the current version with the target version
		values := cm.Data["values"]
		values = strings.ReplaceAll(values, fmt.Sprintf("version: %s", currentVersion), fmt.Sprintf("version: %s", targetRelease))
		cm.Data["values"] = values

		err = k8sclient.CtrlClient().Update(ctx, cm)
		if err != nil {
			return microerror.Mask(err)
		}
		msg = fmt.Sprintf("Cluster '%s' is updated to release version '%s'\n", name, targetRelease)
	} else {
		patches = cluster.PatchOptions{
			PatchSpecs: []cluster.PatchSpec{
				{
					Op:    "add",
					Path:  fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.ReleaseVersion)),
					Value: targetRelease,
				},
			},
		}
		msg = fmt.Sprintf("Cluster '%s' is updated to release version '%s'\n", name, targetRelease)
	}

	if len(patches.PatchSpecs) > 0 {
		err = r.service.Patch(ctx, resource.ClientObject(), patches)

		if err != nil {
			return microerror.Mask(err)
		}
	}

	fmt.Fprintln(r.stdout, msg)
	return nil
}

func replaceToEscape(from string) string {
	return strings.ReplaceAll(from, "/", "~1")
}

func (r *runner) getService() error {
	if r.service != nil {
		return nil
	}

	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	serviceConfig := cluster.Config{
		Client: client.CtrlClient(),
	}
	r.service = cluster.New(serviceConfig)

	return nil
}

func isCapiProvider(cluster *cluster.Cluster) bool {
	labels := cluster.Cluster.GetLabels()
	name, ok := labels["cluster.x-k8s.io/watch-filter"]
	if !ok {
		return false
	}
	if name == "capi" {
		return true
	}
	return false
}

func getReleaseVersion(cluster *cluster.Cluster) string {
	labels := cluster.Cluster.GetLabels()
	return labels[label.ReleaseVersion]
}
