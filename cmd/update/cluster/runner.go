package cluster

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/label"
	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger

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

	name := r.flag.Name
	targetRelease := r.flag.ReleaseVersion
	scheduledTime := r.flag.ScheduledTime
	provider := r.flag.Provider

	config := commonconfig.New(r.flag.config)
	{
		err = r.getService(config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	namespace, _, err := r.flag.config.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return microerror.Mask(err)
	}

	getOptions := cluster.GetOptions{
		Name:      name,
		Namespace: namespace,
		Provider:  provider,
	}

	resource, err := r.service.Get(ctx, getOptions)
	if cluster.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "Cluster with name '%s' cannot be found in the '%s' namespace.\n", getOptions.Name, getOptions.Namespace)
	} else if err != nil {
		return microerror.Mask(err)
	}

	var patches cluster.PatchOptions
	var msg string
	if scheduledTime != "" {
		patches = cluster.PatchOptions{
			PatchSpecs: []cluster.PatchSpec{
				{
					Op:    "add",
					Path:  fmt.Sprintf("/metadata/annotations/%s", replaceToEscape(annotation.UpdateScheduleTargetRelease)),
					Value: targetRelease,
				},
				{
					Op:    "add",
					Path:  fmt.Sprintf("/metadata/annotations/%s", replaceToEscape(annotation.UpdateScheduleTargetTime)),
					Value: scheduledTime,
				},
			},
		}
		msg = fmt.Sprintf("Cluster '%s' is scheduled on '%v' to update to release version '%s'\n", name, scheduledTime, targetRelease)
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

	err = r.service.Patch(ctx, resource.Object(), patches)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprintln(r.stdout, msg)
	return nil
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}

func (r *runner) getService(config *commonconfig.CommonConfig) error {
	if r.service != nil {
		return nil
	}

	client, err := config.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	serviceConfig := cluster.Config{
		Client: client,
	}
	r.service, err = cluster.New(serviceConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
