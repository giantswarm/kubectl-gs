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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/v6/internal/label"
	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v6/pkg/data/domain/cluster"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger

	service cluster.Interface
	client  client.Client

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

	// A Cluster API cluster's release lives in the tag of its release chart
	// or in its values: check the edit before scheduling or applying it, so a
	// typo never reaches the chart.
	var update client.Object
	capi := isCapiProvider(resource)
	if capi {
		update, err = r.releaseUpdate(ctx, resource, targetRelease)
		if err != nil {
			return microerror.Mask(err)
		}

		err = ensureApplicable(update, scheduledTime != "")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var patches cluster.PatchOptions
	var msg string

	if scheduledTime != "" {
		patchSpecs := make([]cluster.PatchSpec, 0)
		if resource.Cluster.GetAnnotations() == nil {
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
	} else if capi {
		err = r.client.Update(ctx, update)
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

	_, _ = fmt.Fprintln(r.stdout, msg)
	return nil
}

// releaseUpdate returns the resource that sets the cluster's release, with
// the release version set to target: the OCIRepository of a cluster deployed
// as a HelmRelease of a release-<provider> chart, else the user config. It
// fails when it can't tell the current release version, when target is not
// higher than it, or when the management cluster has no Release for target.
func (r *runner) releaseUpdate(ctx context.Context, resource *cluster.Cluster, target string) (client.Object, error) {
	hr, err := clusterHelmRelease(ctx, r.client, resource.Cluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var update client.Object
	var current string
	if hr != nil {
		repo, repoCurrent, err := releaseChartUpdate(ctx, r.client, hr, target)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if repo != nil {
			update, current = repo, repoCurrent
		}
	}
	if update == nil {
		update, current, err = r.userConfigUpdate(ctx, resource, target)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	err = ensureNewer(current, target)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	provider, err := releaseProvider(resource.Cluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = ensureRelease(ctx, r.client, provider, target)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return update, nil
}

// userConfigUpdate returns the release version of a cluster deployed as an
// App CR, and its user config with global.release.version set to target.
func (r *runner) userConfigUpdate(ctx context.Context, resource *cluster.Cluster, target string) (*corev1.ConfigMap, string, error) {
	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{Name: fmt.Sprintf("%s-userconfig", resource.Cluster.GetName()), Namespace: resource.Cluster.GetNamespace()}
	err := r.client.Get(ctx, key, cm)
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	current, values, err := setReleaseVersion(cm.Data["values"], target)
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	cm.Data["values"] = values
	return cm, current, nil
}

// ensureApplicable fails when the update can't take effect the way it is
// asked for. Nothing carries out a scheduled update of a release chart: the
// schedule annotations were read by upgrade-schedule-operator, archived before
// clusters were deployed as HelmReleases. And an update applied now to values
// that a Flux controller
// applies would be reverted on its next reconciliation.
func ensureApplicable(update client.Object, scheduled bool) error {
	_, values := update.(*corev1.ConfigMap)
	if scheduled && !values {
		return microerror.Maskf(notAllowedError, "The cluster is deployed as a HelmRelease of a release chart, whose update can't be scheduled. Update it at the time you want instead.")
	}
	if !scheduled && values {
		return microerror.Mask(ensureNotManaged(update, "ConfigMap", "global.release.version"))
	}
	return nil
}

func replaceToEscape(from string) string {
	return strings.ReplaceAll(from, "/", "~1")
}

func (r *runner) getService() error {
	if r.service != nil && r.client != nil {
		return nil
	}

	k8sClient, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	r.client = k8sClient.CtrlClient()
	r.service = cluster.New(cluster.Config{
		Client: r.client,
	})

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
